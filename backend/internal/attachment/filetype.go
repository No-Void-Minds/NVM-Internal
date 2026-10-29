// Package attachment berisi aturan berkas lampiran task (FR-TASK-15, NFR-SEC-09).
package attachment

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/richardlehane/mscfb"
)

// Rejected berarti berkas ditolak karena isinya, bukan karena galat server. Pesannya aman ditampilkan.
type Rejected struct{ Message string }

func (e *Rejected) Error() string { return e.Message }

func reject(format string, args ...any) error {
	return &Rejected{Message: fmt.Sprintf(format, args...)}
}

func mismatch() error {
	return reject("Isi berkas tidak sesuai dengan ekstensinya. Berkas mungkin rusak atau ekstensinya diganti.")
}

func macro() error {
	return reject("Berkas ber-macro tidak diizinkan. Simpan ulang tanpa macro, misalnya sebagai DOCX, XLSX, atau PPTX.")
}

const allowedList = "PDF, CSV, TXT, DOC, DOCX, XLS, XLSX, PPT, PPTX, JPG, JPEG, dan PNG"

// macroExts ditolak dengan pesan khusus supaya pengguna tahu alasannya, bukan sekadar "tipe tidak diizinkan".
var macroExts = map[string]bool{
	".docm": true, ".dotm": true, ".xlsm": true, ".xltm": true, ".xlam": true, ".pptm": true, ".potm": true, ".ppsm": true,
}

type rule struct {
	mime  string
	check func(f *os.File, size int64) error
}

var oleSignature = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}

// Daftar izin, bukan daftar tolak: ekstensi yang tidak ada di sini selalu ditolak.
var rules = map[string]rule{
	".pdf":  {"application/pdf", prefix([]byte("%PDF-"))},
	".png":  {"image/png", prefix([]byte("\x89PNG\r\n\x1a\n"))},
	".jpg":  {"image/jpeg", prefix([]byte{0xFF, 0xD8, 0xFF})},
	".jpeg": {"image/jpeg", prefix([]byte{0xFF, 0xD8, 0xFF})},
	".txt":  {"text/plain", plainText},
	".csv":  {"text/csv", plainText},
	".docx": {"application/vnd.openxmlformats-officedocument.wordprocessingml.document", ooxml("word/document.xml")},
	".xlsx": {"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", ooxml("xl/workbook.xml")},
	".pptx": {"application/vnd.openxmlformats-officedocument.presentationml.presentation", ooxml("ppt/presentation.xml")},
	".doc":  {"application/msword", ole("WordDocument")},
	".xls":  {"application/vnd.ms-excel", ole("Workbook", "Book")},
	// Makro di PPT lama dikompres di dalam stream utama, bukan stream bernama, jadi tidak terdeteksi di sini.
	// Diterima karena SRS mengizinkan PPT; risikonya diredam dengan unduhan yang selalu berupa attachment.
	".ppt": {"application/vnd.ms-powerpoint", ole("PowerPoint Document")},
}

