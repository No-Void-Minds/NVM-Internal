package task

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nvm-internal/internal/attachment"
	"nvm-internal/internal/storage"
)

func (f fixture) attachments(t *testing.T) (*Attachments, *storage.Store) {
	t.Helper()
	store, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return NewAttachments(f.pool, store), store
}

func tmpIsEmpty(t *testing.T, store *storage.Store) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(store.Root(), "tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("tmp/ masih berisi %d berkas sisa unggahan", len(entries))
	}
}

func (f fixture) countAttachments(t *testing.T, id string) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), `SELECT count(*) FROM task_attachments WHERE task_id = $1`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

var pdfContent = []byte("%PDF-1.7\n%brief")

func TestUploadSavesFileChecksumAndAudit(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	att, store := f.attachments(t)
	id := f.newTask(t, f.stafA.ID)

	saved, err := att.Upload(ctx, f.manager, id, KindBrief, `C:\Users\manajer\brief final.pdf`, bytes.NewReader(pdfContent))
	if err != nil {
		t.Fatal(err)
	}
	if saved.Name != "brief final.pdf" || saved.MIME != "application/pdf" || saved.Size != int64(len(pdfContent)) {
		t.Fatalf("hasil unggah %+v", saved)
	}

	p, err := store.PathFor(saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	onDisk, err := os.ReadFile(p)
	if err != nil || !bytes.Equal(onDisk, pdfContent) {
		t.Fatalf("isi berkas di disk tidak sama: %v", err)
	}

	var checksum string
	if err := f.pool.QueryRow(ctx, `SELECT checksum FROM task_attachments WHERE id = $1`, saved.ID).Scan(&checksum); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(pdfContent)
	if checksum != hex.EncodeToString(sum[:]) {
		t.Fatalf("checksum %s, mau %x", checksum, sum)
	}

	var audits int
	if err := f.pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_logs WHERE action = 'attachment_added' AND entity = 'task' AND entity_id = $1`, id,
	).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 1 {
		t.Fatalf("audit attachment_added tercatat %d kali, mau 1", audits)
	}
	tmpIsEmpty(t, store)
}

func TestUploadPermissions(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	att, _ := f.attachments(t)
	id := f.newTask(t, f.stafA.ID)

	cases := []struct {
		who   string
		actor Actor
		kind  AttachmentKind
		want  error
	}{
		{"Manajer brief", f.manager, KindBrief, nil},
		{"Manajer deliverable", f.manager, KindDeliverable, nil},
		{"Staf PIC deliverable", f.stafA, KindDeliverable, nil},
		{"Staf PIC brief", f.stafA, KindBrief, ErrForbidden},
		{"Staf bukan PIC deliverable", f.stafC, KindDeliverable, ErrForbidden},
		{"Admin brief", f.admin, KindBrief, ErrForbidden},
		{"Admin deliverable", f.admin, KindDeliverable, ErrForbidden},
	}
	for _, c := range cases {
		_, err := att.Upload(ctx, c.actor, id, c.kind, "berkas.pdf", bytes.NewReader(pdfContent))
		if !errors.Is(err, c.want) {
			t.Errorf("%s: err %v, mau %v", c.who, err, c.want)
		}
	}
}

type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func TestRejectedUploadLeavesNothingBehind(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	att, store := f.attachments(t)
	id := f.newTask(t, f.stafA.ID)

	tooBig := io.MultiReader(strings.NewReader("%PDF-1.7\n"), io.LimitReader(zeros{}, MaxAttachmentSize))
	if _, err := att.Upload(ctx, f.manager, id, KindBrief, "besar.pdf", tooBig); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("berkas di atas 50 MB: err %v, mau ErrTooLarge", err)
	}
	tmpIsEmpty(t, store)

	exe := append([]byte("MZ"), make([]byte, 64)...)
	var rejected *attachment.Rejected
	if _, err := att.Upload(ctx, f.manager, id, KindBrief, "invoice.pdf", bytes.NewReader(exe)); !errors.As(err, &rejected) {
		t.Fatalf("EXE diganti nama: err %v, mau Rejected", err)
	}
	tmpIsEmpty(t, store)

	if n := f.countAttachments(t, id); n != 0 {
		t.Fatalf("unggahan yang ditolak tetap tercatat: %d baris", n)
	}
}

func TestUploadToDeletedTask(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	att, _ := f.attachments(t)
	id := f.newTask(t, f.stafA.ID)
	if err := f.svc.Delete(ctx, f.manager, id, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := att.Upload(ctx, f.manager, id, KindBrief, "brief.pdf", bytes.NewReader(pdfContent)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unggah ke task terhapus: err %v, mau ErrNotFound", err)
	}
}

func TestOpenListAndSoftDelete(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	att, store := f.attachments(t)
	id := f.newTask(t, f.stafA.ID)
	other := f.newTask(t, f.stafA.ID)

	brief, err := att.Upload(ctx, f.manager, id, KindBrief, "brief.pdf", bytes.NewReader(pdfContent))
	if err != nil {
		t.Fatal(err)
	}
	result, err := att.Upload(ctx, f.stafA, id, KindDeliverable, "hasil.pdf", bytes.NewReader(pdfContent))
	if err != nil {
		t.Fatal(err)
	}

	list, err := att.List(ctx, id)
	if err != nil || len(list) != 2 || list[0].ID != brief.ID || list[0].UploadedByName == "" {
		t.Fatalf("daftar lampiran %+v, err %v", list, err)
	}

	_, file, err := att.Open(ctx, id, brief.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(file)
	_ = file.Close()
	if !bytes.Equal(got, pdfContent) {
		t.Fatal("isi unduhan tidak sama dengan yang diunggah")
	}
	if _, _, err := att.Open(ctx, other, brief.ID); !errors.Is(err, ErrAttachmentNotFound) {
		t.Fatalf("lampiran dibuka lewat task lain: err %v, mau ErrAttachmentNotFound", err)
	}

	deletes := []struct {
		who   string
		actor Actor
		aid   string
		want  error
	}{
		{"Staf PIC menghapus brief", f.stafA, brief.ID, ErrForbidden},
		{"Staf bukan PIC menghapus deliverable", f.stafC, result.ID, ErrForbidden},
		{"Admin menghapus brief", f.admin, brief.ID, ErrForbidden},
		{"Staf PIC menghapus deliverable", f.stafA, result.ID, nil},
		{"Manajer menghapus brief", f.manager, brief.ID, nil},
		{"Manajer menghapus lagi", f.manager, brief.ID, ErrAttachmentNotFound},
	}
	for _, d := range deletes {
		if err := att.Delete(ctx, d.actor, id, d.aid); !errors.Is(err, d.want) {
			t.Errorf("%s: err %v, mau %v", d.who, err, d.want)
		}
	}

	if list, _ := att.List(ctx, id); len(list) != 0 {
		t.Fatalf("lampiran terhapus masih tampil: %d", len(list))
	}
	if _, _, err := att.Open(ctx, id, brief.ID); !errors.Is(err, ErrAttachmentNotFound) {
		t.Fatalf("lampiran terhapus masih bisa diunduh: err %v", err)
	}
	// Soft delete: berkas tetap di disk supaya bisa dipulihkan.
	p, _ := store.PathFor(brief.ID)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("berkas ikut terhapus dari disk: %v", err)
	}

	var audits int
	if err := f.pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_logs WHERE action = 'attachment_deleted' AND entity_id = $1`, id,
	).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 2 {
		t.Fatalf("audit attachment_deleted tercatat %d kali, mau 2", audits)
	}
}

func TestAttachmentsHiddenWhenTaskDeleted(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	att, _ := f.attachments(t)
	id := f.newTask(t, f.stafA.ID)
	saved, err := att.Upload(ctx, f.manager, id, KindBrief, "brief.pdf", bytes.NewReader(pdfContent))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Delete(ctx, f.manager, id, 1); err != nil {
		t.Fatal(err)
	}
	if _, _, err := att.Open(ctx, id, saved.ID); !errors.Is(err, ErrAttachmentNotFound) {
		t.Fatalf("lampiran task terhapus masih bisa diunduh: err %v", err)
	}
}
