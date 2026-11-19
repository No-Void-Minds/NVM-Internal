package task

import (
	"context"
	"testing"
	"time"
)

// Perbaikan bug revisi 19: jadwal yang ditambahkan lewat Ubah membuat task itu jadi kemunculan pertama,
// dengan occurrence_date sama dengan tanggal mulainya, dan ticker tidak membuat kemunculan dobel di tanggal itu.
func TestJadwalLewatUbahMengisiOccurrenceDate(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	f.svc.now = func() time.Time { return at("2026-11-19") }

	// Dibuat tanpa jadwal, dengan tanggal mulai hari ini.
	id, err := f.svc.Create(ctx, f.manager, CreateInput{
		Title: "Rekap harian", Urgency: "medium", PICIDs: []int64{f.stafA.ID}, StartDate: "2026-11-19",
	})
	if err != nil {
		t.Fatal(err)
	}
	d, err := f.svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if d.Task.OccurrenceDate.Valid {
		t.Fatal("task tanpa jadwal tidak boleh punya occurrence_date")
	}

	// Jadwal ditambahkan lewat Ubah.
	if _, err := f.svc.Update(ctx, f.manager, id, UpdateInput{
		Title: "Rekap harian", Urgency: "medium", PICIDs: []int64{f.stafA.ID}, StartDate: "2026-11-19",
		Version: f.version(t, id), Schedule: &ScheduleInput{Freq: FreqDaily, SkipWeekend: true},
	}); err != nil {
		t.Fatal(err)
	}
	d, err = f.svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if d.Series == nil {
		t.Fatal("seri tidak terbentuk lewat Ubah")
	}
	if !d.Task.OccurrenceDate.Valid {
		t.Fatal("task yang diberi jadwal lewat Ubah harus jadi kemunculan pertama")
	}
	if got := d.Task.OccurrenceDate.Time.Format("2006-01-02"); got != "2026-11-19" {
		t.Fatalf("occurrence_date %q, mau tanggal mulai 2026-11-19", got)
	}

	// Ticker di hari yang sama tidak boleh membuat kemunculan kedua untuk tanggal itu.
	created, err := f.svc.CreateDueOccurrences(ctx, at("2026-11-19"))
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 {
		t.Fatalf("ticker membuat kemunculan dobel: %+v", created)
	}
	if n := len(seriesDates(t, f, d.Task.SeriesID.Int64)); n != 1 {
		t.Fatalf("kemunculan di seri %d, mau 1", n)
	}

	// Hari berikutnya (Jumat 20) tetap dibuat seperti biasa.
	created, err = f.svc.CreateDueOccurrences(ctx, at("2026-11-20"))
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 1 {
		t.Fatalf("kemunculan hari berikutnya %+v", created)
	}
}

// Perbaikan bug revisi 19: detail task berulang membawa daftar sub task templat beserta PIC-nya.
func TestDetailMembawaSubTaskTemplat(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	f.svc.now = func() time.Time { return at("2026-11-19") }

	id, err := f.svc.Create(ctx, f.manager, CreateInput{
		Title: "Konten harian", Urgency: "medium", PICIDs: []int64{f.stafA.ID},
		Subtasks: []SubtaskFields{{Title: "Tulis draf", PICID: f.stafA.ID}},
		Schedule: &ScheduleInput{Freq: FreqDaily},
	})
	if err != nil {
		t.Fatal(err)
	}
	d, err := f.svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.TemplateSubtasks) != 1 {
		t.Fatalf("sub task templat %+v, mau satu", d.TemplateSubtasks)
	}
	if d.TemplateSubtasks[0].Title != "Tulis draf" || d.TemplateSubtasks[0].PIC != "staf.a" {
		t.Fatalf("sub task templat %+v", d.TemplateSubtasks[0])
	}
	if !d.TemplateMatchesTask() {
		t.Fatal("tepat setelah dibuat, sub task task dan templat harus sama")
	}

	// Sejak revisi 29 sub task yang ditambahkan lewat panel detail di kemunculan terbaru ikut masuk templat.
	if _, err := f.svc.AddSubtask(ctx, f.stafA, id, SubtaskFields{Title: "Unggah", PICID: f.stafA.ID}); err != nil {
		t.Fatal(err)
	}
	d, err = f.svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.TemplateSubtasks) != 2 || !d.TemplateMatchesTask() {
		t.Fatalf("templat harus ikut kemunculan terbaru: %+v", d.TemplateSubtasks)
	}
}
