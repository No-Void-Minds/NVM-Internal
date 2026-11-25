package whiteboard

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// Objek gambar bebas: kotak batas dari titik dunia (plus setengah ketebalan, minimal 40 px), goresan tambahan
// memperbesar kotak tanpa menggeser gambar, hapus dan ganti warna, batas lebar, dan isi tidak bisa ditimpa lewat
// patch biasa.
func TestWbGambarBebasObjek(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	d, err := f.svc.CreateDrawing(ctx, f.rina, RootID, "", Stroke{Color: "tinta", Width: 4, Points: []int{100, 300, 101, 301}})
	if err != nil {
		t.Fatal(err)
	}
	if d.Kind != KindDrawing || d.W != 40 || d.H != 40 || d.CreatedByName != "rina" {
		t.Fatalf("objek kecil %v,%v %vx%v oleh %q", d.X, d.Y, d.W, d.H, d.CreatedByName)
	}
	d, err = f.svc.Drawing(ctx, f.rina, RootID, d.ID, DrawingOp{Op: "add", Stroke: &Stroke{Color: "biru", Width: 8, Points: []int{20, 50, 400, 600}}})
	if err != nil {
		t.Fatal(err)
	}
	if d.X != 14 || d.Y != 44 || d.W != 392 || d.H != 562 {
		t.Fatalf("kotak batas %v,%v %vx%v", d.X, d.Y, d.W, d.H)
	}
	var c DrawingContent
	_ = json.Unmarshal(d.Content, &c)
	// Goresan pertama tetap di titik dunia yang sama setelah kotaknya membesar.
	if got := absolute(c, d.X, d.Y)[0].Points; got[0] != 100 || got[1] != 300 {
		t.Fatalf("goresan pertama bergeser ke %v", got)
	}
	idx := 0
	if d, err = f.svc.Drawing(ctx, f.rina, RootID, d.ID, DrawingOp{Op: "remove", Index: &idx}); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Drawing(ctx, f.rina, RootID, d.ID, DrawingOp{Op: "remove", Index: &idx})
	wantInvalid(t, err, "goresan terakhir")
	if d, err = f.svc.Drawing(ctx, f.rina, RootID, d.ID, DrawingOp{Op: "color", Color: "merah"}); err != nil || !strings.Contains(string(d.Content), "merah") {
		t.Fatalf("ganti warna %s %v", d.Content, err)
	}
	_, err = f.svc.Drawing(ctx, f.rina, RootID, d.ID, DrawingOp{Op: "color", Color: "ungu"})
	wantInvalid(t, err, "Warna pena")
	_, err = f.svc.Drawing(ctx, f.rina, RootID, d.ID, DrawingOp{Op: "add", Stroke: &Stroke{Color: "tinta", Width: 2, Points: []int{0, 0, 5000, 0}}})
	wantInvalid(t, err, "terlalu lebar")
	_, err = f.svc.PatchCards(ctx, f.rina, RootID, []CardPatch{{ID: d.ID, Content: json.RawMessage(`{"strokes":[]}`), BaseContentVersion: d.ContentVersion}})
	wantInvalid(t, err, "lewat pena")
	w := 200.0
	_, err = f.svc.PatchCards(ctx, f.rina, RootID, []CardPatch{{ID: d.ID, W: &w}})
	wantInvalid(t, err, "lewat pena")
	x := 500.0
	if _, err := f.svc.PatchCards(ctx, f.rina, RootID, []CardPatch{{ID: d.ID, X: &x}}); err != nil {
		t.Fatalf("pindah objek gambar: %v", err)
	}
	_, err = f.svc.CreateCard(ctx, f.rina, RootID, CardInput{Kind: KindDrawing, W: 40, H: 40})
	wantInvalid(t, err, "kosong")
}

