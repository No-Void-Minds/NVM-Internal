package todo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/testdb"
)

type fixture struct {
	pool *pgxpool.Pool
	svc  *Service
	dina int64
	rio  int64
}

func setup(t *testing.T) fixture {
	t.Helper()
	pool := testdb.Pool(t)
	div := testdb.Division(t, pool, "Marketing", true)
	return fixture{
		pool: pool,
		svc:  NewService(pool),
		dina: testdb.User(t, pool, "dina", "staff", div, true),
		rio:  testdb.User(t, pool, "rio", "staff", div, true),
	}
}

func (f fixture) titles(t *testing.T, user int64) []string {
	t.Helper()
	items, err := f.svc.List(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(items))
	for _, i := range items {
		if i.Done {
			out = append(out, i.Title+" (selesai)")
			continue
		}
		out = append(out, i.Title)
	}
	return out
}

func (f fixture) add(t *testing.T, user int64, title string) Item {
	t.Helper()
	item, err := f.svc.Add(context.Background(), user, title)
	if err != nil {
		t.Fatal(err)
	}
	return item
}

// To Do pribadi: hanya pemiliknya yang bisa membaca dan mengubah, dan id milik orang lain diperlakukan
// sebagai tidak ada (404), bukan ditolak (403).
func TestToDoHanyaMilikSendiri(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	milikDina := f.add(t, f.dina, "  Kirim laporan  ")
	if milikDina.Title != "Kirim laporan" {
		t.Fatalf("judul dirapikan jadi %q", milikDina.Title)
	}
	f.add(t, f.rio, "Punya Rio")

	// Daftar masing-masing hanya memuat miliknya.
	if got := f.titles(t, f.dina); len(got) != 1 || got[0] != "Kirim laporan" {
		t.Fatalf("daftar Dina %+v", got)
	}
	if got := f.titles(t, f.rio); len(got) != 1 || got[0] != "Punya Rio" {
		t.Fatalf("daftar Rio %+v", got)
	}

	// Rio menyentuh item Dina: selalu ErrNotFound, bukan galat hak akses.
	for nama, err := range map[string]error{
		"ubah":    f.svc.Rename(ctx, f.rio, milikDina.ID, "Dibajak"),
		"centang": f.svc.SetDone(ctx, f.rio, milikDina.ID, true),
		"hapus":   f.svc.Delete(ctx, f.rio, milikDina.ID),
		"urutkan": f.svc.Reorder(ctx, f.rio, []int64{milikDina.ID}),
	} {
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("%s item orang lain: %v, mau ErrNotFound", nama, err)
		}
	}
	// Item Dina tidak berubah sedikit pun.
	if got := f.titles(t, f.dina); len(got) != 1 || got[0] != "Kirim laporan" {
		t.Fatalf("item Dina berubah: %+v", got)
	}

	// Pemiliknya sendiri boleh semuanya.
	if err := f.svc.Rename(ctx, f.dina, milikDina.ID, "Kirim laporan mingguan"); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.SetDone(ctx, f.dina, milikDina.ID, true); err != nil {
		t.Fatal(err)
	}
	if got := f.titles(t, f.dina); got[0] != "Kirim laporan mingguan (selesai)" {
		t.Fatalf("setelah diubah dan dicentang: %+v", got)
	}
	if err := f.svc.SetDone(ctx, f.dina, milikDina.ID, false); err != nil {
		t.Fatal(err)
	}
	if got := f.titles(t, f.dina); got[0] != "Kirim laporan mingguan" {
		t.Fatalf("setelah batal centang: %+v", got)
	}
}

func TestToDoBatasSeratus(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	for i := 0; i < MaxItems; i++ {
		f.add(t, f.dina, fmt.Sprintf("item %d", i+1))
	}
	_, err := f.svc.Add(ctx, f.dina, "kelebihan")
	if err == nil || !strings.Contains(err.Error(), "penuh") {
		t.Fatalf("item ke-101 harus ditolak dengan pesan penuh, dapat %v", err)
	}
	// Akun lain punya kuotanya sendiri.
	f.add(t, f.rio, "punya Rio")

	// Judul kosong dan terlalu panjang ditolak.
	if _, err := f.svc.Add(ctx, f.rio, "   "); err == nil {
		t.Fatal("judul kosong harus ditolak")
	}
	if _, err := f.svc.Add(ctx, f.rio, strings.Repeat("a", MaxTitle+1)); err == nil {
		t.Fatal("judul lebih dari batas harus ditolak")
	}
}

// Urutkan ulang hanya menerima id milik sendiri, dan satu id asing membatalkan seluruh permintaan.
func TestToDoUrutkanUlang(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	a := f.add(t, f.dina, "a")
	b := f.add(t, f.dina, "b")
	cc := f.add(t, f.dina, "c")
	asing := f.add(t, f.rio, "punya Rio")

	if err := f.svc.Reorder(ctx, f.dina, []int64{cc.ID, a.ID, b.ID}); err != nil {
		t.Fatal(err)
	}
	if got := f.titles(t, f.dina); len(got) != 3 || got[0] != "c" || got[1] != "a" || got[2] != "b" {
		t.Fatalf("urutan setelah disusun: %+v", got)
	}

	// Satu id milik orang lain: ditolak, dan urutan sebelumnya tidak berubah.
	err := f.svc.Reorder(ctx, f.dina, []int64{a.ID, asing.ID, b.ID})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("id orang lain: %v, mau ErrNotFound", err)
	}
	if got := f.titles(t, f.dina); got[0] != "c" {
		t.Fatalf("urutan ikut berubah padahal ditolak: %+v", got)
	}
	// Id kembar ditolak sebagai permintaan yang tidak sah.
	if err := f.svc.Reorder(ctx, f.dina, []int64{a.ID, a.ID}); err == nil {
		t.Fatal("id kembar harus ditolak")
	}
	// Item Rio tidak ikut terurut.
	if got := f.titles(t, f.rio); len(got) != 1 {
		t.Fatalf("daftar Rio %+v", got)
	}
}

// Bersihkan yang selesai hanya menghapus milik sendiri, dan hanya yang selesai.
func TestToDoBersihkanYangSelesai(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	selesai := f.add(t, f.dina, "sudah")
	f.add(t, f.dina, "belum")
	selesaiRio := f.add(t, f.rio, "sudah juga")
	if err := f.svc.SetDone(ctx, f.dina, selesai.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.SetDone(ctx, f.rio, selesaiRio.ID, true); err != nil {
		t.Fatal(err)
	}

	n, err := f.svc.ClearDone(ctx, f.dina)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("dibersihkan %d, mau 1", n)
	}
	if got := f.titles(t, f.dina); len(got) != 1 || got[0] != "belum" {
		t.Fatalf("sisa milik Dina %+v", got)
	}
	// Milik Rio utuh, termasuk yang sudah selesai.
	if got := f.titles(t, f.rio); len(got) != 1 || got[0] != "sudah juga (selesai)" {
		t.Fatalf("milik Rio ikut terhapus: %+v", got)
	}
}

// Item selesai turun ke bawah, dan yang belum selesai tetap urut posisi.
func TestToDoYangSelesaiPindahKeBawah(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	a := f.add(t, f.dina, "a")
	f.add(t, f.dina, "b")
	f.add(t, f.dina, "c")
	if err := f.svc.SetDone(ctx, f.dina, a.ID, true); err != nil {
		t.Fatal(err)
	}
	got := f.titles(t, f.dina)
	if len(got) != 3 || got[0] != "b" || got[1] != "c" || got[2] != "a (selesai)" {
		t.Fatalf("urutan dengan item selesai: %+v", got)
	}
}
