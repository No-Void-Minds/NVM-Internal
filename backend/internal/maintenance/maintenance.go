// Package maintenance berisi job harian S12-07: sesi kedaluwarsa, audit log login lama, berkas yatim, dan hapus
// permanen isi Tempat sampah Whiteboard yang lebih tua dari 30 hari.
// Dijalankan cron host lewat `docker compose run --rm backend maintenance [--dry-run]`.
package maintenance

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/auth"
	"nvm-internal/internal/db/sqlc"
	"nvm-internal/internal/storage"
	"nvm-internal/internal/whiteboard"
)

const (
	// AuditRetention: audit log login lebih tua dari ini dihapus (NFR-RET-03).
	AuditRetention = 365 * 24 * time.Hour
	// orphanGrace melindungi berkas yang sedang ditulis: lampiran sudah ada di disk sebelum transaksinya selesai,
	// dan foto ditulis sebelum database diubah.
	orphanGrace = time.Hour
)

// loginActions adalah jejak login yang boleh dihapus setelah 1 tahun. Ganti password dan perubahan akun tidak
// termasuk, karena itu catatan perubahan keamanan akun, bukan jejak masuk dan keluar.
var loginActions = []string{"login_success", "login_failed", "account_locked", "logout"}

type Report struct {
	DryRun            bool
	ExpiredSessions   int64
	OldLoginAuditLogs int64
	// WhiteboardBatches: jumlah batch Tempat sampah Whiteboard yang dihapus permanen (atau akan, saat dry run).
	WhiteboardBatches int
	OrphanAttachments []string
	OrphanAvatars     []string
	StaleUploads      []string
	// Unknown berisi nama berkas yang bukan bentuk nama aplikasi; hanya dilaporkan, tidak pernah dihapus.
	Unknown []string
}

func Run(ctx context.Context, pool *pgxpool.Pool, store *storage.Store, now time.Time, dryRun bool) (Report, error) {
	r := Report{DryRun: dryRun}
	q := sqlc.New(pool)
	ts := func(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

	ageBefore := ts(now.Add(-auth.SessionMaxAge))
	var err error
	if dryRun {
		r.ExpiredSessions, err = q.CountExpiredSessions(ctx, ageBefore)
	} else {
		r.ExpiredSessions, err = q.DeleteExpiredSessions(ctx, ageBefore)
	}
	if err != nil {
		return r, fmt.Errorf("sesi kedaluwarsa: %w", err)
	}

	before := ts(now.Add(-AuditRetention))
	if dryRun {
		r.OldLoginAuditLogs, err = q.CountOldAuditLogs(ctx, sqlc.CountOldAuditLogsParams{Actions: loginActions, Before: before})
	} else {
		r.OldLoginAuditLogs, err = q.DeleteOldAuditLogs(ctx, sqlc.DeleteOldAuditLogsParams{Actions: loginActions, Before: before})
	}
	if err != nil {
		return r, fmt.Errorf("audit log login: %w", err)
	}

	// Sebelum pembersih berkas yatim: berkas Whiteboard (W3) yang barisnya terhapus di sini ikut dibersihkan
	// pada run berikutnya, setelah orphanGrace.
	r.WhiteboardBatches, err = whiteboard.Purge(ctx, pool, now, dryRun)
	if err != nil {
		return r, fmt.Errorf("tempat sampah whiteboard: %w", err)
	}

	attachments, err := q.ListAttachmentFileIDs(ctx)
	if err != nil {
		return r, fmt.Errorf("daftar lampiran: %w", err)
	}
	avatars, err := q.ListAvatarFileIDs(ctx)
	if err != nil {
		return r, fmt.Errorf("daftar foto: %w", err)
	}
	cutoff := now.Add(-orphanGrace)

	// attachments/<2 huruf>/<uuid>, nama berkas persis id baris task_attachments.
	known := set(attachments)
	err = walkOld(store.AttachmentsDir(), cutoff, func(path, name string) error {
		switch {
		case !storage.ValidID(name):
			r.Unknown = append(r.Unknown, path)
		case !known[name]:
			r.OrphanAttachments = append(r.OrphanAttachments, path)
			return remove(path, dryRun)
		}
		return nil
	})
	if err != nil {
		return r, fmt.Errorf("berkas lampiran: %w", err)
	}

	// avatars/<uuid>.jpg dan sisa berkas sementara .upload-* dari unggahan yang terputus.
	known = set(avatars)
	err = walkOld(store.AvatarsDir(), cutoff, func(path, name string) error {
		id, isJPG := strings.CutSuffix(name, ".jpg")
		switch {
		case strings.HasPrefix(name, ".upload-"):
			r.StaleUploads = append(r.StaleUploads, path)
			return remove(path, dryRun)
		case !isJPG || !storage.ValidID(id):
			r.Unknown = append(r.Unknown, path)
		case !known[id]:
			r.OrphanAvatars = append(r.OrphanAvatars, path)
			return remove(path, dryRun)
		}
		return nil
	})
	if err != nil {
		return r, fmt.Errorf("berkas foto: %w", err)
	}
	return r, nil
}

// walkOld memanggil fn untuk setiap berkas biasa yang lebih tua dari cutoff. Folder yang tidak bisa dibaca
// menggagalkan job, supaya masalah izin tidak tersembunyi (pelajaran backup compro).
func walkOld(root string, cutoff time.Time, fn func(path, name string) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.ModTime().After(cutoff) {
			return nil
		}
		return fn(path, d.Name())
	})
}

func remove(path string, dryRun bool) error {
	if dryRun {
		return nil
	}
	return os.Remove(path)
}

func set(ids []string) map[string]bool {
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}
