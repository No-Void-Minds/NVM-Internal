package driveexport

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/activity"
	"nvm-internal/internal/db/sqlc"
	"nvm-internal/internal/storage"
)

const (
	kindTaskFile     = "task_attachment"
	kindActivityFile = "activity_attachment"
	// manifestName ada di dalam staging dan tidak ikut diunggah (rclone memakai --exclude).
	manifestName = ".nvm-drive-manifest.json"
)

// ManifestFile adalah satu berkas yang akan dicatat di drive_exports setelah unggahan berhasil.
type ManifestFile struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// Manifest ditulis saat staging disusun dan dibaca lagi setelah rclone selesai. Pemisahan ini yang membuat
// aturan "setiap berkas hanya diunggah sekali" tidak berubah jadi "hanya dicoba sekali": baris drive_exports
// baru ditulis setelah unggahan benar-benar berhasil.
type Manifest struct {
	Files []ManifestFile `json:"files"`
	// Extra adalah berkas yang memang diunggah ulang setiap malam (ekspor bulanan dan dump), jadi tidak dicatat.
	Extra []string `json:"extra"`
	Bytes int64    `json:"bytes"`
}

type Service struct {
	pool       *pgxpool.Pool
	store      *storage.Store
	activities *activity.Service
	staging    string
	// recipient adalah kunci publik age untuk dump database. Kosong berarti dump tidak diunggah.
	recipient string
	log       *slog.Logger
	now       func() time.Time
}

func NewService(pool *pgxpool.Pool, store *storage.Store, staging, recipient string, log *slog.Logger) *Service {
	return &Service{
		pool: pool, store: store, activities: activity.NewService(pool, store),
		staging: staging, recipient: recipient, log: log, now: time.Now,
	}
}

// Stage menyusun pohon staging: hardlink untuk berkas aplikasi (tidak memakan disk karena satu filesystem
// dengan storage), ekspor Excel bulan berjalan, dan dump yang datang sebagai tar di stdin.
// dryRun tidak menulis apa pun ke database, jadi strukturnya bisa diperiksa dulu tanpa menetapkan nama folder.
func (s *Service) Stage(ctx context.Context, dumps io.Reader, dryRun bool) (Manifest, error) {
	var m Manifest
	if err := os.RemoveAll(s.staging); err != nil {
		return m, fmt.Errorf("folder staging lama tidak bisa dibuang: %w", err)
	}
	if err := os.MkdirAll(s.staging, 0o750); err != nil {
		return m, fmt.Errorf("folder staging tidak bisa dibuat: %w", err)
	}

	taken := newTakenCache(s.pool)
	if err := s.stageTaskFiles(ctx, &m, taken, dryRun); err != nil {
		return m, err
	}
	if err := s.stageActivityFiles(ctx, &m, taken); err != nil {
		return m, err
	}
	if err := s.stageMonthlyExport(ctx, &m); err != nil {
		return m, err
	}
	if dumps != nil {
		if err := s.stageDumps(&m, dumps); err != nil {
			return m, err
		}
	}
	return m, s.writeManifest(m)
}

func (s *Service) stageTaskFiles(ctx context.Context, m *Manifest, taken *takenCache, dryRun bool) error {
	rows, err := sqlc.New(s.pool).ListTaskFilesForDrive(ctx)
	if err != nil {
		return fmt.Errorf("daftar lampiran task: %w", err)
	}
	q := sqlc.New(s.pool)
	for _, r := range rows {
		folder, err := s.taskFolderPath(ctx, q, r, dryRun)
		if err != nil {
			return err
		}
		if r.SubtaskID.Valid {
			sub, err := s.subtaskFolderPath(ctx, q, folder, r, taken, dryRun)
			if err != nil {
				return err
			}
			folder = sub
		} else if r.Kind == sqlc.AttachmentKindBrief {
			folder += "/Brief"
		} else {
			folder += "/Deliverable"
		}
		name, err := taken.pick(ctx, folder, CleanFile(r.OriginalName))
		if err != nil {
			return err
		}
		s.link(m, kindTaskFile, r.ID, folder+"/"+name, r.SizeBytes)
	}
	return nil
}

// taskFolderPath memakai nama yang sudah tercatat bila ada, jadi judul task yang diganti tidak membuat
// folder baru di Drive (keputusan revisi 25).
func (s *Service) taskFolderPath(ctx context.Context, q *sqlc.Queries, r sqlc.ListTaskFilesForDriveRow, dryRun bool) (string, error) {
	saved, err := q.GetDriveFolder(ctx, sqlc.GetDriveFolderParams{Scope: "task", SourceID: r.TaskID})
	if err == nil {
		return saved, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("folder task %s: %w", r.TaskID, err)
	}
	// ID task ada di nama folder, jadi dua task tidak mungkin bentrok walau judulnya sama.
	folder := TaskFolder(r.TaskCreatedAt.Time, r.TaskID, r.TaskTitle)
	if dryRun {
		return folder, nil
	}
	err = q.InsertDriveFolder(ctx, sqlc.InsertDriveFolderParams{Scope: "task", SourceID: r.TaskID, FolderPath: folder})
	return folder, err
}

