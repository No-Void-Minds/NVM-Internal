package activity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// Revisi 33: rantai sambung jam.

func (f fixture) link(t *testing.T, actor Actor, userID int64, slot int, linked bool) error {
	t.Helper()
	return f.svc.Link(context.Background(), actor, LinkInput{UserID: userID, Date: hariIni, Slot: slot, Linked: linked})
}

func (f fixture) save(t *testing.T, slot int, activity, output string) {
	t.Helper()
	err := f.svc.Save(context.Background(), f.dina, SaveInput{
		UserID: f.dina.ID, Date: hariIni, Slot: slot, Activity: activity, Output: output,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func isConflict(err error) bool {
	var c *Conflict
	return errors.As(err, &c)
}

func TestRantaiPasangDanLepas(t *testing.T) {
	f := setup(t)
	f.save(t, 1, "Edit video", "https://drive.example/1")
	if _, err := f.upload(t, f.dina, f.dina.ID, hariIni, 1, "bukti.png", pngBytes()); err != nil {
		t.Fatal(err)
	}

	if err := f.link(t, f.dina, f.dina.ID, 2, true); err != nil {
		t.Fatal(err)
	}
	got := f.cell(t, f.dina.ID, hariIni, 2)
	if !got.Linked || got.Head != 1 || got.Activity != "Edit video" || got.Output != "https://drive.example/1" ||
		len(got.Files) != 1 || got.Span != "09.00-11.00" {
		t.Fatalf("jam tersambung %+v", got)
	}
	if head := f.cell(t, f.dina.ID, hariIni, 1); head.Span != "09.00-11.00" || head.Linked {
		t.Fatalf("jam kepala %+v", head)
	}
	// Isi tidak disalin ke database: baris jam tersambung tetap kosong.
	var activity *string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT activity FROM daily_activities WHERE user_id = $1 AND activity_date = $2 AND slot = 2`,
		f.dina.ID, hariIni).Scan(&activity); err != nil || activity != nil {
		t.Fatalf("baris jam 2: activity %v, err %v", activity, err)
	}
	// Edit di jam kepala langsung tampil di jam tersambung.
	f.save(t, 1, "Edit video final", "")
	if got := f.cell(t, f.dina.ID, hariIni, 2); got.Activity != "Edit video final" {
		t.Fatalf("jam tersambung tidak mengikuti kepala: %+v", got)
	}
	// Klik ganda tidak mengubah apa pun.
	if err := f.link(t, f.dina, f.dina.ID, 2, true); err != nil {
		t.Fatalf("pasang ulang: %v", err)
	}

	// Jam tersambung tidak bisa diisi atau diberi berkas sendiri.
	err := f.svc.Save(context.Background(), f.dina, SaveInput{UserID: f.dina.ID, Date: hariIni, Slot: 2, Activity: "x"})
	if !isConflict(err) {
		t.Fatalf("isi jam tersambung: mau Conflict, dapat %v", err)
	}
	if _, err := f.upload(t, f.dina, f.dina.ID, hariIni, 2, "b.png", pngBytes()); !isConflict(err) {
		t.Fatalf("unggah ke jam tersambung: mau Conflict, dapat %v", err)
	}

	if err := f.link(t, f.dina, f.dina.ID, 2, false); err != nil {
		t.Fatal(err)
	}
	got = f.cell(t, f.dina.ID, hariIni, 2)
	if got.Linked || got.Activity != "" || len(got.Files) != 0 || got.Span != "10.00-11.00" {
		t.Fatalf("setelah dilepas %+v", got)
	}
	// Setelah lepas, jam itu bisa diisi lagi.
	f.save(t, 2, "Rapat", "")
	if n := f.auditCount(t); n != 0 {
		t.Fatalf("rantai milik sendiri tidak masuk audit, dapat %d", n)
	}
}

func TestRantaiTigaJamDanLepasDiTengah(t *testing.T) {
	f := setup(t)
	f.save(t, 1, "Syuting", "")
	for _, slot := range []int{2, 3} {
		if err := f.link(t, f.dina, f.dina.ID, slot, true); err != nil {
			t.Fatalf("sambung jam %d: %v", slot, err)
		}
	}
	if got := f.cell(t, f.dina.ID, hariIni, 3); got.Head != 1 || got.Activity != "Syuting" || got.Span != "09.00-12.00" {
		t.Fatalf("jam ketiga %+v", got)
	}

	// Lepas jam 10.00-11.00: jam 11.00-12.00 yang tersambung ke jam itu ikut lepas, jam kepala tetap.
	if err := f.link(t, f.dina, f.dina.ID, 2, false); err != nil {
		t.Fatal(err)
	}
	for _, slot := range []int{2, 3} {
		if got := f.cell(t, f.dina.ID, hariIni, slot); got.Linked || got.Activity != "" {
			t.Fatalf("jam %d masih tersambung: %+v", slot, got)
		}
	}
	if head := f.cell(t, f.dina.ID, hariIni, 1); head.Activity != "Syuting" || head.Span != "09.00-10.00" {
		t.Fatalf("jam kepala berubah %+v", head)
	}
}

func TestRantaiDitolak(t *testing.T) {
	f := setup(t)
	f.save(t, 1, "Desain", "")
	f.save(t, 2, "Sudah ada isi", "")

	// Jam berikutnya berisi teks.
	err := f.link(t, f.dina, f.dina.ID, 2, true)
	if !isConflict(err) || !strings.Contains(err.Error(), "Kosongkan jam ini dulu") {
		t.Fatalf("jam berisi: mau Conflict, dapat %v", err)
	}
	if hint := f.cell(t, f.dina.ID, hariIni, 2).LinkHint; hint != hintNotEmpty {
		t.Fatalf("link_hint %q", hint)
	}
	// Jam berikutnya hanya berisi berkas.
	if err := f.link(t, f.dina, f.dina.ID, 1, true); err == nil {
		t.Fatal("jam pertama tidak punya jam sebelumnya")
	}
	if _, err := f.upload(t, f.dina, f.dina.ID, hariIni, 6, "c.png", pngBytes()); err != nil {
		t.Fatal(err)
	}
	f.save(t, 5, "Ada isi", "")
	if err := f.link(t, f.dina, f.dina.ID, 6, true); !isConflict(err) {
		t.Fatalf("jam berisi berkas: mau Conflict, dapat %v", err)
	}
	// Jam sebelumnya kosong: tidak ada yang bisa disambung.
	if err := f.link(t, f.dina, f.dina.ID, 8, true); !isConflict(err) {
		t.Fatalf("jam sebelumnya kosong: mau Conflict, dapat %v", err)
	}

	// Milik orang lain dan hari lalu, sama dengan aturan isian.
	if err := f.link(t, f.rio, f.dina.ID, 3, false); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Staf lain: mau ErrForbidden, dapat %v", err)
	}
	if err := f.link(t, f.admin, f.dina.ID, 3, true); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Admin: mau ErrForbidden, dapat %v", err)
	}
	err = f.svc.Link(context.Background(), f.dina, LinkInput{UserID: f.dina.ID, Date: diLuarRentang, Slot: 2, Linked: true})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("di luar rentang: mau ErrForbidden, dapat %v", err)
	}
}

func TestRantaiManajerTercatatAudit(t *testing.T) {
	f := setup(t)
	f.save(t, 1, "Desain", "")
	if err := f.link(t, f.manajer, f.dina.ID, 2, true); err != nil {
		t.Fatal(err)
	}
	if n := f.auditCount(t); n != 1 {
		t.Fatalf("audit log %d baris, mau 1", n)
	}
}

func TestRantaiTerhitungTerisiDanEksporSatuRentang(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	f.save(t, 1, "Edit video panjang", "https://drive.example/9")
	for _, slot := range []int{2, 3} {
		if err := f.link(t, f.dina, f.dina.ID, slot, true); err != nil {
			t.Fatal(err)
		}
	}

	r, err := f.svc.Range(ctx, hariIni, hariIni)
	if err != nil {
		t.Fatal(err)
	}
	col := -1
	for i, p := range r.People {
		if p.ID != f.dina.ID {
			continue
		}
		col = i
		if p.Filled != 3 {
			t.Fatalf("jam terisi %d, mau 3", p.Filled)
		}
	}
	if col < 0 {
		t.Fatal("kolom Dina tidak ada")
	}

	data, _, err := f.svc.Export(ctx, hariIni, hariIni)
	if err != nil {
		t.Fatal(err)
	}
	sheet := sheetOf(t, data)
	if n := strings.Count(sheet, "Edit video panjang"); n != 1 {
		t.Fatalf("isi rantai tertulis %d kali, mau 1", n)
	}
	// Baris 4 sampai 6 adalah 09.00-10.00 sampai 11.00-12.00 hari pertama; dua kolom Dina digabung ke bawah.
	for _, c := range []string{colName(3 + 2*col), colName(4 + 2*col)} {
		want := fmt.Sprintf(`<mergeCell ref="%s4:%s6"/>`, c, c)
		if !strings.Contains(sheet, want) {
			t.Fatalf("sheet tidak memuat %s", want)
		}
	}
}

// Ubahan revisi 33: rantai boleh melintasi jam istirahat. Span dari awal kepala sampai akhir ekor, hitungan
// terisi tetap per slot, dan ekspor menggabungkan dua baris jamnya.
func TestRantaiBolehMelintasiIstirahat(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	f.save(t, 3, "Rapat klien", "")
	if c := f.cell(t, f.dina.ID, hariIni, 4); !c.Joinable || c.LinkHint != "" {
		t.Fatalf("13.00-14.00 harus bisa disambung: %+v", c)
	}
	if err := f.link(t, f.dina, f.dina.ID, 4, true); err != nil {
		t.Fatal(err)
	}
	got := f.cell(t, f.dina.ID, hariIni, 4)
	if !got.Linked || got.Head != 3 || got.Activity != "Rapat klien" || got.Span != "11.00-14.00" {
		t.Fatalf("jam setelah istirahat %+v", got)
	}

	r, err := f.svc.Range(ctx, hariIni, hariIni)
	if err != nil {
		t.Fatal(err)
	}
	col := -1
	for i, p := range r.People {
		if p.ID == f.dina.ID {
			col = i
			if p.Filled != 2 {
				t.Fatalf("jam terisi %d, mau 2", p.Filled)
			}
		}
	}
	if col < 0 {
		t.Fatal("kolom Dina tidak ada")
	}
	data, _, err := f.svc.Export(ctx, hariIni, hariIni)
	if err != nil {
		t.Fatal(err)
	}
	sheet := sheetOf(t, data)
	// Baris 6 dan 7 adalah 11.00-12.00 dan 13.00-14.00 hari pertama.
	for _, c := range []string{colName(3 + 2*col), colName(4 + 2*col)} {
		want := fmt.Sprintf(`<mergeCell ref="%s6:%s7"/>`, c, c)
		if !strings.Contains(sheet, want) {
			t.Fatalf("sheet tidak memuat %s", want)
		}
	}
}
