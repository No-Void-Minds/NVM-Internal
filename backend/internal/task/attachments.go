package task

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"nvm-internal/internal/apperr"
	"nvm-internal/internal/attachment"
	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
	"nvm-internal/internal/storage"
	"os"
)

type AttachmentKind string

const (
	KindBrief       AttachmentKind = "brief"
	KindDeliverable AttachmentKind = "deliverable"
)

// MaxAttachmentSize sama dengan batas CHECK di tabel task_attachments (FR-TASK-15).
const MaxAttachmentSize = 50 << 20

var ErrTooLarge = errors.New("berkas melebihi 50 MB")

func ParseKind(v string) (AttachmentKind, error) {
	switch k := AttachmentKind(v); k {
	case KindBrief, KindDeliverable:
		return k, nil
	default:
		return "", apperr.Invalid("Jenis lampiran harus brief atau deliverable.")
	}
}

type SavedAttachment struct {
	ID   string
	Kind AttachmentKind
	Name string
	MIME string
	Size int64
}

type Attachments struct {
	pool  *pgxpool.Pool
	store *storage.Store
}

func NewAttachments(pool *pgxpool.Pool, store *storage.Store) *Attachments {
	return &Attachments{pool: pool, store: store}
}

// canAttach: brief hanya Manajer (FR-TASK-23); deliverable Manajer atau Staf yang menjadi PIC (BR-02).
func canAttach(ctx context.Context, q *sqlc.Queries, actor Actor, id string, kind AttachmentKind) (bool, error) {
	if kind == KindBrief {
		return actor.Role == RoleManager, nil
	}
	return canContribute(ctx, q, actor, id)
}

// CheckUpload dipanggil sebelum isi berkas dibaca, supaya unggahan yang pasti ditolak tidak sempat mengirim
// 50 MB lewat jaringan dan tidak menyentuh disk.
func (a *Attachments) CheckUpload(ctx context.Context, actor Actor, taskID string, kind AttachmentKind) error {
	q := sqlc.New(a.pool)
	active, err := q.TaskIsActive(ctx, taskID)
	if err != nil {
		return err
	}
	if !active {
		return ErrNotFound
	}
	ok, err := canAttach(ctx, q, actor, taskID, kind)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

// Upload menyimpan satu berkas: ditulis dulu ke tmp/ sambil dihitung SHA-256-nya, diperiksa tipenya, lalu
// dipindah ke attachments/ di dalam transaksi yang sama dengan baris database dan audit-nya.
func (a *Attachments) Upload(ctx context.Context, actor Actor, taskID string, kind AttachmentKind, originalName string, r io.Reader) (SavedAttachment, error) {
	if err := a.CheckUpload(ctx, actor, taskID, kind); err != nil {
		return SavedAttachment{}, err
	}
	tmp, err := a.store.CreateTemp()
	if err != nil {
		return SavedAttachment{}, err
	}
	tmpPath := tmp.Name()
	// Membersihkan sisa unggahan yang gagal di tahap mana pun. Setelah Commit berhasil, path ini sudah tidak ada.
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	h := sha256.New()
	// Satu byte di atas batas cukup untuk tahu berkasnya kebesaran, tanpa membaca sisanya.
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(r, MaxAttachmentSize+1))
	if err != nil {
		return SavedAttachment{}, err
	}
	if n > MaxAttachmentSize {
		return SavedAttachment{}, ErrTooLarge
	}

	name := attachment.CleanName(originalName)
	mime, err := attachment.Detect(tmp, n, name)
	if err != nil {
		return SavedAttachment{}, err
	}
	if err := tmp.Close(); err != nil {
		return SavedAttachment{}, err
	}

	saved := SavedAttachment{Kind: kind, Name: name, MIME: mime, Size: n}
	err = pgx.BeginFunc(ctx, a.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		// Dicek ulang di dalam transaksi: task bisa dihapus atau PIC diganti selama berkas diunggah.
		if _, err := q.GetTaskForUpdate(ctx, taskID); errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		ok, err := canAttach(ctx, q, actor, taskID, kind)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		id, err := q.InsertAttachment(ctx, sqlc.InsertAttachmentParams{
			TaskID:       taskID,
			Kind:         sqlc.AttachmentKind(kind),
			OriginalName: name,
			MimeType:     mime,
			SizeBytes:    n,
			Checksum:     hex.EncodeToString(h.Sum(nil)),
			UploadedBy:   actor.ID,
		})
		if err != nil {
			return err
		}
		saved.ID = id
		if err := audit.Change(ctx, q, actor.auditActor(), "attachment_added", "task", taskID, nil, map[string]any{
			"attachment_id": id, "kind": kind, "name": name, "size_bytes": n,
		}); err != nil {
			return err
		}
		// Dipindah paling akhir: kalau gagal, transaksi dibatalkan sehingga tidak ada baris tanpa berkas.
		return a.store.Commit(tmpPath, id)
	})
	if err != nil {
		return SavedAttachment{}, err
	}
	return saved, nil
}

