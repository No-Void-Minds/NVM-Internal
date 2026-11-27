package whiteboard

import (
	"encoding/json"
	"fmt"
	"slices"
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
		// Diagram use case (revisi 53): batas sistem (judul di dalam atas) dan use case (elips).
		"uml_system": true, "uml_usecase": true,
		// BPMN (revisi 54a)
		"bpmn_start": true, "bpmn_intermediate": true, "bpmn_end": true, "bpmn_task": true, "bpmn_subprocess": true,
		"bpmn_gateway": true, "bpmn_data_object": true, "bpmn_data_store": true, "bpmn_annotation": true,
		// Wadah ber-lane (revisi 54b)
		"bpmn_pool": true, "uml_swimlane": true,
		// Activity dan state machine (revisi 54c)
		"uml_initial": true, "uml_final": true, "uml_flow_final": true, "uml_action": true, "uml_decision": true,
		"uml_fork": true, "uml_object": true, "uml_send_signal": true, "uml_receive_signal": true, "uml_state": true,
		"uml_choice": true, "uml_state_composite": true,
		// Sequence diagram (revisi 54d)
		"uml_lifeline": true, "uml_lifeline_actor": true, "uml_activation": true, "uml_fragment": true,
		// Panah blok
		"arrow_right": true, "arrow_left": true, "arrow_up": true, "arrow_down": true, "arrow_both": true,
	}
	// ShapeVariants: varian per jenis bentuk (revisi 54), urutan sama dengan katalog klien; yang pertama bawaan.
	// Daftar ini dan ShapeTypes dicek sama dengan frontend/src/utils/wbShapeCatalog.json oleh TestKatalogBentuk.
	ShapeVariants = map[string][]string{
		"bpmn_start":         {"none", "message", "timer", "signal", "error"},
		"bpmn_intermediate":  {"none", "message", "timer", "error", "signal", "cancel"},
		"bpmn_end":           {"none", "message", "error", "signal", "cancel"},
		"bpmn_task":          {"none", "user", "service", "script", "manual", "send", "receive"},
		"bpmn_gateway":       {"exclusive", "parallel", "inclusive", "event"},
		"uml_lifeline":       {"none", "destroyed"},
		"uml_lifeline_actor": {"none", "destroyed"},
		"uml_fragment":       {"alt", "opt", "loop", "par"},
	}
	// laneShapes: bentuk yang punya lane (revisi 54b); bentuk lain tidak boleh membawa lanes.
	laneShapes = map[string]bool{"bpmn_pool": true, "uml_swimlane": true, "uml_fragment": true}
	// Isian memakai nama warna kartu; "" berarti tanpa isian, "putih" latar surface.
	shapeFills = map[string]bool{"": true, "putih": true, "abu": true, "hijau": true, "kuning": true, "biru": true, "krem": true, "merah": true}
	// Garis tepi memakai warna pena (token tinta gelap, minimal 3:1 terhadap canvas).
	shapeStrokes = map[string]bool{"tinta": true, "abu": true, "hijau": true, "kuning": true, "biru": true, "krem": true, "merah": true}
	shapeAligns  = map[string]bool{"left": true, "center": true, "right": true}
	shapeFonts   = map[int]bool{12: true, 14: true, 16: true, 20: true, 24: true, 32: true}
)

type ShapeContent struct {
	Shape string `json:"shape"`
	// Variant: varian bentuk dari ShapeVariants (revisi 54); kosong berarti varian pertama.
	Variant  string `json:"variant,omitempty"`
	Rotation int    `json:"rotation"`
	// Balik mendatar dan tegak (revisi 50): hanya gambar bentuknya, teks tetap terbaca.
	FlipX       bool   `json:"flip_x,omitempty"`
	FlipY       bool   `json:"flip_y,omitempty"`
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
	// Lanes: lane pool dan swimlane (revisi 54b); Size adalah bobot, jadi ubah ukuran kartu membagi ulang.
	Lanes []ShapeLane `json:"lanes,omitempty"`
	// Pasangan dokumen berformat (revisi 49); teks di atas tetap teks polosnya.
	Doc           *Node `json:"doc,omitempty"`
	NameDoc       *Node `json:"name_doc,omitempty"`
	AttributesDoc *Node `json:"attributes_doc,omitempty"`
	MethodsDoc    *Node `json:"methods_doc,omitempty"`
}

type ShapeLane struct {
	Label string `json:"label"`
	Size  int    `json:"size"`
}

const (
	maxLanes     = 20
	maxLaneLabel = 200
)

// cleanLanes: paling banyak 20 lane untuk bentuk ber-lane (tanpa lane berarti satu area), label satu baris, bobot
// 20 sampai 4000.
func cleanLanes(shape string, lanes []ShapeLane) ([]ShapeLane, error) {
	if !laneShapes[shape] {
		if len(lanes) > 0 {
			return nil, apperr.Invalid("Lane hanya untuk pool, swimlane, dan fragmen.")
		}
		return nil, nil
	}
	if len(lanes) > maxLanes {
		return nil, apperr.Invalid(fmt.Sprintf("Lane paling banyak %d.", maxLanes))
	}
	out := make([]ShapeLane, len(lanes))
	for i, l := range lanes {
		label, err := CleanText(l.Label, maxLaneLabel, "Label lane")
		if err != nil {
			return nil, err
		}
		if l.Size < 20 || l.Size > 4000 {
			return nil, apperr.Invalid("Ukuran lane harus 20 sampai 4000.")
		}
		out[i] = ShapeLane{Label: label, Size: l.Size}
	}
	return out, nil
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
	if in.Variant != "" && !slices.Contains(ShapeVariants[in.Shape], in.Variant) {
		return in, "", apperr.Invalid("Varian bentuk tidak dikenal untuk jenis bentuk ini.")
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
		out.Text, out.Doc = "", nil
		if out.Name, out.NameDoc, err = cleanRich(in.Name, in.NameDoc, MaxHeadingText, "Nama kelas", CleanText); err != nil {
			return in, "", err
		}
		if out.Attributes, out.AttributesDoc, err = cleanRich(in.Attributes, in.AttributesDoc, MaxShapeText, "Atribut kelas", cleanMultiline); err != nil {
			return in, "", err
		}
		if out.Methods, out.MethodsDoc, err = cleanRich(in.Methods, in.MethodsDoc, MaxShapeText, "Metode kelas", cleanMultiline); err != nil {
			return in, "", err
		}
		plain := strings.TrimSpace(strings.Join([]string{out.Name, out.Attributes, out.Methods}, "\n"))
		return out, plain, nil
	}
	out.Name, out.Attributes, out.Methods = "", "", ""
	out.NameDoc, out.AttributesDoc, out.MethodsDoc = nil, nil, nil
	if out.Text, out.Doc, err = cleanRich(in.Text, in.Doc, MaxShapeText, "Teks bentuk", cleanMultiline); err != nil {
		return in, "", err
	}
	if out.Lanes, err = cleanLanes(in.Shape, in.Lanes); err != nil {
		return in, "", err
	}
	// Label lane ikut pencarian.
	plain := out.Text
	for _, l := range out.Lanes {
		if l.Label != "" {
			plain = strings.TrimSpace(plain + "\n" + l.Label)
		}
	}
	return out, plain, nil
}

// defaultShape: isi awal bentuk baru dari palet. Jenis bentuk dikirim klien; ini hanya cadangan untuk kartu tanpa isi.
const defaultShape = `{"shape":"rect","rotation":0,"fill":"putih","stroke":"tinta","stroke_width":2,"stroke_style":"solid","text":"","align":"center","font_size":16}`
