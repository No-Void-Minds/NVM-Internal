package whiteboard

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"nvm-internal/internal/apperr"
)

// Bentuk dan diagram (W5, revisi C). Satu jenis kartu "shape"; jenis bentuknya dari daftar putih dan digambar klien
// sebagai SVG dari angka, jadi tidak ada markup dari pengguna yang dirender.

const (
	KindShape = "shape"
	// MaxShapeText: teks di dalam bentuk dan tiap bagian kelas UML.
	MaxShapeText = 2000
)

var (
	ShapeTypes = map[string]bool{
		// Dasar
		"rect": true, "rounded": true, "ellipse": true, "triangle": true, "diamond": true, "hexagon": true, "star": true,
		// Flowchart
		"process": true, "decision": true, "terminator": true, "data": true, "database": true, "document": true,
		// UML
		"uml_class": true, "uml_actor": true, "uml_note": true, "uml_package": true,
		// Panah blok
		"arrow_right": true, "arrow_left": true, "arrow_up": true, "arrow_down": true, "arrow_both": true,
	}
	// Isian memakai nama warna kartu; "" berarti tanpa isian, "putih" latar surface.
	shapeFills = map[string]bool{"": true, "putih": true, "abu": true, "hijau": true, "kuning": true, "biru": true, "krem": true, "merah": true}
	// Garis tepi memakai warna pena (token tinta gelap, minimal 3:1 terhadap canvas).
	shapeStrokes = map[string]bool{"tinta": true, "abu": true, "hijau": true, "kuning": true, "biru": true, "krem": true, "merah": true}
	shapeAligns  = map[string]bool{"left": true, "center": true, "right": true}
	shapeFonts   = map[int]bool{12: true, 14: true, 16: true, 20: true, 24: true, 32: true}
)

type ShapeContent struct {
	Shape       string `json:"shape"`
	Rotation    int    `json:"rotation"`
	Fill        string `json:"fill"`
	Stroke      string `json:"stroke"`
	StrokeWidth int    `json:"stroke_width"`
	StrokeStyle string `json:"stroke_style"`
	Text        string `json:"text"`
	Align       string `json:"align"`
	FontSize    int    `json:"font_size"`
	// Kelas UML: tiga bagian (nama, atribut, metode); Text kosong.
	Name       string `json:"name,omitempty"`
	Attributes string `json:"attributes,omitempty"`
	Methods    string `json:"methods,omitempty"`
}

// cleanMultiline seperti CleanText tetapi mempertahankan baris baru (atribut dan metode UML satu per baris).
func cleanMultiline(v string, max int, label string) (string, error) {
	v = strings.ReplaceAll(v, "\r\n", "\n")
	lines := strings.Split(v, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(stripControl(l, false), " ")
	}
	v = strings.Trim(strings.Join(lines, "\n"), "\n ")
	if utf8.RuneCountInString(v) > max {
		return "", apperr.Invalid(fmt.Sprintf("%s maksimal %d karakter.", label, max))
	}
	return v, nil
}

// CleanShape memvalidasi isi kartu bentuk. Nilai di luar daftar ditolak, bukan dibetulkan diam-diam, supaya klien
// yang salah ketahuan.
func CleanShape(raw json.RawMessage) (ShapeContent, string, error) {
	var in ShapeContent
	if err := decodeStrict(raw, &in); err != nil {
		return in, "", err
	}
	if !ShapeTypes[in.Shape] {
		return in, "", apperr.Invalid("Jenis bentuk tidak dikenal.")
	}
	if in.Rotation < 0 || in.Rotation > 359 {
		return in, "", apperr.Invalid("Sudut putar harus 0 sampai 359 derajat.")
	}
	if !shapeFills[in.Fill] {
		return in, "", apperr.Invalid("Warna isian bentuk tidak dikenal.")
	}
	if !shapeStrokes[in.Stroke] {
		return in, "", apperr.Invalid("Warna garis bentuk tidak dikenal.")
	}
	if in.StrokeWidth < 1 || in.StrokeWidth > 8 {
		return in, "", apperr.Invalid("Tebal garis bentuk harus 1 sampai 8.")
	}
	if in.StrokeStyle != "solid" && in.StrokeStyle != "dashed" {
		return in, "", apperr.Invalid("Gaya garis bentuk harus utuh atau putus-putus.")
	}
	if !shapeAligns[in.Align] {
		return in, "", apperr.Invalid("Perataan teks bentuk tidak dikenal.")
	}
	if !shapeFonts[in.FontSize] {
		return in, "", apperr.Invalid("Ukuran huruf bentuk harus 12, 14, 16, 20, 24, atau 32.")
	}
	out := in
	var err error
	if in.Shape == "uml_class" {
		out.Text = ""
		if out.Name, err = CleanText(in.Name, MaxHeadingText, "Nama kelas"); err != nil {
			return in, "", err
		}
		if out.Attributes, err = cleanMultiline(in.Attributes, MaxShapeText, "Atribut kelas"); err != nil {
			return in, "", err
		}
		if out.Methods, err = cleanMultiline(in.Methods, MaxShapeText, "Metode kelas"); err != nil {
			return in, "", err
		}
		plain := strings.TrimSpace(strings.Join([]string{out.Name, out.Attributes, out.Methods}, "\n"))
		return out, plain, nil
	}
	out.Name, out.Attributes, out.Methods = "", "", ""
	if out.Text, err = cleanMultiline(in.Text, MaxShapeText, "Teks bentuk"); err != nil {
		return in, "", err
	}
	return out, out.Text, nil
}

// defaultShape: isi awal bentuk baru dari palet. Jenis bentuk dikirim klien; ini hanya cadangan untuk kartu tanpa isi.
const defaultShape = `{"shape":"rect","rotation":0,"fill":"putih","stroke":"tinta","stroke_width":2,"stroke_style":"solid","text":"","align":"center","font_size":16}`