var ErrAttachmentNotFound = errors.New("lampiran tidak ditemukan")

// uuidOf mengubah ID teks dari URL menjadi tipe UUID untuk query. Teks yang bukan UUID diperlakukan sama
// dengan lampiran yang tidak ada.
func uuidOf(id string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(id); err != nil {
		return u, ErrAttachmentNotFound
	}
	return u, nil
}

// List mengembalikan lampiran task yang belum dihapus, urut dari yang paling lama diunggah.
func (a *Attachments) List(ctx context.Context, taskID string) ([]sqlc.ListTaskAttachmentsRow, error) {
	return sqlc.New(a.pool).ListTaskAttachments(ctx, taskID)
}

// Open membuka berkas untuk diunduh. Izinnya sama dengan membaca detail task (FR-TASK-18), jadi tidak ada
// pengecekan peran di sini; yang dicek hanya lampiran itu memang milik task ini dan belum dihapus.
func (a *Attachments) Open(ctx context.Context, taskID, attachmentID string) (sqlc.GetAttachmentRow, *os.File, error) {
	uid, err := uuidOf(attachmentID)
	if err != nil {
		return sqlc.GetAttachmentRow{}, nil, err
	}
	row, err := sqlc.New(a.pool).GetAttachment(ctx, sqlc.GetAttachmentParams{ID: uid, TaskID: taskID})
	if errors.Is(err, pgx.ErrNoRows) {
		return row, nil, ErrAttachmentNotFound
	}
	if err != nil {
		return row, nil, err
	}
	p, err := a.store.PathFor(row.ID)
	if err != nil {
		return row, nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		// Baris ada tetapi berkasnya tidak: tanda masalah disk atau restore, bukan "tidak ditemukan" biasa.
		return row, nil, fmt.Errorf("berkas lampiran %s tidak terbaca dari disk: %w", row.ID, err)
	}
	return row, f, nil
}

// Delete menandai lampiran terhapus. Berkasnya tetap di disk supaya bisa dipulihkan, sama seperti task (FR-TASK-16).
// Yang boleh menghapus sama dengan yang boleh mengunggah jenis lampiran itu.
func (a *Attachments) Delete(ctx context.Context, actor Actor, taskID, attachmentID string) error {
	uid, err := uuidOf(attachmentID)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, a.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if _, err := q.GetTaskForUpdate(ctx, taskID); errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		row, err := q.GetAttachment(ctx, sqlc.GetAttachmentParams{ID: uid, TaskID: taskID})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAttachmentNotFound
		}
		if err != nil {
			return err
		}
		ok, err := canAttach(ctx, q, actor, taskID, AttachmentKind(row.Kind))
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if err := q.SoftDeleteAttachment(ctx, sqlc.SoftDeleteAttachmentParams{ID: uid, TaskID: taskID}); err != nil {
			return err
		}
		return audit.Change(ctx, q, actor.auditActor(), "attachment_deleted", "task", taskID,
			map[string]any{"attachment_id": row.ID, "kind": row.Kind, "name": row.OriginalName}, nil)
	})
}
