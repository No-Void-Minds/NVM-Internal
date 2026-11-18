package activity

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/db/sqlc"
	"nvm-internal/internal/testdb"
)

const (
	hariIni   = "2026-11-18"
	kemarin   = "2026-11-17"
	bulanLalu = "2026-10-05"
)

// sheetOf membaca sheet1.xml dari .xlsx hasil ekspor, supaya isinya bisa diperiksa tanpa pustaka tambahan.
func sheetOf(t *testing.T, data []byte) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range zr.File {
		if f.Name != "xl/worksheets/sheet1.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = rc.Close() }()
		body, err := io.ReadAll(rc)
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	t.Fatal("sheet1.xml tidak ada di berkas ekspor")
	return ""
}

type fixture struct {
	pool    *pgxpool.Pool
	svc     *Service
	manajer Actor
	admin   Actor
	dina    Actor // Staf Marketing
	rio     Actor // Staf Creative
}

func setup(t *testing.T) fixture {
	t.Helper()
	pool := testdb.Pool(t)
	svc := NewService(pool)
	// Jam palsu: pukul 10.00 WIB di hariIni, supaya aturan "hanya hari ini" tidak bergantung tanggal sistem.
	svc.now = func() time.Time {
		d, err := time.ParseInLocation(dayLayout, hariIni, officeTZ)
		if err != nil {
			t.Fatal(err)
		}
		return d.Add(10 * time.Hour)
	}
	marketing := testdb.Division(t, pool, "Marketing", true)
	creative := testdb.Division(t, pool, "Creative", true)
	return fixture{
		pool:    pool,
		svc:     svc,
		manajer: Actor{ID: testdb.User(t, pool, "manajer.a", "manager", marketing, true), Role: sqlc.UserRoleManager},
		admin:   Actor{ID: testdb.User(t, pool, "admin.a", "admin", marketing, true), Role: sqlc.UserRoleAdmin},
		dina:    Actor{ID: testdb.User(t, pool, "dina", "staff", marketing, true), Role: sqlc.UserRoleStaff},
		rio:     Actor{ID: testdb.User(t, pool, "rio", "staff", creative, true), Role: sqlc.UserRoleStaff},
	}
}

func (f fixture) cell(t *testing.T, userID int64, date string, slot int) Cell {
	t.Helper()
	mine, err := f.svc.Mine(context.Background(), userID, date)
	if err != nil {
		t.Fatal(err)
	}
	return mine.Cells[slot-1]
}

