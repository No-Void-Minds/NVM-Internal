package maintenance

import (
	"bytes"
	"context"
	"slices"
	"testing"
	"time"

	"nvm-internal/internal/storage"
	"nvm-internal/internal/testdb"
	"nvm-internal/internal/whiteboard"
)

// Salinan template (W4) memakai berkas gambar yang sama dengan templatenya (file_id), tanpa menyalin byte. Template
// yang dihapus permanen tidak boleh membawa berkas yang masih dipakai salinannya: baris wb_files, berkas di disk,
// dan thumbnail-nya bertahan sampai kartu terakhir yang memakainya ikut dihapus permanen.
func TestTemplateBerkasBertahanSetelahTemplateDihapusPermanen(t *testing.T) {
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

	// Aturan FK yang membuat skenario ini aman. wb_cards.file_id tanpa ON DELETE (NO ACTION): baris wb_files tidak
	// bisa terhapus selama ada kartu, termasuk di Tempat sampah, yang memakainya. wb_files.board_id ON DELETE SET
	// NULL: papan asal yang dihapus permanen tidak ikut menghapus baris berkasnya.
	rules := map[string]string{}
	rows, err := pool.Query(ctx, `
		SELECT conrelid::regclass::text || '.' || a.attname, c.confdeltype::text
		FROM pg_constraint c
		JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
		WHERE c.contype = 'f' AND c.confrelid IN ('wb_files'::regclass, 'wb_boards'::regclass)
		  AND conrelid IN ('wb_cards'::regclass, 'wb_files'::regclass)`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var col, rule string
		if err := rows.Scan(&col, &rule); err != nil {
			t.Fatal(err)
		}
		rules[col] = rule
	}
	rows.Close()
	if rules["wb_cards.file_id"] != "a" || rules["wb_files.board_id"] != "n" {
		t.Fatalf("aturan FK berubah (a = NO ACTION, n = SET NULL): %v", rules)
	}

	// 1. Papan A berisi kartu gambar dijadikan template; papan B dibuat dari template itu.
	cardA, err := svc.CreateCard(ctx, actor, whiteboard.RootID, whiteboard.CardInput{Kind: whiteboard.KindBoard, Title: "Template A", W: 200, H: 120})
	if err != nil {
		t.Fatal(err)
	}
	boardA := cardA.Ref.ID
	img, err := svc.Upload(ctx, actor, boardA, whiteboard.CardInput{W: 240, H: 160}, "poster.png", bytes.NewReader(pngBytes(t)))
	if err != nil || !img.File.Thumb {
		t.Fatalf("unggah gambar: %+v %v", img.File, err)
	}
	fileID := img.File.ID
	filePath, _ := store.PathFor(fileID)
	thumbPath, _ := store.ThumbPath(fileID)
	if _, err := svc.SetTemplate(ctx, actor, boardA, true); err != nil {
		t.Fatal(err)
	}
	cardB, err := svc.FromTemplate(ctx, actor, whiteboard.RootID, whiteboard.TemplateInput{TemplateID: boardA, X: 400, W: 200, H: 120})
	if err != nil {
		t.Fatal(err)
	}
	boardB := cardB.Ref.ID
	imageInB := func() *whiteboard.Card {
		t.Helper()
		v, err := svc.Board(ctx, actor, boardB)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range v.Cards {
			if c.Kind == whiteboard.KindImage {
				return &c
			}
		}
		return nil
	}
	if c := imageInB(); c == nil || c.File == nil || c.File.ID != fileID {
		t.Fatalf("kartu gambar di papan B tidak memakai berkas yang sama: %+v", c)
	}

	// 2. Papan A dihapus, lalu job harian dijalankan 31 hari kemudian (lewat 30 hari Tempat sampah).
	if _, _, err := svc.DeleteCards(ctx, actor, whiteboard.RootID, []string{cardA.ID}, nil); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	r, err := Run(ctx, pool, store, now.AddDate(0, 0, 31), false)
	if err != nil {
		t.Fatal(err)
	}
	if r.WhiteboardBatches != 1 {
		t.Fatalf("batch dihapus permanen %d, mau 1", r.WhiteboardBatches)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM wb_boards WHERE id = $1`, boardA).Scan(&n); err != nil || n != 0 {
		t.Fatalf("papan A masih ada (%d, %v)", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM wb_cards WHERE id = $1`, img.ID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("kartu gambar papan A masih ada (%d, %v)", n, err)
	}

	// 3. Kartu gambar papan B, baris berkas, berkas di disk, dan thumbnail-nya masih ada dan masih bisa dibuka.
	if c := imageInB(); c == nil || c.File == nil || c.File.ID != fileID {
		t.Fatalf("kartu gambar papan B hilang setelah template dihapus permanen: %+v", c)
	}
	var boardNull bool
	if err := pool.QueryRow(ctx, `SELECT board_id IS NULL FROM wb_files WHERE id = $1`, fileID).Scan(&boardNull); err != nil {
		t.Fatalf("baris wb_files hilang: %v", err)
	}
	if !boardNull {
		t.Fatal("board_id berkas tidak menjadi NULL setelah papan asalnya dihapus permanen")
	}
	for _, p := range []string{filePath, thumbPath} {
		if !exists(t, p) {
			t.Fatalf("%s terhapus padahal masih dipakai papan B", p)
		}
		if slices.Contains(r.OrphanAttachments, p) {
			t.Fatalf("%s dilaporkan yatim", p)
		}
	}
	for _, thumb := range []bool{false, true} {
		_, f, err := svc.OpenFile(ctx, fileID, thumb)
		if err != nil {
			t.Fatalf("buka berkas (thumbnail %v) setelah template dihapus permanen: %v", thumb, err)
		}
		_ = f.Close()
	}
	// Run berikutnya juga tidak menyentuhnya (pembersih berkas yatim berjalan setelah baris berkas dibuang).
	if r, err = Run(ctx, pool, store, now.AddDate(0, 0, 33), false); err != nil {
		t.Fatal(err)
	}
	if !exists(t, filePath) || !exists(t, thumbPath) {
		t.Fatal("berkas yang dipakai papan B terhapus di run berikutnya")
	}

	// 4. Papan B juga dihapus permanen: baru sekarang baris berkas dibuang dan berkas di disk dianggap yatim.
	if _, _, err := svc.DeleteCards(ctx, actor, whiteboard.RootID, []string{cardB.ID}, nil); err != nil {
		t.Fatal(err)
	}
	if r, err = Run(ctx, pool, store, now.AddDate(0, 0, 70), false); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM wb_files WHERE id = $1`, fileID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("baris wb_files masih ada setelah kartu terakhir dihapus permanen (%d, %v)", n, err)
	}
	for _, p := range []string{filePath, thumbPath} {
		if exists(t, p) {
			t.Fatalf("%s tidak dihapus padahal sudah yatim", p)
		}
		if !slices.Contains(r.OrphanAttachments, p) {
			t.Fatalf("%s tidak tercatat di laporan yatim: %v", p, r.OrphanAttachments)
		}
	}
}