// Detect memeriksa isi berkas terhadap ekstensi nama aslinya dan mengembalikan MIME yang terbukti dari isi.
// Ekstensi yang cocok saja tidak cukup: EXE yang diganti nama menjadi .pdf tetap ditolak (NFR-SEC-09 a).
func Detect(f *os.File, size int64, originalName string) (string, error) {
	ext := strings.ToLower(path.Ext(strings.ReplaceAll(originalName, `\`, "/")))
	r, ok := rules[ext]
	if !ok {
		if macroExts[ext] {
			return "", macro()
		}
		return "", reject("Tipe berkas ini tidak diizinkan. Yang diizinkan: %s.", allowedList)
	}
	if size <= 0 {
		return "", reject("Berkas kosong.")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	if err := r.check(f, size); err != nil {
		return "", err
	}
	return r.mime, nil
}

func prefix(sig []byte) func(*os.File, int64) error {
	return func(f *os.File, _ int64) error {
		head := make([]byte, len(sig))
		if _, err := io.ReadFull(f, head); err != nil || !bytes.Equal(head, sig) {
			return mismatch()
		}
		return nil
	}
}

// plainText menerima UTF-8 maupun ekspor Excel lama (ANSI), tetapi menolak byte kontrol. Berkas biner seperti
// EXE selalu mengandung byte 0, jadi tidak bisa lolos sebagai .txt atau .csv.
func plainText(f *os.File, _ int64) error {
	buf := make([]byte, 64*1024)
	for {
		n, err := f.Read(buf)
		for _, b := range buf[:n] {
			if b < 0x20 && b != '\t' && b != '\n' && b != '\r' && b != '\f' {
				return mismatch()
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// ooxml memeriksa DOCX, XLSX, dan PPTX, yang sebenarnya arsip ZIP. Hanya daftar isinya yang dibaca dan tidak ada
// yang diekstrak ke disk, jadi arsip yang sengaja dibuat sangat besar saat diekstrak (zip bomb) tidak berbahaya.
func ooxml(mainPart string) func(*os.File, int64) error {
	return func(f *os.File, size int64) error {
		zr, err := zip.NewReader(f, size)
		if err != nil {
			return mismatch()
		}
		var hasTypes, hasMain bool
		for _, zf := range zr.File {
			name := strings.ToLower(zf.Name)
			switch {
			case path.Base(name) == "vbaproject.bin":
				return macro()
			case name == "[content_types].xml":
				hasTypes = true
				if err := checkContentTypes(zf); err != nil {
					return err
				}
			case name == mainPart:
				hasMain = true
			}
		}
		if !hasTypes || !hasMain {
			return mismatch()
		}
		return nil
	}
}

func checkContentTypes(zf *zip.File) error {
	rc, err := zf.Open()
	if err != nil {
		return mismatch()
	}
	defer func() { _ = rc.Close() }()
	// Dibatasi 1 MB: [Content_Types].xml normal hanya beberapa KB.
	data, err := io.ReadAll(io.LimitReader(rc, 1<<20))
	if err != nil {
		return mismatch()
	}
	if bytes.Contains(bytes.ToLower(data), []byte("macroenabled")) {
		return macro()
	}
	return nil
}

// ole memeriksa DOC dan XLS lama (format Compound File). Proyek VBA tersimpan sebagai stream dengan nama tetap,
// jadi berkas ber-macro dikenali dari daftar stream tanpa menjalankan apa pun.
func ole(mainStreams ...string) func(*os.File, int64) error {
	return func(f *os.File, size int64) error {
		if err := prefix(oleSignature)(f, size); err != nil {
			return err
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return err
		}
		doc, err := mscfb.New(f)
		if err != nil {
			return mismatch()
		}
		hasMain := false
		for entry, err := doc.Next(); err == nil; entry, err = doc.Next() {
			switch entry.Name {
			case "Macros", "VBA", "_VBA_PROJECT_CUR", "_VBA_PROJECT":
				return macro()
			}
			if slices.Contains(mainStreams, entry.Name) {
				hasMain = true
			}
		}
		if !hasMain {
			return mismatch()
		}
		return nil
	}
}

const maxNameRunes = 200

// CleanName menyiapkan nama asli untuk disimpan dan ditampilkan: folder dibuang, karakter kontrol dihapus, dan
// panjang dibatasi dengan ekstensi tetap utuh. Nama ini tidak pernah dipakai sebagai path di disk.
func CleanName(name string) string {
	name = path.Base(strings.ReplaceAll(name, `\`, "/"))
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == "/" || name == ".." {
		return "berkas"
	}
	if utf8.RuneCountInString(name) > maxNameRunes {
		runes := []rune(name)
		ext := []rune(path.Ext(name))
		if len(ext) > 20 {
			ext = nil
		}
		name = string(runes[:maxNameRunes-len(ext)]) + string(ext)
	}
	return name
}
