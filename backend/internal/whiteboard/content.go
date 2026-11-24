package whiteboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"nvm-internal/internal/apperr"
)

// Isi kartu disimpan sebagai struktur, bukan HTML (docs/whiteboard.md bagian 5). Setiap isi dibangun ulang dari
// daftar putih di sini, jadi field, node, mark, dan atribut yang tidak dikenal dibuang, bukan disimpan.

const (
	MaxContentBytes = 64 << 10
	MaxDocDepth     = 8
	maxDocNodes     = 5000
	MaxHeadingText  = 200
	MaxChecklist    = 200
	MaxItemText     = 500
	MaxHref         = 2048
)

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Node adalah satu node dokumen ProseMirror dalam bentuk yang diizinkan.
type Node struct {
	Type    string         `json:"type"`
	Attrs   map[string]any `json:"attrs,omitempty"`
	Content []Node         `json:"content,omitempty"`
	Text    string         `json:"text,omitempty"`
	Marks   []Mark         `json:"marks,omitempty"`
}

type Mark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

type NoteContent struct {
	Doc Node `json:"doc"`
}

type HeadingContent struct {
	Text  string `json:"text"`
	Level int    `json:"level"`
}

type ChecklistItem struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type ChecklistContent struct {
	Title string          `json:"title"`
	Items []ChecklistItem `json:"items"`
}

// blockChildren: node blok yang boleh berada di bawah setiap node. Teks hanya di paragraf dan heading.
var blockChildren = map[string]map[string]bool{
	"doc":         {"paragraph": true, "heading": true, "blockquote": true, "bulletList": true, "orderedList": true},
	"blockquote":  {"paragraph": true, "heading": true, "bulletList": true, "orderedList": true},
	"bulletList":  {"listItem": true},
	"orderedList": {"listItem": true},
	"listItem":    {"paragraph": true, "bulletList": true, "orderedList": true},
}

var inlineParents = map[string]bool{"paragraph": true, "heading": true}

var allowedMarks = map[string]bool{"bold": true, "italic": true, "underline": true, "strike": true, "code": true, "link": true}

// CleanText merapikan teks satu baris dari pengguna: karakter kontrol dibuang dan spasi di ujung dipangkas.
func CleanText(v string, max int, label string) (string, error) {
	v = strings.TrimSpace(stripControl(v, false))
	if utf8.RuneCountInString(v) > max {
		return "", apperr.Invalid(fmt.Sprintf("%s maksimal %d karakter.", label, max))
	}
	return v, nil
}

func stripControl(v string, keepTab bool) string {
	if !utf8.ValidString(v) {
		v = strings.ToValidUTF8(v, "")
	}
	return strings.Map(func(r rune) rune {
		if r == '\t' && keepTab {
			return r
		}
		if unicode.IsControl(r) || r == ' ' || r == ' ' {
			return -1
		}
		return r
	}, v)
}

// CleanHref menerima hanya http, https, dan mailto. javascript:, data:, dan skema lain ditolak, bukan dibuang,
// supaya pengguna tahu tautannya tidak tersimpan.
func CleanHref(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > MaxHref || strings.ContainsAny(v, "\x00\r\n\t") {
		return "", apperr.Invalid("Tautan tidak sah. Pakai alamat yang diawali http://, https://, atau mailto:.")
	}
	u, err := url.Parse(v)
	if err != nil {
		return "", apperr.Invalid("Tautan tidak sah. Pakai alamat yang diawali http://, https://, atau mailto:.")
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		if u.Host == "" {
			return "", apperr.Invalid("Tautan tidak sah. Pakai alamat yang diawali http://, https://, atau mailto:.")
		}
	case "mailto":
	default:
		return "", apperr.Invalid("Tautan hanya boleh http, https, atau mailto.")
	}
	return v, nil
}

type docCleaner struct {
	nodes int
	plain strings.Builder
}

