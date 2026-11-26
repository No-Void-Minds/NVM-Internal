package whiteboard

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func ptr[T any](v T) *T { return &v }

// jsonEq membandingkan isi JSON, bukan teksnya (jsonb menata ulang kunci dan spasi).
func jsonEq(a []byte, b string) bool {
	var x, y any
	return json.Unmarshal(a, &x) == nil && json.Unmarshal([]byte(b), &y) == nil && reflect.DeepEqual(x, y)
}

const (
	grupA = "aaaaaaaa-0000-4000-8000-000000000001"
	grupB = "bbbbbbbb-0000-4000-8000-000000000002"
)

// Revisi 50: gaya tersimpan dari daftar putih, grup bersarang lewat jalur grup, siklus ditolak, hapus dan pulihkan
// grup satu batch, kunci menolak geser, ukuran, putar, dan hapus tetapi isi tetap bisa diedit dan siapa pun bisa
// membukanya (tercatat di riwayat), dan template membawa grup (dengan id baru), gaya, dan kunci.
func TestWbGayaGrupKunci(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	b := f.board(t, f.root, "Kampanye").Ref.ID
	j1 := f.card(t, b, KindHeading, `{"text":"Satu","level":1}`)
	j2 := f.card(t, b, KindHeading, `{"text":"Dua","level":1}`)
	shape := f.card(t, b, KindShape, defaultShape)

	// Gaya: tersimpan tanpa nilai bawaan; kunci atau nilai di luar daftar ditolak.
	out, err := f.svc.PatchCards(ctx, f.rina, b, []CardPatch{{ID: j1.ID,
		Style: json.RawMessage(`{"bg":"transparan","border":"tebal","border_color":"biru","border_style":"titik","radius":"bulat","shadow":"halus","opacity":75,"padding":"normal"}`)}})
	if err != nil || !jsonEq(out[0].Style, `{"bg":"transparan","border":"tebal","border_color":"biru","border_style":"titik","radius":"bulat","opacity":75}`) {
		t.Fatalf("gaya %s %v", out[0].Style, err)
	}
	for _, bad := range []string{`{"bg":"#fff"}`, `{"opacity":20}`, `{"opacity":77}`, `{"css":"color:red"}`, `{"border_color":"collab-3"}`, `{"shadow":"neon"}`} {
		_, err := f.svc.PatchCards(ctx, f.rina, b, []CardPatch{{ID: j1.ID, Style: json.RawMessage(bad)}})
		if err == nil {
			t.Errorf("gaya %s diterima", bad)
		}
	}

	// Grup bersarang: j1 dan j2 di grup B, grup B di dalam grup A bersama bentuk.
	_, err = f.svc.PatchCards(ctx, f.rina, b, []CardPatch{
		{ID: j1.ID, GroupIDs: &[]string{grupA, grupB}},
		{ID: j2.ID, GroupIDs: &[]string{grupA, grupB}},
		{ID: shape.ID, GroupIDs: &[]string{grupA}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.PatchCards(ctx, f.rina, b, []CardPatch{{ID: j1.ID, GroupIDs: &[]string{grupA, grupB, grupA}}})
	wantInvalid(t, err, "dirinya sendiri")
	nine := []string{}
	for i := 0; i < 9; i++ {
		nine = append(nine, strings.Replace(grupA, "1", string(rune('1'+i)), 1))
	}
	_, err = f.svc.PatchCards(ctx, f.rina, b, []CardPatch{{ID: j1.ID, GroupIDs: &nine}})
	wantInvalid(t, err, "8 tingkat")
	v, _ := f.svc.Board(ctx, f.rina, b)
	for _, c := range v.Cards {
		if c.ID == j1.ID && (len(c.GroupIDs) != 2 || c.GroupIDs[0] != grupA || c.GroupIDs[1] != grupB) {
			t.Fatalf("jalur grup %v", c.GroupIDs)
		}
	}

	// Hapus grup satu batch, lalu pulihkan: jalur grup tetap.
	batch, _, err := f.svc.DeleteCards(ctx, f.rina, b, []string{j1.ID, j2.ID, shape.ID}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Restore(ctx, f.rina, batch); err != nil {
		t.Fatal(err)
	}
	v, _ = f.svc.Board(ctx, f.rina, b)
	if len(v.Cards) != 3 {
		t.Fatalf("pulih %d kartu", len(v.Cards))
	}
	for _, c := range v.Cards {
		if len(c.GroupIDs) == 0 || c.GroupIDs[0] != grupA {
			t.Fatalf("grup hilang setelah pulih: %s %v", c.ID, c.GroupIDs)
		}
	}

	// Kunci: geser, ubah ukuran, putar, dan hapus ditolak; isi dan warna tetap boleh.
	if _, err := f.svc.PatchCards(ctx, f.rina, b, []CardPatch{{ID: j1.ID, Locked: ptr(true)}, {ID: shape.ID, Locked: ptr(true)}}); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.PatchCards(ctx, f.nala, b, []CardPatch{{ID: j1.ID, X: ptr(500.0)}})
	wantInvalid(t, err, "terkunci")
	_, err = f.svc.PatchCards(ctx, f.nala, b, []CardPatch{{ID: j1.ID, W: ptr(500.0)}})
	wantInvalid(t, err, "terkunci")
	_, _, err = f.svc.DeleteCards(ctx, f.nala, b, []string{j1.ID, j2.ID}, nil)
	wantInvalid(t, err, "terkunci")
	rotated := strings.Replace(defaultShape, `"rotation":0`, `"rotation":90`, 1)
	_, err = f.svc.PatchCards(ctx, f.nala, b, []CardPatch{{ID: shape.ID, Content: json.RawMessage(rotated), BaseContentVersion: shape.ContentVersion}})
	wantInvalid(t, err, "terkunci")
	text := strings.Replace(defaultShape, `"text":""`, `"text":"Mulai"`, 1)
	if _, err := f.svc.PatchCards(ctx, f.nala, b, []CardPatch{{ID: shape.ID, Content: json.RawMessage(text), BaseContentVersion: shape.ContentVersion}}); err != nil {
		t.Fatalf("isi kartu terkunci tidak bisa diedit: %v", err)
	}
	if _, err := f.svc.PatchCards(ctx, f.nala, b, []CardPatch{{ID: j1.ID, Content: json.RawMessage(`{"text":"Satu baru","level":1}`), BaseContentVersion: j1.ContentVersion, Color: ptr("biru")}}); err != nil {
		t.Fatalf("isi dan warna kartu terkunci: %v", err)
	}
	// Orang lain membuka kunci sambil menggeser di satu patch: boleh, dan tercatat.
	out, err = f.svc.PatchCards(ctx, f.nala, b, []CardPatch{{ID: j1.ID, Locked: ptr(false), X: ptr(480.0)}})
	if err != nil || out[0].Locked || out[0].X != 480 {
		t.Fatalf("buka kunci %+v %v", out, err)
	}
	if n := f.count(t, `SELECT count(*) FROM wb_activity WHERE action = 'card_unlocked' AND user_id = $1`, f.nala.ID); n != 1 {
		t.Fatalf("buka kunci tidak tercatat: %d", n)
	}

	// Template membawa grup (id baru, tetap satu grup), gaya, dan kunci.
	if _, err := f.svc.SetTemplate(ctx, f.rina, b, true); err != nil {
		t.Fatal(err)
	}
	cp, err := f.svc.FromTemplate(ctx, f.nala, RootID, TemplateInput{TemplateID: b, X: 900, W: 200, H: 120})
	if err != nil {
		t.Fatal(err)
	}
	v, _ = f.svc.Board(ctx, f.nala, cp.Ref.ID)
	outer := map[string]int{}
	for _, c := range v.Cards {
		if len(c.GroupIDs) > 0 {
			outer[c.GroupIDs[0]]++
		}
		if c.GroupIDs[0] == grupA {
			t.Fatal("salinan template memakai id grup yang sama dengan templatenya")
		}
		if c.Kind == KindShape && !c.Locked {
			t.Fatal("kunci tidak ikut tersalin")
		}
		if c.Kind == KindHeading && strings.Contains(string(c.Content), "Satu baru") && !strings.Contains(string(c.Style), `"transparan"`) {
			t.Fatalf("gaya salinan %s", c.Style)
		}
	}
	if len(outer) != 1 {
		t.Fatalf("grup salinan terpecah: %v", outer)
	}
}

// Grab grup semua atau tidak sama sekali: satu anggota dipegang orang lain menolak seluruh seret dengan namanya.
func TestHubGrabGrup(t *testing.T) {
	f := setupWb(t)
	h := NewHub(f.svc)
	c1 := f.card(t, RootID, KindNote, `{"doc":{"type":"doc","content":[{"type":"paragraph"}]}}`)
	c2 := f.card(t, RootID, KindNote, `{"doc":{"type":"doc","content":[{"type":"paragraph"}]}}`)
	a := f.hubClient(t, h, f.rina, "Rina", true)
	b := f.hubClient(t, h, f.nala, "Nala", true)
	for _, c := range []hubClient{a, b} {
		c.do(h, msg{"t": "join", "board": RootID, "tab_id": "t"})
	}
	b.do(h, msg{"t": "grab", "ids": []string{c2.ID}})
	a.drain()
	a.do(h, msg{"t": "grab", "ids": []string{c1.ID, c2.ID}, "all": true})
	r := last(a.drain(), "grab_result")
	granted, _ := r["granted"].([]any)
	denied, _ := r["denied"].([]any)
	if len(granted) != 0 || len(denied) != 1 || !strings.Contains(string(mustJSON(denied)), "Nala") {
		t.Fatalf("grab grup %v", r)
	}
	// c1 tidak ikut terpegang: orang lain masih bisa memegangnya.
	b.do(h, msg{"t": "grab", "ids": []string{c1.ID}})
	if g, _ := last(b.drain(), "grab_result")["granted"].([]any); len(g) != 1 {
		t.Fatal("anggota grup tetap terpegang setelah grab grup ditolak")
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// Ganti gambar: posisi, ukuran, keterangan, gaya, grup, kunci, dan garis tetap; jenis harus cocok; undo memakai
// berkas lama lewat patch file_id; job harian tidak menghapus berkas yang masih dipakai, dan menghapusnya setelah
// tidak dipakai kartu mana pun.
func TestWbGantiGambar(t *testing.T) {
	f := setupWb(t)
	f.withStore(t)
	ctx := context.Background()
	img, err := f.svc.Upload(ctx, f.rina, RootID, CardInput{X: 40, Y: 60, W: 240, H: 160}, "lama.png", bytes.NewReader(testPNG(t, 300, 200)))
	if err != nil {
		t.Fatal(err)
	}
	note := f.card(t, RootID, KindNote, `{"doc":{"type":"doc","content":[{"type":"paragraph"}]}}`)
	if _, err := f.svc.CreateConnector(ctx, f.rina, RootID, ConnectorInput{From: img.ID, To: note.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.PatchCards(ctx, f.rina, RootID, []CardPatch{{ID: img.ID, Content: json.RawMessage(`{"caption":"Poster","crop":{"x":0.1,"y":0,"w":0.5,"h":1}}`),
		BaseContentVersion: img.ContentVersion, Style: json.RawMessage(`{"shadow":"tegas"}`), GroupIDs: &[]string{grupA}, Locked: ptr(true)}}); err != nil {
		t.Fatal(err)
	}
	oldFile := img.File.ID

	got, old, err := f.svc.ReplaceFile(ctx, f.nala, RootID, img.ID, "baru.png", bytes.NewReader(testPNG(t, 100, 100)))
	if err != nil {
		t.Fatal(err)
	}
	var cap CaptionContent
	_ = json.Unmarshal(got.Content, &cap)
	if old != oldFile || got.File.ID == oldFile || got.File.Name != "baru.png" || got.X != 40 || got.Y != 60 || got.W != 240 ||
		cap.Caption != "Poster" || cap.Crop != nil || !jsonEq(got.Style, `{"shadow":"tegas"}`) || !got.Locked || got.GroupIDs[0] != grupA {
		t.Fatalf("ganti gambar %+v isi %s", got, got.Content)
	}
	if v, _ := f.svc.Board(ctx, f.rina, RootID); len(v.Connectors) != 1 {
		t.Fatal("garis hilang setelah ganti gambar")
	}
	_, _, err = f.svc.ReplaceFile(ctx, f.nala, RootID, img.ID, "teks.txt", strings.NewReader("isi"))
	wantInvalid(t, err, "hanya bisa diganti dengan gambar")
	_, _, err = f.svc.ReplaceFile(ctx, f.nala, RootID, note.ID, "x.png", bytes.NewReader(testPNG(t, 10, 10)))
	wantInvalid(t, err, "Hanya kartu gambar dan berkas")

	// Undo: patch file_id kembali ke berkas lama.
	newFile := got.File.ID
	out, err := f.svc.PatchCards(ctx, f.nala, RootID, []CardPatch{{ID: img.ID, FileID: &oldFile}})
	if err != nil || out[0].File.ID != oldFile {
		t.Fatalf("undo ganti gambar %+v %v", out, err)
	}

	// Job harian: berkas lama dipakai lagi, jadi tetap; berkas baru tidak dipakai, jadi terhapus.
	files := func(id string) int { return f.count(t, `SELECT count(*) FROM wb_files WHERE id = $1`, id) }
	later := time.Now().Add(48 * time.Hour)
	if _, err := Purge(ctx, f.pool, later, false); err != nil {
		t.Fatal(err)
	}
	if files(oldFile) != 1 || files(newFile) != 0 {
		t.Fatalf("job harian: berkas lama %d, berkas baru %d", files(oldFile), files(newFile))
	}
	// Setelah berkas lama tidak dipakai kartu mana pun, ikut terhapus.
	if _, _, err := f.svc.ReplaceFile(ctx, f.nala, RootID, img.ID, "ketiga.png", bytes.NewReader(testPNG(t, 50, 50))); err != nil {
		t.Fatal(err)
	}
	if _, err := Purge(ctx, f.pool, later, false); err != nil {
		t.Fatal(err)
	}
	if files(oldFile) != 0 {
		t.Fatal("berkas lama tetap ada setelah tidak dipakai")
	}
}

// Keputusan Manajer (awal revisi 51): kartu terkunci, juga isi kolom yang terkunci, tidak boleh pindah papan.
func TestWbKunciTidakPindahPapan(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	b := f.board(t, f.root, "Asal").Ref.ID
	tujuan := f.board(t, f.root, "Tujuan").Ref.ID
	j := f.card(t, b, KindHeading, `{"text":"Terkunci","level":1}`)
	bebas := f.card(t, b, KindHeading, `{"text":"Bebas","level":1}`)
	col := f.card(t, b, KindColumn, `{"title":"Kolom"}`)
	isi := f.card(t, b, KindHeading, `{"text":"Isi kolom","level":1}`)
	if _, err := f.svc.PatchCards(ctx, f.rina, b, []CardPatch{{ID: j.ID, Locked: ptr(true)}, {ID: isi.ID, ColumnID: ptr(col.ID), ColOrder: ptr(1.0)}}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.PatchCards(ctx, f.rina, b, []CardPatch{{ID: isi.ID, Locked: ptr(true)}}); err != nil {
		t.Fatal(err)
	}
	_, err := f.svc.MoveCards(ctx, f.rina, b, []string{bebas.ID, j.ID}, tujuan)
	wantInvalid(t, err, "Buka kunci dulu")
	_, err = f.svc.MoveCards(ctx, f.rina, b, []string{col.ID}, tujuan)
	wantInvalid(t, err, "Buka kunci dulu")
	if v, _ := f.svc.Board(ctx, f.rina, b); len(v.Cards) != 4 {
		t.Fatalf("pindahan sebagian terjadi: %d kartu tersisa", len(v.Cards))
	}
	if _, err := f.svc.MoveCards(ctx, f.rina, b, []string{bebas.ID}, tujuan); err != nil {
		t.Fatalf("kartu bebas tidak bisa dipindah: %v", err)
	}
}
