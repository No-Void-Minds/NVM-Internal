package whiteboard

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func stroke(points int) string {
	p := make([]string, points*2)
	for i := range p {
		p[i] = "1"
	}
	return `{"color":"tinta","width":2,"points":[` + strings.Join(p, ",") + `]}`
}

func TestCleanDrawing(t *testing.T) {
	ok := `{"strokes":[{"color":"biru","width":4,"points":[0,0,4000,4000]},` + stroke(10) + `]}`
	if _, _, err := CleanDrawing(json.RawMessage(ok)); err != nil {
		t.Fatalf("gambar sah ditolak: %v", err)
	}
	// Field asing tidak ditolak tetapi tidak ikut tersimpan, karena isi dibangun ulang dari struct.
	if v, _, err := CleanDrawing(json.RawMessage(`{"strokes":[{"color":"tinta","width":2,"points":[1,2]}],"script":"x"}`)); err != nil {
		t.Fatal(err)
	} else if b, _ := json.Marshal(v); string(b) != `{"strokes":[{"color":"tinta","width":2,"points":[1,2]}]}` {
		t.Fatalf("field asing ikut: %s", b)
	}
	many := make([]string, MaxStrokes+1)
	for i := range many {
		many[i] = stroke(1)
	}
	tooMany := make([]string, 4)
	for i := range tooMany {
		tooMany[i] = stroke(MaxDrawingPoints / 4)
	}
	for name, raw := range map[string]string{
		"kosong":          `{"strokes":[]}`,
		"warna":           `{"strokes":[{"color":"#ff0000","width":2,"points":[0,0]}]}`,
		"ketebalan":       `{"strokes":[{"color":"tinta","width":3,"points":[0,0]}]}`,
		"titik ganjil":    `{"strokes":[{"color":"tinta","width":2,"points":[0,0,1]}]}`,
		"di luar":         `{"strokes":[{"color":"tinta","width":2,"points":[0,4001]}]}`,
		"negatif":         `{"strokes":[{"color":"tinta","width":2,"points":[-1,0]}]}`,
		"goresan banyak":  `{"strokes":[` + strings.Join(many, ",") + `]}`,
		"goresan panjang": `{"strokes":[` + stroke(MaxStrokePoints+1) + `]}`,
		"total titik":     `{"strokes":[` + strings.Join(tooMany, ",") + `,` + stroke(1) + `]}`,
	} {
		if _, _, err := CleanDrawing(json.RawMessage(raw)); err == nil {
			t.Errorf("%s: seharusnya ditolak", name)
		}
	}
}

func TestCleanTableDanOp(t *testing.T) {
	if _, _, err := CleanTable(json.RawMessage(`{"header":true,"rows":[["a","b"],["c"]]}`)); err == nil {
		t.Error("baris tidak sama lebar diterima")
	}
	wide := `[` + strings.TrimSuffix(strings.Repeat(`"",`, MaxTableCols+1), ",") + `]`
	if _, _, err := CleanTable(json.RawMessage(`{"header":false,"rows":[` + wide + `]}`)); err == nil {
		t.Error("kolom berlebih diterima")
	}
	rows := strings.TrimSuffix(strings.Repeat(`[""],`, MaxTableRows+1), ",")
	if _, _, err := CleanTable(json.RawMessage(`{"header":false,"rows":[` + rows + `]}`)); err == nil {
		t.Error("baris berlebih diterima")
	}
	long := fmt.Sprintf(`{"header":false,"rows":[["%s"]]}`, strings.Repeat("x", MaxCellText+1))
	if _, _, err := CleanTable(json.RawMessage(long)); err == nil {
		t.Error("sel kepanjangan diterima")
	}

	tb := TableContent{Rows: [][]string{{"a", "b"}, {"c", "d"}}}
	one := 1
	for _, op := range []TableOp{
		{Op: "set", Row: 1, Col: 0, Text: "C"},
		{Op: "add_row", Index: &one},
		{Op: "add_col"},
		{Op: "remove_col", Index: &one},
		{Op: "header", On: true},
	} {
		if err := applyTableOp(&tb, op); err != nil {
			t.Fatalf("%s: %v", op.Op, err)
		}
	}
	got, _ := json.Marshal(tb)
	if string(got) != `{"title":"","header":true,"rows":[["a",""],["",""],["C",""]]}` {
		t.Fatalf("hasil op %s", got)
	}
	for _, op := range []TableOp{{Op: "set", Row: 9}, {Op: "geser"}} {
		if err := applyTableOp(&tb, op); err == nil {
			t.Errorf("%+v seharusnya ditolak", op)
		}
	}
	small := TableContent{Rows: [][]string{{"x"}}}
	if applyTableOp(&small, TableOp{Op: "remove_row"}) == nil || applyTableOp(&small, TableOp{Op: "remove_col"}) == nil {
		t.Error("tabel 1x1 bisa dikosongkan")
	}
}
