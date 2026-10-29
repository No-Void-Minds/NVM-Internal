package task

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/attachment"
	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
	"nvm-internal/internal/storage"
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
