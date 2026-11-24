package whiteboard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// hubClient membungkus Client dengan pembaca antrean keluarnya.
type hubClient struct {
	*Client
	t *testing.T
}

func (f wbFixture) hubClient(t *testing.T, h *Hub, a Actor, name string, canWrite bool) hubClient {
	t.Helper()
	c := NewClient(a.ID, name, nil, canWrite)
	if !h.Register(c) {
		t.Fatalf("daftar %s ditolak", name)
	}
	return hubClient{c, t}
}

func (c hubClient) do(h *Hub, m msg) {
	b, _ := json.Marshal(m)
	h.Handle(context.Background(), c.Client, b)
}

// drain membaca semua pesan yang sudah ada di antrean tanpa menunggu.
func (c hubClient) drain() []map[string]any {
	out := []map[string]any{}
	for {
		select {
		case b := <-c.Send:
			var m map[string]any
			_ = json.Unmarshal(b, &m)
			out = append(out, m)
		default:
			return out
		}
	}
}

func last(msgs []map[string]any, t string) map[string]any {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i]["t"] == t {
			return msgs[i]
		}
	}
	return nil
}

func count(msgs []map[string]any, t string) int {
	n := 0
	for _, m := range msgs {
		if m["t"] == t {
			n++
		}
	}
	return n
}

// Kehadiran: tiga koneksi di papan yang sama saling melihat; dua tab akun yang sama tampil satu orang dengan dua tab;
// yang keluar hilang dari daftar.
func TestHubKehadiran(t *testing.T) {
	f := setupWb(t)
	h := NewHub(f.svc)
	a := f.hubClient(t, h, f.rina, "rina", true)
	a2 := f.hubClient(t, h, f.rina, "rina", true)
	b := f.hubClient(t, h, f.nala, "nala", true)
	for i, c := range []hubClient{a, a2, b} {
		c.do(h, msg{"t": "join", "board": RootID, "tab_id": fmt.Sprint("tab", i)})
	}
	people := last(b.drain(), "presence")["people"].([]any)
	if len(people) != 2 {
		t.Fatalf("kehadiran %v", people)
	}
	for _, p := range people {
		pm := p.(map[string]any)
		if pm["name"] == "rina" && pm["tabs"].(float64) != 2 {
			t.Fatalf("dua tab rina: %v", pm)
		}
		if pm["color"].(float64) < 1 || pm["color"].(float64) > 8 {
			t.Fatalf("warna %v", pm["color"])
		}
	}
	a.drain()
	h.Unregister(b.Client)
	people = last(a.drain(), "presence")["people"].([]any)
	if len(people) != 1 || people[0].(map[string]any)["name"] != "rina" {
		t.Fatalf("setelah nala keluar: %v", people)
	}
	// Diam: semua tab rina diam baru tampil diam.
	a.do(h, msg{"t": "idle"})
	if p := last(a2.drain(), "presence")["people"].([]any)[0].(map[string]any); p["idle"] != false {
		t.Fatalf("satu tab diam sudah dianggap diam: %v", p)
	}
	a2.do(h, msg{"t": "idle"})
	if p := last(a.drain(), "presence")["people"].([]any)[0].(map[string]any); p["idle"] != true {
		t.Fatalf("semua tab diam: %v", p)
	}
}

