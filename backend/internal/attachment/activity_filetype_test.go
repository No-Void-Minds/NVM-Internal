package attachment

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func detectActivity(t *testing.T, name string, content []byte) (string, error) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "unggahan")
	if err := os.WriteFile(p, content, 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	return DetectActivity(f, int64(len(content)), name)
}

// Lampiran Aktivitas Harian (revisi 24): jenis media ikut diizinkan, tetapi tetap diperiksa dari magic bytes.
func TestDetectActivity(t *testing.T) {
	pad := func(head []byte) []byte { return append(head, make([]byte, 64)...) }
	exe := pad([]byte("MZ\x90\x00\x03\x00\x00\x00"))
	webp := pad(append([]byte("RIFF\x2c\x00\x00\x00"), []byte("WEBPVP8 ")...))
	// Wadah ISO Base Media: 4 byte panjang kotak, lalu "ftyp".
	mp4 := pad(append([]byte{0, 0, 0, 0x20}, []byte("ftypisom")...))
	mov := pad(append([]byte{0, 0, 0, 0x14}, []byte("ftypqt  ")...))

	cases := []struct {
		name     string
		file     string
		content  []byte
		wantMIME string
		wantErr  string // potongan pesan; kosong berarti harus diterima
	}{
		{"WebP", "tangkapan.webp", webp, "image/webp", ""},
		{"GIF", "animasi.gif", pad([]byte("GIF89a")), "image/gif", ""},
		{"GIF lama", "animasi.gif", pad([]byte("GIF87a")), "image/gif", ""},
		{"MP4", "rekaman.mp4", mp4, "video/mp4", ""},
		{"MOV", "rekaman.mov", mov, "video/quicktime", ""},
		{"WebM", "rekaman.webm", pad([]byte{0x1A, 0x45, 0xDF, 0xA3}), "video/webm", ""},
		{"ZIP", "bahan.zip", pad([]byte("PK\x03\x04")), "application/zip", ""},
		{"ZIP kosong", "kosong.zip", pad([]byte("PK\x05\x06")), "application/zip", ""},
		// Jenis lampiran task tetap jalan lewat daftar yang sama.
		{"PDF", "laporan.pdf", []byte("%PDF-1.7\n%isi"), "application/pdf", ""},
		{"PNG huruf besar", "FOTO.PNG", pad([]byte("\x89PNG\r\n\x1a\n")), "image/png", ""},

		{"SVG ditolak", "ikon.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), "", "tidak diizinkan"},
		{"EXE ditolak", "virus.exe", exe, "", "tidak diizinkan"},
		{"EXE diganti jadi MP4", "video.mp4", exe, "", "tidak sesuai"},
		{"EXE diganti jadi WebP", "foto.webp", exe, "", "tidak sesuai"},
		{"EXE diganti jadi ZIP", "bahan.zip", exe, "", "tidak sesuai"},
		{"GIF dengan isi PNG", "animasi.gif", pad([]byte("\x89PNG\r\n\x1a\n")), "", "tidak sesuai"},
		{"WebP tanpa penanda WEBP", "foto.webp", pad(append([]byte("RIFF\x2c\x00\x00\x00"), []byte("WAVEfmt ")...)), "", "tidak sesuai"},
		{"berkas kosong", "kosong.mp4", nil, "", "kosong"},
		{"tanpa ekstensi", "berkas", pad([]byte("%PDF-")), "", "tidak diizinkan"},
	}

	for _, c := range cases {
		mime, err := detectActivity(t, c.file, c.content)
		if c.wantErr == "" {
			if err != nil {
				t.Errorf("%s: ditolak padahal harus diterima: %v", c.name, err)
			} else if mime != c.wantMIME {
				t.Errorf("%s: mime %q, mau %q", c.name, mime, c.wantMIME)
			}
			continue
		}
		if err == nil {
			t.Errorf("%s: diterima padahal harus ditolak", c.name)
			continue
		}
		var rejected *Rejected
		if !errors.As(err, &rejected) {
			t.Errorf("%s: galat harus Rejected, dapat %T", c.name, err)
			continue
		}
		if !strings.Contains(strings.ToLower(rejected.Message), c.wantErr) {
			t.Errorf("%s: pesan %q tidak memuat %q", c.name, rejected.Message, c.wantErr)
		}
	}

	// Pesan penolakan menyebut jenis yang boleh, supaya pengguna tidak menebak.
	_, err := detectActivity(t, "ikon.svg", []byte("<svg/>"))
	if err == nil || !strings.Contains(err.Error(), "MP4") || !strings.Contains(err.Error(), "ZIP") {
		t.Fatalf("pesan penolakan harus memuat daftar jenis: %v", err)
	}
}
