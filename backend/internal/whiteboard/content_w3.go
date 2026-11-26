package whiteboard

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"nvm-internal/internal/apperr"
)

// Isi jenis kartu W3 (docs/whiteboard.md bagian 9). Sama dengan W1: setiap isi dibangun ulang dari struct, jadi
// field yang tidak dikenal tidak ikut tersimpan.

const (
	KindImage   = "image"
	KindFile    = "file"
	KindLink    = "link"
	KindColumn  = "column"
	KindDrawing = "drawing"
	KindTable   = "table"

	MaxCaption = 300

	// Gambar bebas (revisi A): objek tanpa bingkai di koordinat dunia. Titik disimpan sebagai bilangan bulat relatif
	// ke pojok kotak batas objek, dalam piksel dunia; kotak batasnya dihitung server dari titik. Batas jumlah menjaga
	// isi di bawah 64 KB; lewat dari itu klien memulai objek baru.
	DrawingSpace     = 4000
	MaxStrokes       = 300
	MaxStrokePoints  = 2000
	MaxDrawingPoints = 6000
	MaxTableRows     = 100
	MaxTableCols     = 20
	MaxCellText      = 500
	drawingInk       = "tinta"
)

// DrawingWidths: ketebalan pena yang diizinkan, dalam piksel ruang gambar.
var DrawingWidths = map[int]bool{2: true, 4: true, 8: true}

type CaptionContent struct {
	Caption string `json:"caption"`
}

type LinkContent struct {
	URL       string `json:"url"`
	PreviewID string `json:"preview_id"`
}

type ColumnContent struct {
	Title    string `json:"title"`
	TitleDoc *Node  `json:"title_doc,omitempty"`
}

type Stroke struct {
	Color  string `json:"color"`
	Width  int    `json:"width"`
	Points []int  `json:"points"`
}

type DrawingContent struct {
	Strokes []Stroke `json:"strokes"`
}

type TableContent struct {
	Title    string     `json:"title"`
	TitleDoc *Node      `json:"title_doc,omitempty"`
	Header   bool       `json:"header"`
	Rows     [][]string `json:"rows"`
	// Docs sejajar dengan Rows (revisi 49): dokumen sel yang berformat, null untuk sel teks polos. Tidak disimpan
	// bila tidak ada sel berformat.
	Docs [][]*Node `json:"docs,omitempty"`
}

func cleanCaption(raw json.RawMessage) (CaptionContent, string, error) {
	var in CaptionContent
	if err := decodeStrict(raw, &in); err != nil {
		return in, "", err
	}
	c, err := CleanText(in.Caption, MaxCaption, "Keterangan")
	return CaptionContent{Caption: c}, c, err
}

// CleanLinkURL hanya menerima http dan https untuk kartu tautan (mailto tidak punya halaman untuk dipratinjau).
func CleanLinkURL(v string) (string, error) {
	clean, err := CleanHref(v)
	if err != nil {
		return "", err
	}
	u, _ := url.Parse(clean)
	if s := strings.ToLower(u.Scheme); s != "http" && s != "https" {
		return "", apperr.Invalid("Kartu tautan hanya untuk alamat http:// atau https://.")
	}
	return clean, nil
}

func cleanLink(raw json.RawMessage) (LinkContent, string, error) {
	var in LinkContent
	if err := decodeStrict(raw, &in); err != nil {
		return in, "", err
	}
	u, err := CleanLinkURL(in.URL)
	if err != nil {
		return in, "", err
	}
	pid := strings.ToLower(strings.TrimSpace(in.PreviewID))
	if pid != "" && !uuidPattern.MatchString(pid) {
		return in, "", apperr.Invalid("Pratinjau tautan tidak sah.")
	}
	return LinkContent{URL: u, PreviewID: pid}, u, nil
}

func cleanColumn(raw json.RawMessage) (ColumnContent, string, error) {
	var in ColumnContent
	if err := decodeStrict(raw, &in); err != nil {
		return in, "", err
	}
	t, doc, err := cleanRich(in.Title, in.TitleDoc, MaxHeadingText, "Judul kolom", CleanText)
	return ColumnContent{Title: t, TitleDoc: doc}, t, err
}

