package task

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Revisi 27: Staf dan Manajer membuat task dengan aturan yang sama persis, termasuk sub task dan jadwal.
// Admin tetap ditolak.
func TestStafMembuatTask(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	f.svc.now = func() time.Time { return at("2026-11-19") }
	id, err := f.svc.Create(ctx, f.stafA, CreateInput{
		Title:    "Konten mingguan",
		Urgency:  "medium",
		PICIDs:   []int64{f.stafA.ID},
		Subtasks: []SubtaskFields{{Title: "Riset bahan", PICID: f.stafA.ID, Description: "Kumpulkan referensi"}},
		Schedule: &ScheduleInput{Freq: FreqWeekly, Weekdays: []int{4}},
	})
	if err != nil {
		t.Fatal(err)
	}
	d, err := f.svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if d.Task.RequestedBy != f.stafA.ID {
		t.Fatalf("pembuat task %d, mau %d", d.Task.RequestedBy, f.stafA.ID)
	}
	if len(d.Subtasks) != 1 || d.Subtasks[0].Title != "Riset bahan" {
		t.Fatalf("sub task dari form tidak tersimpan: %+v", d.Subtasks)
	}
	// Jadwal dari Staf jalan sama seperti dari Manajer, dan task itu sendiri jadi kemunculan pertama.
	if d.Series == nil || !d.Task.OccurrenceDate.Valid {
		t.Fatalf("jadwal tidak terpasang: series=%v occurrence=%v", d.Series, d.Task.OccurrenceDate)
	}

	// Admin tidak boleh membuat task.
	_, err = f.svc.Create(ctx, f.admin, CreateInput{Title: "Dari admin", Urgency: "low", PICIDs: []int64{f.stafA.ID}})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Admin harus ditolak, dapat %v", err)
	}
}

// Revisi 27: pembuat task boleh mengubah dan menghapus task buatannya selama masih Requested. Setelah keluar
// dari Requested, berlaku aturan biasa.
func TestPembuatMengubahTaskRequested(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	// Dibuat Staf A, PIC-nya Staf C, jadi pembuatnya sengaja bukan PIC.
	id, err := f.svc.Create(ctx, f.stafA, CreateInput{Title: "Butuh desain", Urgency: "high", PICIDs: []int64{f.stafC.ID}})
	if err != nil {
		t.Fatal(err)
	}

	// Staf lain yang bukan pembuat dan bukan PIC: ditolak.
	_, err = f.svc.Update(ctx, f.stafC, id, UpdateInput{
		Title: "Diubah PIC", Urgency: "high", PICIDs: []int64{f.stafC.ID}, Version: f.version(t, id),
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("PIC yang bukan pembuat harus ditolak, dapat %v", err)
	}

	// Pembuatnya: diterima.
	if _, err := f.svc.Update(ctx, f.stafA, id, UpdateInput{
		Title: "Butuh desain banner", Urgency: "high", PICIDs: []int64{f.stafC.ID}, Version: f.version(t, id),
	}); err != nil {
		t.Fatal(err)
	}
	d, err := f.svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if d.Task.Title != "Butuh desain banner" {
		t.Fatalf("judul %q", d.Task.Title)
	}

	// Setelah pindah ke On Progress, pembuat yang bukan PIC tidak boleh lagi.
	if _, err := f.svc.Move(ctx, f.stafC, id, f.version(t, id), OnProgress, "", nil); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Update(ctx, f.stafA, id, UpdateInput{
		Title: "Diubah lagi", Urgency: "high", PICIDs: []int64{f.stafC.ID}, Version: f.version(t, id),
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("pembuat di On Progress harus ditolak, dapat %v", err)
	}
	if err := f.svc.Delete(ctx, f.stafA, id, f.version(t, id)); !errors.Is(err, ErrForbidden) {
		t.Fatalf("hapus di On Progress harus ditolak, dapat %v", err)
	}

	// Manajer tetap boleh di status apa pun.
	if _, err := f.svc.Update(ctx, f.manager, id, UpdateInput{
		Title: "Diubah Manajer", Urgency: "high", PICIDs: []int64{f.stafC.ID}, Version: f.version(t, id),
	}); err != nil {
		t.Fatal(err)
	}
}

// Pembuat boleh menghapus task buatannya yang masih Requested.
func TestPembuatMenghapusTaskRequested(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	id, err := f.svc.Create(ctx, f.stafA, CreateInput{Title: "Salah buat", Urgency: "low", PICIDs: []int64{f.stafA.ID}})
	if err != nil {
		t.Fatal(err)
	}
	// Staf lain tidak boleh menghapusnya.
	if err := f.svc.Delete(ctx, f.stafC, id, f.version(t, id)); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Staf lain harus ditolak, dapat %v", err)
	}
	if err := f.svc.Delete(ctx, f.stafA, id, f.version(t, id)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Get(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("task terhapus harus hilang, dapat %v", err)
	}
}
