package task

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// at mengembalikan jam palsu: pukul 09.00 WIB di tanggal kantor itu.
func at(date string) time.Time {
	d, err := time.ParseInLocation("2006-01-02", date, officeTZ)
	if err != nil {
		panic(err)
	}
	return d.Add(9 * time.Hour)
}

func seriesDates(t *testing.T, f fixture, seriesID int64) []string {
	t.Helper()
	rows, err := f.pool.Query(context.Background(),
		`SELECT occurrence_date::text FROM tasks WHERE series_id = $1 ORDER BY occurrence_date`, seriesID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			t.Fatal(err)
		}
		out = append(out, d)
	}
	return out
}

// newSeriesTask membuat task berulang lewat Create, dengan "hari ini" palsu.
func newSeriesTask(t *testing.T, f fixture, today string, in CreateInput) (string, int64) {
	t.Helper()
	f.svc.now = func() time.Time { return at(today) }
	id, err := f.svc.Create(context.Background(), f.manager, in)
	if err != nil {
		t.Fatal(err)
	}
	d, err := f.svc.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Task.SeriesID.Valid || d.Series == nil {
		t.Fatalf("task %s tidak tercatat sebagai kemunculan seri", id)
	}
	return id, d.Task.SeriesID.Int64
}

// Revisi 19: ticker membuat tepat satu kemunculan per hari, tidak menyusulkan hari terlewat, menyalin sub task
// tanpa centang dengan jarak tenggat yang sama, dan berhenti setelah seri dihentikan.
func TestSeriesOccurrences(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	first, sid := newSeriesTask(t, f, "2026-11-20", CreateInput{
		Title: "Laporan harian", Urgency: "medium", PICIDs: []int64{f.stafA.ID}, DueDate: "2026-11-22",
		Subtasks: []SubtaskFields{{
			Title: "Rekap angka", PICID: f.stafC.ID, DueDate: "2026-11-21", Reference: "https://contoh.id/templat",
		}},
		Schedule: &ScheduleInput{Freq: FreqDaily},
	})
	if got := seriesDates(t, f, sid); len(got) != 1 || got[0] != "2026-11-20" {
		t.Fatalf("kemunculan pertama harus tanggal hari ini 2026-11-20, dapat %v", got)
	}

	// Sub task kemunculan pertama dicentang; kemunculan berikutnya tetap tanpa centang.
	d, _ := f.svc.Get(ctx, first)
	if _, err := f.svc.Move(ctx, f.stafA, first, d.Task.Version, OnProgress, ""); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.SetSubtaskDone(ctx, f.stafC, first, d.Subtasks[0].ID, true); err != nil {
		t.Fatal(err)
	}

	for i := range 2 {
		created, err := f.svc.CreateDueOccurrences(ctx, at("2026-11-21"))
		if err != nil {
			t.Fatal(err)
		}
		if want := 1 - i; len(created) != want {
			t.Fatalf("panggilan ke-%d: %d kemunculan dibuat, mau %d", i+1, len(created), want)
		}
	}
	if got := seriesDates(t, f, sid); len(got) != 2 || got[1] != "2026-11-21" {
		t.Fatalf("setelah dua panggilan di 21 November: %v", got)
	}

	var next string
	if err := f.pool.QueryRow(ctx, `SELECT id FROM tasks WHERE series_id = $1 AND occurrence_date = '2026-11-21'`, sid).Scan(&next); err != nil {
		t.Fatal(err)
	}
	n, err := f.svc.Get(ctx, next)
	if err != nil {
		t.Fatal(err)
	}
	if n.Task.Title != "Laporan harian" || n.Task.Status != "requested" ||
		dateValue(n.Task.StartDate) != "2026-11-21" || dateValue(n.Task.DueDate) != "2026-11-23" {
		t.Fatalf("isi kemunculan: judul %q status %s mulai %v tenggat %v", n.Task.Title, n.Task.Status,
			dateValue(n.Task.StartDate), dateValue(n.Task.DueDate))
	}
	if len(n.Assignees) != 1 || n.Assignees[0].ID != f.stafA.ID {
		t.Fatalf("PIC task tidak tersalin: %+v", n.Assignees)
	}
	if len(n.Subtasks) != 1 {
		t.Fatalf("sub task tidak tersalin: %d", len(n.Subtasks))
	}
	st := n.Subtasks[0]
	if st.DoneAt.Valid || dateValue(st.DueDate) != "2026-11-22" || st.ReferenceURL != "https://contoh.id/templat" ||
		len(st.Assignees) != 1 || st.Assignees[0].ID != f.stafC.ID {
		t.Fatalf("sub task kemunculan: centang %v tenggat %v acuan %q PIC %+v",
			st.DoneAt.Valid, dateValue(st.DueDate), st.ReferenceURL, st.Assignees)
	}
	var actor *int64
	if err := f.pool.QueryRow(ctx, `SELECT user_id FROM audit_logs WHERE action = 'occurrence_created' AND entity_id = $1`, next).Scan(&actor); err != nil || actor != nil {
		t.Fatalf("audit occurrence_created harus beraktor Sistem (NULL), dapat %v err %v", actor, err)
	}

	// Tidak menyusulkan 22 dan 23: hanya hari ini (24).
	if _, err := f.svc.CreateDueOccurrences(ctx, at("2026-11-24")); err != nil {
		t.Fatal(err)
	}
	if got := seriesDates(t, f, sid); len(got) != 3 || got[2] != "2026-11-24" {
		t.Fatalf("hari terlewat tidak boleh disusulkan, dapat %v", got)
	}

	if err := f.svc.StopSeries(ctx, f.manager, next); err != nil {
		t.Fatal(err)
	}
	if created, err := f.svc.CreateDueOccurrences(ctx, at("2026-11-25")); err != nil || len(created) != 0 {
		t.Fatalf("seri dihentikan tetap membuat %v (err %v)", created, err)
	}
	if err := f.svc.StopSeries(ctx, f.stafA, first); err == nil {
		t.Fatal("Staf menghentikan seri harus ditolak")
	}
}