// Kunci lunak: dua koneksi mengunci kartu yang sama bersamaan, tepat satu menang (diulang 100 kali). Draf hanya
// diteruskan dari pemegang dan hanya bila lolos validasi. Kunci lepas saat pemegang putus dan saat tanpa detak 30
// detik.
func TestHubKunciLunak(t *testing.T) {
	f := setupWb(t)
	h := NewHub(f.svc)
	a := f.hubClient(t, h, f.rina, "rina", true)
	b := f.hubClient(t, h, f.nala, "nala", true)
	a.do(h, msg{"t": "join", "board": RootID})
	b.do(h, msg{"t": "join", "board": RootID})
	card := f.card(t, RootID, KindNote, note("Awal"))

	for i := 0; i < 100; i++ {
		a.drain()
		b.drain()
		var wg sync.WaitGroup
		for _, c := range []hubClient{a, b} {
			wg.Add(1)
			go func(c hubClient) {
				defer wg.Done()
				c.do(h, msg{"t": "lock", "card": card.ID})
			}(c)
		}
		wg.Wait()
		am, bm := a.drain(), b.drain()
		granted := count(am, "lock_granted") + count(bm, "lock_granted")
		denied := count(am, "lock_denied") + count(bm, "lock_denied")
		if granted != 1 || denied != 1 {
			t.Fatalf("putaran %d: granted %d denied %d", i, granted, denied)
		}
		holder := a
		if count(bm, "lock_granted") == 1 {
			holder = b
		}
		holder.do(h, msg{"t": "unlock", "card": card.ID})
	}

	a.do(h, msg{"t": "lock", "card": card.ID})
	b.drain()
	b.do(h, msg{"t": "draft", "card": card.ID, "content": json.RawMessage(note("bukan pemegang"))})
	a.do(h, msg{"t": "draft", "card": card.ID, "content": json.RawMessage(`{"doc":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","attrs":{"href":"javascript:1"}}]}]}]}}`)})
	a.do(h, msg{"t": "draft", "card": card.ID, "content": json.RawMessage(note("Ketikan Rina"))})
	drafts := []string{}
	for _, m := range b.drain() {
		if m["t"] == "draft" {
			raw, _ := json.Marshal(m["content"])
			drafts = append(drafts, string(raw))
		}
	}
	if len(drafts) != 1 || !strings.Contains(drafts[0], "Ketikan Rina") {
		t.Fatalf("draf yang diteruskan: %v", drafts)
	}

	// Tanpa detak 30 detik: kunci lepas.
	base := time.Now()
	h.now = func() time.Time { return base.Add(LockTTL + time.Second) }
	h.Sweep()
	if last(b.drain(), "unlocked") == nil {
		t.Fatal("kunci tanpa detak tidak lepas")
	}
	h.now = time.Now
	b.do(h, msg{"t": "lock", "card": card.ID})
	if last(b.drain(), "lock_granted") == nil {
		t.Fatal("kunci yang sudah lepas tidak bisa diambil")
	}
	a.drain()
	h.Unregister(b.Client)
	if last(a.drain(), "unlocked") == nil {
		t.Fatal("kunci tidak lepas saat pemegang putus")
	}
}

// Dua tab akun yang sama: tab kedua meminta kunci dengan force, tab pertama diminta melepas, lalu kunci pindah.
// Bila tab pertama tidak menjawab, kunci pindah setelah batas tunggu.
func TestHubPindahKunciDuaTab(t *testing.T) {
	f := setupWb(t)
	h := NewHub(f.svc)
	tab1 := f.hubClient(t, h, f.rina, "rina", true)
	tab2 := f.hubClient(t, h, f.rina, "rina", true)
	tab1.do(h, msg{"t": "join", "board": RootID})
	tab2.do(h, msg{"t": "join", "board": RootID})
	card := f.card(t, RootID, KindNote, note("Awal"))
	tab1.do(h, msg{"t": "lock", "card": card.ID})
	tab2.drain()
	tab2.do(h, msg{"t": "lock", "card": card.ID})
	if d := last(tab2.drain(), "lock_denied"); d == nil || d["same_user"] != true {
		t.Fatalf("tab kedua tanpa force: %v", d)
	}
	tab2.do(h, msg{"t": "lock", "card": card.ID, "force": true})
	if last(tab1.drain(), "lock_release") == nil {
		t.Fatal("tab pertama tidak diminta melepas")
	}
	tab1.do(h, msg{"t": "unlock", "card": card.ID})
	if last(tab2.drain(), "lock_granted") == nil {
		t.Fatal("kunci tidak pindah ke tab kedua")
	}
	// Tab kedua memegang; tab pertama minta balik tetapi tab kedua diam: pindah setelah batas tunggu.
	tab1.do(h, msg{"t": "lock", "card": card.ID, "force": true})
	tab2.drain()
	base := time.Now()
	h.now = func() time.Time { return base.Add(transferWait + time.Second) }
	h.Sweep()
	if last(tab2.drain(), "lock_lost") == nil || last(tab1.drain(), "lock_granted") == nil {
		t.Fatal("kunci tidak pindah setelah batas tunggu")
	}
}

