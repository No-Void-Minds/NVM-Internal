// Package storage menyimpan berkas lampiran di disk. Nama berkas di disk selalu UUID dari database, tidak
// pernah nama asli dari pengguna, jadi path tidak bisa diarahkan keluar folder ini (NFR-SEC-09 c).
package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// staleTmpAge adalah umur berkas di tmp/ yang dianggap sisa unggahan gagal. Satu jam jauh di atas waktu
// unggah 50 MB di jaringan kantor, jadi unggahan yang masih berjalan tidak ikut terhapus.
const staleTmpAge = time.Hour

type Store struct {
	root    string
	files   string
	tmp     string
	avatars string
}

// Open menyiapkan folder penyimpanan dan memastikan folder itu bisa ditulis. Dipanggil saat server start,
// supaya izin folder yang salah ketahuan saat itu juga, bukan saat orang pertama mengunggah.
func Open(dir string) (*Store, error) {
	if dir == "" {
		return nil, errors.New("STORAGE_DIR wajib diisi")
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("STORAGE_DIR tidak valid: %w", err)
	}
	s := &Store{
		root:    root,
		files:   filepath.Join(root, "attachments"),
		tmp:     filepath.Join(root, "tmp"),
		avatars: filepath.Join(root, "avatars"),
	}
	for _, d := range []string{s.files, s.tmp, s.avatars} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			return nil, fmt.Errorf("folder penyimpanan %s tidak bisa dibuat: %w", d, err)
		}
	}

	probe, err := os.CreateTemp(s.tmp, "probe-*")
	if err != nil {
		return nil, fmt.Errorf("folder penyimpanan %s tidak bisa ditulis: %w", s.tmp, err)
	}
	name := probe.Name()
	_ = probe.Close()
	if err := os.Remove(name); err != nil {
		return nil, fmt.Errorf("berkas uji di %s tidak bisa dihapus: %w", s.tmp, err)
	}

	if err := s.removeStaleTmp(time.Now()); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Root() string { return s.root }

// removeStaleTmp membuang sisa unggahan yang terputus, misalnya karena server mati di tengah unggahan.
func (s *Store) removeStaleTmp(now time.Time) error {
	entries, err := os.ReadDir(s.tmp)
	if err != nil {
		return fmt.Errorf("folder %s tidak bisa dibaca: %w", s.tmp, err)
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || e.IsDir() || now.Sub(info.ModTime()) < staleTmpAge {
			continue
		}
		_ = os.Remove(filepath.Join(s.tmp, e.Name()))
	}
	return nil
}

var idPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// CreateTemp membuat berkas sementara untuk satu unggahan di tmp/.
func (s *Store) CreateTemp() (*os.File, error) {
	return os.CreateTemp(s.tmp, "upload-*")
}

// PathFor mengembalikan lokasi berkas lampiran. Berkas disebar ke subfolder dua huruf pertama UUID supaya satu
// folder tidak berisi puluhan ribu berkas. id wajib berbentuk UUID, jadi tidak mungkin berisi ".." atau "/".
func (s *Store) PathFor(id string) (string, error) {
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("id lampiran tidak valid: %q", id)
	}
	return filepath.Join(s.files, id[:2], id), nil
}

// Commit memindahkan berkas sementara ke lokasi akhirnya. tmp/ dan attachments/ ada di disk yang sama, jadi
// pemindahan ini atomik: berkas tidak pernah terlihat setengah jadi. Berkas sumber harus sudah ditutup,
// karena Windows menolak memindahkan berkas yang masih terbuka.
func (s *Store) Commit(tmpPath, id string) error {
	dst, err := s.PathFor(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	return os.Rename(tmpPath, dst)
}

func ValidID(id string) bool {
	return idPattern.MatchString(id)
}

// CreateAvatarTemp membuat berkas sementara di avatars/ sendiri, bukan di tmp/, supaya rename ke nama akhirnya
// selalu di dalam satu folder dan tidak pernah terlihat setengah jadi oleh GET foto.
func (s *Store) CreateAvatarTemp() (*os.File, error) {
	return os.CreateTemp(s.avatars, ".upload-*")
}

// AvatarPath mengembalikan lokasi foto profil. id wajib UUID, jadi tidak bisa berisi ".." atau "/".
func (s *Store) AvatarPath(id string) (string, error) {
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("id foto tidak valid: %q", id)
	}
	return filepath.Join(s.avatars, id+".jpg"), nil
}
