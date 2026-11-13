package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenCreatesFolders(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, sub := range []string{"attachments", "tmp"} {
		if info, err := os.Stat(filepath.Join(dir, sub)); err != nil || !info.IsDir() {
			t.Errorf("folder %s belum dibuat: %v", sub, err)
		}
	}
	if !filepath.IsAbs(s.Root()) {
		t.Errorf("Root %q harus path absolut", s.Root())
	}
}

func TestOpenRejectsEmptyDir(t *testing.T) {
	if _, err := Open(""); err == nil {
		t.Fatal("STORAGE_DIR kosong harus ditolak")
	}
}

func TestOpenRemovesOnlyStaleTmp(t *testing.T) {
	dir := t.TempDir()
	tmp := filepath.Join(dir, "tmp")
	if err := os.MkdirAll(tmp, 0o750); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(tmp, "upload-lama")
	fresh := filepath.Join(tmp, "upload-baru")
	for _, p := range []string{stale, fresh} {
		if err := os.WriteFile(p, []byte("x"), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-2 * staleTmpAge)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}

	if _, err := Open(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("sisa unggahan lama harus terhapus")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Error("unggahan yang masih baru tidak boleh terhapus")
	}
}
