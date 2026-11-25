package whiteboard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"net/netip"
	"strings"
	"testing"
	"time"

	"nvm-internal/internal/storage"
)

func (f wbFixture) withStore(t *testing.T) *storage.Store {
	t.Helper()
	st, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f.svc.UseStorage(st)
	return st
}

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestWbUnggahBerkas(t *testing.T) {
	f := setupWb(t)
	f.withStore(t)
	ctx := context.Background()
	at := CardInput{X: 0, Y: 0, W: 240, H: 160}

	img, err := f.svc.Upload(ctx, f.rina, RootID, at, "poster.png", bytes.NewReader(testPNG(t, 400, 100)))
	if err != nil {
		t.Fatal(err)
	}
	if img.Kind != KindImage || img.File == nil || !img.File.Thumb || img.File.Width != 400 || img.H != 60 {
		t.Fatalf("kartu gambar %+v berkas %+v", img, img.File)
	}
	row, file, err := f.svc.OpenFile(ctx, img.File.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	head := make([]byte, 3)
	_, _ = io.ReadFull(file, head)
	_ = file.Close()
	if !bytes.Equal(head, []byte{0xFF, 0xD8, 0xFF}) || row.OriginalName != "poster.png" {
		t.Fatalf("thumbnail bukan JPEG: %x", head)
	}

	doc, err := f.svc.Upload(ctx, f.rina, RootID, at, "catatan.txt", strings.NewReader("isi"))
	if err != nil || doc.Kind != KindFile || doc.File.Thumb {
		t.Fatalf("kartu berkas %+v %v", doc, err)
	}
	if _, _, err := f.svc.OpenFile(ctx, doc.File.ID, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("thumbnail berkas teks: %v", err)
	}

	svg := `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`
	_, err = f.svc.Upload(ctx, f.rina, RootID, at, "logo.svg", strings.NewReader(svg))
	wantInvalid(t, err, "JPG, PNG")
	_, err = f.svc.Upload(ctx, f.rina, RootID, at, "besar.txt", io.LimitReader(zeros{}, MaxFileSize+1))
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("berkas 50 MB lebih: %v", err)
	}

	// Salin tempel memakai berkas yang sama; jenis kartu harus cocok dengan jenis berkas.
	if _, err := f.svc.CreateCard(ctx, f.rina, RootID, CardInput{Kind: KindImage, FileID: img.File.ID, W: 240, H: 60}); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.CreateCard(ctx, f.rina, RootID, CardInput{Kind: KindImage, FileID: doc.File.ID, W: 240, H: 60})
	wantInvalid(t, err, "tidak ditemukan")
	_, err = f.svc.CreateCard(ctx, f.rina, RootID, CardInput{Kind: KindFile, W: 240, H: 60})
	wantInvalid(t, err, "unggahan")
}

