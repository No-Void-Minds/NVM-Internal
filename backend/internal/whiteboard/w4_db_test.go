package whiteboard

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/db/sqlc"
)

func TestWbTemplate(t *testing.T) {
	f := setupWb(t)
	f.withStore(t)
	ctx := context.Background()
	str := func(s string) *string { return &s }

	tpl := f.board(t, f.root, "Rencana kampanye")
	b := tpl.Ref.ID
	col := f.card(t, b, KindColumn, `{"title":"Ide"}`)
	n1 := f.card(t, b, KindNote, note("Tujuan kampanye"))
	h := f.card(t, b, KindHeading, `{"text":"Jadwal"}`)
	if _, err := f.svc.PatchCards(ctx, f.rina, b, []CardPatch{{ID: n1.ID, ColumnID: str(col.ID)}}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.CreateConnector(ctx, f.rina, b, ConnectorInput{From: h.ID, To: col.ID, Label: "lalu"}); err != nil {
		t.Fatal(err)
	}
	img, err := f.svc.Upload(ctx, f.rina, b, CardInput{W: 240, H: 160}, "poster.png", bytes.NewReader(testPNG(t, 40, 40)))
	if err != nil {
		t.Fatal(err)
	}
	sub := f.board(t, b, "Aset")
	f.card(t, sub.Ref.ID, KindChecklist, `{"title":"Daftar aset","items":[]}`)

	at := TemplateInput{TemplateID: b, X: 300, W: 200, H: 120}
	_, err = f.svc.FromTemplate(ctx, f.nala, RootID, at)
	wantInvalid(t, err, "bukan template")
	_, err = f.svc.SetTemplate(ctx, f.rina, RootID, true)
	wantInvalid(t, err, "Papan akar")
	if flag, err := f.svc.SetTemplate(ctx, f.rina, b, true); err != nil || !flag.IsTemplate {
		t.Fatalf("tandai template %+v %v", flag, err)
	}
	list, err := f.svc.Templates(ctx)
	if err != nil || len(list) != 1 || list[0].Title != "Rencana kampanye" || list[0].Path != "Whiteboard" || list[0].Cards != 5 {
		t.Fatalf("daftar template %+v %v", list, err)
	}
	if v, _ := f.svc.Board(ctx, f.rina, b); !v.IsTemplate {
		t.Fatal("is_template tidak ikut di tampilan papan")
	}

	card, err := f.svc.FromTemplate(ctx, f.nala, RootID, at)
	if err != nil || card.Kind != KindBoard || card.Ref == nil || card.Ref.Title != "Rencana kampanye" || card.Ref.ID == b || card.X != 300 {
		t.Fatalf("salin template %+v %v", card, err)
	}
	cp, err := f.svc.Board(ctx, f.nala, card.Ref.ID)
	if err != nil || cp.IsTemplate || len(cp.Cards) != 5 || len(cp.Connectors) != 1 {
		t.Fatalf("salinan: template %v, %d kartu, %d garis, %v", cp.IsTemplate, len(cp.Cards), len(cp.Connectors), err)
	}
	orig := map[string]bool{col.ID: true, n1.ID: true, h.ID: true, img.ID: true, sub.ID: true}
	byKind := map[string]Card{}
	for _, c := range cp.Cards {
		if orig[c.ID] {
			t.Fatalf("kartu salinan memakai id asli %s", c.ID)
		}
		byKind[c.Kind] = c
	}
	if nc := byKind[KindNote]; nc.ColumnID == nil || *nc.ColumnID != byKind[KindColumn].ID {
		t.Fatalf("isi kolom tidak menunjuk kolom salinan: %+v", nc.ColumnID)
	}
	if ic := byKind[KindImage]; ic.File == nil || ic.File.ID != img.File.ID {
		t.Fatalf("kartu gambar salinan tidak memakai berkas yang sama: %+v", ic.File)
	}
	if ln := cp.Connectors[0]; ln.From == nil || *ln.From != byKind[KindHeading].ID || *ln.To != byKind[KindColumn].ID || ln.Label != "lalu" {
		t.Fatalf("garis salinan %+v", ln)
	}
	sb := byKind[KindBoard]
	if sb.Ref == nil || sb.Ref.ID == sub.Ref.ID || sb.Ref.Title != "Aset" {
		t.Fatalf("papan anak salinan %+v", sb.Ref)
	}
	if subView, _ := f.svc.Board(ctx, f.nala, sb.Ref.ID); len(subView.Cards) != 1 || subView.Cards[0].Kind != KindChecklist {
		t.Fatalf("isi papan anak salinan %+v", subView.Cards)
	}
	if subView, _ := f.svc.Board(ctx, f.nala, sb.Ref.ID); len(subView.Crumbs) != 3 || subView.Crumbs[1].ID != card.Ref.ID {
		t.Fatalf("induk papan anak salinan %+v", subView.Crumbs)
	}
	// Mengubah salinan tidak mengubah template.
	if _, err := f.svc.Checklist(ctx, f.nala, sb.Ref.ID, byKindIn(t, f, sb.Ref.ID, KindChecklist).ID, ChecklistOp{Op: "title", Text: str("Diubah")}); err != nil {
		t.Fatal(err)
	}
	if tv, _ := f.svc.Board(ctx, f.rina, sub.Ref.ID); !strings.Contains(string(tv.Cards[0].Content), "Daftar aset") {
		t.Fatalf("template ikut berubah: %s", tv.Cards[0].Content)
	}
	acts, _ := f.svc.Activity(ctx, f.nala, card.Ref.ID)
	if len(acts) == 0 || acts[0].Action != "board_from_template" {
		t.Fatalf("riwayat salinan %+v", acts)
	}

	// Kartu template di Tempat sampah tidak ikut, begitu juga garis yang ujungnya di Tempat sampah.
	if _, _, err := f.svc.DeleteCards(ctx, f.rina, b, []string{h.ID}, nil); err != nil {
		t.Fatal(err)
	}
	card2, err := f.svc.FromTemplate(ctx, f.nala, RootID, at)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := f.svc.Board(ctx, f.nala, card2.Ref.ID); len(v.Cards) != 4 || len(v.Connectors) != 0 {
		t.Fatalf("salinan kedua: %d kartu, %d garis", len(v.Cards), len(v.Connectors))
	}

	// Batas kedalaman: template bertingkat dua tidak muat di papan berkedalaman 7.
	deep := f.root
	for i := 0; i < 7; i++ {
		deep = f.board(t, deep, "Tingkat").Ref.ID
	}
	_, err = f.svc.FromTemplate(ctx, f.nala, deep, at)
	wantInvalid(t, err, "terlalu bertingkat")

	// Template yang dilepas tidak bisa disalin lagi.
	if _, err := f.svc.SetTemplate(ctx, f.rina, b, false); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.FromTemplate(ctx, f.nala, RootID, at)
	wantInvalid(t, err, "bukan template")
}

func byKindIn(t *testing.T, f wbFixture, board, kind string) Card {
	t.Helper()
	v, err := f.svc.Board(context.Background(), f.rina, board)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range v.Cards {
		if c.Kind == kind {
			return c
		}
	}
	t.Fatalf("tidak ada kartu %s", kind)
	return Card{}
}

func TestWbCariIsi(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	jan := f.board(t, f.root, "Januari")
	f.card(t, jan.Ref.ID, KindNote, note("Brief kampanye Ramadan untuk tim desain"))
	f.card(t, f.root, KindChecklist, `{"title":"Persiapan Ramadan","items":[{"id":"7f1c2a5e-0d6b-4c8e-9a43-2b1e5f6d7c80","text":"Cetak brosur","done":false}]}`)
	f.card(t, f.root, KindTable, `{"title":"Anggaran","header":true,"rows":[["Pos","Biaya"],["Iklan ramadan","5 juta"]]}`)
	f.card(t, f.root, KindNote, note("Tidak terkait"))
	gone := f.card(t, f.root, KindHeading, `{"text":"Ramadan lama"}`)
	if _, _, err := f.svc.DeleteCards(ctx, f.rina, RootID, []string{gone.ID}, nil); err != nil {
		t.Fatal(err)
	}
	// Kartu tautan cocok lewat judul pratinjaunya.
	var pid string
	if err := f.pool.QueryRow(ctx, `INSERT INTO wb_link_previews (url_hash, url, title, status) VALUES ('h1', 'https://contoh.test/', 'Panduan Ramadan', 'ok') RETURNING id::text`).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	f.card(t, f.root, KindLink, `{"url":"https://contoh.test/","preview_id":"`+pid+`"}`)
	// Isi papan di Tempat sampah tidak muncul.
	trashed := f.board(t, f.root, "Lama")
	f.card(t, trashed.Ref.ID, KindNote, note("Ramadan tahun lalu"))
	if _, _, err := f.svc.DeleteCards(ctx, f.rina, RootID, []string{trashed.ID}, nil); err != nil {
		t.Fatal(err)
	}

	hits, err := f.svc.SearchCards(ctx, "ramadan")
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]CardHit{}
	for _, h := range hits {
		kinds[h.Kind] = h
	}
	if len(hits) != 4 || kinds[KindNote].Board != "Whiteboard / Januari" || kinds[KindNote].BoardID != jan.Ref.ID ||
		!strings.Contains(kinds[KindChecklist].Snippet, "Persiapan Ramadan") || !strings.Contains(kinds[KindTable].Snippet, "Iklan ramadan") ||
		!strings.Contains(kinds[KindLink].Snippet, "Panduan Ramadan") {
		t.Fatalf("hasil cari isi %+v", hits)
	}
	if h, _ := f.svc.SearchCards(ctx, "100%"); len(h) != 0 {
		t.Fatalf("persen dianggap wildcard: %+v", h)
	}
	_, err = f.svc.SearchCards(ctx, "r")
	wantInvalid(t, err, "minimal 2 huruf")
}

