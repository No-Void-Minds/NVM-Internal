package activity

import (
	"errors"
	"strings"
	"testing"
	"time"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/db/sqlc"
)

func TestSlots(t *testing.T) {
	if len(Slots) != 8 {
		t.Fatalf("jumlah slot %d, mau 8", len(Slots))
	}
	if Slots[0] != "09.00-10.00" || Slots[len(Slots)-1] != "17.00-18.00" {
		t.Fatalf("slot pertama dan terakhir salah: %v", Slots)
	}
	for _, s := range Slots {
		if strings.HasPrefix(s, "12.00") {
			t.Fatal("12.00-13.00 jam istirahat, tidak boleh ada di daftar slot")
		}
	}
	// Nomor slot yang dikirim ke frontend harus 1 sampai 8, urut, dan selalu lengkap.
	cells := emptyCells()
	for i, c := range cells {
		if c.Slot != i+1 || c.Time != Slots[i] {
			t.Fatalf("sel %d: %+v", i, c)
		}
	}
}

func TestClean(t *testing.T) {
	got, err := Clean("  Edit video\r\nlalu unggah  ", "Aktivitas")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Edit video\nlalu unggah" {
		t.Fatalf("hasil %q", got)
	}
	if got, err := Clean("   \n\t ", "Aktivitas"); err != nil || got != "" {
		t.Fatalf("isian hanya spasi harus jadi kosong, dapat %q %v", got, err)
	}

	// Batas dihitung per karakter, bukan per byte, supaya huruf beraksen tidak memakan kuota dua kali.
	if _, err := Clean(strings.Repeat("é", MaxFieldLen), "Aktivitas"); err != nil {
		t.Fatalf("%d karakter harus lolos: %v", MaxFieldLen, err)
	}
	var ve *apperr.Validation
	_, err = Clean(strings.Repeat("a", MaxFieldLen+1), "Aktivitas")
	if !errors.As(err, &ve) || ve.Message != "Aktivitas maksimal 1000 karakter." {
		t.Fatalf("pesan galat: %v", err)
	}
	if _, err := Clean(strings.Repeat("a", MaxFieldLen+1), "Output"); err == nil {
		t.Fatal("Output lebih dari batas harus ditolak juga")
	}
}

func TestCanEdit(t *testing.T) {
	staf := Actor{ID: 4, Role: sqlc.UserRoleStaff}
	manajer := Actor{ID: 7, Role: sqlc.UserRoleManager}
	admin := Actor{ID: 1, Role: sqlc.UserRoleAdmin}
	const today = "2026-11-18"
	cases := []struct {
		name   string
		actor  Actor
		target int64
		date   string
		want   bool
	}{
		{"staf isian sendiri hari ini", staf, 4, today, true},
		{"staf isian sendiri kemarin", staf, 4, "2026-11-17", false},
		{"staf isian orang lain hari ini", staf, 5, today, false},
		{"manajer isian orang lain hari lampau", manajer, 4, "2026-10-01", true},
		{"admin isian orang lain", admin, 4, today, false},
		{"admin isian sendiri", admin, 1, today, false},
	}
	for _, c := range cases {
		if got := CanEdit(c.actor, c.target, c.date, today); got != c.want {
			t.Errorf("%s: dapat %v, mau %v", c.name, got, c.want)
		}
	}
}

func TestParseDay(t *testing.T) {
	s := &Service{now: func() time.Time { return time.Date(2026, 11, 18, 2, 0, 0, 0, time.UTC) }}
	// 02.00 UTC sudah 09.00 WIB di tanggal yang sama, dan 20.00 UTC sudah masuk tanggal berikutnya.
	if got := s.Today(); got != "2026-11-18" {
		t.Fatalf("hari ini %q", got)
	}
	s.now = func() time.Time { return time.Date(2026, 11, 18, 20, 0, 0, 0, time.UTC) }
	if got := s.Today(); got != "2026-11-19" {
		t.Fatalf("hari ini setelah 17.00 UTC %q", got)
	}
	if got, err := s.ParseDay(""); err != nil || got != "2026-11-19" {
		t.Fatalf("tanggal kosong harus jadi hari ini, dapat %q %v", got, err)
	}
	if got, err := s.ParseDay(" 2026-01-05 "); err != nil || got != "2026-01-05" {
		t.Fatalf("dapat %q %v", got, err)
	}
	for _, bad := range []string{"18-11-2026", "2026-13-01", "besok"} {
		if _, err := s.ParseDay(bad); err == nil {
			t.Errorf("%q harus ditolak", bad)
		}
	}
}

