package whiteboard

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func table(name, rows string) json.RawMessage {
	return json.RawMessage(`{"shape":"erd_table","rotation":0,"fill":"putih","stroke":"tinta","stroke_width":2,"stroke_style":"solid","text":"sisa","align":"center","font_size":14,"name":"` + name + `","rows":` + rows + `}`)
}

// Revisi 57a: kolom tabel ERD divalidasi (id, nama, tipe, kunci, jumlah), kunci disimpan dalam urutan tetap, dan
// kolom ditolak di bentuk lain.
func TestTabelERD(t *testing.T) {
	out, plain, err := CleanShape(table(" pesanan ", `[{"id":"a1","name":"id","type":"bigint","keys":["uk","pk"],"nullable":false},{"id":"b2","name":"pelanggan_id","type":"bigint","keys":["fk"],"nullable":true}]`))
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "pesanan" || out.Text != "" || len(out.Rows) != 2 {
		t.Fatalf("isi tabel %+v", out)
	}
	if strings.Join(out.Rows[0].Keys, ",") != "pk,uk" || !out.Rows[1].Nullable {
		t.Errorf("kunci %v, nullable %v", out.Rows[0].Keys, out.Rows[1].Nullable)
	}
	if plain != "pesanan\nid bigint\npelanggan_id bigint" {
		t.Errorf("teks cari %q", plain)
	}
	if out, _, err := CleanShape(table("kosong", `[]`)); err != nil || len(out.Rows) != 0 {
		t.Errorf("tabel tanpa kolom: %v", err)
	}
	many := make([]string, MaxErdRows+1)
	for i := range many {
		many[i] = fmt.Sprintf(`{"id":"r%d","name":"k","type":"","keys":[],"nullable":false}`, i)
	}
	long := strings.Repeat("x", 101)
	for _, bad := range []string{
		`[` + strings.Join(many, ",") + `]`,
		`[{"id":"A","name":"id","type":"","keys":[],"nullable":false}]`,
		`[{"id":"a","name":"id","type":"","keys":[],"nullable":false},{"id":"a","name":"id2","type":"","keys":[],"nullable":false}]`,
		`[{"id":"a","name":"id","type":"","keys":["idx"],"nullable":false}]`,
		`[{"id":"a","name":"` + long + `","type":"","keys":[],"nullable":false}]`,
		`[{"id":"a","name":"id","type":"` + strings.Repeat("t", 61) + `","keys":[],"nullable":false}]`,
	} {
		if _, _, err := CleanShape(table("t", bad)); err == nil {
			t.Errorf("kolom diterima: %.80s", bad)
		}
	}
	if _, _, err := CleanShape(table(long, `[]`)); err == nil {
		t.Error("nama tabel 101 karakter diterima")
	}
	rect := json.RawMessage(`{"shape":"rect","rotation":0,"fill":"putih","stroke":"tinta","stroke_width":2,"stroke_style":"solid","text":"","align":"center","font_size":16,"rows":[{"id":"a","name":"id","type":"","keys":[],"nullable":false}]}`)
	if _, _, err := CleanShape(rect); err == nil {
		t.Error("kolom di persegi diterima")
	}
}

// Ujung garis ke baris tabel: hanya sisi kiri atau kanan, id baris sah, dan ujung yang menempel ke kartu.
func TestUjungGarisKeBaris(t *testing.T) {
	ok := ConnectorGeometry{From: ConnectorEnd{Port: "right", Row: "a1"}, To: ConnectorEnd{Port: "left", Row: "b2"}}
	if _, err := cleanGeometry(ok, true, true); err != nil {
		t.Fatal(err)
	}
	x, y := 10.0, 20.0
	for _, g := range []ConnectorGeometry{
		{From: ConnectorEnd{Port: "top", Row: "a1"}},
		{From: ConnectorEnd{Port: "auto", Row: "a1"}},
		{From: ConnectorEnd{Port: "left", Row: "A-1"}},
		{From: ConnectorEnd{Port: "left", Row: "a1", X: &x, Y: &y}, To: ConnectorEnd{Port: "left"}},
	} {
		attached := g.From.X == nil
		if _, err := cleanGeometry(g, attached, true); err == nil {
			t.Errorf("ujung baris diterima: %+v", g.From)
		}
	}
}