// Judul tabel (revisi B): ikut dibersihkan, ikut teks pencarian, dan diubah lewat operasi tabel.
func TestWbJudulTabel(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	tb := f.card(t, f.root, KindTable, `{"title":"  Harga   paket ","header":true,"rows":[["a"]]}`)
	if !strings.Contains(string(tb.Content), `"title": "Harga   paket"`) {
		t.Fatalf("judul dibersihkan: %s", tb.Content)
	}
	card, err := f.svc.Table(ctx, f.rina, RootID, tb.ID, TableOp{Op: "title", Text: "Rencana Desember"})
	if err != nil || !strings.Contains(string(card.Content), "Rencana Desember") {
		t.Fatalf("ganti judul %s %v", card.Content, err)
	}
	var plain string
	if err := f.pool.QueryRow(ctx, `SELECT plain FROM wb_cards WHERE id = $1`, tb.ID).Scan(&plain); err != nil || !strings.HasPrefix(plain, "Rencana Desember") {
		t.Fatalf("teks pencarian %q %v", plain, err)
	}
	_, err = f.svc.Table(ctx, f.rina, RootID, tb.ID, TableOp{Op: "title", Text: strings.Repeat("x", 201)})
	wantInvalid(t, err, "Judul tabel")
}

// Model garis untuk diagram W5: ujung bebas, port, jalur siku dengan titik belok, tujuh jenis ujung, garis titik-titik,
// dan tiga label. Garis dengan ujung bebas tetap tampil selama ujung yang menempel hidup.
func TestWbGarisModelDiagram(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	a := f.card(t, f.root, KindNote, "")
	x, y := 900.0, 40.0
	conn, err := f.svc.CreateConnector(ctx, f.rina, RootID, ConnectorInput{
		From: a.ID,
		Geometry: ConnectorGeometry{From: ConnectorEnd{Port: "right"}, To: ConnectorEnd{X: &x, Y: &y}, Route: "orthogonal",
			Points: [][2]float64{{600, 40}}},
		Style:      ConnectorStyle{Start: "diamond_open", End: "triangle", Line: "dotted"},
		LabelStart: "1", Label: "punya", LabelEnd: "  banyak ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if conn.To != nil || conn.Geometry.To.X == nil || conn.Geometry.Route != "orthogonal" || conn.LabelEnd != "banyak" ||
		conn.Style.Line != "dotted" || len(conn.Geometry.Points) != 1 {
		t.Fatalf("garis %+v", conn)
	}
	view, _ := f.svc.Board(ctx, f.rina, RootID)
	if len(view.Connectors) != 1 {
		t.Fatalf("garis ujung bebas tidak tampil: %d", len(view.Connectors))
	}
	for name, in := range map[string]ConnectorInput{
		"port":         {From: a.ID, Geometry: ConnectorGeometry{From: ConnectorEnd{Port: "tengah"}, To: ConnectorEnd{X: &x, Y: &y}}},
		"titik bebas":  {From: a.ID},
		"jalur":        {From: a.ID, Geometry: ConnectorGeometry{Route: "zigzag", To: ConnectorEnd{X: &x, Y: &y}}},
		"jenis ujung":  {From: a.ID, Geometry: ConnectorGeometry{To: ConnectorEnd{X: &x, Y: &y}}, Style: ConnectorStyle{End: "bintang"}},
		"terlalu jauh": {From: a.ID, Geometry: ConnectorGeometry{To: ConnectorEnd{X: &x, Y: &y}, Points: [][2]float64{{2e6, 0}}}},
	} {
		if _, err := f.svc.CreateConnector(ctx, f.rina, RootID, in); err == nil {
			t.Errorf("%s: seharusnya ditolak", name)
		}
	}
	// Mengubah jalur: titik bebas tetap wajib untuk ujung yang tidak menempel.
	_, err = f.svc.UpdateConnector(ctx, f.rina, RootID, conn.ID, ConnectorPatch{Geometry: &ConnectorGeometry{Route: "curved"}})
	wantInvalid(t, err, "butuh titik")
	got, err := f.svc.UpdateConnector(ctx, f.rina, RootID, conn.ID, ConnectorPatch{Geometry: &ConnectorGeometry{Route: "curved",
		From: ConnectorEnd{Port: "bottom", X: &x, Y: &y}, To: ConnectorEnd{X: &x, Y: &y}}})
	if err != nil || got.Geometry.Route != "curved" || got.Geometry.From.X != nil || got.Geometry.From.Port != "bottom" {
		t.Fatalf("ubah jalur %+v %v", got.Geometry, err)
	}
}
