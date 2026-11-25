package whiteboard

import (
	"encoding/json"
	"strings"
	"testing"
)

func shapeJSON(t *testing.T, mut func(m map[string]any)) json.RawMessage {
	t.Helper()
	m := map[string]any{}
	if err := json.Unmarshal([]byte(defaultShape), &m); err != nil {
		t.Fatal(err)
	}
	mut(m)
	b, _ := json.Marshal(m)
	return b
}

// Isi bentuk (W5): jenis, sudut, warna, tebal, gaya, perataan, ukuran huruf, dan panjang teks dari daftar putih.
func TestCleanShape(t *testing.T) {
	ok := func(mut func(map[string]any)) ShapeContent {
		t.Helper()
		v, _, err := CleanShape(shapeJSON(t, mut))
		if err != nil {
			t.Fatalf("ditolak: %v", err)
		}
		return v
	}
	bad := func(part string, mut func(map[string]any)) {
		t.Helper()
		_, _, err := CleanShape(shapeJSON(t, mut))
		if err == nil || !strings.Contains(err.Error(), part) {
			t.Fatalf("mau galat %q, dapat %v", part, err)
		}
	}

	for shape := range ShapeTypes {
		ok(func(m map[string]any) { m["shape"] = shape })
	}
	bad("Jenis bentuk", func(m map[string]any) { m["shape"] = "lingkaran_ajaib" })
	bad("Jenis bentuk", func(m map[string]any) { m["shape"] = "<svg>" })
	ok(func(m map[string]any) { m["rotation"] = 359 })
	bad("Sudut putar", func(m map[string]any) { m["rotation"] = 360 })
	bad("Sudut putar", func(m map[string]any) { m["rotation"] = -1 })
	bad("tidak terbaca", func(m map[string]any) { m["rotation"] = 12.5 })
	bad("Warna isian", func(m map[string]any) { m["fill"] = "#ff0000" })
	ok(func(m map[string]any) { m["fill"] = "" })
	bad("Warna garis", func(m map[string]any) { m["stroke"] = "" })
	bad("Tebal garis", func(m map[string]any) { m["stroke_width"] = 0 })
	bad("Tebal garis", func(m map[string]any) { m["stroke_width"] = 9 })
	bad("Gaya garis", func(m map[string]any) { m["stroke_style"] = "dotted" })
	bad("Perataan", func(m map[string]any) { m["align"] = "justify" })
	bad("Ukuran huruf", func(m map[string]any) { m["font_size"] = 18 })
	bad("Teks bentuk maksimal 2000", func(m map[string]any) { m["text"] = strings.Repeat("a", 2001) })

	// Baris baru dipertahankan, karakter kontrol dibuang, CRLF dinormalkan.
	v := ok(func(m map[string]any) { m["text"] = "Mulai\r\nproses\x00 \n" })
	if v.Text != "Mulai\nproses" {
		t.Fatalf("teks %q", v.Text)
	}
	// Bagian kelas hanya untuk kelas UML.
	v = ok(func(m map[string]any) { m["name"] = "Pengguna"; m["methods"] = "masuk()" })
	if v.Name != "" || v.Methods != "" {
		t.Fatalf("bagian kelas ikut tersimpan di persegi: %+v", v)
	}
	v, plain, err := CleanShape(shapeJSON(t, func(m map[string]any) {
		m["shape"] = "uml_class"
		m["text"] = "dibuang"
		m["name"] = " Pengguna "
		m["attributes"] = "+ nama: string\n+ email: string"
		m["methods"] = "+ masuk(): bool"
	}))
	if err != nil || v.Text != "" || v.Name != "Pengguna" || plain != "Pengguna\n+ nama: string\n+ email: string\n+ masuk(): bool" {
		t.Fatalf("kelas UML %+v %q %v", v, plain, err)
	}
	bad("Nama kelas maksimal", func(m map[string]any) { m["shape"] = "uml_class"; m["name"] = strings.Repeat("a", 201) })
	bad("Metode kelas maksimal", func(m map[string]any) {
		m["shape"] = "uml_class"
		m["methods"] = strings.Repeat("a", 2001)
	})
}
