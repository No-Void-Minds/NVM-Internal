package httpapi

import (
	"fmt"
	"net/http"
	"testing"
)

// Revisi 31 dan 32 lewat HTTP: Staf yang mengirim stop_series ditolak 403, Manajer berhasil dan seri berhenti;
// detail task mengirim stop_series di can, dan PIC task mendapat can_edit untuk sub task buatan orang lain.
func TestDeleteStopSeriesHTTP(t *testing.T) {
	tm := newTaskTeam(t)
	body := fmt.Sprintf(`{"title":"Unggah harian","urgency":"medium","pic_ids":[%d],"label_ids":[],
		"subtasks":[{"title":"Potong video","pic_id":0}],
		"schedule":{"freq":"daily","skip_weekend":false,"weekdays":[],"month_day":0,"end_date":""}}`, tm.picID)
	var created struct {
		ID string `json:"id"`
	}
	if code := tm.sendJSON(t, tm.manager, http.MethodPost, "/api/tasks", body, &created); code != http.StatusCreated {
		t.Fatalf("Manajer membuat task berulang: %d", code)
	}
	id := created.ID

	type detail struct {
		Task struct {
			Version int32 `json:"version"`
			Series  *struct {
				Active bool `json:"active"`
			} `json:"series"`
			Can struct {
				StopSeries bool `json:"stop_series"`
			} `json:"can"`
			Subtasks []struct {
				CanEdit bool `json:"can_edit"`
			} `json:"subtasks"`
		} `json:"task"`
	}
	get := func(c *http.Client) detail {
		var d detail
		if code := tm.sendJSON(t, c, http.MethodGet, "/api/tasks/"+id, "", &d); code != http.StatusOK {
			t.Fatalf("GET task: %d", code)
		}
		return d
	}
	pic := get(tm.pic)
	if pic.Task.Can.StopSeries || !get(tm.manager).Task.Can.StopSeries {
		t.Fatal("can.stop_series harus true hanya untuk Manajer")
	}
	if len(pic.Task.Subtasks) != 1 || !pic.Task.Subtasks[0].CanEdit {
		t.Fatalf("PIC task harus mendapat can_edit untuk sub task buatan Manajer: %+v", pic.Task.Subtasks)
	}
	if o := get(tm.other); len(o.Task.Subtasks) != 1 || o.Task.Subtasks[0].CanEdit {
		t.Fatal("Staf yang bukan PIC tidak boleh mendapat can_edit")
	}

	path := fmt.Sprintf("/api/tasks/%s?version=%d&stop_series=true", id, pic.Task.Version)
	for name, c := range map[string]*http.Client{"PIC task": tm.pic, "Staf lain": tm.other} {
		if code := tm.sendJSON(t, c, http.MethodDelete, path, "", nil); code != http.StatusForbidden {
			t.Fatalf("%s mengirim stop_series: %d, mau 403", name, code)
		}
	}
	if code := tm.sendJSON(t, tm.manager, http.MethodDelete, path, "", nil); code != http.StatusNoContent {
		t.Fatalf("Manajer hapus dan hentikan seri: %d, mau 204", code)
	}
	var active bool
	if err := tm.pool.QueryRow(t.Context(), `SELECT s.active FROM task_series s JOIN tasks t ON t.series_id = s.id WHERE t.id = $1`, id).Scan(&active); err != nil || active {
		t.Fatalf("seri harus berhenti: active=%v err %v", active, err)
	}
}

// Bug produksi revisi 31 (T-0026) per peran lewat HTTP. Seri harian lewati akhir pekan dibuat Staf sebagai peminta,
// PIC orang lain, status Requested versi 1. Tidak boleh ada jawaban 500 untuk peran mana pun.
func TestDeleteStopSeriesPerPeran(t *testing.T) {
	tm := newTaskTeam(t)
	newTask := func(schedule string) (string, int32) {
		t.Helper()
		body := fmt.Sprintf(`{"title":"TEslagi","urgency":"medium","pic_ids":[%d],"label_ids":[],"subtasks":[]%s}`, tm.picID, schedule)
		var out struct {
			ID string `json:"id"`
		}
		if code := tm.sendJSON(t, tm.other, http.MethodPost, "/api/tasks", body, &out); code != http.StatusCreated {
			t.Fatalf("Staf membuat task: %d", code)
		}
		return out.ID, tm.get(t, tm.other, out.ID).Version
	}
	daily := `,"schedule":{"freq":"daily","skip_weekend":true,"weekdays":[],"month_day":0,"end_date":""}`
	del := func(c *http.Client, id string, v int32, stop bool) int {
		t.Helper()
		path := fmt.Sprintf("/api/tasks/%s?version=%d", id, v)
		if stop {
			path += "&stop_series=true"
		}
		return tm.sendJSON(t, c, http.MethodDelete, path, "", nil)
	}

	// Admin tidak boleh menghapus task sama sekali; peminta (Staf) boleh menghapus tetapi tidak menghentikan seri.
	id, v := newTask(daily)
	for name, c := range map[string]*http.Client{"Admin": tm.admin, "Staf peminta": tm.other} {
		if code := del(c, id, v, true); code != http.StatusForbidden {
			t.Fatalf("%s dengan stop_series: %d, mau 403", name, code)
		}
	}
	if code := del(tm.manager, id, v, true); code != http.StatusNoContent {
		t.Fatalf("Manajer dengan stop_series: %d, mau 204", code)
	}
	// Task yang sudah terhapus: 404, bukan 500.
	if code := del(tm.manager, id, v, true); code != http.StatusNotFound {
		t.Fatalf("hapus ulang task terhapus: %d, mau 404", code)
	}

	// Hapus biasa oleh peminta: seri tetap aktif.
	id, v = newTask(daily)
	if code := del(tm.other, id, v, false); code != http.StatusNoContent {
		t.Fatalf("Staf peminta hapus biasa: %d, mau 204", code)
	}
	var active bool
	if err := tm.pool.QueryRow(t.Context(), `SELECT s.active FROM task_series s JOIN tasks t ON t.series_id = s.id WHERE t.id = $1`, id).Scan(&active); err != nil || !active {
		t.Fatalf("hapus biasa tidak boleh menghentikan seri: active=%v err %v", active, err)
	}

	// Task bukan seri: stop_series ditolak 400 dan task tetap ada; hapus biasa berhasil.
	id, v = newTask("")
	if code := del(tm.manager, id, v, true); code != http.StatusBadRequest {
		t.Fatalf("stop_series untuk task biasa: %d, mau 400", code)
	}
	if code := del(tm.manager, id, v, false); code != http.StatusNoContent {
		t.Fatalf("hapus task biasa: %d, mau 204", code)
	}
}
