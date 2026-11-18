// Package driveexport menyusun salinan rapi semua berkas aplikasi untuk Google Drive (revisi 25). Isinya dua
// bagian: aturan nama dan path (berkas ini), dan penyusun pohon staging plus pencatatnya (export.go).
package driveexport

import (
	"fmt"
	"path"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// MaxSegment membatasi satu bagian nama (folder atau nama berkas). Drive sendiri mengizinkan lebih panjang,
// tetapi path yang panjang menyulitkan sinkronisasi di Windows dan sulit dibaca orang.
const MaxSegment = 120

// Karakter yang dilarang di nama berkas Windows dan membingungkan di Drive.
const forbidden = `/\:*?"<>|`

var monthNames = [...]string{
	"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

var dayNames = map[time.Weekday]string{
	time.Monday: "Senin", time.Tuesday: "Selasa", time.Wednesday: "Rabu", time.Thursday: "Kamis",
	time.Friday: "Jumat", time.Saturday: "Sabtu", time.Sunday: "Minggu",
}

func MonthName(m time.Month) string { return monthNames[int(m)-1] }

func DayName(d time.Weekday) string { return dayNames[d] }

// Clean menyiapkan satu bagian nama: karakter terlarang dan kontrol dihapus, spasi dirapikan, lalu dipotong
// 120 karakter. Nama yang habis jadi kosong diganti "tanpa nama", supaya path tetap sah. Garis miring ikut
// dibuang, bukan dipakai memotong nama, supaya judul seperti "Rekap 2026/11" tetap satu folder.
func Clean(name string) string {
	return cut(scrub(name))
}

// CleanFile sama dengan Clean, tetapi ekstensi dipertahankan saat nama dipotong.
func CleanFile(name string) string {
	name = scrub(name)
	if utf8.RuneCountInString(name) <= MaxSegment {
		return name
	}
	ext := path.Ext(name)
	if utf8.RuneCountInString(ext) > 12 {
		ext = ""
	}
	runes := []rune(strings.TrimSuffix(name, ext))
	return strings.TrimRight(string(runes[:MaxSegment-utf8.RuneCountInString(ext)]), " ") + ext
}

func scrub(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		// Tab dan baris baru ikut dihitung spasi, bukan dibuang sebagai karakter kontrol, supaya judul
		// berbaris ganda tidak menempel jadi satu kata.
		case unicode.IsSpace(r):
			b.WriteRune(' ')
		case r < 0x20 || r == 0x7f:
			// Karakter kontrol lain dibuang.
		case strings.ContainsRune(forbidden, r):
			// Karakter terlarang dibuang, bukan diganti tanda lain, supaya nama tidak jadi penuh garis.
		default:
			b.WriteRune(r)
		}
	}
	name = strings.Join(strings.Fields(b.String()), " ")
	// Titik di awal menyembunyikan berkas di banyak alat, titik di akhir dibuang Windows.
	name = strings.Trim(name, ". ")
	if name == "" {
		return "tanpa nama"
	}
	return name
}

func cut(name string) string {
	runes := []rune(name)
	if len(runes) <= MaxSegment {
		return name
	}
	return strings.TrimRight(string(runes[:MaxSegment]), " ")
}

// Unique menambahkan " (2)", " (3)", dan seterusnya bila nama sudah dipakai di folder yang sama. Ekstensi
// tetap di belakang, jadi "foto.png" menjadi "foto (2).png", bukan "foto.png (2)".
func Unique(name string, taken map[string]bool) string {
	if !taken[strings.ToLower(name)] {
		return name
	}
	ext := path.Ext(name)
	if utf8.RuneCountInString(ext) > 12 {
		ext = ""
	}
	base := strings.TrimSuffix(name, ext)
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s (%d)%s", base, n, ext)
		if !taken[strings.ToLower(candidate)] {
			return candidate
		}
	}
}

// ActivityMonthDir: "Aktivitas/2026-11 November".
func ActivityMonthDir(day time.Time) string {
	return fmt.Sprintf("Aktivitas/%04d-%02d %s", day.Year(), int(day.Month()), MonthName(day.Month()))
}

// ActivityDayDir: "Aktivitas/2026-11 November/18 Rabu".
func ActivityDayDir(day time.Time) string {
	return fmt.Sprintf("%s/%d %s", ActivityMonthDir(day), day.Day(), DayName(day.Weekday()))
}

// ActivityFileName: "09.00-10.00 - tangkapan.png".
func ActivityFileName(slotLabel, fileName string) string {
	return CleanFile(fmt.Sprintf("%s - %s", slotLabel, fileName))
}

// MonthlyExportPath: ekspor Excel bulan berjalan, ditimpa setiap malam.
func MonthlyExportPath(day time.Time) string {
	return fmt.Sprintf("%s/Aktivitas Harian %s %d.xlsx", ActivityMonthDir(day), MonthName(day.Month()), day.Year())
}

// TaskFolder: "Task/2026-11/T-0012 Foto katalog". Bulannya bulan task dibuat, jadi folder task tidak pindah
// walau lampirannya ditambah bulan berikutnya.
func TaskFolder(created time.Time, id, title string) string {
	return fmt.Sprintf("Task/%04d-%02d/%s %s", created.Year(), int(created.Month()), id, Clean(title))
}

// SubtaskFolderName: "Sub task 1 - Edit video (Dina)". Nomornya position + 1, sama dengan nomor di aplikasi.
func SubtaskFolderName(position int, title, pic string) string {
	name := fmt.Sprintf("Sub task %d - %s", position+1, Clean(title))
	if pic != "" {
		name += fmt.Sprintf(" (%s)", Clean(pic))
	}
	return Clean(name)
}

// DumpPath: "Backup Database/2026-11-18.dump.age", dari nama berkas dump "db-20261118T200000Z.dump".
func DumpPath(dumpName string) string {
	stamp := strings.TrimSuffix(strings.TrimPrefix(path.Base(dumpName), "db-"), ".dump")
	if t, err := time.Parse("20060102T150405Z", stamp); err == nil {
		return fmt.Sprintf("Backup Database/%s.dump.age", t.Format("2006-01-02"))
	}
	// Nama yang tidak terbaca tetap diunggah dengan namanya sendiri, supaya tidak ada dump yang hilang diam-diam.
	return "Backup Database/" + CleanFile(path.Base(dumpName)) + ".age"
}
