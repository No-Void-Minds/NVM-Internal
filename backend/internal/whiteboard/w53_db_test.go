package whiteboard

import (
	"context"
	"math"
	"testing"
)

// Revisi 53: z garis bawaan null (otomatis), urutan manual tersimpan dan tervalidasi, z_auto mengembalikannya ke
// otomatis (jalur undo), siaran dan sinkron ulang membawa z, template menyalinnya, pindah papan melepasnya, dan
// bentuk diagram use case diterima daftar putih.
func TestWbUrutanGaris(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	b := f.board(t, f.root, "Use case").Ref.ID
	a := f.card(t, b, KindNote, note("Aktor"))
	u := f.card(t, b, KindNote, note("Use case"))

	h := NewHub(f.svc)
	cl := f.hubClient(t, h, f.nala, "nala", true)
	cl.do(h, msg{"t": "join", "board": b})
	cl.drain()

	conn, err := f.svc.CreateConnector(ctx, f.rina, b, ConnectorInput{From: a.ID, To: u.ID})
	if err != nil || conn.Z != nil {
		t.Fatalf("garis baru harus otomatis: %+v %v", conn.Z, err)
	}
	seq0 := conn.Seq

	// Manual: tersimpan dan ikut siaran.
	up, err := f.svc.UpdateConnector(ctx, f.rina, b, conn.ID, ConnectorPatch{Z: ptr(-3.5)})
	if err != nil || up.Z == nil || *up.Z != -3.5 {
		t.Fatalf("z manual %+v %v", up.Z, err)
	}
	h.Connectors(b, []Connector{up})
	got := last(cl.drain(), "connectors")
	if got == nil || got["items"].([]any)[0].(map[string]any)["z"] != -3.5 {
		t.Fatalf("siaran z %v", got)
	}
	v, _ := f.svc.Board(ctx, f.rina, b)
	if len(v.Connectors) != 1 || v.Connectors[0].Z == nil || *v.Connectors[0].Z != -3.5 {
		t.Fatalf("muat papan %+v", v.Connectors)
	}
	sync, err := f.svc.Since(ctx, b, seq0)
	if err != nil || len(sync.Connectors) != 1 || sync.Connectors[0].Z == nil || *sync.Connectors[0].Z != -3.5 {
		t.Fatalf("sinkron ulang %+v %v", sync.Connectors, err)
	}

	// Patch tanpa z tidak menyentuh z.
	up, err = f.svc.UpdateConnector(ctx, f.rina, b, conn.ID, ConnectorPatch{Label: ptr("include")})
	if err != nil || up.Z == nil || *up.Z != -3.5 {
		t.Fatalf("label menghapus z %+v %v", up.Z, err)
	}

	// Validasi.
	for _, bad := range []float64{math.Inf(1), math.NaN(), 2e9, -2e9} {
		_, err := f.svc.UpdateConnector(ctx, f.rina, b, conn.ID, ConnectorPatch{Z: ptr(bad)})
		wantInvalid(t, err, "Urutan tumpuk garis")
	}
	_, err = f.svc.UpdateConnector(ctx, f.rina, b, conn.ID, ConnectorPatch{Z: ptr(1.0), ZAuto: true})
	wantInvalid(t, err, "bukan keduanya")
	_, err = f.svc.CreateConnector(ctx, f.rina, b, ConnectorInput{From: a.ID, To: u.ID, Z: ptr(math.Inf(-1))})
	wantInvalid(t, err, "Urutan tumpuk garis")

	// Template menyalin z manual.
	if _, err := f.svc.SetTemplate(ctx, f.rina, b, true); err != nil {
		t.Fatal(err)
	}
	cp, err := f.svc.FromTemplate(ctx, f.rina, RootID, TemplateInput{TemplateID: b, X: 500, W: 200, H: 120})
	if err != nil {
		t.Fatal(err)
	}
	cv, _ := f.svc.Board(ctx, f.rina, cp.Ref.ID)
	if len(cv.Connectors) != 1 || cv.Connectors[0].Z == nil || *cv.Connectors[0].Z != -3.5 {
		t.Fatalf("salinan template %+v", cv.Connectors)
	}
	if _, err := f.svc.SetTemplate(ctx, f.rina, b, false); err != nil {
		t.Fatal(err)
	}

	// z_auto kembali ke otomatis.
	up, err = f.svc.UpdateConnector(ctx, f.rina, b, conn.ID, ConnectorPatch{ZAuto: true})
	if err != nil || up.Z != nil {
		t.Fatalf("z_auto %+v %v", up.Z, err)
	}

	// Garis kedua ujung bebas memakai z sendiri sejak dibuat.
	x1, y1, x2, y2 := 0.0, 0.0, 100.0, 0.0
	free, err := f.svc.CreateConnector(ctx, f.rina, b, ConnectorInput{Z: ptr(7.0),
		Geometry: ConnectorGeometry{From: ConnectorEnd{X: &x1, Y: &y1}, To: ConnectorEnd{X: &x2, Y: &y2}}})
	if err != nil || free.Z == nil || *free.Z != 7 {
		t.Fatalf("garis bebas %+v %v", free.Z, err)
	}

	// Pindah papan: urutan manual dilepas karena z kartu di papan tujuan berbeda.
	if _, err := f.svc.UpdateConnector(ctx, f.rina, b, conn.ID, ConnectorPatch{Z: ptr(9.0)}); err != nil {
		t.Fatal(err)
	}
	dest := f.board(t, f.root, "Tujuan").Ref.ID
	if _, err := f.svc.MoveCards(ctx, f.rina, b, []string{a.ID, u.ID}, dest); err != nil {
		t.Fatal(err)
	}
	dv, _ := f.svc.Board(ctx, f.rina, dest)
	if len(dv.Connectors) != 1 || dv.Connectors[0].Z != nil {
		t.Fatalf("garis pindahan %+v", dv.Connectors)
	}

	// Jalur bebas dengan anchor tersimpan; anchor di luar papan dan jalur tak dikenal ditolak.
	geo := ConnectorGeometry{Route: "bebas", Points: [][2]float64{{50, 80}, {120, -30}},
		Anchor: &[2][2]float64{{0, 0}, {200, 0}}}
	bv, err := f.svc.UpdateConnector(ctx, f.rina, dest, dv.Connectors[0].ID, ConnectorPatch{Geometry: &geo})
	if err != nil || bv.Geometry.Route != "bebas" || bv.Geometry.Anchor == nil || bv.Geometry.Anchor[1][0] != 200 ||
		len(bv.Geometry.Points) != 2 {
		t.Fatalf("jalur bebas %+v %v", bv.Geometry, err)
	}
	bad := geo
	bad.Anchor = &[2][2]float64{{0, 0}, {math.Inf(1), 0}}
	_, err = f.svc.UpdateConnector(ctx, f.rina, dest, bv.ID, ConnectorPatch{Geometry: &bad})
	wantInvalid(t, err, "Titik acuan")
	bad = geo
	bad.Route = "zigzag"
	_, err = f.svc.UpdateConnector(ctx, f.rina, dest, bv.ID, ConnectorPatch{Geometry: &bad})
	wantInvalid(t, err, "Jalur garis")

	// Bentuk diagram use case.
	for _, s := range []string{"uml_system", "uml_usecase"} {
		f.card(t, b, KindShape, `{"shape":"`+s+`","rotation":0,"fill":"","stroke":"tinta","stroke_width":2,"stroke_style":"solid","text":"Sistem","align":"center","font_size":16}`)
	}
}
