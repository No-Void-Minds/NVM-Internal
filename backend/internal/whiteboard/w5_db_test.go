package whiteboard

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// Bentuk (W5): dibuat dan diubah lewat jalur kartu yang sudah ada (patch isi dengan content_version, 409 bila
// basi), isinya divalidasi server, teksnya ikut pencarian dan template, dan garis menempel ke port sudut bentuk
// dengan jalur siku dan titik belok tersimpan.
func TestWbBentuk(t *testing.T) {
	f := setupWb(t)
	ctx := context.Background()
	b := f.board(t, f.root, "Alur pesanan").Ref.ID

	decision := f.card(t, b, KindShape, `{"shape":"decision","rotation":0,"fill":"kuning","stroke":"tinta","stroke_width":2,"stroke_style":"solid","text":"Stok tersedia?","align":"center","font_size":16}`)
	class := f.card(t, b, KindShape, `{"shape":"uml_class","rotation":0,"fill":"putih","stroke":"tinta","stroke_width":1,"stroke_style":"solid","text":"","align":"left","font_size":14,"name":"Pesanan","attributes":"+ id: int","methods":"+ bayar()"}`)
	var got ShapeContent
	if err := json.Unmarshal(decision.Content, &got); err != nil || got.Shape != "decision" || got.Text != "Stok tersedia?" {
		t.Fatalf("isi bentuk %s %v", decision.Content, err)
	}

	// Isi tidak sah ditolak saat dibuat dan saat diubah.
	_, err := f.svc.CreateCard(ctx, f.rina, b, CardInput{Kind: KindShape, W: 160, H: 100, Content: json.RawMessage(`{"shape":"awan","rotation":0,"fill":"putih","stroke":"tinta","stroke_width":2,"stroke_style":"solid","text":"","align":"center","font_size":16}`)})
	wantInvalid(t, err, "Jenis bentuk")
	_, err = f.svc.PatchCards(ctx, f.rina, b, []CardPatch{{ID: decision.ID, Content: json.RawMessage(`{"shape":"decision","rotation":400,"fill":"kuning","stroke":"tinta","stroke_width":2,"stroke_style":"solid","text":"x","align":"center","font_size":16}`), BaseContentVersion: decision.ContentVersion}})
	wantInvalid(t, err, "Sudut putar")

	// Putar 45 derajat lewat patch isi; versi isi naik, versi basi ditolak 409 dengan isi terbaru.
	rotated := json.RawMessage(`{"shape":"decision","rotation":45,"fill":"kuning","stroke":"tinta","stroke_width":2,"stroke_style":"solid","text":"Stok tersedia?","align":"center","font_size":16}`)
	out, err := f.svc.PatchCards(ctx, f.rina, b, []CardPatch{{ID: decision.ID, Content: rotated, BaseContentVersion: decision.ContentVersion}})
	if err != nil || len(out) != 1 || out[0].ContentVersion != decision.ContentVersion+1 {
		t.Fatalf("putar %+v %v", out, err)
	}
	_, err = f.svc.PatchCards(ctx, f.nala, b, []CardPatch{{ID: decision.ID, Content: rotated, BaseContentVersion: decision.ContentVersion}})
	var conflict *ConflictError
	if !errors.As(err, &conflict) || conflict.Card.ContentVersion != out[0].ContentVersion {
		t.Fatalf("versi basi tidak 409: %v", err)
	}

	// Teks bentuk dan bagian kelas ikut pencarian isi.
	for _, q := range []string{"Stok tersedia", "bayar()"} {
		hits, err := f.svc.SearchCards(ctx, q)
		if err != nil || len(hits) != 1 || hits[0].Kind != KindShape {
			t.Fatalf("cari %q: %+v %v", q, hits, err)
		}
	}

	// Garis dari sudut kanan bawah bentuk ke port atas kelas, siku, dengan satu titik belok dan tiga label.
	conn, err := f.svc.CreateConnector(ctx, f.rina, b, ConnectorInput{
		From: decision.ID, To: class.ID,
		Geometry:   ConnectorGeometry{From: ConnectorEnd{Port: "bottom_right"}, To: ConnectorEnd{Port: "top"}, Route: "orthogonal"},
		Style:      ConnectorStyle{Start: "diamond_open", End: "triangle", Line: "dashed", Color: "biru"},
		Label:      "ya",
		LabelStart: "1",
		LabelEnd:   "0..*",
	})
	if err != nil || conn.Geometry.Route != "orthogonal" || conn.Geometry.From.Port != "bottom_right" || conn.LabelEnd != "0..*" {
		t.Fatalf("garis %+v %v", conn, err)
	}
	geo := ConnectorGeometry{From: ConnectorEnd{Port: "bottom_right"}, To: ConnectorEnd{Port: "top"}, Route: "curved", Points: [][2]float64{{300, 400}}}
	upd, err := f.svc.UpdateConnector(ctx, f.rina, b, conn.ID, ConnectorPatch{Geometry: &geo})
	if err != nil || upd.Geometry.Route != "curved" || len(upd.Geometry.Points) != 1 || upd.Geometry.Points[0] != [2]float64{300, 400} {
		t.Fatalf("titik belok %+v %v", upd.Geometry, err)
	}
	bad := ConnectorGeometry{From: ConnectorEnd{Port: "tengah"}, To: ConnectorEnd{Port: "top"}, Route: "orthogonal"}
	_, err = f.svc.UpdateConnector(ctx, f.rina, b, conn.ID, ConnectorPatch{Geometry: &bad})
	if err == nil {
		t.Fatal("port tidak dikenal diterima")
	}

	// Template membawa bentuk beserta garisnya.
	if _, err := f.svc.SetTemplate(ctx, f.rina, b, true); err != nil {
		t.Fatal(err)
	}
	copyCard, err := f.svc.FromTemplate(ctx, f.nala, RootID, TemplateInput{TemplateID: b, X: 600, W: 200, H: 120})
	if err != nil {
		t.Fatal(err)
	}
	v, err := f.svc.Board(ctx, f.nala, copyCard.Ref.ID)
	if err != nil || len(v.Cards) != 2 || v.Cards[0].Kind != KindShape || len(v.Connectors) != 1 || v.Connectors[0].Geometry.Route != "curved" {
		t.Fatalf("salinan template: %d kartu, %d garis, %v", len(v.Cards), len(v.Connectors), err)
	}
}
