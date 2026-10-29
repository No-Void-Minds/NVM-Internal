package task

import (
	"context"
	"errors"
	"strings"
	"testing"

	"nvm-internal/internal/apperr"
)

// TestLifecycleWithDatabase menjalankan satu task dari Requested sampai dibuka kembali,
// termasuk percobaan yang harus ditolak di setiap tahap (AC-04, AC-08, FR-TASK-22).
func TestLifecycleWithDatabase(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	id := f.newTask(t, f.stafA.ID)
	v := int32(1)

	move := func(actor Actor, to Status, note string) (MoveResult, error) {
		return f.svc.Move(ctx, actor, id, v, to, note)
	}
	mustMove := func(actor Actor, to Status, note string) MoveResult {
		t.Helper()
		r, err := move(actor, to, note)
		if err != nil {
			t.Fatalf("%s ke %s: %v", actor.Role, to, err)
		}
		v = r.Version
		return r
	}

	if _, err := move(f.stafC, OnProgress, ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Staf bukan PIC: err %v, mau ErrForbidden", err)
	}
	if _, err := move(f.admin, OnProgress, ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Admin: err %v, mau ErrForbidden", err)
	}
	if _, err := move(f.manager, Completed, ""); !errors.Is(err, ErrInvalidMove) {
		t.Fatalf("lompat ke Completed: err %v, mau ErrInvalidMove", err)
	}

	mustMove(f.stafA, OnProgress, "")
	mustMove(f.stafA, Ready, "")

	if _, err := move(f.stafA, Completed, ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Staf menutup task: err %v, mau ErrForbidden", err)
	}
	var ve *apperr.Validation
	if _, err := move(f.stafA, OnProgress, ""); !errors.As(err, &ve) {
		t.Fatalf("revisi tanpa catatan: err %v, mau Validation", err)
	}
	mustMove(f.stafA, OnProgress, "Ukuran foto belum sesuai")
	mustMove(f.stafA, Ready, "")

	done := mustMove(f.manager, Completed, "")
	if !done.Solved.SolvedDate.Valid {
		t.Fatal("Completed harus mengisi solved_date")
	}
	reopened := mustMove(f.manager, OnProgress, "")
	if reopened.Solved.SolvedDate.Valid {
		t.Fatal("dibuka kembali harus mengosongkan solved_date")
	}

	if _, err := f.svc.Move(ctx, f.stafA, id, 1, Ready, ""); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("versi basi: err %v, mau ErrVersionConflict", err)
	}

	d, err := f.svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	// Riwayat terbaru di atas: buka kembali, Completed, Ready, revisi, Ready, On Progress, lalu task_created.
	if len(d.History) != 7 || d.History[0].Action != "task_status_changed" || d.History[6].Action != "task_created" {
		t.Fatalf("riwayat tidak lengkap atau urutannya salah: %d baris", len(d.History))
	}
	if !strings.Contains(string(d.History[3].NewValue), "Ukuran foto belum sesuai") {
		t.Fatalf("catatan revisi tidak tersimpan di riwayat: %s", d.History[3].NewValue)
	}
}
