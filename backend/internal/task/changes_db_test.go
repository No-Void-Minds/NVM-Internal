package task

import (
	"context"
	"errors"
	"testing"

	"nvm-internal/internal/apperr"
)

func (f fixture) label(t *testing.T, name string, division *int64, active bool) int64 {
	t.Helper()
	var id int64
	if err := f.pool.QueryRow(context.Background(),
		`INSERT INTO labels (name, color, division_id, is_active) VALUES ($1, '#E6E1D7', $2, $3) RETURNING id`,
		name, division, active).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f fixture) version(t *testing.T, id string) int32 {
	t.Helper()
	d, err := f.svc.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return d.Task.Version
}

func TestManagerUpdateMovesDivisionAndBumpsVersion(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	id := f.newTask(t, f.stafA.ID)

	v, err := f.svc.Update(ctx, f.manager, id, UpdateInput{
		Version: 1, Title: "Foto katalog Core", Urgency: "urgent", PICIDs: []int64{f.stafC.ID, f.stafA.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	d, _ := f.svc.Get(ctx, id)
	if v != 2 || d.Task.DivisionID != f.creative || d.Task.Title != "Foto katalog Core" {
		t.Fatalf("versi=%d divisi=%d judul=%q; mau versi 2, divisi Creative, judul baru", v, d.Task.DivisionID, d.Task.Title)
	}

	_, err = f.svc.Update(ctx, f.manager, id, UpdateInput{Version: 1, Title: "Lama", Urgency: "low", PICIDs: []int64{f.stafA.ID}})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("versi lama: err %v, mau ErrVersionConflict", err)
	}
	if _, err := f.svc.Update(ctx, f.stafA, id, UpdateInput{Version: 2, Title: "X", Urgency: "low", PICIDs: []int64{f.stafA.ID}}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Staf mengubah atribut task: err %v, mau ErrForbidden (AC-04)", err)
	}
}

func TestStaffLabelScope(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	semua := f.label(t, "TikTok", nil, true)
	marketing := f.label(t, "Ads", &f.marketing, true)
	creative := f.label(t, "Desain", &f.creative, true)
	id := f.newTask(t, f.stafA.ID)

	// Manajer memasang label Creative di task milik Staf Marketing.
	v, err := f.svc.SetLabels(ctx, f.manager, id, 1, []int64{creative})
	if err != nil {
		t.Fatal(err)
	}
	// Staf Marketing mengatur labelnya sendiri; label Creative tetap menempel walau tidak dikirim.
	v, err = f.svc.SetLabels(ctx, f.stafA, id, v, []int64{semua, marketing})
	if err != nil {
		t.Fatal(err)
	}
	d, _ := f.svc.Get(ctx, id)
	if len(d.Labels) != 3 {
		t.Fatalf("label di task %+v, mau TikTok, Ads, dan Desain", d.Labels)
	}

	if _, err := f.svc.SetLabels(ctx, f.stafA, id, v, []int64{}); err != nil {
		t.Fatal(err)
	}
	d, _ = f.svc.Get(ctx, id)
	if len(d.Labels) != 1 || d.Labels[0].ID != creative {
		t.Fatalf("Staf melepas semua labelnya: tersisa %+v, mau hanya Desain", d.Labels)
	}

	other := f.newTask(t, f.stafA.ID)
	if _, err := f.svc.SetLabels(ctx, f.stafA, other, 1, []int64{creative}); !errors.Is(err, ErrLabelOutOfScope) {
		t.Fatalf("Staf memasang label divisi lain: err %v, mau ErrLabelOutOfScope", err)
	}
	if _, err := f.svc.SetLabels(ctx, f.stafC, other, 1, []int64{creative}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Staf bukan PIC memasang label: err %v, mau ErrForbidden", err)
	}
	if _, err := f.svc.SetLabels(ctx, f.admin, other, 1, []int64{semua}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Admin memasang label: err %v, mau ErrForbidden", err)
	}
}

func TestInactiveLabelStaysButCannotBeAdded(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	lbl := f.label(t, "Konten", nil, true)
	id := f.newTask(t, f.stafA.ID)
	v, err := f.svc.SetLabels(ctx, f.manager, id, 1, []int64{lbl})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE labels SET is_active = false WHERE id = $1`, lbl); err != nil {
		t.Fatal(err)
	}

	// Tetap terpasang di task lama dan boleh dikirim ulang tanpa galat.
	if _, err := f.svc.SetLabels(ctx, f.manager, id, v, []int64{lbl}); err != nil {
		t.Fatalf("label nonaktif yang sudah terpasang harus boleh tetap ada: %v", err)
	}
	other := f.newTask(t, f.stafA.ID)
	var ve *apperr.Validation
	if _, err := f.svc.SetLabels(ctx, f.manager, other, 1, []int64{lbl}); !errors.As(err, &ve) {
		t.Fatalf("memasang label nonaktif ke task baru: err %v, mau Validation", err)
	}
}

func TestDeliverableByPIC(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	id := f.newTask(t, f.stafA.ID)

	v, err := f.svc.SetDeliverable(ctx, f.stafA, id, 1, "https://drive.google.com/folder/abc")
	if err != nil || v != 2 {
		t.Fatalf("PIC mengisi deliverable: versi %d err %v", v, err)
	}
	var ve *apperr.Validation
	if _, err := f.svc.SetDeliverable(ctx, f.stafA, id, v, "javascript:alert(1)"); !errors.As(err, &ve) {
		t.Fatalf("tautan javascript: err %v, mau Validation", err)
	}
	if _, err := f.svc.SetDeliverable(ctx, f.stafC, id, v, "https://contoh.id"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Staf bukan PIC: err %v, mau ErrForbidden", err)
	}
}

func TestSoftDelete(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	id := f.newTask(t, f.stafA.ID)

	if err := f.svc.Delete(ctx, f.stafA, id, 1); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Staf menghapus task: err %v, mau ErrForbidden", err)
	}
	if err := f.svc.Delete(ctx, f.manager, id, f.version(t, id)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Get(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("task terhapus masih bisa dibuka: err %v", err)
	}
	rows, _ := f.svc.List(ctx, nil)
	if len(rows) != 0 {
		t.Fatal("task terhapus masih muncul di daftar")
	}
	var n int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM tasks WHERE id = $1`, id).Scan(&n); err != nil || n != 1 {
		t.Fatal("soft delete tidak boleh menghapus baris dari database")
	}
}
