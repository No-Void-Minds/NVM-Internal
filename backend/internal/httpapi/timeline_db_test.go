package httpapi

import (
	"fmt"
	"net/http"
	"testing"
)

type timelineEventView struct {
	ID           int64   `json:"id"`
	Category     string  `json:"category"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	EndDate      *string `json:"end_date"`
	AbsenceType  *string `json:"absence_type"`
	AbsenceLabel string  `json:"absence_label"`
	User         *struct {
		ID int64 `json:"id"`
	} `json:"user"`
}

type timelineMonth struct {
	Events   []timelineEventView `json:"events"`
	CanWrite bool                `json:"can_write"`
}

func (tm taskTeam) month(t *testing.T, c *http.Client, month string) timelineMonth {
	t.Helper()
	var out timelineMonth
	if code := tm.sendJSON(t, c, http.MethodGet, "/api/timeline/events?month="+month, "", &out); code != http.StatusOK {
		t.Fatalf("GET bulan %s: %d", month, code)
	}
	return out
}

func (tm taskTeam) createEvent(t *testing.T, body string) int64 {
	t.Helper()
	var out struct {
		ID int64 `json:"id"`
	}
	if code := tm.sendJSON(t, tm.manager, http.MethodPost, "/api/timeline/events", body, &out); code != http.StatusCreated {
		t.Fatalf("Manajer membuat event %s: %d", body, code)
	}
	return out.ID
}

func findEvent(m timelineMonth, id int64) *timelineEventView {
	for i := range m.Events {
		if m.Events[i].ID == id {
			return &m.Events[i]
		}
	}
	return nil
}

// Revisi 30: hanya Manajer yang menulis Timeline. Staf dan Admin ditolak 403 di semua rute tulis, tetapi tetap
// bisa membaca.
func TestTimelineHanyaManajerMenulis(t *testing.T) {
	tm := newTaskTeam(t)
	id := tm.createEvent(t, `{"category":"launching","title":"Rilis katalog","start_date":"2026-11-10"}`)
	var note struct {
		Note struct {
			ID int64 `json:"id"`
		} `json:"note"`
	}
	if code := tm.sendJSON(t, tm.manager, http.MethodPost, "/api/timeline/notes", `{"title":"Libur kantor","body":"25 Desember"}`, &note); code != http.StatusCreated {
		t.Fatalf("Manajer membuat catatan: %d", code)
	}
	nid := note.Note.ID

	writes := []struct{ method, path, body string }{
		{http.MethodPost, "/api/timeline/events", `{"category":"produksi","title":"Syuting","start_date":"2026-11-11"}`},
		{http.MethodPut, fmt.Sprintf("/api/timeline/events/%d", id), `{"category":"launching","title":"Ganti","start_date":"2026-11-10"}`},
		{http.MethodDelete, fmt.Sprintf("/api/timeline/events/%d", id), ""},
		{http.MethodPost, "/api/timeline/notes", `{"title":"Catatan"}`},
		{http.MethodPut, fmt.Sprintf("/api/timeline/notes/%d", nid), `{"title":"Ganti"}`},
		{http.MethodPut, "/api/timeline/notes/order", fmt.Sprintf(`{"ids":[%d]}`, nid)},
		{http.MethodDelete, fmt.Sprintf("/api/timeline/notes/%d", nid), ""},
	}
	for _, who := range []struct {
		name string
		c    *http.Client
	}{{"Staf", tm.other}, {"Admin", tm.admin}} {
		for _, w := range writes {
			if code := tm.sendJSON(t, who.c, w.method, w.path, w.body, nil); code != http.StatusForbidden {
				t.Errorf("%s %s %s: %d, mau 403", who.name, w.method, w.path, code)
			}
		}
		m := tm.month(t, who.c, "2026-11")
		if m.CanWrite || findEvent(m, id) == nil {
			t.Errorf("%s: can_write %v, event terbaca %v", who.name, m.CanWrite, findEvent(m, id) != nil)
		}
	}
	if !tm.month(t, tm.manager, "2026-11").CanWrite {
		t.Error("Manajer harus mendapat can_write true")
	}
}

// Revisi 30: rentang tanggal. Sampai tidak boleh sebelum mulai, paling panjang 62 hari, dan Sampai yang sama
// dengan mulai disimpan sebagai satu hari.
func TestTimelineRentangTanggal(t *testing.T) {
	tm := newTaskTeam(t)
	bad := []string{
		`{"category":"produksi","title":"Mundur","start_date":"2026-11-10","end_date":"2026-11-09"}`,
		`{"category":"produksi","title":"Kepanjangan","start_date":"2026-11-01","end_date":"2027-01-02"}`,
		`{"category":"produksi","title":"","start_date":"2026-11-01"}`,
		`{"category":"produksi","title":"Tanpa tanggal","start_date":""}`,
		`{"category":"rapat","title":"Jenis salah","start_date":"2026-11-01"}`,
		`{"category":"kehadiran","start_date":"2026-11-01","absence_type":"sakit"}`,
		fmt.Sprintf(`{"category":"kehadiran","start_date":"2026-11-01","user_id":%d,"absence_type":"liburan"}`, tm.picID),
	}
	for _, b := range bad {
		if code := tm.sendJSON(t, tm.manager, http.MethodPost, "/api/timeline/events", b, nil); code != http.StatusBadRequest {
			t.Errorf("%s: %d, mau 400", b, code)
		}
	}
	// 1 November sampai 1 Januari: 62 hari termasuk keduanya, tepat di batas.
	edge := tm.createEvent(t, `{"category":"produksi","title":"Produksi panjang","start_date":"2026-11-01","end_date":"2027-01-01"}`)
	one := tm.createEvent(t, `{"category":"produksi","title":"Sehari","start_date":"2026-11-05","end_date":"2026-11-05"}`)
	m := tm.month(t, tm.manager, "2026-11")
	if e := findEvent(m, edge); e == nil || e.EndDate == nil || *e.EndDate != "2027-01-01" {
		t.Fatalf("rentang 62 hari: %+v", e)
	}
	if e := findEvent(m, one); e == nil || e.EndDate != nil {
		t.Fatalf("Sampai sama dengan mulai harus jadi satu hari: %+v", e)
	}
}

// Revisi 30: bulan menangkap event rentang yang melintasi batas bulan, di kedua bulannya, dan event yang dihapus
// tidak muncul lagi.
func TestTimelineBulanMenangkapRentang(t *testing.T) {
	tm := newTaskTeam(t)
	cross := tm.createEvent(t, `{"category":"launching","title":"Kampanye akhir tahun","start_date":"2026-11-28","end_date":"2026-12-03"}`)
	nov := tm.createEvent(t, `{"category":"launching","title":"Hanya November","start_date":"2026-11-02"}`)
	for _, c := range []struct {
		month       string
		cross, only bool
	}{{"2026-10", false, false}, {"2026-11", true, true}, {"2026-12", true, false}, {"2027-01", false, false}} {
		m := tm.month(t, tm.other, c.month)
		if (findEvent(m, cross) != nil) != c.cross || (findEvent(m, nov) != nil) != c.only {
			t.Errorf("bulan %s: rentang %v (mau %v), November %v (mau %v)",
				c.month, findEvent(m, cross) != nil, c.cross, findEvent(m, nov) != nil, c.only)
		}
	}
	if code := tm.sendJSON(t, tm.manager, http.MethodDelete, fmt.Sprintf("/api/timeline/events/%d", cross), "", nil); code != http.StatusNoContent {
		t.Fatalf("hapus: %d", code)
	}
	if findEvent(tm.month(t, tm.manager, "2026-12"), cross) != nil {
		t.Fatal("event yang dihapus masih muncul")
	}
	if code := tm.sendJSON(t, tm.manager, http.MethodDelete, fmt.Sprintf("/api/timeline/events/%d", cross), "", nil); code != http.StatusNotFound {
		t.Fatalf("hapus kedua kali: %d, mau 404", code)
	}
}

// Revisi 30: alasan kehadiran yang pribadi hanya sampai ke Manajer dan orangnya sendiri. Staf lain dan Admin
// menerima "Tidak hadir" tanpa jenis dan keterangan. Tukar shift tampil apa adanya untuk semua.
func TestTimelinePrivasiKehadiran(t *testing.T) {
	tm := newTaskTeam(t)
	sick := tm.createEvent(t, fmt.Sprintf(`{"category":"kehadiran","start_date":"2026-11-12","user_id":%d,"absence_type":"sakit","description":"Demam, surat dokter menyusul"}`, tm.picID))
	swap := tm.createEvent(t, fmt.Sprintf(`{"category":"kehadiran","start_date":"2026-11-13","user_id":%d,"absence_type":"tukar_shift","description":"Masuk Sabtu"}`, tm.picID))

	for _, who := range []struct {
		name string
		c    *http.Client
		full bool
	}{{"Manajer", tm.manager, true}, {"orangnya sendiri", tm.pic, true}, {"Staf lain", tm.other, false}, {"Admin", tm.admin, false}} {
		m := tm.month(t, who.c, "2026-11")
		e := findEvent(m, sick)
		if e == nil || e.User == nil || e.User.ID != tm.picID {
			t.Fatalf("%s: kehadiran tidak terbaca: %+v", who.name, e)
		}
		if who.full {
			if e.AbsenceType == nil || *e.AbsenceType != "sakit" || e.AbsenceLabel != "Sakit" || e.Description == "" {
				t.Errorf("%s harus menerima alasan lengkap: %+v", who.name, e)
			}
		} else if e.AbsenceType != nil || e.AbsenceLabel != "Tidak hadir" || e.Description != "" {
			t.Errorf("%s tidak boleh menerima alasan: %+v", who.name, e)
		}
		s := findEvent(m, swap)
		if s == nil || s.AbsenceType == nil || *s.AbsenceType != "tukar_shift" || s.Description != "Masuk Sabtu" {
			t.Errorf("%s: tukar shift harus tampil apa adanya: %+v", who.name, s)
		}
	}
}