// queryLog menangkap SQL dan argumen query terakhir, supaya EXPLAIN memakai teks query sqlc yang sama persis.
type queryLog struct {
	mu   sync.Mutex
	sql  string
	args []any
}

func (l *queryLog) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !strings.HasPrefix(d.SQL, "EXPLAIN") {
		l.sql, l.args = d.SQL, d.Args
	}
	return ctx
}

func (l *queryLog) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// TestWbExplain5000 memuat papan dan mencari di 5.000 kartu (rancangan bagian 7) lewat EXPLAIN ANALYZE pada query
// sqlc yang sebenarnya. Batas waktunya longgar (test DB lokal); rencananya dicetak dengan -v untuk laporan.
func TestWbExplain5000(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	board := f.board(t, f.root, "Performa")
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO wb_cards (board_id, kind, x, y, w, h, z, content, plain, created_by, updated_by)
		SELECT $1::uuid, 'heading', (i % 100) * 260, (i / 100) * 180, 240, 160, i,
		       jsonb_build_object('text', t), t, $2, $2
		FROM generate_series(1, 5000) AS i,
		     LATERAL (SELECT CASE WHEN i % 100 = 0 THEN 'Kampanye nomor ' || i ELSE 'Catatan rapat ke ' || i END AS t) x`,
		board.Ref.ID, f.rina.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `ANALYZE wb_cards`); err != nil {
		t.Fatal(err)
	}

	log := &queryLog{}
	cfg := f.pool.Config().Copy()
	cfg.ConnConfig.Tracer = log
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	q := sqlc.New(pool)
	id, _ := ParseID(board.Ref.ID)

	explain := func(name string, limit time.Duration) {
		t.Helper()
		log.mu.Lock()
		sql, args := log.sql, log.args
		log.mu.Unlock()
		var raw []byte
		if err := pool.QueryRow(ctx, "EXPLAIN (ANALYZE, FORMAT JSON) "+sql, args...).Scan(&raw); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var plan []struct {
			Plan          map[string]any `json:"Plan"`
			ExecutionTime float64        `json:"Execution Time"`
		}
		if err := json.Unmarshal(raw, &plan); err != nil || len(plan) == 0 {
			t.Fatalf("%s: rencana tidak terbaca %v", name, err)
		}
		ms := time.Duration(plan[0].ExecutionTime * float64(time.Millisecond))
		t.Logf("%s: %.1f ms, indeks %v", name, plan[0].ExecutionTime, indexes(plan[0].Plan))
		if ms > limit {
			t.Fatalf("%s %v, batas %v", name, ms, limit)
		}
	}

	rows, err := q.ListWbCards(ctx, id)
	if err != nil || len(rows) != 5000 {
		t.Fatalf("muat papan %d %v", len(rows), err)
	}
	explain("muat papan 5000 kartu", 500*time.Millisecond)

	hits, err := q.SearchWbCards(ctx, "kampanye")
	if err != nil || len(hits) != 50 {
		t.Fatalf("cari %d %v", len(hits), err)
	}
	explain("cari isi di 5000 kartu", 200*time.Millisecond)
}

func indexes(n map[string]any) []string {
	var out []string
	if name, ok := n["Index Name"].(string); ok {
		out = append(out, name)
	}
	if kids, ok := n["Plans"].([]any); ok {
		for _, k := range kids {
			if m, ok := k.(map[string]any); ok {
				out = append(out, indexes(m)...)
			}
		}
	}
	return out
}