// CleanDrawing memeriksa setiap goresan: warna dari daftar warna kartu atau tinta, ketebalan 2, 4, atau 8, jumlah
// koordinat genap, setiap titik di dalam ruang gambar, dan batas jumlah goresan serta titik.
func CleanDrawing(raw json.RawMessage) (DrawingContent, string, error) {
	var in DrawingContent
	if err := decodeStrict(raw, &in); err != nil {
		return in, "", err
	}
	if len(in.Strokes) == 0 {
		return in, "", apperr.Invalid("Gambar bebas kosong.")
	}
	if len(in.Strokes) > MaxStrokes {
		return in, "", apperr.Invalid(fmt.Sprintf("Gambar maksimal %d goresan. Hapus sebagian, atau mulai kartu gambar baru.", MaxStrokes))
	}
	out := DrawingContent{Strokes: make([]Stroke, 0, len(in.Strokes))}
	total := 0
	for _, s := range in.Strokes {
		if s.Color != drawingInk && !Colors[s.Color] {
			return in, "", apperr.Invalid("Warna pena tidak dikenal.")
		}
		if !DrawingWidths[s.Width] {
			return in, "", apperr.Invalid("Ketebalan pena harus 2, 4, atau 8.")
		}
		n := len(s.Points)
		if n < 2 || n%2 != 0 {
			return in, "", apperr.Invalid("Goresan gambar tidak lengkap.")
		}
		if n/2 > MaxStrokePoints {
			return in, "", apperr.Invalid(fmt.Sprintf("Satu goresan maksimal %d titik.", MaxStrokePoints))
		}
		for _, p := range s.Points {
			if p < 0 || p > DrawingSpace {
				return in, "", apperr.Invalid("Titik gambar di luar kartu.")
			}
		}
		total += n / 2
		if total > MaxDrawingPoints {
			return in, "", apperr.Invalid(fmt.Sprintf("Gambar maksimal %d titik. Hapus sebagian, atau mulai kartu gambar baru.", MaxDrawingPoints))
		}
		out.Strokes = append(out.Strokes, Stroke{Color: s.Color, Width: s.Width, Points: append([]int{}, s.Points...)})
	}
	return out, "", nil
}

// CleanTable memeriksa tabel teks polos: 1 sampai 100 baris, 1 sampai 20 kolom, semua baris selebar baris
// pertama, setiap sel maksimal 500 karakter.
func CleanTable(raw json.RawMessage) (TableContent, string, error) {
	var in TableContent
	if err := decodeStrict(raw, &in); err != nil {
		return in, "", err
	}
	if len(in.Rows) == 0 || len(in.Rows) > MaxTableRows {
		return in, "", apperr.Invalid(fmt.Sprintf("Tabel harus 1 sampai %d baris.", MaxTableRows))
	}
	cols := len(in.Rows[0])
	if cols == 0 || cols > MaxTableCols {
		return in, "", apperr.Invalid(fmt.Sprintf("Tabel harus 1 sampai %d kolom.", MaxTableCols))
	}
	title, titleDoc, err := cleanRich(in.Title, in.TitleDoc, MaxHeadingText, "Judul tabel", CleanText)
	if err != nil {
		return in, "", err
	}
	out := TableContent{Title: title, TitleDoc: titleDoc, Header: in.Header, Rows: make([][]string, 0, len(in.Rows))}
	docs := make([][]*Node, 0, len(in.Rows))
	anyDoc := false
	var plain strings.Builder
	if title != "" {
		plain.WriteString(title)
		plain.WriteString(" ")
	}
	for _, row := range in.Rows {
		if len(row) != cols {
			return in, "", apperr.Invalid("Semua baris tabel harus punya jumlah kolom yang sama.")
		}
		clean := make([]string, cols)
		rowDocs := make([]*Node, cols)
		for i, cell := range row {
			var d *Node
			if r := len(out.Rows); r < len(in.Docs) && i < len(in.Docs[r]) {
				d = in.Docs[r][i]
			}
			t, doc, err := cleanRich(cell, d, MaxCellText, "Isi sel", CleanText)
			if err != nil {
				return in, "", err
			}
			clean[i] = t
			rowDocs[i] = doc
			anyDoc = anyDoc || doc != nil
			if t != "" {
				plain.WriteString(t)
				plain.WriteString(" ")
			}
		}
		out.Rows = append(out.Rows, clean)
		docs = append(docs, rowDocs)
	}
	if anyDoc {
		out.Docs = docs
	}
	return out, strings.TrimSpace(plain.String()), nil
}