func TestWeekRange(t *testing.T) {
	cases := []struct{ date, from, to string }{
		// Minggu biasa: Senin 16 sampai Jumat 20 November 2026, dari tanggal mana pun di minggu itu.
		{"2026-11-16", "2026-11-16", "2026-11-20"},
		{"2026-11-18", "2026-11-16", "2026-11-20"},
		{"2026-11-20", "2026-11-16", "2026-11-20"},
		// Sabtu dan Minggu masih masuk minggu Senin yang sudah lewat, bukan minggu berikutnya.
		{"2026-11-21", "2026-11-16", "2026-11-20"},
		{"2026-11-22", "2026-11-16", "2026-11-20"},
		// Pergantian bulan: Senin 30 November 2026, Jumat jatuh di Desember.
		{"2026-12-02", "2026-11-30", "2026-12-04"},
		// Pergantian tahun: Senin 28 Desember 2026, Jumat 1 Januari 2027.
		{"2026-12-31", "2026-12-28", "2027-01-01"},
		// Tahun kabisat: 29 Februari 2028 jatuh hari Selasa.
		{"2028-03-01", "2028-02-28", "2028-03-03"},
	}
	for _, c := range cases {
		from, to, err := WeekRange(c.date)
		if err != nil {
			t.Fatalf("%s: %v", c.date, err)
		}
		if from != c.from || to != c.to {
			t.Errorf("%s: dapat %s sampai %s, mau %s sampai %s", c.date, from, to, c.from, c.to)
		}
	}
	if _, _, err := WeekRange("16-11-2026"); err == nil {
		t.Fatal("tanggal tidak sah harus ditolak")
	}
}

func TestExportRows(t *testing.T) {
	people := []Person{{ID: 4, Name: "Dina"}, {ID: 5, Name: "Rio"}}
	cells := emptyCells()
	cells[0].Activity = "Edit video"
	cells[0].Output = "https://drive.example/1"
	day := exportDay{
		Date:  time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC), // Senin
		Cells: map[int64][]Cell{4: cells},
	}
	rows := exportRows("Daily Activity Monitoring - November 2026", people, []exportDay{day})

	if len(rows) != 3+len(Slots) {
		t.Fatalf("jumlah baris %d, mau %d", len(rows), 3+len(Slots))
	}
	// Tiga kolom kiri, lalu dua kolom per orang.
	if len(rows[0]) != 3+2*len(people) {
		t.Fatalf("lebar baris %d", len(rows[0]))
	}
	if rows[0][0] != "Daily Activity Monitoring - November 2026" {
		t.Fatalf("judul %q", rows[0][0])
	}
	if rows[1][3] != "Dina" || rows[1][5] != "Rio" || rows[1][4] != "" {
		t.Fatalf("baris nama %+v", rows[1])
	}
	want := []string{"Week", "Date", "Time", "Activity", "Output", "Activity", "Output"}
	for i, v := range want {
		if rows[2][i] != v {
			t.Fatalf("kepala kolom %d %q, mau %q", i, rows[2][i], v)
		}
	}
	first := rows[3]
	if first[0] != "Week 45" || first[1] != "Monday - 2" || first[2] != Slots[0] {
		t.Fatalf("baris slot pertama %+v", first[:3])
	}
	if first[3] != "Edit video" || first[4] != "https://drive.example/1" {
		t.Fatalf("isian tidak masuk kolom pemiliknya: %+v", first)
	}
	if first[5] != "" || first[6] != "" {
		t.Fatalf("kolom orang yang tidak mengisi harus kosong: %+v", first)
	}
	// Nomor minggu dan tanggal hanya ditulis di baris slot pertama.
	if rows[4][0] != "" || rows[4][1] != "" || rows[4][2] != Slots[1] {
		t.Fatalf("baris slot kedua %+v", rows[4][:3])
	}
}

func TestExportTitleDanColName(t *testing.T) {
	nov := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	if got := exportTitle(nov, nov.AddDate(0, 0, 10)); got != "Daily Activity Monitoring - November 2026" {
		t.Fatalf("judul satu bulan %q", got)
	}
	if got := exportTitle(nov, nov.AddDate(0, 1, 0)); !strings.Contains(got, "sampai") {
		t.Fatalf("judul dua bulan %q", got)
	}
	for i, want := range map[int]string{0: "A", 25: "Z", 26: "AA", 27: "AB"} {
		if got := colName(i); got != want {
			t.Errorf("colName(%d) = %q, mau %q", i, got, want)
		}
	}
}
