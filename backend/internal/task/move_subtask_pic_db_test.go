package task

import (
	"context"
	"errors"
	"slices"
	"testing"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/testdb"
)

// Revisi 17: PIC sub task boleh Requested ke On Progress dan On Progress ke Ready (dengan Ready gate), tidak
// boleh perpindahan lain. move_to yang dihitung dari data yang sama harus cocok dengan hasil pindah sebenarnya.
func TestSubtaskPICMoves(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	outsider := Actor{ID: testdb.User(t, f.pool, "staf.luar", "staff", f.marketing, true), Role: RoleStaff, DivisionID: f.marketing}
	id := f.newTask(t, f.stafA.ID)
	sub, err := f.svc.AddSubtask(ctx, f.stafA, id, SubtaskFields{Title: "Desain poster", PICID: f.stafC.ID})
	if err != nil {
		t.Fatal(err)
	}

	// moveTo menghitung move_to untuk actor dengan cara yang sama dengan handler detail task.
	moveTo := func(actor Actor) ([]Status, int32) {
		t.Helper()
		d, err := f.svc.Get(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		c := PermissionsFor(actor.Role, d.IsPIC(actor.ID), d.IsSubtaskPIC(actor.ID), Status(d.Task.Status))
		return c.MoveTo, d.Task.Version
	}
	move := func(actor Actor, to Status, note string) error {
		t.Helper()
		_, v := moveTo(actor)
		_, err := f.svc.Move(ctx, actor, id, v, to, note)
		return err
	}

	if got, _ := moveTo(outsider); len(got) != 0 {
		t.Fatalf("orang luar: move_to %v, mau kosong", got)
	}
	if err := move(outsider, OnProgress, ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("orang luar memulai task: err %v, mau ErrForbidden", err)
	}

	if got, _ := moveTo(f.stafC); !slices.Equal(got, []Status{OnProgress}) {
		t.Fatalf("PIC sub task di Requested: move_to %v, mau [on_progress]", got)
	}
	if err := move(f.stafC, OnProgress, ""); err != nil {
		t.Fatalf("PIC sub task memulai task: %v", err)
	}
	if got, _ := moveTo(f.stafC); !slices.Equal(got, []Status{Ready}) {
		t.Fatalf("PIC sub task di On Progress: move_to %v, mau [ready]", got)
	}
	if err := move(f.stafC, Requested, ""); err == nil {
		t.Fatal("PIC sub task mengembalikan ke Requested harus ditolak")
	}

	var ve *apperr.Validation
	if err := move(f.stafC, Ready, ""); !errors.As(err, &ve) {
		t.Fatalf("Ready dengan sub task terbuka: err %v, mau Validation (Ready gate)", err)
	}
	if err := f.svc.SetSubtaskDone(ctx, f.stafC, id, sub, true); err != nil {
		t.Fatal(err)
	}
	if err := move(f.stafC, Ready, ""); err != nil {
		t.Fatalf("PIC sub task menandai Ready setelah semua tercentang: %v", err)
	}

	if got, _ := moveTo(f.stafC); len(got) != 0 {
		t.Fatalf("PIC sub task di Ready: move_to %v, mau kosong", got)
	}
	if err := move(f.stafC, OnProgress, "Perlu revisi"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("PIC sub task mengembalikan Ready: err %v, mau ErrForbidden", err)
	}
	if err := move(f.stafC, Completed, ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("PIC sub task menutup task: err %v, mau ErrForbidden", err)
	}

	// Papan dan Arsip menghitung hubungan yang sama lewat satu query.
	m, err := f.svc.SubtaskPICTasks(ctx, f.stafC.ID, []string{id})
	if err != nil || !m[id] {
		t.Fatalf("SubtaskPICTasks: %v, err %v", m, err)
	}
	if m, _ := f.svc.SubtaskPICTasks(ctx, outsider.ID, []string{id}); m[id] {
		t.Fatal("SubtaskPICTasks menganggap orang luar PIC sub task")
	}
}