func (f fixture) countRows(t *testing.T, userID int64, date string, slot int) int {
	t.Helper()
	var n int
	err := f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM daily_activities WHERE user_id = $1 AND activity_date = $2 AND slot = $3`,
		userID, date, slot).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func (f fixture) auditCount(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_logs WHERE action = 'activity_updated'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestStafMenyimpanIsianHariIni(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	err := f.svc.Save(ctx, f.dina, SaveInput{
		UserID: f.dina.ID, Date: hariIni, Slot: 1, Activity: " Edit video  ", Output: "https://drive.example/1",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := f.cell(t, f.dina.ID, hariIni, 1)
	if got.Activity != "Edit video" || got.Output != "https://drive.example/1" {
		t.Fatalf("isian tersimpan %+v", got)
	}
	// Isian sendiri tidak masuk audit log (keputusan revisi 21).
	if n := f.auditCount(t); n != 0 {
		t.Fatalf("audit log %d baris, mau 0", n)
	}

	// Simpan kedua di sel yang sama hanya memperbarui satu baris, bukan menambah baris.
	if err := f.svc.Save(ctx, f.dina, SaveInput{UserID: f.dina.ID, Date: hariIni, Slot: 1, Activity: "Revisi video"}); err != nil {
		t.Fatal(err)
	}
	if n := f.countRows(t, f.dina.ID, hariIni, 1); n != 1 {
		t.Fatalf("baris di sel yang sama %d, mau 1", n)
	}
	after := f.cell(t, f.dina.ID, hariIni, 1)
	if after.Activity != "Revisi video" || after.Output != "" {
		t.Fatalf("isian setelah simpan kedua %+v", after)
	}
}

func TestStafTidakBisaMengubahIsianLain(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	err := f.svc.Save(ctx, f.dina, SaveInput{UserID: f.dina.ID, Date: kemarin, Slot: 2, Activity: "Lupa diisi"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("isian kemarin harus ditolak, dapat %v", err)
	}
	err = f.svc.Save(ctx, f.dina, SaveInput{UserID: f.rio.ID, Date: hariIni, Slot: 2, Activity: "Punya orang lain"})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("isian orang lain harus ditolak, dapat %v", err)
	}
	// Admin hanya membaca, walau isian itu miliknya sendiri.
	if err := f.svc.Save(ctx, f.admin, SaveInput{UserID: f.admin.ID, Date: hariIni, Slot: 1}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Admin harus ditolak, dapat %v", err)
	}
	if n := f.countRows(t, f.dina.ID, kemarin, 2); n != 0 {
		t.Fatalf("baris tersimpan padahal ditolak: %d", n)
	}
}

func TestManajerMengubahIsianOrangLain(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	if err := f.svc.Save(ctx, f.dina, SaveInput{UserID: f.dina.ID, Date: hariIni, Slot: 3, Activity: "Riset"}); err != nil {
		t.Fatal(err)
	}
	// Isian yang disimpan pemiliknya sendiri tidak menampilkan "diubah <nama>".
	if sendiri := f.cell(t, f.dina.ID, hariIni, 3); sendiri.UpdatedBy != "" {
		t.Fatalf("isian sendiri tidak boleh bertanda diubah: %+v", sendiri)
	}
	// Manajer boleh tanggal lampau sekalipun.
	if err := f.svc.Save(ctx, f.manajer, SaveInput{
		UserID: f.dina.ID, Date: bulanLalu, Slot: 4, Activity: "Rapat klien", Output: "Notulen",
	}); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Save(ctx, f.manajer, SaveInput{
		UserID: f.dina.ID, Date: hariIni, Slot: 3, Activity: "Riset pasar",
	}); err != nil {
		t.Fatal(err)
	}

	diubah := f.cell(t, f.dina.ID, hariIni, 3)
	if diubah.Activity != "Riset pasar" || diubah.UpdatedBy != "manajer.a" {
		t.Fatalf("sel yang diubah Manajer %+v, mau menampilkan nama Manajer", diubah)
	}
	if lampau := f.cell(t, f.dina.ID, bulanLalu, 4); lampau.Activity != "Rapat klien" || lampau.Output != "Notulen" {
		t.Fatalf("isian tanggal lampau %+v", lampau)
	}
	if n := f.auditCount(t); n != 2 {
		t.Fatalf("audit log %d baris, mau 2 (dua perubahan Manajer)", n)
	}
}

func TestManajerDanAdminBukanKolom(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	if err := f.svc.Save(ctx, f.dina, SaveInput{UserID: f.dina.ID, Date: hariIni, Slot: 1, Activity: "Edit video"}); err != nil {
		t.Fatal(err)
	}
	day, err := f.svc.Range(ctx, hariIni, hariIni)
	if err != nil {
		t.Fatal(err)
	}
	if len(day.People) != 2 {
		t.Fatalf("kolom %d, mau 2 (hanya Staf aktif): %+v", len(day.People), day.People)
	}
	// Urutan menurut divisi lalu nama: Creative sebelum Marketing.
	if day.People[0].Name != "rio" || day.People[1].Name != "dina" {
		t.Fatalf("urutan kolom %s, %s", day.People[0].Name, day.People[1].Name)
	}
	if len(day.Dates) != 1 || day.Dates[0] != hariIni {
		t.Fatalf("tanggal yang tampil %+v", day.Dates)
	}
	for _, p := range day.People {
		if p.ID == f.manajer.ID || p.ID == f.admin.ID {
			t.Fatalf("Manajer atau Admin ikut jadi kolom: %+v", p)
		}
		if len(p.Days[hariIni]) != len(Slots) {
			t.Fatalf("%s punya %d sel, mau %d", p.Name, len(p.Days[hariIni]), len(Slots))
		}
		if p.Total != len(Slots) {
			t.Fatalf("%s total %d, mau %d", p.Name, p.Total, len(Slots))
		}
	}
	if day.People[1].Days[hariIni][0].Activity != "Edit video" || day.People[1].Filled != 1 {
		t.Fatalf("isian Dina tidak masuk kolomnya: %+v", day.People[1])
	}
	if day.People[0].Filled != 0 {
		t.Fatal("Rio belum mengisi apa pun, filled harus 0")
	}
}

// Minggu dengan isian Sabtu: Senin sampai Jumat selalu tampil, Sabtu ikut karena ada isinya, Minggu tidak.
func TestRangeMingguMelewatiAkhirPekanKosong(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if err := f.svc.Save(ctx, f.manajer, SaveInput{UserID: f.dina.ID, Date: "2026-11-21", Slot: 1, Activity: "Sabtu lembur"}); err != nil {
		t.Fatal(err)
	}
	r, err := f.svc.Range(ctx, "2026-11-16", "2026-11-22")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"2026-11-16", "2026-11-17", "2026-11-18", "2026-11-19", "2026-11-20", "2026-11-21"}
	if len(r.Dates) != len(want) {
		t.Fatalf("tanggal yang tampil %+v, mau %+v", r.Dates, want)
	}
	for i, d := range want {
		if r.Dates[i] != d {
			t.Fatalf("tanggal ke-%d %q, mau %q", i, r.Dates[i], d)
		}
	}
	dina := r.People[1]
	if dina.Name != "dina" || dina.Filled != 1 || dina.Total != len(want)*len(Slots) {
		t.Fatalf("ringkasan Dina %+v", dina)
	}
	// Minggu yang tersaring tidak boleh ikut terkirim.
	if _, ada := dina.Days["2026-11-22"]; ada {
		t.Fatal("tanggal yang tidak tampil masih terkirim")
	}
	if _, err := f.svc.Range(ctx, "2026-01-01", "2026-03-01"); err == nil {
		t.Fatal("rentang lebih dari 31 hari harus ditolak")
	}

	// Mode Hari: satu tanggal yang diminta tegas tetap tampil walau Sabtu dan kosong, supaya ada sel untuk diisi.
	satu, err := f.svc.Range(ctx, "2026-11-28", "2026-11-28")
	if err != nil {
		t.Fatal(err)
	}
	if len(satu.Dates) != 1 || satu.Dates[0] != "2026-11-28" {
		t.Fatalf("Sabtu kosong di mode Hari harus tetap tampil, dapat %+v", satu.Dates)
	}
	if len(satu.People[0].Days["2026-11-28"]) != len(Slots) {
		t.Fatal("sel di tanggal itu harus lengkap walau kosong")
	}
	// Rentang: Sabtu kosong tetap disaring.
	minggu, err := f.svc.Range(ctx, "2026-11-23", "2026-11-29")
	if err != nil {
		t.Fatal(err)
	}
	if len(minggu.Dates) != 5 {
		t.Fatalf("akhir pekan kosong di rentang harus disaring, dapat %+v", minggu.Dates)
	}
}

func TestManajerTidakBisaMenitipkanKeManajer(t *testing.T) {
	f := setup(t)
	err := f.svc.Save(context.Background(), f.manajer, SaveInput{
		UserID: f.manajer.ID, Date: hariIni, Slot: 1, Activity: "Rapat",
	})
	if err == nil {
		t.Fatal("Manajer tidak punya tabel aktivitas, isiannya harus ditolak")
	}
	if n := f.countRows(t, f.manajer.ID, hariIni, 1); n != 0 {
		t.Fatalf("baris tersimpan untuk Manajer: %d", n)
	}
}

func TestEksporMelewatiAkhirPekanKosong(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	// 2026-11-16 Senin sampai 2026-11-22 Minggu. Hanya Senin dan Sabtu yang diisi.
	if err := f.svc.Save(ctx, f.manajer, SaveInput{UserID: f.dina.ID, Date: "2026-11-16", Slot: 1, Activity: "Senin"}); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Save(ctx, f.manajer, SaveInput{UserID: f.dina.ID, Date: "2026-11-21", Slot: 1, Activity: "Sabtu lembur"}); err != nil {
		t.Fatal(err)
	}
	data, name, err := f.svc.Export(ctx, "2026-11-16", "2026-11-22")
	if err != nil {
		t.Fatal(err)
	}
	if name != "aktivitas-harian-2026-11-16-sampai-2026-11-22.xlsx" {
		t.Fatalf("nama berkas %q", name)
	}
	sheet := sheetOf(t, data)
	for _, want := range []string{"Monday - 16", "Saturday - 21", "Sabtu lembur", "Daily Activity Monitoring"} {
		if !strings.Contains(sheet, want) {
			t.Fatalf("sheet tidak memuat %q", want)
		}
	}
	// Minggu tanpa isian tidak ditampilkan (keputusan revisi 21); Jumat tetap ada walau kosong.
	if strings.Contains(sheet, "Sunday - 22") {
		t.Fatal("hari Minggu tanpa isian tidak boleh muncul di ekspor")
	}
	if !strings.Contains(sheet, "Friday - 20") {
		t.Fatal("hari kerja tanpa isian tetap harus muncul")
	}

	// Rentang lebih dari 31 hari ditolak.
	if _, _, err := f.svc.Export(ctx, "2026-01-01", "2026-03-01"); err == nil {
		t.Fatal("rentang lebih dari 31 hari harus ditolak")
	}
	if _, _, err := f.svc.Export(ctx, "2026-11-20", "2026-11-19"); err == nil {
		t.Fatal("tanggal akhir sebelum tanggal awal harus ditolak")
	}
}