func (c *docCleaner) block(n Node, parent string, depth int) (Node, error) {
	c.nodes++
	if c.nodes > maxDocNodes {
		return Node{}, apperr.Invalid("Isi catatan terlalu panjang.")
	}
	if depth > MaxDocDepth {
		return Node{}, apperr.Invalid(fmt.Sprintf("Isi catatan terlalu bertingkat (maksimal %d tingkat).", MaxDocDepth))
	}
	if !blockChildren[parent][n.Type] {
		return Node{}, apperr.Invalid("Isi catatan memuat format yang tidak didukung.")
	}
	out := Node{Type: n.Type}
	switch n.Type {
	case "heading":
		level := intAttr(n.Attrs, "level", 1)
		if level < 1 || level > 3 {
			level = 1
		}
		out.Attrs = map[string]any{"level": level}
	case "orderedList":
		start := intAttr(n.Attrs, "start", 1)
		if start < 1 || start > 10000 {
			start = 1
		}
		if start != 1 {
			out.Attrs = map[string]any{"start": start}
		}
	}
	if inlineParents[n.Type] {
		for _, ch := range n.Content {
			in, keep, err := c.inline(ch)
			if err != nil {
				return Node{}, err
			}
			if keep {
				out.Content = append(out.Content, in)
			}
		}
		c.plain.WriteString("\n")
		return out, nil
	}
	for _, ch := range n.Content {
		b, err := c.block(ch, n.Type, depth+1)
		if err != nil {
			return Node{}, err
		}
		out.Content = append(out.Content, b)
	}
	// Daftar dan butir daftar tanpa isi tidak sah di skema ProseMirror; diganti struktur kosong yang sah.
	switch n.Type {
	case "bulletList", "orderedList":
		if len(out.Content) == 0 {
			out.Content = []Node{{Type: "listItem", Content: []Node{{Type: "paragraph"}}}}
		}
	case "listItem", "blockquote":
		if len(out.Content) == 0 || out.Content[0].Type != "paragraph" && n.Type == "listItem" {
			out.Content = append([]Node{{Type: "paragraph"}}, out.Content...)
		}
	}
	return out, nil
}

func (c *docCleaner) inline(n Node) (Node, bool, error) {
	c.nodes++
	if c.nodes > maxDocNodes {
		return Node{}, false, apperr.Invalid("Isi catatan terlalu panjang.")
	}
	switch n.Type {
	case "hardBreak":
		c.plain.WriteString("\n")
		return Node{Type: "hardBreak"}, true, nil
	case "text":
	default:
		return Node{}, false, apperr.Invalid("Isi catatan memuat format yang tidak didukung.")
	}
	text := stripControl(n.Text, true)
	if text == "" {
		return Node{}, false, nil
	}
	out := Node{Type: "text", Text: text}
	seen := map[string]bool{}
	for _, m := range n.Marks {
		if !allowedMarks[m.Type] {
			return Node{}, false, apperr.Invalid("Isi catatan memuat format yang tidak didukung.")
		}
		if seen[m.Type] {
			continue
		}
		seen[m.Type] = true
		mark := Mark{Type: m.Type}
		if m.Type == "link" {
			href, _ := m.Attrs["href"].(string)
			clean, err := CleanHref(href)
			if err != nil {
				return Node{}, false, err
			}
			mark.Attrs = map[string]any{"href": clean}
		}
		out.Marks = append(out.Marks, mark)
	}
	c.plain.WriteString(text)
	return out, true, nil
}

func intAttr(attrs map[string]any, key string, def int) int {
	switch v := attrs[key].(type) {
	case float64:
		if v == float64(int(v)) {
			return int(v)
		}
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return int(n)
		}
	}
	return def
}

// CleanNote membangun ulang dokumen catatan dari daftar putih dan mengembalikan teks polosnya.
func CleanNote(raw json.RawMessage) (NoteContent, string, error) {
	var in NoteContent
	if err := decodeStrict(raw, &in); err != nil {
		return in, "", err
	}
	if in.Doc.Type != "doc" {
		return in, "", apperr.Invalid("Isi catatan tidak terbaca.")
	}
	c := &docCleaner{}
	out := Node{Type: "doc"}
	for _, ch := range in.Doc.Content {
		b, err := c.block(ch, "doc", 1)
		if err != nil {
			return in, "", err
		}
		out.Content = append(out.Content, b)
	}
	if len(out.Content) == 0 {
		out.Content = []Node{{Type: "paragraph"}}
	}
	return NoteContent{Doc: out}, strings.TrimSpace(c.plain.String()), nil
}

