package attachment

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func detect(t *testing.T, name string, content []byte) (string, error) {
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
	return Detect(f, int64(len(content)), name)
}

func TestDetect(t *testing.T) {
	exe := append([]byte("MZ\x90\x00\x03\x00\x00\x00"), make([]byte, 64)...)
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 16)...)
	jpeg := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 16)...)
	wordTypes := `<Types><Override ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`
	docmTypes := `<Types><Override ContentType="application/vnd.ms-word.document.macroEnabled.main+xml"/></Types>`
	docx := zipOf(t, map[string]string{"[Content_Types].xml": wordTypes, "word/document.xml": "<w:document/>"})
	withVBA := zipOf(t, map[string]string{"[Content_Types].xml": wordTypes, "word/document.xml": "<w:document/>", "word/vbaProject.bin": "x"})
	macroTypes := zipOf(t, map[string]string{"[Content_Types].xml": docmTypes, "word/document.xml": "<w:document/>"})
	xlsx := zipOf(t, map[string]string{"[Content_Types].xml": "<Types/>", "xl/workbook.xml": "<workbook/>"})
	oleGarbage := append(append([]byte{}, oleSignature...), make([]byte, 600)...)

	cases := []struct {
		name     string
		file     string
		content  []byte
		wantMIME string
		wantErr  string // potongan pesan; kosong berarti harus diterima
	}{
		{"PDF", "brief.pdf", []byte("%PDF-1.7\n%isi"), "application/pdf", ""},
		{"ekstensi huruf besar", "BRIEF.PDF", []byte("%PDF-1.7\n%isi"), "application/pdf", ""},
		{"EXE diganti nama jadi PDF", "invoice.pdf", exe, "", "tidak sesuai"},
		{"PNG", "foto.png", png, "image/png", ""},
		{"JPG", "foto.jpg", jpeg, "image/jpeg", ""},
		{"JPEG", "foto.jpeg", jpeg, "image/jpeg", ""},
		{"PNG diberi nama JPG", "foto.jpg", png, "", "tidak sesuai"},
		{"TXT UTF-8", "catatan.txt", []byte("Halo\nbaris kedua ü"), "text/plain", ""},
		{"CSV ANSI dari Excel lama", "data.csv", []byte("nama;harga\r\ncaf\xe9;10\r\n"), "text/csv", ""},
		{"EXE diganti nama jadi CSV", "data.csv", exe, "", "tidak sesuai"},
		{"DOCX", "laporan.docx", docx, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", ""},
		{"DOCX berisi vbaProject.bin", "laporan.docx", withVBA, "", "macro"},
		{"DOCM diganti nama jadi DOCX", "laporan.docx", macroTypes, "", "macro"},
		{"XLSX diberi nama DOCX", "laporan.docx", xlsx, "", "tidak sesuai"},
		{"ekstensi DOCM", "laporan.docm", docx, "", "macro"},
		{"ekstensi XLSM", "data.XLSM", xlsx, "", "macro"},
		{"EXE", "setup.exe", exe, "", "tidak diizinkan"},
		{"tanpa ekstensi", "README", []byte("teks"), "", "tidak diizinkan"},
		{"DOC lama yang rusak", "lama.doc", oleGarbage, "", "tidak sesuai"},
		{"EXE diganti nama jadi DOC", "lama.doc", exe, "", "tidak sesuai"},
		{"berkas kosong", "kosong.pdf", []byte{}, "", "kosong"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := detect(t, c.file, c.content)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("harus diterima, ditolak: %v", err)
				}
				if got != c.wantMIME {
					t.Fatalf("MIME %q, mau %q", got, c.wantMIME)
				}
				return
			}
			var rj *Rejected
			if !errors.As(err, &rj) {
				t.Fatalf("harus ditolak dengan Rejected, dapat err %v", err)
			}
			if !strings.Contains(rj.Message, c.wantErr) {
				t.Fatalf("pesan %q tidak memuat %q", rj.Message, c.wantErr)
			}
		})
	}
}

func TestCleanName(t *testing.T) {
	cases := map[string]string{
		`C:\Users\staf\Desktop\brief.pdf`: "brief.pdf",
		"../../etc/passwd":                "passwd",
		"lapo\x00ran\n.txt":               "laporan.txt",
		"   ":                             "berkas",
		"..":                              "berkas",
		"foto produk 07.jpg":              "foto produk 07.jpg",
	}
	for in, want := range cases {
		if got := CleanName(in); got != want {
			t.Errorf("CleanName(%q) = %q, mau %q", in, got, want)
		}
	}

	long := CleanName(strings.Repeat("a", 300) + ".pdf")
	if utf8.RuneCountInString(long) != maxNameRunes || !strings.HasSuffix(long, ".pdf") {
		t.Errorf("nama panjang dipotong salah: %d karakter, akhiran %q", utf8.RuneCountInString(long), long[len(long)-4:])
	}
}