type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func TestWbKolomDanPenghubung(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	col := f.card(t, f.root, KindColumn, `{"title":"Ide"}`)
	a := f.card(t, f.root, KindNote, "")
	b := f.card(t, f.root, KindHeading, `{"text":"Judul"}`)
	str := func(s string) *string { return &s }

	if _, err := f.svc.PatchCards(ctx, f.rina, RootID, []CardPatch{{ID: a.ID, ColumnID: str(col.ID)}}); err != nil {
		t.Fatal(err)
	}
	col2 := f.card(t, f.root, KindColumn, `{"title":"Lain"}`)
	_, err := f.svc.PatchCards(ctx, f.rina, RootID, []CardPatch{{ID: col2.ID, ColumnID: str(col.ID)}})
	wantInvalid(t, err, "kolom lain")
	_, err = f.svc.PatchCards(ctx, f.rina, RootID, []CardPatch{{ID: b.ID, ColumnID: str(a.ID)}})
	wantInvalid(t, err, "Kolom tujuan")

	_, err = f.svc.CreateConnector(ctx, f.rina, RootID, ConnectorInput{From: a.ID, To: a.ID})
	wantInvalid(t, err, "berbeda")
	_, err = f.svc.CreateConnector(ctx, f.rina, RootID, ConnectorInput{From: a.ID, To: b.ID, Style: ConnectorStyle{End: "bintang"}})
	wantInvalid(t, err, "Gaya garis")
	conn, err := f.svc.CreateConnector(ctx, f.rina, RootID, ConnectorInput{From: b.ID, To: a.ID, Label: "  lalu  "})
	if err != nil || conn.Label != "lalu" || conn.Style.End != "arrow" {
		t.Fatalf("garis %+v %v", conn, err)
	}
	view, _ := f.svc.Board(ctx, f.rina, RootID)
	seq := view.Seq

	// Hapus kolom: isinya ikut, garis yang menempel ikut ke batch yang sama, dan semuanya kembali saat dipulihkan.
	batch, _, err := f.svc.DeleteCards(ctx, f.rina, RootID, []string{col.ID}, nil)
	if err != nil {
		t.Fatal(err)
	}
	view, _ = f.svc.Board(ctx, f.rina, RootID)
	if len(view.Cards) != 2 || len(view.Connectors) != 0 {
		t.Fatalf("setelah hapus kolom: %d kartu, %d garis", len(view.Cards), len(view.Connectors))
	}
	sync, err := f.svc.Since(ctx, f.root, seq)
	if err != nil || len(sync.Deleted) != 2 || len(sync.ConnectorsGone) != 1 {
		t.Fatalf("sinkron ulang %+v %v", sync, err)
	}
	trash, _ := f.svc.Trash(ctx)
	if len(trash) != 1 || trash[0].Connectors != 1 || trash[0].Cards != 2 {
		t.Fatalf("tempat sampah %+v", trash)
	}
	if _, err := f.svc.Restore(ctx, f.rina, batch); err != nil {
		t.Fatal(err)
	}
	view, _ = f.svc.Board(ctx, f.rina, RootID)
	if len(view.Cards) != 4 || len(view.Connectors) != 1 {
		t.Fatalf("setelah pulihkan: %d kartu, %d garis", len(view.Cards), len(view.Connectors))
	}
	for _, c := range view.Cards {
		if c.ID == a.ID && (c.ColumnID == nil || *c.ColumnID != col.ID) {
			t.Fatalf("kartu keluar dari kolom setelah dipulihkan: %+v", c)
		}
	}

	if _, err := f.svc.DeleteConnector(ctx, f.rina, f.root, conn.ID); err != nil {
		t.Fatal(err)
	}
	view, _ = f.svc.Board(ctx, f.rina, RootID)
	if len(view.Connectors) != 0 {
		t.Fatal("garis masih tampil setelah dihapus")
	}
}