// Seret: dua koneksi memegang kartu yang sama bersamaan, tepat satu menang; gerakan dari yang kalah tidak
// diteruskan; gerakan di atas 30 per detik dibuang.
func TestHubSeret(t *testing.T) {
	f := setupWb(t)
	h := NewHub(f.svc)
	a := f.hubClient(t, h, f.rina, "rina", true)
	b := f.hubClient(t, h, f.nala, "nala", true)
	a.do(h, msg{"t": "join", "board": RootID})
	b.do(h, msg{"t": "join", "board": RootID})
	card := f.card(t, RootID, KindNote, note("Seret"))
	var wg sync.WaitGroup
	for _, c := range []hubClient{a, b} {
		wg.Add(1)
		go func(c hubClient) {
			defer wg.Done()
			c.do(h, msg{"t": "grab", "ids": []string{card.ID}})
		}(c)
	}
	wg.Wait()
	ar, br := last(a.drain(), "grab_result"), last(b.drain(), "grab_result")
	winner, loser := a, b
	if len(br["granted"].([]any)) == 1 {
		winner, loser = b, a
		ar, br = br, ar
	}
	if len(ar["granted"].([]any)) != 1 || len(br["denied"].([]any)) != 1 {
		t.Fatalf("hasil grab %v %v", ar, br)
	}
	move := msg{"t": "move", "moves": []Move{{ID: card.ID, X: 1, Y: 2, W: 240, H: 160}}}
	loser.do(h, move)
	if last(winner.drain(), "moving") != nil {
		t.Fatal("gerakan dari yang kalah diteruskan")
	}
	for i := 0; i < 100; i++ {
		winner.do(h, move)
	}
	if n := count(loser.drain(), "moving"); n > ratePerSecond+5 || n == 0 {
		t.Fatalf("gerakan diteruskan %d kali, mau sekitar %d", n, ratePerSecond)
	}
	winner.do(h, msg{"t": "release", "ids": []string{card.ID}})
	if last(loser.drain(), "released") == nil {
		t.Fatal("release tidak disiarkan")
	}
}

// Admin boleh terhubung dan melihat, tetapi lock, grab, draft, dan select ditolak.
func TestHubAdminHanyaMelihat(t *testing.T) {
	f := setupWb(t)
	h := NewHub(f.svc)
	adm := f.hubClient(t, h, f.rina, "admin", false)
	other := f.hubClient(t, h, f.nala, "nala", true)
	adm.do(h, msg{"t": "join", "board": RootID})
	other.do(h, msg{"t": "join", "board": RootID})
	card := f.card(t, RootID, KindNote, note("x"))
	adm.drain()
	other.drain()
	for _, m := range []msg{
		{"t": "lock", "card": card.ID},
		{"t": "grab", "ids": []string{card.ID}},
		{"t": "select", "ids": []string{card.ID}},
		{"t": "draft", "card": card.ID, "content": json.RawMessage(note("y"))},
	} {
		adm.do(h, m)
	}
	if n := count(adm.drain(), "denied"); n != 4 {
		t.Fatalf("penolakan Admin %d, mau 4", n)
	}
	if got := other.drain(); count(got, "locked")+count(got, "grabbed")+count(got, "select")+count(got, "draft") != 0 {
		t.Fatalf("tulisan Admin diteruskan: %v", got)
	}
	adm.do(h, msg{"t": "cursor", "x": 10, "y": 20})
	if last(other.drain(), "cursor") == nil {
		t.Fatal("kursor Admin tidak terlihat")
	}
}

