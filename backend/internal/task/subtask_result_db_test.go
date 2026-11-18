package task

import (
	"context"
	"errors"
	"strings"
	"testing"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/testdb"
)

// Revisi 18: catatan hasil sub task. stafC bukan PIC task, hanya PIC sub task (revisi 11), dan tetap boleh
// mengisi catatan dan tautan hasil dalam satu permintaan serta mengunggah berkas sub task-nya.
func TestSubtaskResultNote(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	outsider := Actor{ID: testdb.User(t, f.pool, "staf.luar", "staff", f.marketing, true), Role: RoleStaff, DivisionID: f.marketing}
	id := f.newTask(t, f.stafA.ID)
	sub, err := f.svc.AddSubtask(ctx, f.stafA, id, SubtaskFields{Title: "Desain poster", PICID: f.stafC.ID})
	if err != nil {
		t.Fatal(err)
	}
	ptr := func(s string) *string { return &s }
	save := func(actor Actor, note *string, link string) error {
		return f.svc.UpdateSubtask(ctx, actor, id, sub, SubtaskFields{
			Title: "Desain poster", PICID: f.stafC.ID, Deliverable: link, ResultNote: note,
		})
	}
	stored := func() (note *string, link string) {
		t.Helper()
		if err := f.pool.QueryRow(ctx, `SELECT result_note, deliverable_url FROM task_subtasks WHERE id = $1`, sub).Scan(&note, &link); err != nil {
			t.Fatal(err)
		}
		return
	}

	if err := save(f.stafC, ptr("  Final di Drive Konten/Desember, versi B.  "), "https://drive.contoh.id/b"); err != nil {
		t.Fatalf("PIC sub task dari luar task menyimpan hasil: %v", err)
	}
	if note, link := stored(); note == nil || *note != "Final di Drive Konten/Desember, versi B." || link != "https://drive.contoh.id/b" {
		t.Fatalf("catatan dan tautan harus tersimpan bersama dan dirapikan, dapat %v %q", note, link)
	}

	// Klien lama dan form Ubah sub task tidak mengirim result_note: isi lama dipertahankan.
	if err := save(f.stafC, nil, "https://drive.contoh.id/b"); err != nil {
		t.Fatal(err)
	}
	if note, _ := stored(); note == nil {
		t.Fatal("result_note yang tidak dikirim tidak boleh menghapus catatan lama")
	}

	if err := save(f.stafC, ptr("   "), ""); err != nil {
		t.Fatal(err)
	}
	if note, _ := stored(); note != nil {
		t.Fatalf("catatan kosong harus disimpan NULL, dapat %q", *note)
	}

	var ve *apperr.Validation
	if err := save(f.stafC, ptr(strings.Repeat("a", 2001)), ""); !errors.As(err, &ve) {
		t.Fatalf("2001 karakter: err %v, mau Validation", err)
	}
	if err := save(f.stafC, ptr(strings.Repeat("é", 2000)), ""); err != nil {
		t.Fatalf("2000 karakter (dihitung per huruf, bukan byte) harus diterima: %v", err)
	}
	if err := save(f.stafC, ptr("ok"), "javascript:alert(1)"); !errors.As(err, &ve) {
		t.Fatalf("tautan javascript: err %v, mau Validation (cleanURL)", err)
	}

	if err := save(outsider, ptr("bukan urusanku"), ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("orang tanpa hak: err %v, mau ErrForbidden", err)
	}

	// Berkas sub task: PIC sub task dari luar task lolos pemeriksaan awal unggahan, orang tanpa hak tidak.
	att := NewAttachments(f.pool, nil)
	for _, kind := range []AttachmentKind{KindDeliverable, KindBrief} {
		if err := att.CheckUpload(ctx, f.stafC, id, kind, sub); err != nil {
			t.Errorf("PIC sub task mengunggah %s ke sub task-nya: %v", kind, err)
		}
		if err := att.CheckUpload(ctx, outsider, id, kind, sub); !errors.Is(err, ErrForbidden) {
			t.Errorf("orang tanpa hak mengunggah %s: err %v, mau ErrForbidden", kind, err)
		}
	}
	// Brief task induk tetap hanya Manajer.
	if err := att.CheckUpload(ctx, f.stafA, id, KindBrief, 0); !errors.Is(err, ErrForbidden) {
		t.Errorf("PIC task mengunggah brief task induk: err %v, mau ErrForbidden", err)
	}
}