func TestWbKomentar(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	staf := Actor{ID: f.rina.ID}
	other := Actor{ID: f.nala.ID}
	manager := Actor{ID: f.nala.ID, Manager: true}
	note := f.card(t, f.root, KindNote, "")

	card, err := f.svc.AddComment(ctx, staf, RootID, note.ID, "  Cek ukuran gambar  ")
	if err != nil || card.Comments != 1 {
		t.Fatalf("komentar %+v %v", card, err)
	}
	_, err = f.svc.AddComment(ctx, staf, RootID, note.ID, "   ")
	wantInvalid(t, err, "kosong")
	list, _ := f.svc.Comments(ctx, staf, RootID, note.ID)
	if len(list) != 1 || list[0].Body != "Cek ukuran gambar" {
		t.Fatalf("daftar %+v", list)
	}
	other2 := f.board(t, f.root, "Lain")
	if _, err := f.svc.Comments(ctx, staf, other2.Ref.ID, note.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("komentar terbaca lewat papan lain: %v", err)
	}
	if _, err := f.svc.EditComment(ctx, other, list[0].ID, "ubah"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("orang lain mengubah komentar: %v", err)
	}
	if _, err := f.svc.EditComment(ctx, staf, list[0].ID, "Cek ukuran"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.DeleteComment(ctx, other, list[0].ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("bukan Manajer menghapus komentar orang lain: %v", err)
	}
	card, err = f.svc.DeleteComment(ctx, manager, list[0].ID)
	if err != nil || card.Comments != 0 {
		t.Fatalf("Manajer menghapus: %+v %v", card, err)
	}
}

func TestWbPindahPapan(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	child := f.board(t, f.root, "Anak")
	grand := f.board(t, child.Ref.ID, "Cucu")
	a := f.card(t, f.root, KindNote, "")
	b := f.card(t, f.root, KindNote, "")
	c := f.card(t, f.root, KindNote, "")
	keep, _ := f.svc.CreateConnector(ctx, f.rina, RootID, ConnectorInput{From: a.ID, To: b.ID})
	cut, _ := f.svc.CreateConnector(ctx, f.rina, RootID, ConnectorInput{From: b.ID, To: c.ID})
	view, _ := f.svc.Board(ctx, f.rina, RootID)
	seq := view.Seq

	res, err := f.svc.MoveCards(ctx, f.rina, RootID, []string{a.ID, b.ID}, child.Ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Moved) != 2 || len(res.ConnectorsGone) != 2 || res.Batch == "" {
		t.Fatalf("hasil pindah %+v", res)
	}
	dst, _ := f.svc.Board(ctx, f.rina, child.Ref.ID)
	if len(dst.Cards) != 3 || len(dst.Connectors) != 1 || dst.Connectors[0].ID != keep.ID {
		t.Fatalf("papan tujuan: %d kartu, garis %+v", len(dst.Cards), dst.Connectors)
	}
	src, _ := f.svc.Board(ctx, f.rina, RootID)
	for _, n := range src.Connectors {
		if n.ID == cut.ID {
			t.Fatal("garis yang terputus masih tampil di papan asal")
		}
	}
	// Klien papan asal yang tersambung ulang tidak bisa disusulkan per kartu, jadi diminta memuat ulang.
	if sync, err := f.svc.Since(ctx, f.root, seq); err != nil || !sync.TooMany {
		t.Fatalf("sinkron papan asal %+v %v", sync, err)
	}

	_, err = f.svc.MoveCards(ctx, f.rina, RootID, []string{child.ID}, grand.Ref.ID)
	wantInvalid(t, err, "dirinya sendiri")
	_, err = f.svc.MoveCards(ctx, f.rina, RootID, []string{c.ID}, RootID)
	wantInvalid(t, err, "sudah berada")

	// Kartu papan membawa papan anaknya: Cucu pindah ke akar, judul breadcrumb-nya ikut berubah.
	if _, err := f.svc.MoveCards(ctx, f.rina, child.Ref.ID, []string{grand.ID}, RootID); err != nil {
		t.Fatal(err)
	}
	g, _ := f.svc.Board(ctx, f.rina, grand.Ref.ID)
	if len(g.Crumbs) != 2 {
		t.Fatalf("breadcrumb Cucu setelah pindah: %+v", g.Crumbs)
	}
}

func TestWbTabel(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	tb := f.card(t, f.root, KindTable, "")
	card, err := f.svc.Table(ctx, f.rina, RootID, tb.ID, TableOp{Op: "set", Row: 0, Col: 1, Text: " Harga "})
	var got TableContent
	if err != nil || json.Unmarshal(card.Content, &got) != nil || !got.Header || got.Rows[0][1] != "Harga" || len(got.Rows) != 2 {
		t.Fatalf("tabel %s %v", card.Content, err)
	}
	note := f.card(t, f.root, KindNote, "")
	_, err = f.svc.Table(ctx, f.rina, RootID, note.ID, TableOp{Op: "header"})
	wantInvalid(t, err, "bukan tabel")
}

// Pratinjau ke alamat internal disimpan sebagai blocked tanpa menyentuh jaringan, dan dipakai ulang dari cache.
func TestWbPratinjauDiblokir(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	calls := 0
	f.svc.preview.Resolve = func(context.Context, string) ([]netip.Addr, error) {
		calls++
		return []netip.Addr{netip.MustParseAddr("10.0.0.7")}, nil
	}
	for range 2 {
		p, err := f.svc.Preview(ctx, f.rina, "https://Intranet.Contoh.test/a#b")
		if err != nil || p.Status != "blocked" || !strings.Contains(p.Message, "jaringan internal") {
			t.Fatalf("pratinjau %+v %v", p, err)
		}
	}
	if calls != 1 {
		t.Fatalf("resolve %d kali, mau 1 (cache)", calls)
	}
	_, err := f.svc.Preview(ctx, f.rina, "javascript:alert(1)")
	wantInvalid(t, err, "http")

	for range previewPerHour {
		f.svc.rate.allow(f.nala.ID, time.Now())
	}
	_, err = f.svc.Preview(ctx, f.nala, "https://lain.contoh.test/")
	wantInvalid(t, err, "Terlalu banyak")
}
