package activity

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

// pngBytes adalah PNG terkecil yang lolos pemeriksaan magic bytes.
func pngBytes() []byte {
	return append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)
}

func (f fixture) upload(t *testing.T, actor Actor, userID int64, date string, slot int, name string, content []byte) (FileInfo, error) {
	t.Helper()
	return f.svc.Upload(context.Background(), actor, userID, date, slot, name, strings.NewReader(string(content)))
}

func (f fixture) fileCount(t *testing.T, userID int64, date string) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM activity_attachments WHERE user_id = $1 AND activity_date = $2`,
		userID, date).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (f fixture) onDisk(t *testing.T, id string) bool {
	t.Helper()
	p, err := f.store.PathFor(id)
	if err != nil {
		t.Fatal(err)
	}
	_, err = os.Stat(p)
	return err == nil
}

func TestLampiranStafHariIni(t *testing.T) {
	f := setup(t)

	got, err := f.upload(t, f.dina, f.dina.ID, hariIni, 1, "tangkapan.png", pngBytes())
	if err != nil {
		t.Fatal(err)
	}
	if got.MIME != "image/png" || got.Name != "tangkapan.png" || got.Size == 0 {
		t.Fatalf("lampiran tersimpan %+v", got)
	}
	if !f.onDisk(t, got.ID) {
		t.Fatal("berkasnya tidak ada di disk")
	}
	// Lampiran ikut terbaca di selnya, dan jam itu dihitung terisi walau teksnya kosong.
	mine, err := f.svc.Mine(context.Background(), f.dina.ID, hariIni)
	if err != nil {
		t.Fatal(err)
	}
	if len(mine.Cells[0].Files) != 1 || mine.Cells[0].Files[0].ID != got.ID {
		t.Fatalf("sel jam pertama %+v", mine.Cells[0])
	}
	if mine.Cells[0].Files[0].UploadedBy != "" {
		t.Fatal("lampiran sendiri tidak boleh bertanda pengunggah")
	}
	day, err := f.svc.Range(context.Background(), hariIni, hariIni)
	if err != nil {
		t.Fatal(err)
	}
	if day.People[1].Filled != 1 {
		t.Fatalf("jam dengan lampiran saja harus dihitung terisi, dapat %d", day.People[1].Filled)
	}
	if n := f.auditCount(t); n != 0 {
		t.Fatalf("lampiran sendiri tidak masuk audit, dapat %d baris", n)
	}
}

func TestLampiranStafDitolak(t *testing.T) {
	f := setup(t)

	if _, err := f.upload(t, f.dina, f.dina.ID, kemarin, 1, "a.png", pngBytes()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("lampiran kemarin harus ditolak, dapat %v", err)
	}
	if _, err := f.upload(t, f.dina, f.rio.ID, hariIni, 1, "a.png", pngBytes()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("lampiran milik orang lain harus ditolak, dapat %v", err)
	}
	if n := f.fileCount(t, f.dina.ID, kemarin) + f.fileCount(t, f.rio.ID, hariIni); n != 0 {
		t.Fatalf("baris tersimpan padahal ditolak: %d", n)
	}
	// Jenis yang tidak diizinkan ditolak dari isinya, bukan dari ekstensinya saja.
	if _, err := f.upload(t, f.dina, f.dina.ID, hariIni, 1, "ikon.svg", []byte("<svg/>")); err == nil {
		t.Fatal("SVG harus ditolak")
	}
	if _, err := f.upload(t, f.dina, f.dina.ID, hariIni, 1, "foto.png", []byte("MZ\x90\x00program")); err == nil {
		t.Fatal("EXE yang diganti ekstensi harus ditolak")
	}
}

func TestLampiranManajerDanHapus(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	// Manajer boleh menaruh bukti di jam orang lain, tanggal lampau sekalipun, dan itu tercatat.
	got, err := f.upload(t, f.manajer, f.dina.ID, bulanLalu, 2, "bukti.png", pngBytes())
	if err != nil {
		t.Fatal(err)
	}
	// Unggah dan hapus berkas punya aksi audit sendiri (activity_file_added dan activity_file_deleted, kategori
	// Aktivitas), bukan activity_updated yang khusus untuk isian teks.
	if n := f.auditActionCount(t, "activity_file_added"); n != 1 {
		t.Fatalf("audit activity_file_added %d baris, mau 1", n)
	}
	if n := f.auditCount(t); n != 0 {
		t.Fatalf("unggah berkas tidak boleh tercatat sebagai activity_updated, dapat %d", n)
	}
	mine, err := f.svc.Mine(ctx, f.dina.ID, bulanLalu)
	if err != nil {
		t.Fatal(err)
	}
	if len(mine.Cells[1].Files) != 1 || mine.Cells[1].Files[0].UploadedBy != "manajer.a" {
		t.Fatalf("lampiran dari Manajer harus bertanda namanya: %+v", mine.Cells[1].Files)
	}

	// Staf tidak boleh menghapus lampiran di tanggal lampau miliknya sendiri.
	if err := f.svc.DeleteFile(ctx, f.dina, got.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("hapus di tanggal lampau oleh Staf harus ditolak, dapat %v", err)
	}
	if err := f.svc.DeleteFile(ctx, f.manajer, got.ID); err != nil {
		t.Fatal(err)
	}
	if f.fileCount(t, f.dina.ID, bulanLalu) != 0 {
		t.Fatal("baris lampiran harus hilang setelah dihapus")
	}
	if f.onDisk(t, got.ID) {
		t.Fatal("berkasnya harus ikut hilang dari disk")
	}
	if n := f.auditActionCount(t, "activity_file_deleted"); n != 1 {
		t.Fatalf("audit activity_file_deleted %d baris, mau 1", n)
	}
	if n := f.auditActionCount(t, "activity_file_added"); n != 1 {
		t.Fatalf("audit activity_file_added %d baris setelah hapus, mau tetap 1", n)
	}
	if err := f.svc.DeleteFile(ctx, f.manajer, got.ID); !errors.Is(err, ErrFileMissing) {
		t.Fatalf("hapus dua kali harus 404, dapat %v", err)
	}
}

func TestLampiranMaksimalSepuluhPerJam(t *testing.T) {
	f := setup(t)

	for i := 0; i < MaxFilesPerSlot; i++ {
		if _, err := f.upload(t, f.dina, f.dina.ID, hariIni, 3, "a.png", pngBytes()); err != nil {
			t.Fatalf("unggahan ke-%d gagal: %v", i+1, err)
		}
	}
	_, err := f.upload(t, f.dina, f.dina.ID, hariIni, 3, "a.png", pngBytes())
	if err == nil || !strings.Contains(err.Error(), "maksimal 10 berkas") {
		t.Fatalf("berkas ke-11 harus ditolak dengan pesan batas, dapat %v", err)
	}
	if n := f.fileCount(t, f.dina.ID, hariIni); n != MaxFilesPerSlot {
		t.Fatalf("baris tersimpan %d, mau %d", n, MaxFilesPerSlot)
	}
	// Jam lain masih punya kuotanya sendiri.
	if _, err := f.upload(t, f.dina, f.dina.ID, hariIni, 4, "a.png", pngBytes()); err != nil {
		t.Fatal(err)
	}
}

func TestLampiranAkhirPekanTampilDiMinggu(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	// Sabtu 21 November 2026, hanya berisi lampiran tanpa teks.
	if _, err := f.upload(t, f.manajer, f.dina.ID, "2026-11-21", 1, "lembur.png", pngBytes()); err != nil {
		t.Fatal(err)
	}
	r, err := f.svc.Range(ctx, "2026-11-16", "2026-11-22")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Dates) != 6 || r.Dates[5] != "2026-11-21" {
		t.Fatalf("Sabtu dengan lampiran harus tampil, dapat %+v", r.Dates)
	}
}

func (f fixture) auditActionCount(t *testing.T, action string) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_logs WHERE action = $1`, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
