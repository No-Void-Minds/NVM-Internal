package driveexport

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// Senin 2 November 2026 pukul 03.45 WIB, jam job Drive malam: Staf masih boleh mengisi Jumat 30 dan Kamis
// 29 Oktober (revisi 37).
var seninAwalBulan = time.Date(2026, 11, 1, 20, 45, 0, 0, time.UTC)

func TestExportMonths(t *testing.T) {
	cases := []struct {
		name string
		now  time.Time
		want [][2]string
	}{
		{"pertengahan bulan", time.Date(2026, 11, 18, 2, 0, 0, 0, time.UTC), [][2]string{{"2026-11-01", "2026-11-18"}}},
		{"Senin awal bulan menulis ulang Oktober", seninAwalBulan, [][2]string{{"2026-10-01", "2026-10-31"}, {"2026-11-01", "2026-11-02"}}},
		// Rabu 2 Desember: Senin 30 November masih boleh diisi, jadi November ikut.
		{"Rabu awal bulan", time.Date(2026, 12, 1, 20, 45, 0, 0, time.UTC), [][2]string{{"2026-11-01", "2026-11-30"}, {"2026-12-01", "2026-12-02"}}},
		// Kamis 3 Desember: terawal Selasa 1 Desember, kemarin 2 Desember, jadi hanya Desember.
		{"Kamis awal bulan", time.Date(2026, 12, 2, 20, 45, 0, 0, time.UTC), [][2]string{{"2026-12-01", "2026-12-03"}}},
		// Selasa 1 Desember dini hari: kemarin 30 November belum pernah terekspor lengkap.
		{"tanggal 1", time.Date(2026, 11, 30, 20, 45, 0, 0, time.UTC), [][2]string{{"2026-11-01", "2026-11-30"}, {"2026-12-01", "2026-12-01"}}},
	}
	for _, c := range cases {
		if got := ExportMonths(c.now); !slices.Equal(got, c.want) {
			t.Errorf("%s: %v, mau %v", c.name, got, c.want)
		}
	}
}

// Revisi 37: isian dan lampiran yang ditambah Staf untuk tanggal lalu ikut terkirim pada ekspor berikutnya,
// termasuk bila tanggalnya di bulan lalu.
func TestEksporMengambilIsianTerlambat(t *testing.T) {
	f := setup(t)
	f.svc.now = func() time.Time { return seninAwalBulan }
	ctx := context.Background()
	var div, user int64
	if err := f.pool.QueryRow(ctx, `INSERT INTO divisions (name, is_active) VALUES ('Marketing', true) RETURNING id`).Scan(&div); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO users (name, username, password_hash, role, division_id, is_active, must_change_password)
		VALUES ('Dina', 'dina', 'x', 'staff', $1, true, false) RETURNING id`, div).Scan(&user); err != nil {
		t.Fatal(err)
	}
	f.exec(t, `INSERT INTO daily_activities (user_id, activity_date, slot, activity, output, updated_by)
		VALUES ($1, '2026-10-30', 2, 'Isian terlambat Jumat', '', $1)`, user)
	const late = "44444444-4444-4444-8444-444444444444"
	f.exec(t, `INSERT INTO activity_attachments (id, user_id, activity_date, slot, original_name, mime_type, size_bytes, checksum, uploaded_by)
		VALUES ($1, $2, '2026-10-29', 1, 'bukti.pdf', 'application/pdf', 10, repeat('a', 64), $2)`, late, user)
	f.file(t, late)

	m, err := f.svc.Stage(ctx, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	oktober := "Aktivitas/2026-10 Oktober/Aktivitas Harian Oktober 2026.xlsx"
	if !slices.Contains(m.Extra, oktober) || len(m.Extra) != 2 {
		t.Fatalf("ekspor bulanan %v, mau Oktober dan November", m.Extra)
	}
	if !strings.Contains(xlsxText(t, filepath.Join(f.svc.staging, filepath.FromSlash(oktober))), "Isian terlambat Jumat") {
		t.Fatal("isian terlambat 30 Oktober tidak ada di ekspor Oktober")
	}
	found := false
	for _, p := range m.Listing() {
		if strings.HasPrefix(p, "Aktivitas/2026-10 Oktober/") && strings.HasSuffix(p, "bukti.pdf") {
			found = true
		}
	}
	if !found {
		t.Fatalf("lampiran terlambat 29 Oktober tidak ikut disusun: %v", m.Listing())
	}
}

// xlsxText menggabungkan semua XML di dalam .xlsx, cukup untuk memeriksa sebuah teks ada di dalamnya.
func xlsxText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(strings.NewReader(string(data)), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, zf := range zr.File {
		rc, err := zf.Open()
		if err != nil {
			t.Fatal(err)
		}
		x, _ := io.ReadAll(rc)
		_ = rc.Close()
		b.Write(x)
	}
	return b.String()
}
