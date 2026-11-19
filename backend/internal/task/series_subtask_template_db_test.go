package task

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func readTemplate(t *testing.T, f fixture, sid int64) SeriesTemplate {
	t.Helper()
	var raw []byte
	if err := f.pool.QueryRow(context.Background(), `SELECT template FROM task_series WHERE id = $1`, sid).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var tm SeriesTemplate
	if err := json.Unmarshal(raw, &tm); err != nil {
		t.Fatal(err)
	}
	return tm
}

func templateTitles(t *testing.T, f fixture, sid int64) []string {
	t.Helper()
	return subtaskTitles(readTemplate(t, f, sid).Subtasks)
}

// Revisi 29: tambah, ubah, hapus, dan urutkan sub task di kemunculan terbaru seri aktif ikut memperbarui templat,
// hanya bagian rencananya, dengan jarak deadline terhadap occurrence_date.
func TestSubTaskKemunculanTerbaruMasukTemplat(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	// Senin 23 November, deadline Rabu 25.
	id, sid := newSeriesTask(t, f, "2026-11-23", CreateInput{
		Title: "Posting mingguan", Urgency: "medium", PICIDs: []int64{f.stafA.ID},
		StartDate: "2026-11-23", DueDate: "2026-11-25",
		Schedule: &ScheduleInput{Freq: FreqWeekly, Weekdays: []int{1}},
	})

	a, err := f.svc.AddSubtask(ctx, f.stafA, id, SubtaskFields{
		Title: "Tulis caption", Description: "Dua versi", Reference: "https://contoh.test/acuan",
		PICID: f.stafA.ID, DueDate: "2026-11-24",
	})
	if err != nil {
		t.Fatal(err)
	}
	tm := readTemplate(t, f, sid)
	if len(tm.Subtasks) != 1 {
		t.Fatalf("tambah: templat %+v, mau satu sub task", tm.Subtasks)
	}
	st := tm.Subtasks[0]
	if st.Title != "Tulis caption" || st.Description != "Dua versi" || st.Reference != "https://contoh.test/acuan" ||
		st.PICID != f.stafA.ID || st.DueOffsetDays == nil || *st.DueOffsetDays != 1 {
		t.Fatalf("tambah: sub task templat %+v, mau rencana lengkap dengan jarak deadline 1 hari", st)
	}

	// Pelakunya Staf yang menambah, dan tercatat sebagai pembaruan seri.
	var actor int64
	var source string
	if err := f.pool.QueryRow(ctx, `SELECT user_id, new_value->>'source' FROM audit_logs
		WHERE action = 'series_updated' AND entity_id = $1 ORDER BY id DESC LIMIT 1`, id).Scan(&actor, &source); err != nil {
		t.Fatal(err)
	}
	if actor != f.stafA.ID || source != "subtask" {
		t.Fatalf("audit series_updated aktor %d sumber %q, mau staf.a dan subtask", actor, source)
	}

	// Hasil pengerjaan (tautan hasil, catatan, centang) tidak masuk templat dan tidak menambah catatan audit.
	count := func() int {
		t.Helper()
		var n int
		if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE action = 'series_updated' AND entity_id = $1`, id).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := count()
	note := "Sudah dikirim"
	if err := f.svc.UpdateSubtask(ctx, f.stafA, id, a, SubtaskFields{
		Title: "Tulis caption", Description: "Dua versi", Reference: "https://contoh.test/acuan",
		Deliverable: "https://contoh.test/hasil", PICID: f.stafA.ID, DueDate: "2026-11-24", ResultNote: &note,
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.SetSubtaskDone(ctx, f.stafA, id, a, true); err != nil {
		t.Fatal(err)
	}
	if count() != before {
		t.Fatal("hasil pengerjaan sub task tidak boleh mencatat pembaruan templat")
	}
	raw, _ := json.Marshal(readTemplate(t, f, sid).Subtasks)
	for _, bocor := range []string{"hasil", "Sudah dikirim", "done"} {
		if strings.Contains(string(raw), bocor) {
			t.Fatalf("templat memuat hasil pengerjaan %q: %s", bocor, raw)
		}
	}
	if err := f.svc.SetSubtaskDone(ctx, f.stafA, id, a, false); err != nil {
		t.Fatal(err)
	}

	// Ubah.
	if err := f.svc.UpdateSubtask(ctx, f.stafA, id, a, SubtaskFields{
		Title: "Tulis caption final", PICID: f.stafA.ID, DueDate: "2026-11-25",
	}); err != nil {
		t.Fatal(err)
	}
	tm = readTemplate(t, f, sid)
	if tm.Subtasks[0].Title != "Tulis caption final" || *tm.Subtasks[0].DueOffsetDays != 2 {
		t.Fatalf("ubah: sub task templat %+v", tm.Subtasks[0])
	}

	// Urutkan lewat form Ubah tanpa centang "Pakai perubahan ini": judul task di templat tetap.
	b, err := f.svc.AddSubtask(ctx, f.manager, id, SubtaskFields{Title: "Desain"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Update(ctx, f.manager, id, UpdateInput{
		Version: f.version(t, id), Title: "Posting mingguan (khusus)", Urgency: "medium", PICIDs: []int64{f.stafA.ID},
		StartDate: "2026-11-23", DueDate: "2026-11-25",
		Subtasks: []SubtaskInput{{ID: b, Title: "Desain"}, {ID: a, Title: "Tulis caption final"}},
	}); err != nil {
		t.Fatal(err)
	}
	if got := templateTitles(t, f, sid); !slices.Equal(got, []string{"Desain", "Tulis caption final"}) {
		t.Fatalf("urutkan: templat %v", got)
	}
	if got := readTemplate(t, f, sid).Title; got != "Posting mingguan" {
		t.Fatalf("judul task induk di templat berubah jadi %q tanpa centang", got)
	}

	// Hapus.
	if err := f.svc.DeleteSubtask(ctx, f.manager, id, b); err != nil {
		t.Fatal(err)
	}
	if got := templateTitles(t, f, sid); !slices.Equal(got, []string{"Tulis caption final"}) {
		t.Fatalf("hapus: templat %v", got)
	}

	d, err := f.svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !d.LatestOccurrence || !d.TemplateMatchesTask() {
		t.Fatalf("detail: terbaru %v, sama dengan templat %v", d.LatestOccurrence, d.TemplateMatchesTask())
	}
}

// Revisi 29: kemunculan lama tidak memperbarui templat; kemunculan terbaru memakai jarak terhadap tanggalnya sendiri.
func TestSubTaskKemunculanLamaTidakMasukTemplat(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	old, sid := newSeriesTask(t, f, "2026-11-23", CreateInput{
		Title: "Posting mingguan", Urgency: "medium", PICIDs: []int64{f.stafA.ID},
		Schedule: &ScheduleInput{Freq: FreqWeekly, Weekdays: []int{1}},
	})
	created, err := f.svc.CreateDueOccurrences(ctx, at("2026-11-30"))
	if err != nil || len(created) != 1 {
		t.Fatalf("kemunculan 30 November %v err %v", created, err)
	}
	latest := created[0]

	if _, err := f.svc.AddSubtask(ctx, f.manager, old, SubtaskFields{Title: "Susulan minggu lalu"}); err != nil {
		t.Fatal(err)
	}
	if got := templateTitles(t, f, sid); len(got) != 0 {
		t.Fatalf("kemunculan lama mengubah templat: %v", got)
	}
	d, err := f.svc.Get(ctx, old)
	if err != nil {
		t.Fatal(err)
	}
	if d.LatestOccurrence || d.TemplateMatchesTask() {
		t.Fatalf("kemunculan lama: terbaru %v, sama dengan templat %v", d.LatestOccurrence, d.TemplateMatchesTask())
	}

	if _, err := f.svc.AddSubtask(ctx, f.manager, latest, SubtaskFields{Title: "Rekap", DueDate: "2026-12-02"}); err != nil {
		t.Fatal(err)
	}
	tm := readTemplate(t, f, sid)
	if len(tm.Subtasks) != 1 || tm.Subtasks[0].DueOffsetDays == nil || *tm.Subtasks[0].DueOffsetDays != 2 {
		t.Fatalf("kemunculan terbaru: templat %+v, mau Rekap dengan jarak deadline 2 hari dari 30 November", tm.Subtasks)
	}
}

// Revisi 29: seri yang dihentikan tidak memperbarui templat.
func TestSubTaskSeriBerhentiTidakMasukTemplat(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	id, sid := newSeriesTask(t, f, "2026-11-23", CreateInput{
		Title: "Posting mingguan", Urgency: "medium", PICIDs: []int64{f.stafA.ID},
		Schedule: &ScheduleInput{Freq: FreqWeekly, Weekdays: []int{1}},
	})
	if err := f.svc.StopSeries(ctx, f.manager, id); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.AddSubtask(ctx, f.manager, id, SubtaskFields{Title: "Setelah berhenti"}); err != nil {
		t.Fatal(err)
	}
	if got := templateTitles(t, f, sid); len(got) != 0 {
		t.Fatalf("seri berhenti mengubah templat: %v", got)
	}
}
