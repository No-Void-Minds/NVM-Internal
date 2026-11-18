package driveexport

import (
	"strings"
	"testing"
	"time"
)

func TestClean(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Foto katalog", "Foto katalog"},
		{`Rekap 2026/11 "final" <v2>`, "Rekap 202611 final v2"},
		{`a\b:c*d?e"f<g>h|i`, "abcdefghi"},
		{"spasi    ganda\tdan\nbaris", "spasi ganda dan baris"},
		{"  rapi  ", "rapi"},
		{".tersembunyi.", "tersembunyi"},
		{"///", "tanpa nama"},
		{"", "tanpa nama"},
		// Garis miring dibuang, jadi nama ini tidak bisa keluar dari folder induknya.
		{"../../etc/passwd", "etcpasswd"},
	}
	for _, c := range cases {
		if got := Clean(c.in); got != c.want {
			t.Errorf("Clean(%q) = %q, mau %q", c.in, got, c.want)
		}
	}
	panjang := strings.Repeat("a", 200)
	if got := Clean(panjang); len([]rune(got)) != MaxSegment {
		t.Errorf("nama panjang dipotong jadi %d karakter, mau %d", len([]rune(got)), MaxSegment)
	}
}

func TestCleanFile(t *testing.T) {
	name := strings.Repeat("b", 200) + ".png"
	got := CleanFile(name)
	if len([]rune(got)) != MaxSegment {
		t.Fatalf("panjang %d, mau %d", len([]rune(got)), MaxSegment)
	}
	if !strings.HasSuffix(got, ".png") {
		t.Fatalf("ekstensi hilang saat dipotong: %q", got)
	}
	if got := CleanFile("laporan akhir.pdf"); got != "laporan akhir.pdf" {
		t.Fatalf("nama normal berubah: %q", got)
	}
}

func TestUnique(t *testing.T) {
	taken := map[string]bool{}
	add := func(n string) string {
		u := Unique(n, taken)
		taken[strings.ToLower(u)] = true
		return u
	}
	if got := add("foto.png"); got != "foto.png" {
		t.Fatalf("pertama %q", got)
	}
	if got := add("foto.png"); got != "foto (2).png" {
		t.Fatalf("kedua %q", got)
	}
	if got := add("foto.png"); got != "foto (3).png" {
		t.Fatalf("ketiga %q", got)
	}
	// Bentrok tidak peduli huruf besar kecil, karena Drive dan Windows memperlakukannya sama.
	if got := add("FOTO.png"); got != "FOTO (4).png" {
		t.Fatalf("huruf besar %q", got)
	}
	// Folder tanpa ekstensi.
	if got := add("Foto katalog"); got != "Foto katalog" {
		t.Fatalf("folder pertama %q", got)
	}
	if got := add("Foto katalog"); got != "Foto katalog (2)" {
		t.Fatalf("folder kedua %q", got)
	}
}

func TestBulanDanHariIndonesia(t *testing.T) {
	// 18 November 2026 jatuh hari Rabu.
	hari := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	if got := MonthName(hari.Month()); got != "November" {
		t.Fatalf("bulan %q", got)
	}
	if got := DayName(hari.Weekday()); got != "Rabu" {
		t.Fatalf("hari %q", got)
	}
	if got := DayName(time.Friday); got != "Jumat" {
		t.Fatalf("Jumat ditulis %q", got)
	}
	for m := time.January; m <= time.December; m++ {
		if MonthName(m) == "" {
			t.Fatalf("bulan %d tidak punya nama", m)
		}
	}
}

func TestPathSetiapJenis(t *testing.T) {
	hari := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	if got := ActivityDayDir(hari); got != "Aktivitas/2026-11 November/18 Rabu" {
		t.Fatalf("folder hari %q", got)
	}
	if got := ActivityFileName("09.00-10.00", "tangkapan.png"); got != "09.00-10.00 - tangkapan.png" {
		t.Fatalf("nama berkas aktivitas %q", got)
	}
	if got := MonthlyExportPath(hari); got != "Aktivitas/2026-11 November/Aktivitas Harian November 2026.xlsx" {
		t.Fatalf("ekspor bulanan %q", got)
	}

	dibuat := time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)
	folder := TaskFolder(dibuat, "T-0012", "Foto katalog")
	if folder != "Task/2026-11/T-0012 Foto katalog" {
		t.Fatalf("folder task %q", folder)
	}
	if got := folder + "/Brief/brief.pdf"; got != "Task/2026-11/T-0012 Foto katalog/Brief/brief.pdf" {
		t.Fatalf("path brief %q", got)
	}
	if got := folder + "/Deliverable/hasil.png"; !strings.HasSuffix(got, "/Deliverable/hasil.png") {
		t.Fatalf("path deliverable %q", got)
	}
	if got := SubtaskFolderName(0, "Edit video", "Dina"); got != "Sub task 1 - Edit video (Dina)" {
		t.Fatalf("folder sub task %q", got)
	}
	// Sub task tanpa PIC tidak menyisakan tanda kurung kosong.
	if got := SubtaskFolderName(2, "Riset", ""); got != "Sub task 3 - Riset" {
		t.Fatalf("sub task tanpa PIC %q", got)
	}
	// Judul dengan karakter terlarang tetap jadi satu bagian path, tidak membuat folder baru.
	if got := SubtaskFolderName(0, "Edit video/final", "Dina"); strings.Contains(got, "/") {
		t.Fatalf("judul dengan garis miring membuat folder baru: %q", got)
	}
}

func TestDumpPath(t *testing.T) {
	if got := DumpPath("db-20261118T200000Z.dump"); got != "Backup Database/2026-11-18.dump.age" {
		t.Fatalf("dump %q", got)
	}
	if got := DumpPath("/srv/nvm-internal/backup/db-20261101T200000Z.dump"); got != "Backup Database/2026-11-01.dump.age" {
		t.Fatalf("dump dengan path %q", got)
	}
	// Nama yang tidak terbaca tetap punya path, supaya tidak ada dump yang hilang diam-diam.
	if got := DumpPath("aneh.bin"); got != "Backup Database/aneh.bin.age" {
		t.Fatalf("dump aneh %q", got)
	}
}