// TableOp mengubah tabel per sel, baris, atau kolom di server. Tabel memakai kunci lunak, jadi indeks tidak
// bergeser oleh orang lain selama satu orang mengedit.
type TableOp struct {
	Op    string `json:"op"`
	Row   int    `json:"row"`
	Col   int    `json:"col"`
	Index *int   `json:"index"`
	Text  string `json:"text"`
	On    bool   `json:"on"`
	// Doc: isi berformat untuk set dan title (revisi 49); tanpa Doc teksnya polos.
	Doc *Node `json:"doc"`
}

func applyTableOp(t *TableContent, op TableOp) error {
	rows := len(t.Rows)
	cols := 0
	if rows > 0 {
		cols = len(t.Rows[0])
	}
	at := func(n int) int {
		if op.Index == nil || *op.Index < 0 || *op.Index > n {
			return n
		}
		return *op.Index
	}
	// Docs disejajarkan dengan Rows dulu (data lama tanpa Docs, atau Docs yang lebih pendek), supaya tambah dan hapus
	// baris atau kolom menggeser format sel bersama teksnya.
	docs := make([][]*Node, rows)
	for r := range docs {
		docs[r] = make([]*Node, cols)
		if r < len(t.Docs) {
			copy(docs[r], t.Docs[r])
		}
	}
	t.Docs = docs
	switch op.Op {
	case "set":
		if op.Row < 0 || op.Row >= rows || op.Col < 0 || op.Col >= cols {
			return apperr.Invalid("Sel tabel tidak ditemukan. Muat ulang papan, lalu coba lagi.")
		}
		t.Rows[op.Row][op.Col] = op.Text
		t.Docs[op.Row][op.Col] = op.Doc
	case "add_row":
		if rows >= MaxTableRows {
			return apperr.Invalid(fmt.Sprintf("Tabel maksimal %d baris.", MaxTableRows))
		}
		i := at(rows)
		t.Rows = append(t.Rows[:i], append([][]string{make([]string, cols)}, t.Rows[i:]...)...)
		t.Docs = append(t.Docs[:i], append([][]*Node{make([]*Node, cols)}, t.Docs[i:]...)...)
	case "remove_row":
		if rows <= 1 {
			return apperr.Invalid("Tabel minimal satu baris.")
		}
		i := at(rows - 1)
		t.Rows = append(t.Rows[:i], t.Rows[i+1:]...)
		t.Docs = append(t.Docs[:i], t.Docs[i+1:]...)
	case "add_col":
		if cols >= MaxTableCols {
			return apperr.Invalid(fmt.Sprintf("Tabel maksimal %d kolom.", MaxTableCols))
		}
		i := at(cols)
		for r := range t.Rows {
			t.Rows[r] = append(t.Rows[r][:i], append([]string{""}, t.Rows[r][i:]...)...)
			t.Docs[r] = append(t.Docs[r][:i], append([]*Node{nil}, t.Docs[r][i:]...)...)
		}
	case "remove_col":
		if cols <= 1 {
			return apperr.Invalid("Tabel minimal satu kolom.")
		}
		i := at(cols - 1)
		for r := range t.Rows {
			t.Rows[r] = append(t.Rows[r][:i], t.Rows[r][i+1:]...)
			t.Docs[r] = append(t.Docs[r][:i], t.Docs[r][i+1:]...)
		}
	case "header":
		t.Header = op.On
	case "title":
		t.Title = op.Text
		t.TitleDoc = op.Doc
	default:
		return apperr.Invalid("Perubahan tabel tidak dikenal.")
	}
	// Tanpa sel berformat, Docs tidak disimpan (sama dengan CleanTable).
	for _, row := range t.Docs {
		for _, d := range row {
			if d != nil {
				return nil
			}
		}
	}
	t.Docs = nil
	return nil
}

// cleanW3 dipanggil CleanContent untuk jenis W3.
func cleanW3(kind string, raw json.RawMessage) (any, string, bool, error) {
	switch kind {
	case KindImage, KindFile:
		v, p, err := cleanCaption(raw)
		return v, p, true, err
	case KindLink:
		v, p, err := cleanLink(raw)
		return v, p, true, err
	case KindColumn:
		v, p, err := cleanColumn(raw)
		return v, p, true, err
	case KindDrawing:
		v, p, err := CleanDrawing(raw)
		return v, p, true, err
	case KindTable:
		v, p, err := CleanTable(raw)
		return v, p, true, err
	case KindShape:
		v, p, err := CleanShape(raw)
		return v, p, true, err
	}
	return nil, "", false, nil
}