func (s *Service) subtaskFolderPath(ctx context.Context, q *sqlc.Queries, taskFolder string, r sqlc.ListTaskFilesForDriveRow, taken *takenCache, dryRun bool) (string, error) {
	id := fmt.Sprintf("%d", r.SubtaskID.Int64)
	saved, err := q.GetDriveFolder(ctx, sqlc.GetDriveFolderParams{Scope: "subtask", SourceID: id})
	if err == nil {
		return saved, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("folder sub task %s: %w", id, err)
	}
	name := SubtaskFolderName(int(r.SubtaskPosition.Int32), r.SubtaskTitle.String, r.SubtaskPic)
	unique, err := taken.pickFolder(ctx, taskFolder, name)
	if err != nil {
		return "", err
	}
	folder := taskFolder + "/" + unique
	if dryRun {
		return folder, nil
	}
	err = q.InsertDriveFolder(ctx, sqlc.InsertDriveFolderParams{Scope: "subtask", SourceID: id, FolderPath: folder})
	return folder, err
}

func (s *Service) stageActivityFiles(ctx context.Context, m *Manifest, taken *takenCache) error {
	rows, err := sqlc.New(s.pool).ListActivityFilesForDrive(ctx)
	if err != nil {
		return fmt.Errorf("daftar lampiran aktivitas: %w", err)
	}
	for _, r := range rows {
		slotLabel := "jam lain"
		if i := int(r.Slot) - 1; i >= 0 && i < len(activity.Slots) {
			slotLabel = activity.Slots[i]
		}
		folder := ActivityDayDir(r.ActivityDate.Time) + "/" + Clean(r.UserName)
		name, err := taken.pick(ctx, folder, ActivityFileName(slotLabel, r.OriginalName))
		if err != nil {
			return err
		}
		s.link(m, kindActivityFile, r.ID, folder+"/"+name, r.SizeBytes)
	}
	return nil
}

// stageMonthlyExport menulis ekspor Excel bulan berjalan, memakai kode ekspor Aktivitas yang sudah ada.
// Berkas ini ditimpa setiap malam, jadi tidak dicatat di drive_exports.
func (s *Service) stageMonthlyExport(ctx context.Context, m *Manifest) error {
	now := s.now()
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	data, _, err := s.activities.Export(ctx, first.Format("2006-01-02"), now.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("ekspor Excel bulanan: %w", err)
	}
	rel := MonthlyExportPath(now)
	if err := s.write(rel, data); err != nil {
		return err
	}
	m.Extra = append(m.Extra, rel)
	m.Bytes += int64(len(data))
	return nil
}

// stageDumps membaca tar dari stdin dan menulis tiap dump terenkripsi age ke staging. Dump mengalir lewat pipe,
// jadi tidak ada salinan plaintext tambahan di disk dan container ini tidak perlu bisa membaca folder backup.
func (s *Service) stageDumps(m *Manifest, r io.Reader) error {
	if s.recipient == "" {
		s.log.Warn("dump tidak diunggah: DRIVE_BACKUP_AGE_RECIPIENT kosong")
		return nil
	}
	rec, err := age.ParseX25519Recipient(s.recipient)
	if err != nil {
		return fmt.Errorf("DRIVE_BACKUP_AGE_RECIPIENT tidak valid: %w", err)
	}
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("tar dump: %w", err)
		}
		if h.Typeflag != tar.TypeReg || !strings.HasSuffix(h.Name, ".dump") {
			continue
		}
		rel := DumpPath(h.Name)
		full, err := s.create(rel)
		if err != nil {
			return err
		}
		enc, err := age.Encrypt(full, rec)
		if err != nil {
			_ = full.Close()
			return fmt.Errorf("enkripsi dump: %w", err)
		}
		n, err := io.Copy(enc, tr)
		if err == nil {
			err = enc.Close()
		}
		if cerr := full.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return fmt.Errorf("tulis dump terenkripsi: %w", err)
		}
		m.Extra = append(m.Extra, rel)
		m.Bytes += n
	}
}

