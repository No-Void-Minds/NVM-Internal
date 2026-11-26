package whiteboard

import (
	"encoding/json"
	"strings"
	"testing"
)

func richPara(marks string, attrs string, text string) string {
	if attrs == "" {
		attrs = "{}"
	}
	return `{"type":"doc","content":[{"type":"paragraph","attrs":` + attrs + `,"content":[{"type":"text","text":"` + text + `","marks":` + marks + `}]}]}`
}

// Revisi 49: mark dan atribut format baru dari daftar putih; warna dan stabilo hanya nama token.
func TestCleanNoteFormatBaru(t *testing.T) {
	ok := `{"doc":{"type":"doc","content":[
		{"type":"paragraph","attrs":{"textAlign":"center"},"content":[
			{"type":"text","text":"Tebal","marks":[{"type":"bold"},{"type":"textStyle","attrs":{"color":"merah","fontSize":"besar"}},{"type":"highlight","attrs":{"color":"kuning"}}]},
			{"type":"text","text":"2","marks":[{"type":"superscript"}]},
			{"type":"text","text":"x","marks":[{"type":"subscript"},{"type":"underline"},{"type":"strike"},{"type":"code"}]}]},
		{"type":"horizontalRule"},
		{"type":"heading","attrs":{"level":3,"textAlign":"justify"},"content":[{"type":"text","text":"Judul"}]}]}}`
	v, plain, err := CleanNote(json.RawMessage(ok))
	if err != nil {
		t.Fatalf("ditolak: %v", err)
	}
	b, _ := json.Marshal(v)
	for _, want := range []string{`"textAlign":"center"`, `"color":"merah"`, `"fontSize":"besar"`, `"type":"highlight","attrs":{"color":"kuning"}`, `"superscript"`, `"subscript"`, `"horizontalRule"`, `"textAlign":"justify"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("hilang %s dari %s", want, b)
		}
	}
	if plain != "Tebal2x\nJudul" {
		t.Errorf("plain %q", plain)
	}

	bad := map[string]string{
		"Warna teks":     `[{"type":"textStyle","attrs":{"color":"#ff0000"}}]`,
		"Warna teks ":    `[{"type":"textStyle","attrs":{"color":"collab-3"}}]`,
		"Warna stabilo":  `[{"type":"highlight","attrs":{"color":"neon"}}]`,
		"stabilo kosong": `[{"type":"highlight"}]`,
		"Ukuran huruf":   `[{"type":"textStyle","attrs":{"fontSize":"72px"}}]`,
		"Tautan":         `[{"type":"link","attrs":{"href":"javascript:alert(1)"}}]`,
		"data":           `[{"type":"link","attrs":{"href":"data:text/html,x"}}]`,
		"tidak didukung": `[{"type":"fontFamily"}]`,
	}
	for name, marks := range bad {
		if _, _, err := CleanNote(json.RawMessage(`{"doc":` + richPara(marks, "", "x") + `}`)); err == nil {
			t.Errorf("%s diterima", name)
		}
	}
	if _, _, err := CleanNote(json.RawMessage(`{"doc":` + richPara(`[]`, `{"textAlign":"middle"}`, "x") + `}`)); err == nil {
		t.Error("perataan tidak dikenal diterima")
	}
	// Kiri adalah bawaan dan tidak disimpan; textStyle tanpa atribut dibuang.
	v, _, err = CleanNote(json.RawMessage(`{"doc":` + richPara(`[{"type":"textStyle","attrs":{}}]`, `{"textAlign":"left"}`, "x") + `}`))
	if b, _ := json.Marshal(v); err != nil || strings.Contains(string(b), "textAlign") || strings.Contains(string(b), "textStyle") {
		t.Errorf("bawaan ikut tersimpan: %s %v", b, err)
	}
}

// Field yang tadinya teks polos: data lama (teks saja) tetap sah tanpa migrasi, dokumen berformat disimpan dengan
// teks polos yang dihasilkan server, dan dokumen tanpa format apa pun disimpan sebagai teks saja.
func TestCleanFieldBerformat(t *testing.T) {
	rich := richPara(`[{"type":"bold"}]`, "", "Penting")
	plainOnly := richPara(`[]`, "", "Biasa")
	id1 := "11111111-1111-4111-8111-111111111111"
	id2 := "22222222-2222-4222-8222-222222222222"

	old := []struct{ kind, raw string }{
		{KindHeading, `{"text":"Judul lama","level":1}`},
		{KindChecklist, `{"title":"Rilis","items":[{"id":"` + id1 + `","text":"Foto","done":false}]}`},
		{KindTable, `{"title":"Harga","header":true,"rows":[["a","b"]]}`},
		{KindColumn, `{"title":"Ide"}`},
		{KindShape, defaultShape},
	}
	for _, o := range old {
		out, _, err := CleanContent(o.kind, json.RawMessage(o.raw))
		if err != nil || strings.Contains(string(out), "doc") {
			t.Errorf("data lama %s: %s %v", o.kind, out, err)
		}
	}

	out, plain, err := CleanContent(KindChecklist, json.RawMessage(`{"title":"","title_doc":`+rich+`,"items":[
		{"id":"`+id1+`","text":"diabaikan","done":false,"doc":`+rich+`},
		{"id":"`+id2+`","text":"","done":true,"doc":`+plainOnly+`}]}`))
	if err != nil {
		t.Fatal(err)
	}
	var c ChecklistContent
	_ = json.Unmarshal(out, &c)
	if c.Title != "Penting" || c.TitleDoc == nil || c.Items[0].Text != "Penting" || c.Items[0].Doc == nil {
		t.Errorf("checklist berformat: %s", out)
	}
	if c.Items[1].Text != "Biasa" || c.Items[1].Doc != nil {
		t.Errorf("dokumen tanpa format harus jadi teks saja: %s", out)
	}
	if plain != "Penting\nPenting\nBiasa" {
		t.Errorf("plain %q", plain)
	}

	// Batas panjang dihitung dari teks polos hasil dokumen.
	long := richPara(`[{"type":"italic"}]`, "", strings.Repeat("a", MaxItemText+1))
	if _, _, err := CleanContent(KindChecklist, json.RawMessage(`{"title":"","items":[{"id":"`+id1+`","text":"","doc":`+long+`}]}`)); err == nil {
		t.Error("butir lebih dari batas diterima")
	}
	// Dokumen di field kecil tetap melalui daftar putih yang sama.
	evil := richPara(`[{"type":"link","attrs":{"href":"javascript:x"}}]`, "", "klik")
	if _, _, err := CleanContent(KindHeading, json.RawMessage(`{"text":"","level":1,"doc":`+evil+`}`)); err == nil {
		t.Error("tautan berbahaya di kartu judul diterima")
	}

	out, _, err = CleanContent(KindShape, json.RawMessage(`{"shape":"uml_class","rotation":0,"fill":"putih","stroke":"tinta","stroke_width":2,"stroke_style":"solid","text":"","align":"left","font_size":14,"name":"","name_doc":`+rich+`,"attributes":"+ id","methods":""}`))
	var sh ShapeContent
	_ = json.Unmarshal(out, &sh)
	if err != nil || sh.Name != "Penting" || sh.NameDoc == nil || sh.Attributes != "+ id" || sh.AttributesDoc != nil {
		t.Errorf("kelas UML berformat: %s %v", out, err)
	}
}

// Operasi tabel menggeser format sel bersama teksnya.
func TestTableOpFormatSel(t *testing.T) {
	doc := &Node{Type: "doc", Content: []Node{{Type: "paragraph", Content: []Node{{Type: "text", Text: "B", Marks: []Mark{{Type: "bold"}}}}}}}
	tb := TableContent{Rows: [][]string{{"a", "b"}, {"c", "d"}}}
	if err := applyTableOp(&tb, TableOp{Op: "set", Row: 0, Col: 1, Text: "B", Doc: doc}); err != nil {
		t.Fatal(err)
	}
	zero := 0
	if err := applyTableOp(&tb, TableOp{Op: "add_col", Index: &zero}); err != nil {
		t.Fatal(err)
	}
	if err := applyTableOp(&tb, TableOp{Op: "add_row", Index: &zero}); err != nil {
		t.Fatal(err)
	}
	if tb.Rows[1][2] != "B" || tb.Docs[1][2] != doc || tb.Docs[1][1] != nil || len(tb.Docs) != 3 {
		t.Fatalf("format sel tidak ikut bergeser: %+v %+v", tb.Rows, tb.Docs)
	}
	if err := applyTableOp(&tb, TableOp{Op: "set", Row: 1, Col: 2, Text: "B"}); err != nil || tb.Docs != nil {
		t.Fatalf("sel tanpa format: Docs harus kosong lagi, %+v %v", tb.Docs, err)
	}
}
