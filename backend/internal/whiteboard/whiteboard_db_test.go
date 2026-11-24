package whiteboard

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/testdb"
)

type wbFixture struct {
	pool       *pgxpool.Pool
	svc        *Service
	rina, nala Actor
	root       string
}

func setupWb(t *testing.T) wbFixture {
	t.Helper()
	pool := testdb.Pool(t)
	div := testdb.Division(t, pool, "Marketing", true)
	f := wbFixture{
		pool: pool,
		svc:  NewService(pool),
		rina: Actor{ID: testdb.User(t, pool, "rina", "staff", div, true)},
		nala: Actor{ID: testdb.User(t, pool, "nala", "manager", div, true)},
	}
	// Test DB dikosongkan lewat TRUNCATE users CASCADE, jadi papan akar dibuat ulang saat pertama dibuka.
	root, err := f.svc.Board(context.Background(), f.rina, RootID)
	if err != nil {
		t.Fatal(err)
	}
	f.root = root.ID
	return f
}

// board membuat papan baru sebagai kartu di papan parent dan mengembalikan kartunya.
func (f wbFixture) board(t *testing.T, parent, title string) Card {
	t.Helper()
	c, err := f.svc.CreateCard(context.Background(), f.rina, parent, CardInput{Kind: KindBoard, Title: title, X: 0, Y: 0, W: 200, H: 120})
	if err != nil {
		t.Fatalf("buat papan %s: %v", title, err)
	}
	return c
}

func (f wbFixture) card(t *testing.T, board string, kind string, content string) Card {
	t.Helper()
	c, err := f.svc.CreateCard(context.Background(), f.rina, board, CardInput{Kind: kind, X: 10, Y: 20, W: 240, H: 160, Content: json.RawMessage(content)})
	if err != nil {
		t.Fatalf("buat kartu: %v", err)
	}
	return c
}