// link membuat hardlink dari storage ke staging. Berkas yang barisnya ada tetapi berkasnya hilang dari disk
// dicatat di log dan dilewati, supaya satu berkas hilang tidak membatalkan seluruh salinan malam itu.
func (s *Service) link(m *Manifest, kind, id, rel string, size int64) {
	src, err := s.store.PathFor(id)
	if err != nil {
		s.log.Warn("id lampiran tidak valid, dilewati", "kind", kind, "id", id, "err", err)
		return
	}
	dst := filepath.Join(s.staging, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		s.log.Warn("folder staging tidak bisa dibuat, dilewati", "path", rel, "err", err)
		return
	}
	if err := os.Link(src, dst); err != nil {
		s.log.Warn("berkas tidak bisa ditautkan, dilewati", "kind", kind, "id", id, "err", err)
		return
	}
	m.Files = append(m.Files, ManifestFile{Kind: kind, ID: id, Path: rel, Size: size})
	m.Bytes += size
}

func (s *Service) create(rel string) (*os.File, error) {
	dst := filepath.Join(s.staging, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return nil, err
	}
	return os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
}

func (s *Service) write(rel string, data []byte) error {
	f, err := s.create(rel)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func (s *Service) writeManifest(m Manifest) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return s.write(manifestName, data)
}

func (s *Service) ReadManifest() (Manifest, error) {
	var m Manifest
	data, err := os.ReadFile(filepath.Join(s.staging, manifestName))
	if err != nil {
		return m, fmt.Errorf("manifest staging tidak terbaca: %w", err)
	}
	return m, json.Unmarshal(data, &m)
}

// Record dijalankan setelah rclone selesai: baris drive_exports ditulis, status run disimpan untuk halaman
// Status, lalu staging dibuang.
func (s *Service) Record(ctx context.Context, m Manifest) error {
	q := sqlc.New(s.pool)
	for _, f := range m.Files {
		if err := q.InsertDriveExport(ctx, sqlc.InsertDriveExportParams{
			SourceKind: f.Kind, SourceID: f.ID, DrivePath: f.Path, SizeBytes: f.Size,
		}); err != nil {
			return fmt.Errorf("catat %s: %w", f.Path, err)
		}
	}
	total := len(m.Files) + len(m.Extra)
	if err := q.InsertDriveExportRun(ctx, sqlc.InsertDriveExportRunParams{
		Ok: true, Files: int32(total), Bytes: m.Bytes,
	}); err != nil {
		return err
	}
	return os.RemoveAll(s.staging)
}

// Fail hanya menulis baris status, supaya halaman Status menunjukkan kegagalan tanpa harus membuka log.
// Staging dibiarkan untuk diperiksa; run berikutnya membuangnya.
func (s *Service) Fail(ctx context.Context, message string) error {
	return sqlc.New(s.pool).InsertDriveExportRun(ctx, sqlc.InsertDriveExportRunParams{
		Ok: false, Message: pgtype.Text{String: message, Valid: message != ""},
	})
}

// Listing adalah daftar path untuk uji kering, urut supaya mudah dibaca.
func (m Manifest) Listing() []string {
	out := make([]string, 0, len(m.Files)+len(m.Extra))
	for _, f := range m.Files {
		out = append(out, f.Path)
	}
	out = append(out, m.Extra...)
	return out
}

// takenCache menyimpan nama yang sudah dipakai per folder, dari database (berkas yang sudah terunggah
// sebelumnya) dan dari run ini sendiri.
type takenCache struct {
	pool    *pgxpool.Pool
	files   map[string]map[string]bool
	folders map[string]map[string]bool
}

func newTakenCache(pool *pgxpool.Pool) *takenCache {
	return &takenCache{pool: pool, files: map[string]map[string]bool{}, folders: map[string]map[string]bool{}}
}

func (t *takenCache) pick(ctx context.Context, folder, name string) (string, error) {
	set, ok := t.files[folder]
	if !ok {
		rows, err := sqlc.New(t.pool).ListDrivePathsIn(ctx, pgtype.Text{String: folder + "/", Valid: true})
		if err != nil {
			return "", fmt.Errorf("nama berkas di %s: %w", folder, err)
		}
		set = map[string]bool{}
		for _, p := range rows {
			set[strings.ToLower(path.Base(p))] = true
		}
		t.files[folder] = set
	}
	unique := Unique(name, set)
	set[strings.ToLower(unique)] = true
	return unique, nil
}

func (t *takenCache) pickFolder(ctx context.Context, parent, name string) (string, error) {
	set, ok := t.folders[parent]
	if !ok {
		rows, err := sqlc.New(t.pool).ListDriveFolderPaths(ctx, sqlc.ListDriveFolderPathsParams{
			Scope: "subtask", Prefix: pgtype.Text{String: parent + "/", Valid: true},
		})
		if err != nil {
			return "", fmt.Errorf("nama folder di %s: %w", parent, err)
		}
		set = map[string]bool{}
		for _, p := range rows {
			set[strings.ToLower(path.Base(p))] = true
		}
		t.folders[parent] = set
	}
	unique := Unique(name, set)
	set[strings.ToLower(unique)] = true
	return unique, nil
}