// end_date: setelah lewat, seri otomatis tidak aktif.
func TestSeriesEndDate(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	_, sid := newSeriesTask(t, f, "2026-11-20", CreateInput{
		Title: "Promo akhir pekan", Urgency: "low", PICIDs: []int64{f.stafA.ID},
		Schedule: &ScheduleInput{Freq: FreqDaily, EndDate: "2026-11-21"},
	})
	if created, _ := f.svc.CreateDueOccurrences(ctx, at("2026-11-22")); len(created) != 0 {
		t.Fatalf("lewat end_date tetap membuat %v", created)
	}
	var active bool
	if err := f.pool.QueryRow(ctx, `SELECT active FROM task_series WHERE id = $1`, sid).Scan(&active); err != nil || active {
		t.Fatalf("seri lewat end_date harus nonaktif, active=%v err %v", active, err)
	}
	if got := seriesDates(t, f, sid); len(got) != 1 {
		t.Fatalf("21 November terlewat dan tidak disusulkan, dapat %v", got)
	}
}

// Ubah task dengan centang memperbarui templat; tanpa centang tidak.
func TestSeriesUpdateTemplate(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	id, sid := newSeriesTask(t, f, "2026-11-23", CreateInput{
		Title: "Posting Senin", Urgency: "medium", PICIDs: []int64{f.stafA.ID},
		Schedule: &ScheduleInput{Freq: FreqWeekly, Weekdays: []int{1}},
	})
	title := func() string {
		t.Helper()
		var raw []byte
		if err := f.pool.QueryRow(ctx, `SELECT template FROM task_series WHERE id = $1`, sid).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var tm SeriesTemplate
		if err := json.Unmarshal(raw, &tm); err != nil {
			t.Fatal(err)
		}
		return tm.Title
	}
	update := func(newTitle string, apply bool) {
		t.Helper()
		d, err := f.svc.Get(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.svc.Update(ctx, f.manager, id, UpdateInput{
			Version: d.Task.Version, Title: newTitle, Urgency: "medium", PICIDs: []int64{f.stafA.ID},
			Subtasks: []SubtaskInput{},
			Schedule: &ScheduleInput{Freq: FreqWeekly, Weekdays: []int{1, 4}, ApplyToSeries: apply},
		}); err != nil {
			t.Fatal(err)
		}
	}

	update("Posting Senin (khusus minggu ini)", false)
	if got := title(); got != "Posting Senin" {
		t.Fatalf("tanpa centang templat berubah jadi %q", got)
	}
	update("Posting Senin dan Kamis", true)
	if got := title(); got != "Posting Senin dan Kamis" {
		t.Fatalf("dengan centang templat %q, mau judul baru", got)
	}
	var weekdays []int16
	if err := f.pool.QueryRow(ctx, `SELECT weekdays FROM task_series WHERE id = $1`, sid).Scan(&weekdays); err != nil ||
		len(weekdays) != 2 {
		t.Fatalf("aturan seri harus ikut diperbarui jadi Senin dan Kamis, dapat %v err %v", weekdays, err)
	}
}
