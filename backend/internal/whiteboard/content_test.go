package whiteboard

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"nvm-internal/internal/apperr"
)

func invalid(err error) bool {
	var v *apperr.Validation
	return errors.As(err, &v)
}

// Dokumen catatan dibangun ulang dari daftar putih: atribut dan field asing hilang, isi yang sah tetap.
func TestCleanNoteMembangunUlang(t *testing.T) {
	raw := `{"doc":{"type":"doc","content":[
		{"type":"heading","attrs":{"level":2,"onclick":"x"},"content":[{"type":"text","text":"Judul"}]},
		{"type":"paragraph","attrs":{"style":"color:red"},"content":[
			{"type":"text","text":"tebal","marks":[{"type":"bold"},{"type":"bold"}]},
			{"type":"hardBreak"},
			{"type":"text","text":"tautan","marks":[{"type":"link","attrs":{"href":"https://nvm.test/a","target":"_self","class":"x"}}]}
		]},
		{"type":"bulletList","content":[]}
	]},"extra":"<script>"}`
	out, plain, err := CleanContent(KindNote, json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, bad := range []string{"onclick", "style", "target", "class", "extra", "script"} {
		if strings.Contains(s, bad) {
			t.Fatalf("%q masih ada di %s", bad, s)
		}
	}
	if strings.Count(s, `"bold"`) != 1 || !strings.Contains(s, `"href":"https://nvm.test/a"`) || !strings.Contains(s, `"level":2`) {
		t.Fatalf("isi sah hilang: %s", s)
	}
	// Daftar kosong diganti struktur sah, karena skema ProseMirror menolak daftar tanpa butir.
	if !strings.Contains(s, `{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph"}]}]}`) {
		t.Fatalf("daftar kosong tidak dirapikan: %s", s)
	}
	if plain != "Judul\ntebal\ntautan" {
		t.Fatalf("teks polos %q", plain)
	}
}

func TestCleanNoteMenolakYangBerbahaya(t *testing.T) {
	cases := map[string]string{
		"javascript":   `{"doc":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","attrs":{"href":"javascript:alert(1)"}}]}]}]}}`,
		"JaVaScRiPt":   `{"doc":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","attrs":{"href":" JaVaScRiPt:alert(1)"}}]}]}]}}`,
		"data":         `{"doc":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","attrs":{"href":"data:text/html,<b>"}}]}]}]}}`,
		"tanpa host":   `{"doc":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","attrs":{"href":"https:///x"}}]}]}]}}`,
		"node html":    `{"doc":{"type":"doc","content":[{"type":"html","content":[]}]}}`,
		"node gambar":  `{"doc":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"image","attrs":{"src":"x"}}]}]}}`,
		"mark asing":   `{"doc":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"textStyle","attrs":{"color":"red"}}]}]}]}}`,
		"teks di doc":  `{"doc":{"type":"doc","content":[{"type":"text","text":"x"}]}}`,
		"bukan doc":    `{"doc":{"type":"paragraph"}}`,
		"json rusak":   `{"doc":`,
		"kosong":       ``,
		"terlalu jauh": `{"doc":{"type":"doc","content":[` + strings.Repeat(`{"type":"blockquote","content":[`, 9) + `{"type":"paragraph"}` + strings.Repeat(`]}`, 9) + `]}}`,
	}
	for name, raw := range cases {
		if _, _, err := CleanContent(KindNote, json.RawMessage(raw)); !invalid(err) {
			t.Errorf("%s: mau galat validasi, dapat %v", name, err)
		}
	}
	big := `{"doc":{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"` + strings.Repeat("a", MaxContentBytes) + `"}]}]}}`
	if _, _, err := CleanContent(KindNote, json.RawMessage(big)); !invalid(err) {
		t.Errorf("isi lebih dari 64 KB diterima: %v", err)
	}
}

func TestCleanHref(t *testing.T) {
	for _, ok := range []string{"https://novoidminds.xyz/a?b=1", "http://contoh.test", "mailto:tim@nvm.test"} {
		if _, err := CleanHref(ok); err != nil {
			t.Errorf("%s ditolak: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "javascript:alert(1)", "vbscript:x", "file:///etc/passwd", "//evil.test", "/relatif", "https://a\nb.test", "ftp://x.test"} {
		if _, err := CleanHref(bad); err == nil {
			t.Errorf("%q diterima", bad)
		}
	}
}

func TestCleanChecklistDanHeading(t *testing.T) {
	id := "3f2b6c1e-8a4d-4c2b-9e1f-0a1b2c3d4e5f"
	out, plain, err := CleanContent(KindChecklist, json.RawMessage(`{"title":" Rilis \u0007","items":[{"id":"`+strings.ToUpper(id)+`","text":" Brief ","done":true,"x":1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"title":"Rilis","items":[{"id":"`+id+`","text":"Brief","done":true}]}` || plain != "Rilis\nBrief" {
		t.Fatalf("checklist %s %q", out, plain)
	}
	for name, raw := range map[string]string{
		"id bukan uuid": `{"title":"","items":[{"id":"1","text":"a"}]}`,
		"id ganda":      `{"title":"","items":[{"id":"` + id + `","text":"a"},{"id":"` + id + `","text":"b"}]}`,
		"teks panjang":  `{"title":"","items":[{"id":"` + id + `","text":"` + strings.Repeat("a", MaxItemText+1) + `"}]}`,
	} {
		if _, _, err := CleanContent(KindChecklist, json.RawMessage(raw)); !invalid(err) {
			t.Errorf("%s: mau galat validasi, dapat %v", name, err)
		}
	}
	h, _, err := CleanContent(KindHeading, json.RawMessage(`{"text":"Kampanye","level":7}`))
	if err != nil || string(h) != `{"text":"Kampanye","level":1}` {
		t.Fatalf("heading %s %v", h, err)
	}
	if _, _, err := CleanContent("image", json.RawMessage(`{}`)); !invalid(err) {
		t.Fatalf("jenis tidak dikenal diterima: %v", err)
	}
}

func TestApplyChecklistOp(t *testing.T) {
	a, b := "00000000-0000-4000-8000-00000000000a", "00000000-0000-4000-8000-00000000000b"
	c := ChecklistContent{Items: []ChecklistItem{{ID: a, Text: "A"}, {ID: b, Text: "B"}}}
	done, zero, text := true, 0, "C"
	steps := []ChecklistOp{
		{Op: "set", ItemID: b, Done: &done},
		{Op: "add", ItemID: "00000000-0000-4000-8000-00000000000c", Text: &text, Index: &zero},
		{Op: "move", ItemID: a, Index: &zero},
		{Op: "remove", ItemID: b},
	}
	for _, op := range steps {
		if err := applyChecklistOp(&c, op); err != nil {
			t.Fatal(err)
		}
	}
	if len(c.Items) != 2 || c.Items[0].ID != a || c.Items[1].Text != "C" {
		t.Fatalf("hasil %+v", c.Items)
	}
	if err := applyChecklistOp(&c, ChecklistOp{Op: "set", ItemID: b, Done: &done}); !invalid(err) {
		t.Fatalf("butir yang sudah dihapus: %v", err)
	}
}
