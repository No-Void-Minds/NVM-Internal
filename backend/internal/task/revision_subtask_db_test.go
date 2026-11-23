package task

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/testdb"
)

// Revisi 22: PIC sub task (bukan PIC task) mengembalikan task Ready ke On Progress dan membuka lagi sub task
// miliknya yang dia pilih. stafC memegang dua sub task, staf.b satu sub task; stafA PIC task.
func TestSubtaskPICRevision(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	stafB := Actor{ID: testdb.User(t, f.pool, "staf.b", "staff", f.marketing, true), Role: RoleStaff, DivisionID: f.marketing}
	outsider := Actor{ID: testdb.User(t, f.pool, "staf.luar", "staff", f.marketing, true), Role: RoleStaff, DivisionID: f.marketing}
	id := f.newTask(t, f.stafA.ID)

	add := func(title string, pic int64) int64 {
		t.Helper()
		sid, err := f.svc.AddSubtask(ctx, f.stafA, id, SubtaskFields{Title: title, PICID: pic})
		if err != nil {
			t.Fatal(err)
		}
		return sid
	}
	naskah, desain, foto := add("Naskah", f.stafC.ID), add("Desain", f.stafC.ID), add("Foto", stafB.ID)

	get := func() Detail {
		t.Helper()
		d, err := f.svc.Get(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	move := func(actor Actor, to Status, note string, subs []int64) error {
		t.Helper()
		_, err := f.svc.Move(ctx, actor, id, get().Task.Version, to, note, subs)
		return err
	}

	if err := move(f.stafA, OnProgress, "", nil); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		actor Actor
		sub   int64
	}{{f.stafC, naskah}, {f.stafC, desain}, {stafB, foto}} {
		if err := f.svc.SetSubtaskDone(ctx, c.actor, id, c.sub, true); err != nil {
			t.Fatal(err)
		}
	}
	if err := move(f.stafA, Ready, "", nil); err != nil {
		t.Fatal(err)
	}

	// move_to dan pilihan dari data yang sama dengan handler.
	d := get()
	can, choices := MoveOptions(f.stafC.Role, d.IsPIC(f.stafC.ID), false, d.MySubtasks(f.stafC.ID), Status(d.Task.Status))
	if !slices.Equal(can.MoveTo, []Status{OnProgress}) || len(choices) != 2 || choices[0].ID != naskah || choices[1].ID != desain {
		t.Fatalf("stafC di Ready: move_to %v, pilihan %v, mau [on_progress] dan [Naskah Desain]", can.MoveTo, choices)
	}

	var ve *apperr.Validation
	if err := move(f.stafC, OnProgress, "", nil); !errors.As(err, &ve) {
		t.Fatalf("daftar kosong: err %v, mau Validation", err)
	}
	if err := move(f.stafC, OnProgress, "", []int64{naskah, foto}); !errors.As(err, &ve) {
		t.Fatalf("memilih sub task milik orang lain: err %v, mau Validation", err)
	}
	if err := move(outsider, OnProgress, "", []int64{naskah}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("bukan PIC apa pun: err %v, mau ErrForbidden", err)
	}
	if _, err := f.svc.Move(ctx, f.stafC, id, get().Task.Version-1, OnProgress, "", []int64{naskah}); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("version basi: err %v, mau ErrVersionConflict", err)
	}
	// PIC task memakai jalur revisi biasa: catatan tetap wajib walau memilih sub task (revisi 34 membolehkan
	// pilihannya, diuji di TestRevisiPICTaskMembukaSubTask).
	if err := move(f.stafA, OnProgress, "", []int64{naskah}); !errors.As(err, &ve) {
		t.Fatalf("PIC task memilih sub task tanpa catatan: err %v, mau Validation", err)
	}
	if d := get(); d.Task.Status != "ready" {
		t.Fatalf("setelah semua penolakan status %s, mau ready", d.Task.Status)
	}

	if err := move(f.stafC, OnProgress, "  Logo terlalu kecil  ", []int64{naskah}); err != nil {
		t.Fatalf("PIC sub task mengembalikan dengan satu sub task: %v", err)
	}
	d = get()
	if d.Task.Status != "on_progress" {
		t.Fatalf("status %s, mau on_progress", d.Task.Status)
	}
	for _, st := range d.Subtasks {
		if want := st.ID != naskah; st.DoneAt.Valid != want {
			t.Errorf("sub task %s: tercentang %v, mau %v", st.Title, st.DoneAt.Valid, want)
		}
	}

	// Riwayat: satu task_status_changed dengan sub task yang dibuka dan alasannya (bukan "note").
	var after struct {
		Status   string       `json:"status"`
		Reopened []SubtaskRef `json:"reopened_subtasks"`
		Reason   string       `json:"reason"`
		Note     *string      `json:"note"`
	}
	found := false
	for _, h := range d.History {
		if h.Action == "task_status_changed" && json.Unmarshal(h.NewValue, &after) == nil && after.Reopened != nil {
			found = true
			break
		}
	}
	if !found || after.Status != "on_progress" || len(after.Reopened) != 1 || after.Reopened[0].Title != "Naskah" ||
		after.Reason != "Logo terlalu kecil" || after.Note != nil {
		t.Fatalf("riwayat revisi sub task: %+v (ketemu %v)", after, found)
	}

	// Setelah kembali ke On Progress, move_to stafC kembali ke Ready seperti revisi 17.
	can, _ = MoveOptions(f.stafC.Role, d.IsPIC(f.stafC.ID), false, d.MySubtasks(f.stafC.ID), Status(d.Task.Status))
	if !slices.Equal(can.MoveTo, []Status{Ready}) {
		t.Fatalf("stafC di On Progress: move_to %v, mau [ready]", can.MoveTo)
	}
}
