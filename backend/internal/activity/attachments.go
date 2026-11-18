package activity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/attachment"
	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
)

// MaxFileSize 95 MB: Cloudflare menolak unggahan di atas 100 MB, jadi batasnya di bawah itu supaya penolakan
// datang dari aplikasi dengan pesan yang jelas, bukan dari Cloudflare dengan halaman galatnya sendiri.
const MaxFileSize = 95 << 20

// MaxFilesPerSlot menjaga satu jam tidak dipakai sebagai gudang berkas.
const MaxFilesPerSlot = 10

var (
	ErrTooLarge    = errors.New("berkas melebihi 95 MB")
	ErrFileMissing = errors.New("lampiran tidak ditemukan")
)

// uuidOf mengubah ID teks dari URL menjadi tipe UUID. Teks yang bukan UUID diperlakukan sama dengan lampiran
// yang tidak ada, jadi URL acak tidak pernah menjadi galat 500.
func uuidOf(id string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(id); err != nil {
		return u, ErrFileMissing
	}
	return u, nil
}

// FileInfo dikirim apa adanya sebagai JSON di dalam sel.
type FileInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	MIME string `json:"mime_type"`
	Size int64  `json:"size_bytes"`
	// Nama pengunggah, dipakai keterangan kecil di sel bila bukan pemilik jamnya.
	UploadedBy string `json:"uploaded_by"`
}

// requireStaffOwner memastikan jam yang disentuh memang milik akun Staf yang aktif. Dipakai isian aktivitas
// dan lampiran, supaya keduanya tidak bisa berbeda aturan.
func requireStaffOwner(ctx context.Context, q *sqlc.Queries, userID int64) (sqlc.GetUserForUpdateRow, error) {
	owner, err := q.GetUserForUpdate(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return owner, apperr.Invalid("Akun tidak ditemukan.")
	}
	if err != nil {
		return owner, err
	}
	// Tabel aktivitas hanya ada untuk Staf, jadi Manajer pun tidak bisa menitipkan isian ke Manajer lain.
	if owner.Role != sqlc.UserRoleStaff {
		return owner, apperr.Invalid("Isian aktivitas hanya ada untuk akun Staf.")
	}
	if !owner.IsActive {
		return owner, apperr.Invalid("Akun ini sudah nonaktif, isiannya tidak bisa diubah.")
	}
	return owner, nil
}

// CheckUpload dipanggil sebelum isi berkas dibaca, supaya unggahan yang pasti ditolak tidak sempat mengirim
// 95 MB lewat jaringan dan tidak menyentuh disk. Haknya sama dengan isian aktivitas (CanEdit).
func (s *Service) CheckUpload(ctx context.Context, actor Actor, userID int64, date string, slot int) (string, error) {
	day, err := s.ParseDay(date)
	if err != nil {
		return "", err
	}
	if slot < 1 || slot > len(Slots) {
		return "", apperr.Invalid("Slot jam tidak dikenal.")
	}
	if !CanEdit(actor, userID, day, s.Today()) {
		return "", ErrForbidden
	}
	q := sqlc.New(s.pool)
	if _, err := requireStaffOwner(ctx, q, userID); err != nil {
		return "", err
	}
	pgDay, err := pgDate(day)
	if err != nil {
		return "", err
	}
	n, err := q.CountActivityAttachments(ctx, sqlc.CountActivityAttachmentsParams{
		UserID: userID, ActivityDate: pgDay, Slot: int16(slot),
	})
	if err != nil {
		return "", err
	}
	if n >= MaxFilesPerSlot {
		return "", apperr.Invalid(fmt.Sprintf("Satu jam maksimal %d berkas. Hapus salah satu dulu.", MaxFilesPerSlot))
	}
	return day, nil
}

// Upload menyimpan satu berkas: ditulis dulu ke tmp/ sambil dihitung SHA-256-nya, diperiksa tipenya dari isi,
// lalu dipindah ke attachments/ di dalam transaksi yang sama dengan baris database dan audit-nya.
func (s *Service) Upload(ctx context.Context, actor Actor, userID int64, date string, slot int, originalName string, r io.Reader) (FileInfo, error) {
	day, err := s.CheckUpload(ctx, actor, userID, date, slot)
	if err != nil {
		return FileInfo{}, err
	}
	pgDay, err := pgDate(day)
	if err != nil {
		return FileInfo{}, err
	}

	tmp, err := s.files.CreateTemp()
	if err != nil {
		return FileInfo{}, err
	}
	tmpPath := tmp.Name()
	// Membersihkan sisa unggahan yang gagal di tahap mana pun. Setelah Commit berhasil, path ini sudah tidak ada.
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	h := sha256.New()
	// Satu byte di atas batas cukup untuk tahu berkasnya kebesaran, tanpa membaca sisanya.
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(r, MaxFileSize+1))
	if err != nil {
		return FileInfo{}, err
	}
	if n > MaxFileSize {
		return FileInfo{}, ErrTooLarge
	}
	name := attachment.CleanName(originalName)
	mime, err := attachment.DetectActivity(tmp, n, name)
	if err != nil {
		return FileInfo{}, err
	}
	if err := tmp.Close(); err != nil {
		return FileInfo{}, err
	}

	out := FileInfo{Name: name, MIME: mime, Size: n}
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		// Dicek ulang di dalam transaksi: akun bisa dinonaktifkan selama berkas diunggah.
		owner, err := requireStaffOwner(ctx, q, userID)
		if err != nil {
			return err
		}
		if !CanEdit(actor, userID, day, s.Today()) {
			return ErrForbidden
		}
		id, err := q.InsertActivityAttachment(ctx, sqlc.InsertActivityAttachmentParams{
			UserID: userID, ActivityDate: pgDay, Slot: int16(slot), OriginalName: name,
			MimeType: mime, SizeBytes: n, Checksum: hex.EncodeToString(h.Sum(nil)), UploadedBy: actor.ID,
		})
		if err != nil {
			return err
		}
		out.ID = id
		// Hanya unggahan Manajer pada jam orang lain yang dicatat, sama dengan isian aktivitas.
		if userID != actor.ID {
			if err := auditFile(ctx, q, actor, "activity_file_added", userID, owner.Name, day, slot, name, id); err != nil {
				return err
			}
		}
		// Dipindah paling akhir: kalau gagal, transaksi dibatalkan sehingga tidak ada baris tanpa berkas.
		return s.files.Commit(tmpPath, id)
	})
	if err != nil {
		return FileInfo{}, err
	}
	return out, nil
}

