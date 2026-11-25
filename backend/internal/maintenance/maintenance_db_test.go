package maintenance

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"

	"nvm-internal/internal/storage"
	"nvm-internal/internal/testdb"
	"nvm-internal/internal/whiteboard"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 40, 20))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func exists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return err == nil
}

// Job harian tidak boleh menghapus berkas Whiteboard yang masih dipakai kartu (termasuk kartu di Tempat sampah)
// maupun thumbnail .thumb.jpg-nya, dan tetap menghapus berkas yatim sungguhan beserta thumbnail yatim.
func TestRunMenjagaBerkasWhiteboard(t *testing.T) {
	ctx := context.Background()
	pool := testdb.Pool(t)
	div := testdb.Division(t, pool, "Marketing", true)
	actor := whiteboard.Actor{ID: testdb.User(t, pool, "rina", "staff", div, true)}
	store, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := whiteboard.NewService(pool)
	svc.UseStorage(store)
	upload := func(name string, data []byte) whiteboard.Card {
		t.Helper()
		c, err := svc.Upload(ctx, actor, whiteboard.RootID, whiteboard.CardInput{X: 0, Y: 0, W: 240, H: 160}, name, bytes.NewReader(data))
		if err != nil {
			t.Fatalf("unggah %s: %v", name, err)
		}
		return c
	}
	paths := func(c whiteboard.Card) (string, string) {
		t.Helper()
		p, _ := store.PathFor(c.File.ID)
		th, _ := store.ThumbPath(c.File.ID)
		return p, th
	}

	img := upload("poster.png", pngBytes(t))
	if img.Kind != whiteboard.KindImage || !img.File.Thumb {
		t.Fatalf("kartu gambar: %+v", img.File)
	}
	imgPath, imgThumb := paths(img)

	trashed := upload("catatan.txt", []byte("isi berkas teks"))
	if _, _, err := svc.DeleteCards(ctx, actor, whiteboard.RootID, []string{trashed.ID}, nil); err != nil {
		t.Fatal(err)
	}
	trashedPath, _ := paths(trashed)

	// Berkas yang kartunya sudah hilang dari database (dihapus permanen): baris wb_files ikut dibuang Purge setelah
	// sehari, lalu berkas dan thumbnail-nya dibersihkan sebagai berkas yatim.
	gone := upload("lama.png", pngBytes(t))
	if _, err := pool.Exec(ctx, `DELETE FROM wb_cards WHERE id = $1`, gone.ID); err != nil {
		t.Fatal(err)
	}
	gonePath, goneThumb := paths(gone)

	orphanID := "0b0b0b0b-0000-4000-8000-000000000001"
	orphan, _ := store.PathFor(orphanID)
	orphanThumb, _ := store.ThumbPath(orphanID)
	if err := os.MkdirAll(strings.TrimSuffix(orphan, orphanID), 0o750); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{orphan, orphanThumb} {
		if err := os.WriteFile(p, []byte("yatim"), 0o640); err != nil {
			t.Fatal(err)
		}
	}

	// Dua hari ke depan: lewat batas berkas tanpa kartu (1 hari) dan orphanGrace, belum lewat 30 hari Tempat sampah.
	r, err := Run(ctx, pool, store, time.Now().Add(48*time.Hour), false)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{imgPath, imgThumb, trashedPath} {
		if !exists(t, p) {
			t.Errorf("%s terhapus padahal masih dipakai kartu", p)
		}
	}
	for _, p := range []string{orphan, orphanThumb, gonePath, goneThumb} {
		if exists(t, p) {
			t.Errorf("%s tidak terhapus padahal yatim", p)
		}
	}
	if len(r.OrphanAttachments) != 4 || len(r.Unknown) != 0 {
		t.Errorf("laporan: yatim %v, tak dikenal %v", r.OrphanAttachments, r.Unknown)
	}
	var rows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM wb_files`).Scan(&rows); err != nil || rows != 2 {
		t.Errorf("baris wb_files tersisa %d (%v), mau 2", rows, err)
	}
}
