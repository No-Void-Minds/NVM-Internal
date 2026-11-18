package driveexport

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"nvm-internal/internal/storage"
)

// butuhSymlink melewati test di Windows, yang menolak membuat symlink tanpa privilege khusus. Kode ini hanya
// pernah berjalan di container Linux, jadi di sana test ini yang membuktikan pohon staging-nya benar.
func butuhSymlink(t *testing.T) {
	t.Helper()
	d := t.TempDir()
	if err := os.Symlink(filepath.Join(d, "x"), filepath.Join(d, "y")); err != nil {
		t.Skipf("symlink tidak diizinkan di sistem ini, dilewati: %v", err)
	}
}

// Pohon staging diuji tanpa database: link dan clearStaging tidak menyentuh database sama sekali.
func stagingFixture(t *testing.T) (*Service, string) {
	t.Helper()
	root := t.TempDir()
	store, err := storage.Open(filepath.Join(root, "storage"))
	if err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(root, "staging")
	svc := NewService(nil, store, staging, "", slog.New(slog.NewTextHandler(io.Discard, nil)))
	return svc, staging
}

const idFoto = "55555555-5555-4555-8555-555555555555"

// link harus membuat symlink ke berkas di storage, bukan hardlink: di dalam container storage dan staging
// adalah dua bind mount berbeda, dan hardlink antar mount ditolak kernel dengan EXDEV.
func TestLinkMembuatSymlinkKeStorage(t *testing.T) {
	butuhSymlink(t)
	svc, staging := stagingFixture(t)
	src, err := svc.store.PathFor(idFoto)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(src), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("isi berkas uji"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := svc.clearStaging(); err != nil {
		t.Fatal(err)
	}

	var m Manifest
	rel := "Task/2026-11/T-0012 Foto katalog/Brief/brief.pdf"
	svc.link(&m, kindTaskFile, idFoto, rel, 14)
	if len(m.Files) != 1 || m.Bytes != 14 {
		t.Fatalf("manifest %+v", m)
	}

	dst := filepath.Join(staging, filepath.FromSlash(rel))
	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("berkas staging harus symlink, dapat mode %v", info.Mode())
	}
	target, err := os.Readlink(dst)
	if err != nil {
		t.Fatal(err)
	}
	if target != src {
		t.Fatalf("symlink menunjuk %q, mau %q", target, src)
	}
	// Yang penting bagi rclone --copy-links: isi berkas terbaca lewat symlink itu.
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("isi berkas tidak terbaca lewat symlink: %v", err)
	}
	if string(data) != "isi berkas uji" {
		t.Fatalf("isi berkas lewat symlink %q", data)
	}
}

// Berkas yang barisnya ada tetapi berkasnya hilang dari storage dilewati, tidak membatalkan seluruh salinan.
func TestLinkMelewatiBerkasYangHilang(t *testing.T) {
	butuhSymlink(t)
	svc, staging := stagingFixture(t)
	if err := svc.clearStaging(); err != nil {
		t.Fatal(err)
	}
	var m Manifest
	svc.link(&m, kindTaskFile, idFoto, "Task/2026-11/T-0012 X/Brief/hilang.pdf", 10)
	if len(m.Files) != 0 || m.Bytes != 0 {
		t.Fatalf("berkas yang hilang tidak boleh masuk manifest: %+v", m)
	}
	if _, err := os.Lstat(filepath.Join(staging, "Task")); !os.IsNotExist(err) {
		t.Fatal("tidak boleh ada symlink menggantung di staging")
	}
	// ID yang bukan UUID juga dilewati, bukan jadi galat.
	svc.link(&m, kindTaskFile, "bukan-uuid", "Task/x/Brief/a.pdf", 10)
	if len(m.Files) != 0 {
		t.Fatalf("id tidak sah tidak boleh masuk manifest: %+v", m)
	}
}

// clearStaging mengosongkan isi, tetapi tidak pernah menghapus folder staging-nya sendiri: di container itu
// titik mount, dan menghapusnya gagal dengan EROFS karena induknya read-only.
func TestClearStagingTidakMenghapusFoldernya(t *testing.T) {
	svc, staging := stagingFixture(t)
	if err := svc.clearStaging(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(staging, "Task", "lama"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "sisa.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	before, err := os.Stat(staging)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.clearStaging(); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(staging)
	if err != nil {
		t.Fatalf("folder staging ikut terhapus: %v", err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("folder staging dibuat ulang, padahal titik mount tidak boleh disentuh")
	}
	entries, err := os.ReadDir(staging)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("isi staging masih ada: %+v", entries)
	}
}