// Kartu yang sedang diedit dihapus orang lain: draf terakhir pemegang ikut tersimpan di kartu yang masuk Tempat
// sampah, dan pulihkan mengembalikan isi draf itu.
func TestHubHapusSaatDiedit(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	h := NewHub(f.svc)
	a := f.hubClient(t, h, f.rina, "rina", true)
	a.do(h, msg{"t": "join", "board": RootID})
	card := f.card(t, RootID, KindNote, note("Awal"))
	a.do(h, msg{"t": "lock", "card": card.ID})
	a.do(h, msg{"t": "draft", "card": card.ID, "content": json.RawMessage(note("Ketikan terakhir"))})
	root, _ := f.svc.ResolveBoard(ctx, f.nala, RootID)
	batch, _, err := f.svc.DeleteCards(ctx, f.nala, root, []string{card.ID}, h.Drafts([]string{card.ID}))
	if err != nil {
		t.Fatal(err)
	}
	h.Deleted(root, []string{card.ID}, batch, "nala", nil)
	d := last(a.drain(), "deleted")
	if d == nil || d["by"] != "nala" || d["batch"] != batch {
		t.Fatalf("pemberitahuan hapus %v", d)
	}
	if _, err := f.svc.Restore(ctx, f.rina, batch); err != nil {
		t.Fatal(err)
	}
	view, _ := f.svc.Board(ctx, f.rina, RootID)
	if len(view.Cards) != 1 || !strings.Contains(string(view.Cards[0].Content), "Ketikan terakhir") {
		t.Fatalf("isi setelah pulih %s", view.Cards[0].Content)
	}
}

// Sinkron ulang: koneksi yang tertinggal menerima tepat kartu yang berubah dan dihapus sejak seq-nya; selisih di
// atas MaxSyncRows meminta muat ulang utuh.
func TestHubSinkronUlang(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	h := NewHub(f.svc)
	keep := f.card(t, RootID, KindNote, note("tetap"))
	gone := f.card(t, RootID, KindNote, note("dihapus"))
	view, _ := f.svc.Board(ctx, f.rina, RootID)
	seq := view.Seq
	x := 50.0
	if _, err := f.svc.PatchCards(ctx, f.nala, RootID, []CardPatch{{ID: keep.ID, X: &x}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.svc.DeleteCards(ctx, f.nala, RootID, []string{gone.ID}, nil); err != nil {
		t.Fatal(err)
	}
	added := f.card(t, RootID, KindHeading, `{"text":"baru","level":1}`)
	a := f.hubClient(t, h, f.rina, "rina", true)
	a.do(h, msg{"t": "join", "board": RootID, "last_seq": seq})
	j := last(a.drain(), "joined")
	cards, _ := json.Marshal(j["cards"])
	deleted, _ := json.Marshal(j["deleted"])
	if !strings.Contains(string(cards), keep.ID) || !strings.Contains(string(cards), added.ID) || strings.Contains(string(cards), gone.ID) ||
		string(deleted) != `["`+gone.ID+`"]` {
		t.Fatalf("sinkron: cards %s deleted %s", cards, deleted)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE wb_cards SET seq = seq + 1000000`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < MaxSyncRows; i++ {
		if _, err := f.pool.Exec(ctx, `INSERT INTO wb_cards (board_id, kind, x, y, w, h, content, seq, created_by, updated_by)
			SELECT board_id, kind, x, y, w, h, content, seq, created_by, updated_by FROM wb_cards WHERE id = $1`, keep.ID); err != nil {
			t.Fatal(err)
		}
	}
	a.do(h, msg{"t": "join", "board": RootID, "last_seq": seq})
	if j := last(a.drain(), "joined"); j["resync"] != true {
		t.Fatalf("selisih besar tanpa resync: %v", j["resync"])
	}
}

// Antrean keluar penuh menutup koneksi itu tanpa menahan koneksi lain.
func TestHubAntreanPenuh(t *testing.T) {
	f := setupWb(t)
	h := NewHub(f.svc)
	slow := f.hubClient(t, h, f.rina, "rina", true)
	fast := f.hubClient(t, h, f.nala, "nala", true)
	kicked := make(chan string, 1)
	slow.Kick = func(reason string) { kicked <- reason }
	slow.do(h, msg{"t": "join", "board": RootID})
	fast.do(h, msg{"t": "join", "board": RootID})
	root, _ := f.svc.ResolveBoard(context.Background(), f.rina, RootID)
	for i := 0; i < sendQueue+10; i++ {
		h.Changed(root, []Card{{ID: "x"}}, 0)
		fast.drain()
	}
	select {
	case <-kicked:
	case <-time.After(2 * time.Second):
		t.Fatal("koneksi lambat tidak ditutup")
	}
}
