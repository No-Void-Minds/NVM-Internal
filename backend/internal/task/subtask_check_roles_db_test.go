package task

import (
	"context"
	"errors"
	"testing"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/testdb"
)

// Revisi 34: centang dan batal centang sub task untuk PIC sub task, PIC task, pembuat task Staf, dan Manajer;
// Admin dan Staf yang tidak terlibat ditolak. "Dicentang <nama>" dan audit memakai orang yang mencentang.
func TestCentangSubTaskPerPeran(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	peminta := Actor{ID: testdb.User(t, f.pool, "staf.peminta", "staff", f.marketing, true), Role: RoleStaff, DivisionID: f.marketing}
	luar := Actor{ID: testdb.User(t, f.pool, "staf.luar", "staff", f.marketing, true), Role: RoleStaff, DivisionID: f.marketing}

	// Task dibuat Staf (revisi 27), PIC task staf.a, sub task dipegang staf.c dan dibuat Manajer.
	id, err := f.svc.Create(ctx, peminta, CreateInput{Title: "Kampanye", Urgency: "medium", PICIDs: []int64{f.stafA.ID}})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := f.svc.AddSubtask(ctx, f.manager, id, SubtaskFields{Title: "Desain poster", PICID: f.stafC.ID})
	if err != nil {
		t.Fatal(err)
	}

	// Bukan On Progress: semua peran ditolak dengan pesan status, bukan pesan hak.
	var ve *apperr.Validation
	if err := f.svc.SetSubtaskDone(ctx, f.manager, id, sub, true); !errors.As(err, &ve) {
		t.Fatalf("centang saat Requested: err %v, mau galat validasi", err)
	}
	d, _ := f.svc.Get(ctx, id)
	if can := CheckableSubtasks(f.manager, Status(d.Task.Status), false, false, d.Subtasks); can[sub] {
		t.Fatal("can_check harus false saat Requested")
	}
	if _, err := f.svc.Move(ctx, f.stafA, id, d.Task.Version, OnProgress, "", nil); err != nil {
		t.Fatal(err)
	}

	boleh := []struct {
		name  string
		actor Actor
		nama  string
	}{
		{"PIC sub task", f.stafC, "staf.c"},
		{"PIC task", f.stafA, "staf.a"},
		{"pembuat task Staf", peminta, "staf.peminta"},
		{"Manajer", f.manager, "manajer.a"},
	}
	for _, c := range boleh {
		d, _ := f.svc.Get(ctx, id)
		can := CheckableSubtasks(c.actor, Status(d.Task.Status), d.IsPIC(c.actor.ID), d.Task.RequestedBy == c.actor.ID, d.Subtasks)
		if !can[sub] {
			t.Fatalf("%s: can_check false, mau true", c.name)
		}
		if err := f.svc.SetSubtaskDone(ctx, c.actor, id, sub, true); err != nil {
			t.Fatalf("%s mencentang: %v", c.name, err)
		}
		d, _ = f.svc.Get(ctx, id)
		if got := d.Subtasks[0].DoneBy; got != c.nama {
			t.Fatalf("%s: tercatat dicentang %q, mau %q", c.name, got, c.nama)
		}
		var pelaku int64
		if err := f.pool.QueryRow(ctx, `SELECT user_id FROM audit_logs WHERE action = 'subtask_checked' ORDER BY id DESC LIMIT 1`).Scan(&pelaku); err != nil || pelaku != c.actor.ID {
			t.Fatalf("%s: audit subtask_checked beraktor %d (err %v), mau %d", c.name, pelaku, err, c.actor.ID)
		}
		if err := f.svc.SetSubtaskDone(ctx, c.actor, id, sub, false); err != nil {
			t.Fatalf("%s membatalkan centang: %v", c.name, err)
		}
	}

	// Centang dulu oleh PIC sub task, supaya penolakan batal centang juga teruji.
	if err := f.svc.SetSubtaskDone(ctx, f.stafC, id, sub, true); err != nil {
		t.Fatal(err)
	}
	for name, a := range map[string]Actor{"Admin": f.admin, "Staf tidak terlibat": luar} {
		d, _ := f.svc.Get(ctx, id)
		if CheckableSubtasks(a, Status(d.Task.Status), d.IsPIC(a.ID), d.Task.RequestedBy == a.ID, d.Subtasks)[sub] {
			t.Fatalf("%s: can_check true, mau false", name)
		}
		for _, done := range []bool{false, true} {
			if err := f.svc.SetSubtaskDone(ctx, a, id, sub, done); !errors.Is(err, ErrNotSubtaskPIC) {
				t.Fatalf("%s mengubah centang (%v): err %v, mau ErrNotSubtaskPIC", name, done, err)
			}
		}
	}
}