// Open membuka berkas untuk diunduh atau dipratinjau. Izinnya sama dengan membaca tabel Aktivitas, yang
// terbuka untuk semua pengguna yang login, jadi di sini tidak ada pemeriksaan peran.
func (s *Service) Open(ctx context.Context, id string) (sqlc.GetActivityAttachmentRow, *os.File, error) {
	uid, err := uuidOf(id)
	if err != nil {
		return sqlc.GetActivityAttachmentRow{}, nil, err
	}
	row, err := sqlc.New(s.pool).GetActivityAttachment(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return row, nil, ErrFileMissing
	}
	if err != nil {
		return row, nil, err
	}
	p, err := s.files.PathFor(row.ID)
	if err != nil {
		return row, nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		// Baris ada tetapi berkasnya tidak: tanda masalah disk atau restore, bukan "tidak ditemukan" biasa.
		return row, nil, fmt.Errorf("berkas lampiran aktivitas %s tidak terbaca dari disk: %w", row.ID, err)
	}
	return row, f, nil
}

// DeleteFile menghapus baris dan berkasnya. Beda dari lampiran task yang hanya ditandai terhapus: berkas di
// sini bisa 95 MB, jadi menyimpannya terus akan menghabiskan disk VM.
func (s *Service) DeleteFile(ctx context.Context, actor Actor, id string) error {
	uid, err := uuidOf(id)
	if err != nil {
		return err
	}
	row, err := sqlc.New(s.pool).GetActivityAttachment(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrFileMissing
	}
	if err != nil {
		return err
	}
	day := row.ActivityDate.Time.Format(dayLayout)
	if !CanEdit(actor, row.UserID, day, s.Today()) {
		return ErrForbidden
	}
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		qt := sqlc.New(tx)
		owner, err := requireStaffOwner(ctx, qt, row.UserID)
		if err != nil {
			return err
		}
		if err := qt.DeleteActivityAttachment(ctx, uid); err != nil {
			return err
		}
		if row.UserID == actor.ID {
			return nil
		}
		return auditFile(ctx, qt, actor, "activity_file_deleted", row.UserID, owner.Name, day, int(row.Slot), row.OriginalName, row.ID)
	})
	if err != nil {
		return err
	}
	// Berkasnya dibuang setelah barisnya hilang: kalau langkah ini gagal, yang tertinggal hanya berkas yatim,
	// yang dibersihkan job harian, bukan baris yang menunjuk berkas yang sudah tidak ada.
	p, err := s.files.PathFor(row.ID)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// fileRows mengambil semua lampiran di rentang tanggal. Pemanggilnya yang menempelkan ke sel, karena tampilan
// mingguan dan tampilan isian sendiri menyusun selnya berbeda.
func (s *Service) fileRows(ctx context.Context, from, to string) ([]sqlc.ListActivityAttachmentsRow, error) {
	fromDay, err := pgDate(from)
	if err != nil {
		return nil, err
	}
	toDay, err := pgDate(to)
	if err != nil {
		return nil, err
	}
	return sqlc.New(s.pool).ListActivityAttachments(ctx, sqlc.ListActivityAttachmentsParams{
		FromDate: fromDay, ToDate: toDay,
	})
}

func toFileInfo(r sqlc.ListActivityAttachmentsRow) FileInfo {
	f := FileInfo{ID: r.ID, Name: r.OriginalName, MIME: r.MimeType, Size: r.SizeBytes}
	// Nama pengunggah hanya berguna bila bukan pemilik jamnya, mis. Manajer yang menambahkan bukti.
	if r.UploadedByID != r.UserID {
		f.UploadedBy = r.UploadedByName
	}
	return f
}

// addFile menempelkan satu lampiran ke selnya. Slot di luar daftar jam diabaikan, mis. sisa data lama setelah
// daftar slot berubah.
func addFile(cells []Cell, r sqlc.ListActivityAttachmentsRow) {
	i := int(r.Slot) - 1
	if i < 0 || i >= len(cells) {
		return
	}
	cells[i].Files = append(cells[i].Files, toFileInfo(r))
}

// auditFile memakai entity dan entity_id yang sama dengan isian aktivitas (user:tanggal:slot), jadi satu sel
// bisa dilacak dari satu tempat di Audit Log.
func auditFile(ctx context.Context, q *sqlc.Queries, actor Actor, action string, userID int64, ownerName, date string, slot int, fileName, fileID string) error {
	detail := map[string]any{
		"name": ownerName, "date": date, "time": Slots[slot-1], "file": fileName, "attachment_id": fileID,
	}
	entity := fmt.Sprintf("%d:%s:%d", userID, date, slot)
	if action == "activity_file_deleted" {
		return audit.Change(ctx, q, audit.Actor{ID: actor.ID, IP: actor.IP}, action, "daily_activity", entity, detail, nil)
	}
	return audit.Change(ctx, q, audit.Actor{ID: actor.ID, IP: actor.IP}, action, "daily_activity", entity, nil, detail)
}
