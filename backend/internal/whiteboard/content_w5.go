package whiteboard

import (
	"encoding/json"
	"fmt"
	"math"
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
		// ERD (revisi 54e)
		"erd_entity": true, "erd_weak_entity": true, "erd_attribute": true, "erd_key_attribute": true,
		"erd_relationship": true,
		// Tabel ERD (revisi 57a): nama tabel dan baris kolom dengan tipe, kunci, dan nullable.
		"erd_table": true,
		// Lainnya (revisi 54f): flowchart lanjutan, arsitektur, wireframe, mind map, dan bentuk presentasi
		"fc_manual_input": true, "fc_predefined": true, "fc_offpage": true, "fc_delay": true, "fc_preparation": true,
		"fc_merge": true, "fc_sort": true,
		"arch_server": true, "arch_database": true, "arch_cloud": true, "arch_user": true, "arch_laptop": true,
		"arch_phone": true, "arch_firewall": true, "arch_component": true, "arch_node": true,
		"wf_browser": true, "wf_phone": true, "wf_button": true, "wf_input": true, "wf_checkbox": true,
		"wf_dropdown": true, "wf_image": true,
		"mm_central": true, "mm_topic": true,
		"callout": true, "chevron": true, "funnel": true, "timeline": true,
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
		// Kelas UML (revisi 54e): kelas lama tanpa varian tetap kelas biasa.
		"uml_class":   {"class", "abstract", "interface", "enum"},
		"wf_checkbox": {"unchecked", "checked"},
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
	// Rows: kolom tabel ERD (revisi 57a); nama tabel di Name.
	Rows []ShapeRow `json:"rows,omitempty"`
	// Image: cara gambar bingkai ditampilkan (revisi 57b); berkasnya sendiri di wb_cards.file_id, bukan di sini.
	Image *ShapeImage `json:"image,omitempty"`
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

// ShapeRow: satu kolom tabel ERD (revisi 57a). ID stabil dibuat klien, supaya garis yang menempel ke baris tetap di
// baris itu saat baris diurutkan ulang.
type ShapeRow struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Keys     []string `json:"keys"`
	Nullable bool     `json:"nullable"`
}

const (
	MaxErdRows    = 60
	maxTableName  = 100
	maxColumnName = 100
	maxColumnType = 60
)

// FrameShapes: bentuk yang bisa diisi gambar (revisi 57b).
var FrameShapes = map[string]bool{"wf_phone": true, "wf_browser": true, "wf_image": true}

// ShapeImage: cover mengisi penuh bingkai (terpotong), contain memuat utuh; X dan Y posisi gambar 0 sampai 100 persen
// (seperti object-position), untuk menggeser bagian yang terlihat.
type ShapeImage struct {
	Fit string  `json:"fit"`
	X   float64 `json:"x"`
	Y   float64 `json:"y"`
}

func cleanImage(shape string, im *ShapeImage) (*ShapeImage, error) {
	if im == nil {
		return nil, nil
	}
	if !FrameShapes[shape] {
		return nil, apperr.Invalid("Pengaturan gambar hanya untuk bingkai ponsel, jendela browser, dan placeholder gambar.")
	}
	if im.Fit != "cover" && im.Fit != "contain" {
		return nil, apperr.Invalid("Cara isi gambar harus penuh atau muat utuh.")
	}
	if !finite(im.X) || !finite(im.Y) || im.X < 0 || im.X > 100 || im.Y < 0 || im.Y > 100 {
		return nil, apperr.Invalid("Posisi gambar harus 0 sampai 100 persen.")
	}
	return &ShapeImage{Fit: im.Fit, X: math.Round(im.X*10) / 10, Y: math.Round(im.Y*10) / 10}, nil
}

// shapeOf: jenis bentuk dari isi kartu tersimpan (kosong bila bukan bentuk).
func shapeOf(content []byte) string {
	var c struct {
		Shape string `json:"shape"`
	}
	_ = json.Unmarshal(content, &c)
	return c.Shape
}

// tableKeys: urutan kunci tetap (PK, FK, UK), jadi isi yang sama selalu tersimpan sama.
var tableKeys = []string{"pk", "fk", "uk"}

// ValidRowID: id baris dan rujukan baris di ujung garis, huruf kecil dan angka saja.
func ValidRowID(id string) bool {
	if id == "" || len(id) > 16 {
		return false
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func cleanRows(rows []ShapeRow) ([]ShapeRow, error) {
	if len(rows) > MaxErdRows {
		return nil, apperr.Invalid(fmt.Sprintf("Tabel paling banyak %d kolom.", MaxErdRows))
	}
	out := make([]ShapeRow, len(rows))
	seen := map[string]bool{}
	for i, r := range rows {
		if !ValidRowID(r.ID) || seen[r.ID] {
			return nil, apperr.Invalid("Id kolom tabel tidak sah atau ganda.")
		}
		seen[r.ID] = true
		name, err := CleanText(r.Name, maxColumnName, "Nama kolom")
		if err != nil {
			return nil, err
		}
		typ, err := CleanText(r.Type, maxColumnType, "Tipe data kolom")
		if err != nil {
			return nil, err
		}
		keys := []string{}
		for _, k := range r.Keys {
			if !slices.Contains(tableKeys, k) {
				return nil, apperr.Invalid("Kunci kolom harus PK, FK, atau UK.")
			}
		}
		for _, k := range tableKeys {
			if slices.Contains(r.Keys, k) {
				keys = append(keys, k)
			}
		}
		out[i] = ShapeRow{ID: r.ID, Name: name, Type: typ, Keys: keys, Nullable: r.Nullable}
	}
	return out, nil
}

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
	if in.Shape == "erd_table" {
		out.Text, out.Doc, out.Lanes = "", nil, nil
		out.Attributes, out.Methods, out.NameDoc, out.AttributesDoc, out.MethodsDoc = "", "", nil, nil, nil
		if out.Name, err = CleanText(in.Name, maxTableName, "Nama tabel"); err != nil {
			return in, "", err
		}
		if out.Rows, err = cleanRows(in.Rows); err != nil {
			return in, "", err
		}
		parts := []string{out.Name}
		for _, r := range out.Rows {
			parts = append(parts, strings.TrimSpace(r.Name+" "+r.Type))
		}
		return out, strings.TrimSpace(strings.Join(parts, "\n")), nil
	}
	if len(in.Rows) > 0 {
		return in, "", apperr.Invalid("Kolom hanya untuk bentuk tabel.")
	}
	if out.Image, err = cleanImage(in.Shape, in.Image); err != nil {
		return in, "", err
	}
	out.Rows = nil
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
