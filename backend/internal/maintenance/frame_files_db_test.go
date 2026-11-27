package maintenance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/storage"
	"nvm-internal/internal/testdb"
	"nvm-internal/internal/whiteboard"
)

func frameContent(shape string) json.RawMessage {
	return json.RawMessage(`{"shape":"` + shape + `","rotation":0,"fill":"putih","stroke":"tinta","stroke_width":2,"stroke_style":"solid","text":"","align":"center","font_size":16}`)
}

// Revisi 57b: gambar di bingkai memakai wb_cards.file_id. Gambarnya bertahan setelah job pembersih, setelah template
// dibuat dari papan, dan setelah kartu gambar asalnya dihapus permanen. Gambar yang dilepas lalu dibersihkan job
// harian membuat undo "isi lagi" gagal dengan pesan 400, bukan 500.
func TestBingkaiBergambar(t *testing.T) {
	ctx := context.Background()
	pool := testdb.Pool(t)
	div := testdb.Division(t, pool, "Marketing", true)
	actor := whiteboard.Actor{ID: testdb.User(t, pool, "nala", "staff", div, true)}
	store, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := whiteboard.NewService(pool)
	svc.UseStorage(store)
	holder, err := svc.CreateCard(ctx, actor, whiteboard.RootID, whiteboard.CardInput{Kind: whiteboard.KindBoard, Title: "Wireframe", W: 200, H: 120})
	if err != nil {
		t.Fatal(err)
	}
	board := holder.Ref.ID
	shape := func(s string) whiteboard.Card {
		t.Helper()
		c, err := svc.CreateCard(ctx, actor, board, whiteboard.CardInput{Kind: whiteboard.KindShape, W: 180, H: 340, Content: frameContent(s)})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	phone := shape("wf_phone")
	rect := shape("rect")
	src, err := svc.Upload(ctx, actor, board, whiteboard.CardInput{X: 400, W: 240, H: 160}, "layar.png", bytes.NewReader(pngBytes(t)))
	if err != nil {
		t.Fatal(err)
	}
	fileID := src.File.ID
	filePath, _ := store.PathFor(fileID)
	patch := func(id, file string) error {
		_, err := svc.PatchCards(ctx, actor, board, []whiteboard.CardPatch{{ID: id, FileID: &file}})
		return err
	}
	frameFile := func(id string) string {
		t.Helper()
		v, err := svc.Board(ctx, actor, board)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range v.Cards {
			if c.ID == id && c.File != nil {
				return c.File.ID
			}
		}
		return ""
	}

	// 1. Validasi: bingkai menerima gambar; persegi dan berkas bukan gambar ditolak 400.
	if err := patch(phone.ID, fileID); err != nil || frameFile(phone.ID) != fileID {
		t.Fatalf("isi bingkai: %v", err)
	}
	var ae *apperr.Validation
	if err := patch(rect.ID, fileID); !errors.As(err, &ae) {
		t.Fatalf("persegi diisi gambar: %v", err)
	}
	doc, err := svc.Upload(ctx, actor, board, whiteboard.CardInput{X: 700, W: 240, H: 120}, "catatan.pdf", bytes.NewReader([]byte("%PDF-1.4\n%%EOF\n")))
	if err == nil {
		if err := patch(phone.ID, doc.File.ID); !errors.As(err, &ae) {
			t.Fatalf("bingkai diisi PDF: %v", err)
		}
	}
	bad := frameContent("wf_phone")
	bad = json.RawMessage(strings.Replace(string(bad), `"font_size":16`, `"font_size":16,"image":{"fit":"stretch","x":50,"y":50}`, 1))
	if _, err := svc.PatchCards(ctx, actor, board, []whiteboard.CardPatch{{ID: phone.ID, Content: bad, BaseContentVersion: phone.ContentVersion}}); !errors.As(err, &ae) {
		t.Fatalf("cara isi tak dikenal diterima: %v", err)
	}

	// 2. Kartu gambar asal dihapus dan dihapus permanen, papan dijadikan template dan disalin, job jalan 31 hari
	// kemudian: bingkai di papan asal dan di salinan template tetap memakai berkas yang sama, berkas di disk tetap.
	if _, _, err := svc.DeleteCards(ctx, actor, board, []string{src.ID}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetTemplate(ctx, actor, board, true); err != nil {
		t.Fatal(err)
	}
	copyCard, err := svc.FromTemplate(ctx, actor, whiteboard.RootID, whiteboard.TemplateInput{TemplateID: board, X: 400, W: 200, H: 120})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if _, err := Run(ctx, pool, store, now.AddDate(0, 0, 31), false); err != nil {
		t.Fatal(err)
	}
	if frameFile(phone.ID) != fileID || !exists(t, filePath) {
		t.Fatal("gambar bingkai hilang setelah kartu asalnya dihapus permanen dan job pembersih")
	}
	cv, err := svc.Board(ctx, actor, copyCard.Ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	inCopy := false
	for _, c := range cv.Cards {
		if c.Kind == whiteboard.KindShape && c.File != nil && c.File.ID == fileID {
			inCopy = true
		}
	}
	if !inCopy {
		t.Fatal("bingkai di salinan template tidak memakai gambar yang sama")
	}
	if _, err := svc.SetTemplate(ctx, actor, board, false); err != nil {
		t.Fatal(err)
	}
	// Salinan template dihapus permanen juga, supaya bingkai di papan asal satu-satunya pemakai berkas itu.
	if _, _, err := svc.DeleteCards(ctx, actor, whiteboard.RootID, []string{copyCard.ID}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(ctx, pool, store, now.AddDate(0, 0, 62), false); err != nil {
		t.Fatal(err)
	}

	// 3. Lepas gambar, job harian berjalan (berkas sudah lebih dari sehari dan tidak dipakai kartu mana pun), lalu undo
	// "isi lagi" dengan berkas lama: ditolak dengan pesan galat pengguna (400), bukan galat server.
	if err := patch(phone.ID, ""); err != nil || frameFile(phone.ID) != "" {
		t.Fatalf("lepas gambar: %v", err)
	}
	if _, err := Run(ctx, pool, store, now.AddDate(0, 0, 63), false); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM wb_files WHERE id = $1`, fileID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("baris berkas yang dilepas tidak dibersihkan (%d, %v)", n, err)
	}
	err = patch(phone.ID, fileID)
	if !errors.As(err, &ae) || !strings.Contains(err.Error(), "sudah tidak ada") {
		t.Fatalf("undo isi gambar yang sudah dibersihkan: mau galat 400 berpesan, dapat %v", err)
	}

	// 4. Migrasi 000030 down gagal dengan jumlah baris selama ada bentuk bergambar (di transaksi yang dibatalkan).
	img, err := svc.Upload(ctx, actor, board, whiteboard.CardInput{X: 900, W: 240, H: 160}, "lagi.png", bytes.NewReader(pngBytes(t)))
	if err != nil {
		t.Fatal(err)
	}
	if err := patch(phone.ID, img.File.ID); err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000030_whiteboard_shape_image.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, string(down)); err == nil || !strings.Contains(err.Error(), "bentuk bergambar") {
		t.Fatalf("down 000030 dengan bentuk bergambar: %v", err)
	}
}
