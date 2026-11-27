package whiteboard

import (
	"context"
	"testing"
)

// Revisi 54d: pesan ke diri sendiri hanya untuk jalur message (layanan dan CHECK migrasi 000029); jalurnya tidak bisa
// diganti ke jalur lain; pesan antar lifeline tersimpan dengan tingginya; lifeline, activation bar, dan fragmen ber-
// varian dan ber-operand diterima.
func TestWbPesanSequence(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	b := f.board(t, f.root, "Sequence").Ref.ID
	life := func(text string) Card {
		return f.card(t, b, KindShape, `{"shape":"uml_lifeline","variant":"none","rotation":0,"fill":"putih","stroke":"tinta","stroke_width":2,"stroke_style":"solid","text":"`+text+`","align":"center","font_size":16}`)
	}
	a := life("Pembeli")
	c := life("Toko")
	f.card(t, b, KindShape, `{"shape":"uml_activation","rotation":0,"fill":"putih","stroke":"tinta","stroke_width":2,"stroke_style":"solid","text":"","align":"center","font_size":16}`)
	f.card(t, b, KindShape, `{"shape":"uml_fragment","variant":"alt","rotation":0,"fill":"","stroke":"tinta","stroke_width":2,"stroke_style":"solid","text":"","align":"center","font_size":16,"lanes":[{"label":"ya","size":120},{"label":"tidak","size":120}]}`)

	msg := ConnectorGeometry{Route: "message", Points: [][2]float64{{0, 120}}}
	m, err := f.svc.CreateConnector(ctx, f.rina, b, ConnectorInput{From: a.ID, To: c.ID, Geometry: msg})
	if err != nil || m.Geometry.Route != "message" || len(m.Geometry.Points) != 1 || m.Geometry.Points[0][1] != 120 {
		t.Fatalf("pesan %+v %v", m.Geometry, err)
	}
	self, err := f.svc.CreateConnector(ctx, f.rina, b, ConnectorInput{From: a.ID, To: a.ID, Geometry: msg})
	if err != nil || self.From == nil || self.To == nil || *self.From != *self.To {
		t.Fatalf("pesan ke diri sendiri %+v %v", self, err)
	}
	_, err = f.svc.CreateConnector(ctx, f.rina, b, ConnectorInput{From: a.ID, To: a.ID, Geometry: ConnectorGeometry{Route: "straight"}})
	wantInvalid(t, err, "dua kartu yang berbeda")
	straight := ConnectorGeometry{Route: "orthogonal"}
	_, err = f.svc.UpdateConnector(ctx, f.rina, b, self.ID, ConnectorPatch{Geometry: &straight})
	wantInvalid(t, err, "jalur pesan")
	// Tinggi pesan diubah lewat titiknya, tetap pesan.
	moved := ConnectorGeometry{Route: "message", Points: [][2]float64{{0, 200}}}
	up, err := f.svc.UpdateConnector(ctx, f.rina, b, self.ID, ConnectorPatch{Geometry: &moved})
	if err != nil || up.Geometry.Points[0][1] != 200 {
		t.Fatalf("geser pesan %+v %v", up.Geometry, err)
	}
	v, _ := f.svc.Board(ctx, f.rina, b)
	if len(v.Connectors) != 2 {
		t.Fatalf("garis di papan %d", len(v.Connectors))
	}
}
