package driveexport

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/storage"
	"nvm-internal/internal/testdb"
)

type fixture struct {
	pool  *pgxpool.Pool
	svc   *Service
	store *storage.Store
}

func setup(t *testing.T) fixture {
	t.Helper()
	// Pohon staging berisi symlink, jadi test ini juga dilewati di sistem yang tidak mengizinkannya.
	butuhSymlink(t)
	pool := testdb.Pool(t)
	root := t.TempDir()
	store, err := storage.Open(filepath.Join(root, "storage"))
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(pool, store, filepath.Join(root, "staging"), "", log)
	return fixture{pool: pool, svc: svc, store: store}
}

// file membuat berkas di storage seperti hasil unggahan sungguhan, supaya symlink-nya tidak menggantung.
func (f fixture) file(t *testing.T, id string) {
	t.Helper()
	p, err := f.store.PathFor(id)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("isi berkas uji"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (f fixture) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

const idBrief = "11111111-1111-4111-8111-111111111111"
const idHapus = "22222222-2222-4222-8222-222222222222"
const idSub = "33333333-3333-4333-8333-333333333333"

// seed membuat satu task dengan tiga lampiran: brief, satu yang ditandai terhapus, dan satu di sub task.
func (f fixture) seed(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	var div, user int64
	if err := f.pool.QueryRow(ctx,
		`INSERT INTO divisions (name, is_active) VALUES ('Marketing', true) RETURNING id`).Scan(&div); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO users (name, username, password_hash, role, division_id, is_active, must_change_password)
		VALUES ('Dina', 'dina', 'x', 'staff', $1, true, false) RETURNING id`, div).Scan(&user); err != nil {
		t.Fatal(err)
	}
	f.exec(t, `
		INSERT INTO tasks (id, title, division_id, requested_by, urgency, created_at)
		VALUES ('T-0012', 'Foto katalog', $1, $2, 'high', '2026-11-02 09:00+07')`, div, user)
	var sub int64
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO task_subtasks (task_id, title, position) VALUES ('T-0012', 'Edit video', 0) RETURNING id`,
	).Scan(&sub); err != nil {
		t.Fatal(err)
	}
	f.exec(t, `INSERT INTO subtask_assignees (subtask_id, user_id, assigned_by) VALUES ($1, $2, $2)`, sub, user)

	add := func(id, kind, name string, subtask *int64, deleted bool) {
		f.exec(t, `
			INSERT INTO task_attachments (id, task_id, subtask_id, kind, original_name, mime_type, size_bytes, checksum, uploaded_by, is_deleted)
			VALUES ($1, 'T-0012', $2, $3, $4, 'application/pdf', 10, repeat('a', 64), $5, $6)`,
			id, subtask, kind, name, user, deleted)
		f.file(t, id)
	}
	add(idBrief, "brief", "brief.pdf", nil, false)
	add(idHapus, "deliverable", "dibuang.pdf", nil, true)
	add(idSub, "deliverable", "hasil.pdf", &sub, false)
}

func TestStageMenyusunPathSetiapJenis(t *testing.T) {
	f := setup(t)
	f.seed(t)
	ctx := context.Background()

	m, err := f.svc.Stage(ctx, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	paths := m.Listing()
	mau := []string{
		"Task/2026-11/T-0012 Foto katalog/Brief/brief.pdf",
		"Task/2026-11/T-0012 Foto katalog/Sub task 1 - Edit video (Dina)/hasil.pdf",
	}
	for _, p := range mau {
		if !slices.Contains(paths, p) {
			t.Fatalf("path %q tidak ada di %+v", p, paths)
		}
	}
	// Lampiran yang ditandai terhapus tidak diunggah lagi.
	for _, p := range paths {
		if filepath.Base(p) == "dibuang.pdf" {
			t.Fatalf("lampiran is_deleted ikut disusun: %q", p)
		}
	}
	// Ekspor Excel bulanan selalu ikut, dan tidak dicatat sebagai berkas sekali unggah.
	if len(m.Extra) != 1 {
		t.Fatalf("berkas yang ditimpa %+v, mau satu ekspor bulanan", m.Extra)
	}
	// Pohon staging benar-benar ada, dan isi berkasnya terbaca lewat symlink-nya (yang dipakai rclone).
	for _, p := range mau {
		if _, err := os.Stat(filepath.Join(f.svc.staging, filepath.FromSlash(p))); err != nil {
			t.Fatalf("berkas staging %q: %v", p, err)
		}
	}
}

func TestRecordMembuatBerkasTidakDisusunLagi(t *testing.T) {
	f := setup(t)
	f.seed(t)
	ctx := context.Background()

	m, err := f.svc.Stage(ctx, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Files) != 2 {
		t.Fatalf("berkas baru %d, mau 2", len(m.Files))
	}
	if err := f.svc.Record(ctx, m); err != nil {
		t.Fatal(err)
	}
	// Staging dibuang setelah dicatat.
	if _, err := os.Stat(f.svc.staging); !os.IsNotExist(err) {
		t.Fatal("staging harus dibuang setelah record")
	}

	kedua, err := f.svc.Stage(ctx, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(kedua.Files) != 0 {
		t.Fatalf("berkas yang sudah tercatat disusun lagi: %+v", kedua.Listing())
	}
	// Ekspor bulanan tetap disusun, karena memang ditimpa setiap malam.
	if len(kedua.Extra) != 1 {
		t.Fatalf("ekspor bulanan tidak ikut di run kedua: %+v", kedua.Extra)
	}

	var n int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM drive_exports`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("baris drive_exports %d, mau 2", n)
	}
	var ok bool
	var files int
	if err := f.pool.QueryRow(ctx,
		`SELECT ok, files FROM drive_export_runs ORDER BY id DESC LIMIT 1`).Scan(&ok, &files); err != nil {
		t.Fatal(err)
	}
	if !ok || files != 3 {
		t.Fatalf("baris status ok=%v files=%d, mau true dan 3", ok, files)
	}
}

func TestGantiJudulTidakMengubahPath(t *testing.T) {
	f := setup(t)
	f.seed(t)
	ctx := context.Background()

	m, err := f.svc.Stage(ctx, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Record(ctx, m); err != nil {
		t.Fatal(err)
	}

	// Judul task dan sub task diganti, lalu satu lampiran baru ditambahkan.
	f.exec(t, `UPDATE tasks SET title = 'Foto katalog revisi besar' WHERE id = 'T-0012'`)
	f.exec(t, `UPDATE task_subtasks SET title = 'Edit video versi dua' WHERE task_id = 'T-0012'`)
	const idBaru = "44444444-4444-4444-8444-444444444444"
	var user int64
	if err := f.pool.QueryRow(ctx, `SELECT id FROM users LIMIT 1`).Scan(&user); err != nil {
		t.Fatal(err)
	}
	f.exec(t, `
		INSERT INTO task_attachments (id, task_id, kind, original_name, mime_type, size_bytes, checksum, uploaded_by)
		VALUES ($1, 'T-0012', 'brief', 'brief.pdf', 'application/pdf', 10, repeat('b', 64), $2)`, idBaru, user)
	f.file(t, idBaru)

	lagi, err := f.svc.Stage(ctx, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(lagi.Files) != 1 {
		t.Fatalf("berkas baru %d, mau 1", len(lagi.Files))
	}
	got := lagi.Files[0].Path
	// Folder lama dipakai lagi, dan nama berkas yang bentrok diberi nomor.
	if got != "Task/2026-11/T-0012 Foto katalog/Brief/brief (2).pdf" {
		t.Fatalf("path berkas baru %q", got)
	}
}