// Revisi 34 memperluas revisi 22: PIC task dan Manajer boleh membuka lagi sub task tercentang mana pun saat
// mengembalikan Ready ke On Progress. Pilihannya opsional dan catatan revisi tetap wajib.
func TestRevisiPICTaskMembukaSubTask(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	id := f.newTask(t, f.stafA.ID)
	a, err := f.svc.AddSubtask(ctx, f.stafA, id, SubtaskFields{Title: "Naskah", PICID: f.stafC.ID})
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.svc.AddSubtask(ctx, f.stafA, id, SubtaskFields{Title: "Rekam suara", PICID: f.stafC.ID})
	if err != nil {
		t.Fatal(err)
	}
	toReady := func() int32 {
		t.Helper()
		d, _ := f.svc.Get(ctx, id)
		if Status(d.Task.Status) == Requested {
			if _, err := f.svc.Move(ctx, f.stafA, id, d.Task.Version, OnProgress, "", nil); err != nil {
				t.Fatal(err)
			}
		}
		for _, s := range []int64{a, b} {
			if err := f.svc.SetSubtaskDone(ctx, f.stafA, id, s, true); err != nil {
				t.Fatal(err)
			}
		}
		d, _ = f.svc.Get(ctx, id)
		res, err := f.svc.Move(ctx, f.stafA, id, d.Task.Version, Ready, "", nil)
		if err != nil {
			t.Fatal(err)
		}
		return res.Version
	}

	v := toReady()
	d, _ := f.svc.Get(ctx, id)
	if got := TaskRevisionChoices(f.manager.Role, false, Ready, d.Subtasks); len(got) != 2 {
		t.Fatalf("pilihan revisi Manajer %d, mau 2", len(got))
	}
	if RevisionNoteRequired(f.stafC.Role, false, true) || !RevisionNoteRequired(f.stafA.Role, true, false) {
		t.Fatal("catatan wajib untuk PIC task, opsional untuk jalur PIC sub task")
	}
	if _, err := f.svc.Move(ctx, f.manager, id, v, OnProgress, "", []int64{a}); err == nil {
		t.Fatal("tanpa catatan harus ditolak")
	}
	if _, err := f.svc.Move(ctx, f.manager, id, v, OnProgress, "Naskah perlu diganti", []int64{a}); err != nil {
		t.Fatalf("Manajer membuka sub task milik orang lain: %v", err)
	}
	d, _ = f.svc.Get(ctx, id)
	if d.Subtasks[0].DoneAt.Valid || !d.Subtasks[1].DoneAt.Valid {
		t.Fatal("hanya sub task yang dipilih yang dibuka lagi")
	}

	// PIC task, tanpa pilihan: semua sub task tetap tercentang.
	v = toReady()
	if _, err := f.svc.Move(ctx, f.stafA, id, v, OnProgress, "Cek ulang", nil); err != nil {
		t.Fatal(err)
	}
	d, _ = f.svc.Get(ctx, id)
	if !d.Subtasks[0].DoneAt.Valid || !d.Subtasks[1].DoneAt.Valid {
		t.Fatal("tanpa pilihan, centang tidak boleh berubah")
	}
	// Pilihan sub task di luar Ready ke On Progress tetap ditolak.
	if _, err := f.svc.Move(ctx, f.stafA, id, d.Task.Version, Ready, "", []int64{a}); err == nil {
		t.Fatal("subtask_ids di perpindahan lain harus ditolak")
	}
}
