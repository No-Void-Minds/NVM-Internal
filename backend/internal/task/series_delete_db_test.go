package task

import (
	"context"
	"errors"
	"slices"
	"testing"

	"nvm-internal/internal/testdb"
)

func seriesActive(t *testing.T, f fixture, sid int64) bool {
	t.Helper()
	var active bool
	if err := f.pool.QueryRow(context.Background(), `SELECT active FROM task_series WHERE id = $1`, sid).Scan(&active); err != nil {
		t.Fatal(err)
	}
	return active
}

// Revisi 31: hapus dengan stop_series menghentikan seri dalam transaksi yang sama, sehingga ticker tidak membuat
// apa pun; hapus biasa tetap membiarkan seri berjalan; Staf tidak boleh memakai stop_series.
func TestDeleteAndStopSeries(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	daily := CreateInput{Title: "Unggah harian", Urgency: "medium", PICIDs: []int64{f.stafA.ID}, Schedule: &ScheduleInput{Freq: FreqDaily}}

	// Hapus biasa: seri tetap aktif dan besok tetap ada kemunculan baru.
	keep, keepSID := newSeriesTask(t, f, "2026-11-20", daily)
	if err := f.svc.Delete(ctx, f.manager, keep, 1); err != nil {
		t.Fatal(err)
	}
	if !seriesActive(t, f, keepSID) {
		t.Fatal("hapus tanpa stop_series tidak boleh menghentikan seri")
	}
	if created, err := f.svc.CreateDueOccurrences(ctx, at("2026-11-21")); err != nil || len(created) != 1 {
		t.Fatalf("hapus biasa: kemunculan berikutnya %v (err %v), mau satu", created, err)
	}

	stop, stopSID := newSeriesTask(t, f, "2026-11-21", daily)
	// Staf ditolak sebelum apa pun berubah, walau dia PIC task.
	if err := f.svc.DeleteAndStopSeries(ctx, f.stafA, stop, 1); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Staf memakai stop_series: err %v, mau ErrForbidden", err)
	}
	if err := f.svc.DeleteAndStopSeries(ctx, f.manager, stop, 1); err != nil {
		t.Fatal(err)
	}
	if seriesActive(t, f, stopSID) {
		t.Fatal("hapus dengan stop_series harus menghentikan seri")
	}
	var deleted bool
	if err := f.pool.QueryRow(ctx, `SELECT is_deleted FROM tasks WHERE id = $1`, stop).Scan(&deleted); err != nil || !deleted {
		t.Fatalf("task harus terhapus, is_deleted=%v err %v", deleted, err)
	}
	var n int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE entity_id = $1 AND action IN ('task_deleted', 'series_stopped')`, stop).Scan(&n); err != nil || n != 2 {
		t.Fatalf("audit hapus dan henti seri: %d baris (err %v), mau 2", n, err)
	}
	if got := seriesDates(t, f, stopSID); len(got) != 1 {
		t.Fatalf("seri yang dihentikan tidak boleh punya kemunculan baru: %v", got)
	}
	if _, err := f.svc.CreateDueOccurrences(ctx, at("2026-11-22")); err != nil {
		t.Fatal(err)
	}
	if got := seriesDates(t, f, stopSID); len(got) != 1 {
		t.Fatalf("ticker tetap membuat kemunculan untuk seri yang dihentikan: %v", got)
	}

	// Task biasa: stop_series ditolak dan task tidak ikut terhapus, karena satu transaksi.
	plain := f.newTask(t, f.stafA.ID)
	if err := f.svc.DeleteAndStopSeries(ctx, f.manager, plain, 1); err == nil {
		t.Fatal("stop_series untuk task tidak berulang harus ditolak")
	}
	if err := f.pool.QueryRow(ctx, `SELECT is_deleted FROM tasks WHERE id = $1`, plain).Scan(&deleted); err != nil || deleted {
		t.Fatalf("task biasa ikut terhapus walau permintaannya ditolak: %v err %v", deleted, err)
	}
}

// Revisi 31: seri aktif yang semua kemunculannya sudah dihapus lebih dari 7 hari dilewati ticker.
func TestTickerSkipsAbandonedSeries(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	id, sid := newSeriesTask(t, f, "2026-11-01", CreateInput{
		Title: "Rekap pekanan", Urgency: "low", PICIDs: []int64{f.stafA.ID}, Schedule: &ScheduleInput{Freq: FreqDaily},
	})
	if err := f.svc.Delete(ctx, f.manager, id, 1); err != nil {
		t.Fatal(err)
	}
	if created, _ := f.svc.CreateDueOccurrences(ctx, at("2026-11-08")); len(created) != 1 {
		t.Fatalf("7 hari setelah kemunculan terakhir masih dibuat, dapat %v", created)
	}
	if err := f.svc.Delete(ctx, f.manager, mustOccurrence(t, f, sid, "2026-11-08"), 1); err != nil {
		t.Fatal(err)
	}
	created, abandoned, err := f.svc.createDueOccurrences(ctx, at("2026-11-16"))
	if err != nil || len(created) != 0 || !slices.Equal(abandoned, []int64{sid}) {
		t.Fatalf("seri ditinggalkan: dibuat %v dilewati %v err %v", created, abandoned, err)
	}
	if !seriesActive(t, f, sid) {
		t.Fatal("pengaman ticker tidak boleh menghentikan seri; itu hak Manajer")
	}
}

func mustOccurrence(t *testing.T, f fixture, sid int64, date string) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT id FROM tasks WHERE series_id = $1 AND occurrence_date = $2`, sid, date).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// Revisi 32: di kemunculan buatan ticker, PIC task mengatur semua sub task, termasuk yang tercatat dibuat pembuat
// seri dan dipegang orang lain, tetapi tidak mencentang sub task milik orang lain. move_to-nya berisi On Progress.
func TestTaskPICManagesTickerSubtasks(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	outsider := Actor{ID: testdb.User(t, f.pool, "staf.luar", "staff", f.marketing, true), Role: RoleStaff, DivisionID: f.marketing}
	_, sid := newSeriesTask(t, f, "2026-11-19", CreateInput{
		Title: "DAILY VIDEO EDIT", Urgency: "medium", PICIDs: []int64{f.stafA.ID},
		Subtasks: []SubtaskFields{{Title: "Edit Video 1", PICID: f.stafC.ID}, {Title: "Edit Video 2", PICID: f.stafC.ID}},
		Schedule: &ScheduleInput{Freq: FreqDaily},
	})
	if created, err := f.svc.CreateDueOccurrences(ctx, at("2026-11-20")); err != nil || len(created) != 1 {
		t.Fatalf("kemunculan ticker: %v err %v", created, err)
	}
	id := mustOccurrence(t, f, sid, "2026-11-20")
	d, err := f.svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if d.Task.RequestedBy == f.stafA.ID || d.Subtasks[0].CreatedBy == f.stafA.ID {
		t.Fatal("prasyarat: task dan sub task kemunculan harus tercatat dibuat orang lain")
	}

	can, _ := MoveOptions(f.stafA.Role, d.IsPIC(f.stafA.ID), false, d.MySubtasks(f.stafA.ID), Status(d.Task.Status))
	if !slices.Contains(can.MoveTo, OnProgress) || !can.AddSubtask || !can.EditLabels {
		t.Fatalf("hak PIC task di kemunculan ticker: %+v", can)
	}

	first, second := d.Subtasks[0], d.Subtasks[1]
	edit := SubtaskFields{Title: "Edit Video 1 revisi", PICID: f.stafC.ID}
	if err := f.svc.UpdateSubtask(ctx, outsider, id, first.ID, edit); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Staf bukan PIC mengubah sub task: err %v, mau ErrForbidden", err)
	}
	if err := f.svc.DeleteSubtask(ctx, outsider, id, second.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Staf bukan PIC menghapus sub task: err %v, mau ErrForbidden", err)
	}
	if err := f.svc.UpdateSubtask(ctx, f.stafA, id, first.ID, edit); err != nil {
		t.Fatalf("PIC task mengubah sub task buatan orang lain: %v", err)
	}
	if err := f.svc.DeleteSubtask(ctx, f.stafA, id, second.ID); err != nil {
		t.Fatalf("PIC task menghapus sub task buatan orang lain: %v", err)
	}
	// Kemunculan terbaru: perubahan PIC task ikut masuk templat (revisi 29).
	d, _ = f.svc.Get(ctx, id)
	if len(d.TemplateSubtasks) != 1 || d.TemplateSubtasks[0].Title != "Edit Video 1 revisi" {
		t.Fatalf("templat seri tidak mengikuti perubahan PIC task: %+v", d.TemplateSubtasks)
	}

	// Centang tetap: PIC task tidak mencentang sub task yang dipegang orang lain.
	if _, err := f.svc.Move(ctx, f.stafA, id, d.Task.Version, OnProgress, "", nil); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.SetSubtaskDone(ctx, f.stafA, id, first.ID, true); !errors.Is(err, ErrForbidden) {
		t.Fatalf("PIC task mencentang sub task milik orang lain: err %v, mau ErrForbidden", err)
	}
}