func (f wbFixture) count(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func note(text string) string {
	return `{"doc":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"` + text + `"}]}]}}`
}

func wantInvalid(t *testing.T, err error, part string) {
	t.Helper()
	if !invalid(err) || !strings.Contains(err.Error(), part) {
		t.Fatalf("mau galat validasi berisi %q, dapat %v", part, err)
	}
}

// Papan akar bersama, papan di dalam papan sebagai kartu, breadcrumb, ringkasan isi, batas kedalaman, ganti judul.
func TestWbPapanBersarang(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	again, err := f.svc.Board(ctx, f.nala, RootID)
	if err != nil || again.ID != f.root || !again.IsRoot || again.Title != "Whiteboard" {
		t.Fatalf("papan akar kedua kali: %+v %v", again, err)
	}
	if n := f.count(t, `SELECT count(*) FROM wb_boards WHERE is_root`); n != 1 {
		t.Fatalf("papan akar %d", n)
	}

	tahun := f.board(t, RootID, "2026")
	if tahun.Kind != KindBoard || tahun.Ref == nil || tahun.Ref.Title != "2026" || tahun.BoardID != f.root {
		t.Fatalf("kartu papan %+v", tahun)
	}
	jan := f.board(t, tahun.Ref.ID, "Januari")
	f.card(t, tahun.Ref.ID, KindNote, note("Rencana tahunan"))
	view, err := f.svc.Board(ctx, f.rina, jan.Ref.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Crumbs) != 3 || view.Crumbs[0].Title != "Whiteboard" || view.Crumbs[1].Title != "2026" || view.Crumbs[2].Title != "Januari" {
		t.Fatalf("breadcrumb %+v", view.Crumbs)
	}
	root, _ := f.svc.Board(ctx, f.rina, RootID)
	if c := root.Cards[0]; c.Ref == nil || c.Ref.Boards != 1 || c.Ref.Cards != 1 {
		t.Fatalf("ringkasan kartu papan %+v", c.Ref)
	}

	// Rantai sampai kedalaman 8 boleh, papan ke-9 ditolak.
	parent := jan.Ref.ID
	for depth := 3; depth <= MaxDepth; depth++ {
		parent = f.board(t, parent, "Tingkat").Ref.ID
	}
	_, err = f.svc.CreateCard(ctx, f.rina, parent, CardInput{Kind: KindBoard, Title: "Terlalu dalam", W: 200, H: 120})
	wantInvalid(t, err, "paling dalam")
	_, err = f.svc.CreateCard(ctx, f.rina, RootID, CardInput{Kind: KindBoard, Title: "  ", W: 200, H: 120})
	wantInvalid(t, err, "wajib")

	renamed, err := f.svc.RenameBoard(ctx, f.rina, jan.Ref.ID, "Januari 2026", 1)
	if err != nil || renamed.Version != 2 {
		t.Fatalf("ganti judul %+v %v", renamed, err)
	}
	if _, err := f.svc.RenameBoard(ctx, f.nala, jan.Ref.ID, "Tertimpa", 1); !errors.Is(err, ErrStale) {
		t.Fatalf("versi basi: %v", err)
	}
	if _, err := f.svc.RenameBoard(ctx, f.nala, RootID, "Papan tim", 1); err != nil {
		t.Fatalf("ganti judul papan akar: %v", err)
	}
	_, err = f.svc.PatchCards(ctx, f.rina, f.root, []CardPatch{{ID: tahun.ID, Content: json.RawMessage(`{}`), BaseContentVersion: 1}})
	wantInvalid(t, err, "Judul papan")

	res, err := f.svc.Search(ctx, "januari")
	if err != nil || len(res) != 1 || res[0].Path != "Papan tim / 2026" {
		t.Fatalf("cari: %+v %v", res, err)
	}
	if res, _ := f.svc.Search(ctx, "100%"); len(res) != 0 {
		t.Fatalf("persen dianggap wildcard: %+v", res)
	}
	// Papan akar tidak bisa masuk Tempat sampah, juga lewat SQL langsung.
	if _, err := f.pool.Exec(ctx, `UPDATE wb_boards SET deleted_at = now(), deleted_by = $1, deleted_batch = gen_random_uuid() WHERE is_root`, f.rina.ID); err == nil {
		t.Fatal("papan akar bisa ditandai terhapus")
	}
}

// Posisi, ukuran, dan warna digabung per field (yang terakhir menang); isi dijaga content_version.
func TestWbKartuPerFieldDanKonflikIsi(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	c := f.card(t, RootID, KindNote, note("Awal"))
	if c.ContentVersion != 1 {
		t.Fatalf("versi isi awal %d", c.ContentVersion)
	}

	x1, x2, color := 100.0, 300.0, "kuning"
	if _, err := f.svc.PatchCards(ctx, f.rina, RootID, []CardPatch{{ID: c.ID, X: &x1}}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.PatchCards(ctx, f.nala, RootID, []CardPatch{{ID: c.ID, Color: &color}}); err != nil {
		t.Fatal(err)
	}
	out, err := f.svc.PatchCards(ctx, f.nala, RootID, []CardPatch{{ID: c.ID, X: &x2}})
	if err != nil {
		t.Fatal(err)
	}
	if out[0].X != x2 || out[0].Y != 20 || out[0].Color == nil || *out[0].Color != "kuning" || out[0].ContentVersion != 1 {
		t.Fatalf("gabung per field: %+v", out[0])
	}

	edited, err := f.svc.PatchCards(ctx, f.rina, RootID, []CardPatch{{ID: c.ID, Content: json.RawMessage(note("Versi Rina")), BaseContentVersion: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if edited[0].ContentVersion != 2 {
		t.Fatalf("versi isi setelah edit %d", edited[0].ContentVersion)
	}
	_, err = f.svc.PatchCards(ctx, f.nala, RootID, []CardPatch{{ID: c.ID, Content: json.RawMessage(note("Versi Nala")), BaseContentVersion: 1}})
	var conflict *ConflictError
	if !errors.As(err, &conflict) || !strings.Contains(string(conflict.Card.Content), "Versi Rina") || conflict.Card.ContentVersion != 2 {
		t.Fatalf("konflik isi: %v", err)
	}

	empty := ""
	cleared, err := f.svc.PatchCards(ctx, f.rina, RootID, []CardPatch{{ID: c.ID, Color: &empty}})
	if err != nil || cleared[0].Color != nil {
		t.Fatalf("kosongkan warna: %+v %v", cleared, err)
	}
	bad := 10.0
	_, err = f.svc.PatchCards(ctx, f.rina, RootID, []CardPatch{{ID: c.ID, W: &bad}})
	wantInvalid(t, err, "Ukuran")
	_, err = f.svc.PatchCards(ctx, f.rina, RootID, []CardPatch{{ID: c.ID, Content: json.RawMessage(`{"doc":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","attrs":{"href":"javascript:alert(1)"}}]}]}]}}`), BaseContentVersion: 2}})
	wantInvalid(t, err, "Tautan")

	// Id dari klien: boleh, tetapi tidak boleh dipakai dua kali.
	id := "6b0f1e2a-3c4d-4e5f-8a9b-0c1d2e3f4a5b"
	if _, err := f.svc.CreateCard(ctx, f.rina, RootID, CardInput{ID: id, Kind: KindHeading, X: 0, Y: 0, W: 320, H: 64}); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.CreateCard(ctx, f.rina, RootID, CardInput{ID: id, Kind: KindHeading, X: 0, Y: 0, W: 320, H: 64})
	wantInvalid(t, err, "sudah dipakai")
}

// Centang butir berbeda oleh dua orang tidak saling menimpa, dan tidak bentrok dengan versi isi.
func TestWbChecklistPerButir(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	a, b := "00000000-0000-4000-8000-0000000000aa", "00000000-0000-4000-8000-0000000000bb"
	c := f.card(t, RootID, KindChecklist, `{"title":"Rilis","items":[{"id":"`+a+`","text":"Brief"},{"id":"`+b+`","text":"Desain"}]}`)
	yes := true
	if _, err := f.svc.Checklist(ctx, f.rina, RootID, c.ID, ChecklistOp{Op: "set", ItemID: a, Done: &yes}); err != nil {
		t.Fatal(err)
	}
	out, err := f.svc.Checklist(ctx, f.nala, RootID, c.ID, ChecklistOp{Op: "set", ItemID: b, Done: &yes})
	if err != nil {
		t.Fatal(err)
	}
	var got ChecklistContent
	if err := json.Unmarshal(out.Content, &got); err != nil {
		t.Fatal(err)
	}
	if !got.Items[0].Done || !got.Items[1].Done {
		t.Fatalf("centang tertimpa: %+v", got.Items)
	}
	n := f.card(t, RootID, KindNote, note("bukan checklist"))
	_, err = f.svc.Checklist(ctx, f.rina, RootID, n.ID, ChecklistOp{Op: "set", ItemID: a, Done: &yes})
	wantInvalid(t, err, "bukan checklist")
}

// Hapus kartu papan membawa papan anak dan seluruh isinya ke satu batch; pulihkan ditolak selama papan asalnya
// juga di Tempat sampah.
func TestWbTempatSampah(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	kampanye := f.board(t, RootID, "Kampanye")
	inner := kampanye.Ref.ID
	f.board(t, inner, "Desember")
	c1 := f.card(t, inner, KindNote, note("Satu"))
	f.card(t, inner, KindNote, note("Dua"))

	cardBatch, _, err := f.svc.DeleteCards(ctx, f.rina, inner, []string{c1.ID}, nil)
	if err != nil {
		t.Fatal(err)
	}
	boardBatch, _, err := f.svc.DeleteCards(ctx, f.nala, RootID, []string{kampanye.ID}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Board(ctx, f.rina, inner); !errors.Is(err, ErrNotFound) {
		t.Fatalf("papan di Tempat sampah masih terbuka: %v", err)
	}
	if root, _ := f.svc.Board(ctx, f.rina, RootID); len(root.Cards) != 0 {
		t.Fatalf("papan akar masih berisi %+v", root.Cards)
	}

	trash, err := f.svc.Trash(ctx)
	if err != nil || len(trash) != 2 {
		t.Fatalf("Tempat sampah %+v %v", trash, err)
	}
	for _, b := range trash {
		switch b.Batch {
		case boardBatch:
			if strings.Join(b.Boards, ",") != "Kampanye,Desember" || b.Cards != 1 || b.BlockedBy != "" || b.BoardTitle != "Whiteboard" {
				t.Fatalf("batch papan %+v", b)
			}
		case cardBatch:
			if len(b.Boards) != 0 || b.Cards != 1 || b.BlockedBy != "Kampanye" || b.Excerpts[0] != "Satu" {
				t.Fatalf("batch kartu %+v", b)
			}
		}
	}

	_, err = f.svc.Restore(ctx, f.rina, cardBatch)
	wantInvalid(t, err, "Pulihkan dulu")
	res, err := f.svc.Restore(ctx, f.rina, boardBatch)
	if err != nil || res.BoardID != f.root {
		t.Fatalf("pulihkan papan %+v %v", res, err)
	}
	if res, err := f.svc.Restore(ctx, f.rina, cardBatch); err != nil || res.BoardID != inner {
		t.Fatalf("pulihkan kartu %+v %v", res, err)
	}
	view, err := f.svc.Board(ctx, f.rina, inner)
	if err != nil || len(view.Cards) != 3 || len(view.Crumbs) != 2 {
		t.Fatalf("papan setelah pulih %+v %v", view, err)
	}
	if _, err := f.svc.Restore(ctx, f.rina, cardBatch); !errors.Is(err, ErrNotFound) {
		t.Fatalf("pulihkan dua kali: %v", err)
	}
	if n := f.count(t, `SELECT count(*) FROM audit_logs WHERE action IN ('wb_deleted', 'wb_restored')`); n != 4 {
		t.Fatalf("audit hapus dan pulihkan: %d baris", n)
	}
}

// Job harian menghapus permanen batch lebih dari 30 hari; 29 hari tetap ada.
func TestWbHapusPermanen(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	lama := f.board(t, RootID, "Lama")
	f.card(t, lama.Ref.ID, KindNote, note("ikut hilang"))
	old := f.card(t, RootID, KindNote, note("kartu lama"))
	young := f.card(t, RootID, KindNote, note("kartu muda"))
	for _, id := range []string{lama.ID, old.ID, young.ID} {
		if _, _, err := f.svc.DeleteCards(ctx, f.rina, RootID, []string{id}, nil); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	age := func(table, id string, days int) {
		if _, err := f.pool.Exec(ctx, `UPDATE `+table+` SET deleted_at = $2 WHERE id = $1`, id, now.AddDate(0, 0, -days)); err != nil {
			t.Fatal(err)
		}
	}
	age("wb_cards", lama.ID, 31)
	age("wb_boards", lama.Ref.ID, 31)
	age("wb_cards", old.ID, 31)
	age("wb_cards", young.ID, 29)

	n, err := Purge(ctx, f.pool, now, true)
	if err != nil || n != 2 || f.count(t, `SELECT count(*) FROM wb_cards`) != 4 {
		t.Fatalf("dry run: %d %v", n, err)
	}
	if n, err = Purge(ctx, f.pool, now, false); err != nil || n != 2 {
		t.Fatalf("hapus permanen: %d %v", n, err)
	}
	if got := f.count(t, `SELECT count(*) FROM wb_boards`); got != 1 {
		t.Fatalf("papan tersisa %d, mau hanya papan akar", got)
	}
	if got := f.count(t, `SELECT count(*) FROM wb_cards`); got != 1 {
		t.Fatalf("kartu tersisa %d, mau hanya kartu 29 hari", got)
	}
	if got := f.count(t, `SELECT count(*) FROM wb_activity WHERE board_id = $1`, lama.Ref.ID); got != 0 {
		t.Fatalf("riwayat papan yang dihapus permanen tersisa %d", got)
	}
	if got := f.count(t, `SELECT count(*) FROM audit_logs WHERE action = 'wb_purged' AND user_id IS NULL`); got != 2 {
		t.Fatalf("audit hapus permanen %d", got)
	}
}

// Riwayat: seret beruntun dan edit beruntun dari orang yang sama digabung; orang lain mendapat baris sendiri.
func TestWbRiwayatDigabung(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	c1 := f.card(t, RootID, KindNote, note("A"))
	c2 := f.card(t, RootID, KindNote, note("B"))
	x := 5.0
	for _, id := range []string{c1.ID, c2.ID, c1.ID} {
		if _, err := f.svc.PatchCards(ctx, f.rina, RootID, []CardPatch{{ID: id, X: &x}}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.svc.PatchCards(ctx, f.nala, RootID, []CardPatch{{ID: c1.ID, X: &x}}); err != nil {
		t.Fatal(err)
	}
	cv := int32(1)
	for _, text := range []string{"A1", "A2"} {
		out, err := f.svc.PatchCards(ctx, f.rina, RootID, []CardPatch{{ID: c1.ID, Content: json.RawMessage(note(text)), BaseContentVersion: cv}})
		if err != nil {
			t.Fatal(err)
		}
		cv = out[0].ContentVersion
	}
	acts, err := f.svc.Activity(ctx, f.rina, RootID)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, a := range acts {
		counts[a.UserName+" "+a.Action]++
		if a.UserName == "rina" && a.Action == "cards_arranged" {
			var sum struct {
				Count int      `json:"count"`
				IDs   []string `json:"ids"`
			}
			if err := json.Unmarshal(a.Summary, &sum); err != nil || sum.Count != 2 || len(sum.IDs) != 2 {
				t.Fatalf("ringkasan seret %s", a.Summary)
			}
		}
		if a.Action == "card_edited" && !strings.Contains(string(a.Summary), "A2") {
			t.Fatalf("edit digabung harus memuat isi terakhir: %s", a.Summary)
		}
	}
	if counts["rina cards_arranged"] != 1 || counts["nala cards_arranged"] != 1 || counts["rina card_edited"] != 1 || counts["rina card_created"] != 2 {
		t.Fatalf("riwayat %v", counts)
	}
}