func CleanHeading(raw json.RawMessage) (HeadingContent, string, error) {
	var in HeadingContent
	if err := decodeStrict(raw, &in); err != nil {
		return in, "", err
	}
	text, err := CleanText(in.Text, MaxHeadingText, "Judul")
	if err != nil {
		return in, "", err
	}
	level := in.Level
	if level != 2 {
		level = 1
	}
	return HeadingContent{Text: text, Level: level}, text, nil
}

func CleanChecklist(raw json.RawMessage) (ChecklistContent, string, error) {
	var in ChecklistContent
	if err := decodeStrict(raw, &in); err != nil {
		return in, "", err
	}
	title, err := CleanText(in.Title, MaxHeadingText, "Judul checklist")
	if err != nil {
		return in, "", err
	}
	if len(in.Items) > MaxChecklist {
		return in, "", apperr.Invalid(fmt.Sprintf("Checklist maksimal %d butir.", MaxChecklist))
	}
	out := ChecklistContent{Title: title, Items: make([]ChecklistItem, 0, len(in.Items))}
	seen := map[string]bool{}
	var plain strings.Builder
	plain.WriteString(title)
	for _, it := range in.Items {
		id := strings.ToLower(strings.TrimSpace(it.ID))
		if !uuidPattern.MatchString(id) || seen[id] {
			return in, "", apperr.Invalid("Butir checklist tidak sah. Muat ulang papan, lalu coba lagi.")
		}
		seen[id] = true
		text, err := CleanText(it.Text, MaxItemText, "Butir checklist")
		if err != nil {
			return in, "", err
		}
		out.Items = append(out.Items, ChecklistItem{ID: id, Text: text, Done: it.Done})
		plain.WriteString("\n")
		plain.WriteString(text)
	}
	return out, strings.TrimSpace(plain.String()), nil
}

// CleanContent memilih pembersih sesuai jenis kartu dan mengembalikan isi kanonik dalam JSON beserta teks polosnya.
func CleanContent(kind string, raw json.RawMessage) ([]byte, string, error) {
	if len(raw) > MaxContentBytes {
		return nil, "", apperr.Invalid("Isi kartu terlalu besar (maksimal 64 KB).")
	}
	var (
		v     any
		plain string
		err   error
	)
	switch kind {
	case KindNote:
		v, plain, err = CleanNote(raw)
	case KindHeading:
		v, plain, err = CleanHeading(raw)
	case KindChecklist:
		v, plain, err = CleanChecklist(raw)
	default:
		return nil, "", apperr.Invalid("Jenis kartu tidak dikenal.")
	}
	if err != nil {
		return nil, "", err
	}
	out, err := json.Marshal(v)
	if err != nil {
		return nil, "", err
	}
	if len(out) > MaxContentBytes {
		return nil, "", apperr.Invalid("Isi kartu terlalu besar (maksimal 64 KB).")
	}
	return out, plain, nil
}

// decodeStrict menolak JSON rusak. Field yang tidak dikenal tidak ditolak, tetapi tidak ikut tersimpan karena isi
// dibangun ulang dari struct.
func decodeStrict(raw json.RawMessage, v any) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return apperr.Invalid("Isi kartu kosong.")
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return apperr.Invalid("Isi kartu tidak terbaca.")
	}
	return nil
}

// DefaultContent adalah isi awal kartu baru tanpa isi.
func DefaultContent(kind string) json.RawMessage {
	switch kind {
	case KindNote:
		return json.RawMessage(`{"doc":{"type":"doc","content":[{"type":"paragraph"}]}}`)
	case KindHeading:
		return json.RawMessage(`{"text":"","level":1}`)
	default:
		return json.RawMessage(`{"title":"","items":[]}`)
	}
}
