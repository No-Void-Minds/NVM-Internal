package driveexport

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

// Ringkasan harus menjelaskan kenapa sebuah berkas tidak ikut disalin. Sebelum ringkasan ini ada, job yang
// tidak menyalin apa pun terlihat sama dengan job yang berhasil: di produksi ada 6 lampiran aktif dan nol
// yang masuk staging, tanpa satu pun alasan tercetak.
func TestRingkasanMenjelaskanYangDilewati(t *testing.T) {
	butuhSymlink(t)
	f := setup(t)
	f.seed(t)
	ctx := context.Background()

	var div, user int64
	if err := f.pool.QueryRow(ctx, `SELECT division_id, id FROM users LIMIT 1`).Scan(&div, &user); err != nil {
		t.Fatal(err)
	}
	// Task yang sudah dihapus: lampirannya tidak disalin, tetapi alasannya harus terbaca.
	f.exec(t, `
		INSERT INTO tasks (id, title, division_id, requested_by, urgency, is_deleted, created_at)
		VALUES ('T-0013', 'Task dibuang', $1, $2, 'low', true, '2026-11-03 09:00+07')`, div, user)
	const idTaskHapus = "66666666-6666-4666-8666-666666666666"
	f.exec(t, `
		INSERT INTO task_attachments (id, task_id, kind, original_name, mime_type, size_bytes, checksum, uploaded_by)
		VALUES ($1, 'T-0013', 'brief', 'ikut.pdf', 'application/pdf', 10, repeat('c', 64), $2)`, idTaskHapus, user)
	f.file(t, idTaskHapus)

	// Lampiran yang barisnya ada tetapi berkasnya tidak ada di storage (mis. setelah restore sebagian).
	const idTanpaBerkas = "77777777-7777-4777-8777-777777777777"
	f.exec(t, `
		INSERT INTO task_attachments (id, task_id, kind, original_name, mime_type, size_bytes, checksum, uploaded_by)
		VALUES ($1, 'T-0012', 'deliverable', 'hilang.pdf', 'application/pdf', 10, repeat('d', 64), $2)`,
		idTanpaBerkas, user)

	m, err := f.svc.Stage(ctx, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Jenis dihitung per kelompok, dan jenis yang kosong tetap muncul sebagai nol.
	mau := map[string]int{
		KindBrief: 1, KindDeliverable: 0, KindSubtask: 1, KindActivity: 0, KindExport: 1, KindDump: 0,
	}
	for kind, n := range mau {
		got, ada := m.Counts[kind]
		if !ada {
			t.Errorf("jenis %q tidak ada di ringkasan", kind)
			continue
		}
		if got != n {
			t.Errorf("jenis %q dihitung %d, mau %d", kind, got, n)
		}
	}

	if m.Skipped["task sudah dihapus"] != 1 {
		t.Errorf("lampiran di task yang dihapus harus terhitung sekali, dapat %d", m.Skipped["task sudah dihapus"])
	}
	if m.Skipped["lampiran sudah dihapus"] != 1 {
		t.Errorf("lampiran is_deleted harus terhitung sekali, dapat %d", m.Skipped["lampiran sudah dihapus"])
	}
	if m.Skipped["berkas hilang di storage"] != 1 {
		t.Errorf("berkas yang hilang harus terhitung sekali, dapat %d", m.Skipped["berkas hilang di storage"])
	}
	for _, p := range m.Listing() {
		if strings.Contains(p, "ikut.pdf") || strings.Contains(p, "hilang.pdf") {
			t.Fatalf("berkas yang harus dilewati ikut disalin: %q", p)
		}
	}

	// Setelah dicatat, alasan "sudah tercatat" yang muncul, bukan nol tanpa keterangan.
	if err := f.svc.Record(ctx, m); err != nil {
		t.Fatal(err)
	}
	lagi, err := f.svc.Stage(ctx, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if lagi.Skipped["sudah tercatat di drive_exports"] != 2 {
		t.Errorf("dua berkas yang sudah tercatat harus terhitung, dapat %d", lagi.Skipped["sudah tercatat di drive_exports"])
	}
	if lagi.Counts[KindExport] != 1 {
		t.Errorf("ekspor bulanan tetap disusun setiap run, dapat %d", lagi.Counts[KindExport])
	}
}

// Uji kering tidak boleh meninggalkan manifest. Sebelumnya manifest tetap ditulis, jadi `record` yang
// dijalankan setelah uji kering mencatat semua lampiran di drive_exports tanpa pernah diunggah, dan sejak
// itu lampiran tersebut hilang dari staging tanpa peringatan apa pun.
func TestUjiKeringTidakMeninggalkanManifest(t *testing.T) {
	f := setup(t)
	f.seed(t)
	ctx := context.Background()

	m, err := f.svc.Stage(ctx, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Files) != 2 {
		t.Fatalf("uji kering harus tetap menyusun 2 lampiran, dapat %d", len(m.Files))
	}
	if _, err := f.svc.ReadManifest(); err == nil {
		t.Fatal("uji kering menulis manifest, jadi record sesudahnya akan mencatat berkas yang belum terunggah")
	}
	var n int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM drive_exports`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("uji kering mencatat %d baris drive_exports", n)
	}
}
