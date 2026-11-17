package task

import (
	"context"
	"errors"
	"testing"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/testdb"
)

// Revisi 13: kandidat PIC sub task sama dengan PIC task, yaitu Staf dan Manajer aktif. Admin dan akun
// nonaktif ditolak, tetapi PIC lama yang kemudian dinonaktifkan tetap bertahan saat hanya nama yang diubah.
func TestSubtaskPICEligibility(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	managerB := testdb.User(t, f.pool, "manajer.b", "manager", f.creative, true)
	inactive := testdb.User(t, f.pool, "staf.lama", "staff", f.marketing, false)
	id := f.newTask(t, f.stafA.ID)

	if _, err := f.svc.AddSubtask(ctx, f.stafA, id, SubtaskFields{Title: "Desain banner", PICID: managerB}); err != nil {
		t.Fatalf("Manajer sebagai PIC sub task harus diterima: %v", err)
	}
	var ve *apperr.Validation
	for name, pic := range map[string]int64{"Admin": f.admin.ID, "akun nonaktif": inactive} {
		if _, err := f.svc.AddSubtask(ctx, f.stafA, id, SubtaskFields{Title: "X", PICID: pic}); !errors.As(err, &ve) {
			t.Errorf("%s sebagai PIC sub task: err %v, mau Validation", name, err)
		}
	}

	subID, err := f.svc.AddSubtask(ctx, f.stafA, id, SubtaskFields{Title: "Foto produk", PICID: f.stafC.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE users SET is_active = false WHERE id = $1`, f.stafC.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.UpdateSubtask(ctx, f.stafA, id, subID, SubtaskFields{Title: "Foto produk final", PICID: f.stafC.ID}); err != nil {
		t.Fatalf("ubah nama dengan PIC lama yang nonaktif harus lolos: %v", err)
	}
	d, err := f.svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range d.Subtasks {
		if st.ID == subID && (len(st.Assignees) != 1 || st.Assignees[0].ID != f.stafC.ID) {
			t.Fatalf("PIC lama harus bertahan, dapat %+v", st.Assignees)
		}
	}
}

// Centang antar Manajer: Manajer lain tidak bisa mencentang sub task yang dipegang Manajer, kecuali dia
// pembuat sub task itu. can_check (CheckableSubtasks) harus menjawab sama dengan SetSubtaskDone.
func TestSubtaskCheckBetweenManagers(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	managerB := Actor{ID: testdb.User(t, f.pool, "manajer.b", "manager", f.creative, true), Role: RoleManager, DivisionID: f.creative}
	id := f.newTask(t, f.stafA.ID)

	byStaf, err := f.svc.AddSubtask(ctx, f.stafA, id, SubtaskFields{Title: "Dibuat Staf", PICID: managerB.ID})
	if err != nil {
		t.Fatal(err)
	}
	byManager, err := f.svc.AddSubtask(ctx, f.manager, id, SubtaskFields{Title: "Dibuat Manajer A", PICID: managerB.ID})
	if err != nil {
		t.Fatal(err)
	}
	d, err := f.svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Move(ctx, f.stafA, id, d.Task.Version, OnProgress, ""); err != nil {
		t.Fatalf("ke On Progress: %v", err)
	}
	can, err := f.svc.CheckableSubtasks(ctx, f.manager, OnProgress, d.IsPIC(f.manager.ID), d.Subtasks)
	if err != nil {
		t.Fatal(err)
	}
	if can[byStaf] || !can[byManager] {
		t.Fatalf("can_check Manajer A: dibuat Staf %v (mau false), dibuat dia sendiri %v (mau true)", can[byStaf], can[byManager])
	}

	if err := f.svc.SetSubtaskDone(ctx, f.manager, id, byStaf, true); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Manajer A mencentang sub task milik Manajer B: err %v, mau ErrForbidden", err)
	}
	if err := f.svc.SetSubtaskDone(ctx, f.manager, id, byManager, true); err != nil {
		t.Fatalf("Manajer A mencentang sub task buatannya sendiri: %v", err)
	}
	if err := f.svc.SetSubtaskDone(ctx, managerB, id, byStaf, true); err != nil {
		t.Fatalf("Manajer B mencentang sub task miliknya: %v", err)
	}
	if err := f.svc.SetSubtaskDone(ctx, f.stafA, id, byStaf, false); err != nil {
		t.Fatalf("pembuat (Staf) melepas centang: %v", err)
	}
}
