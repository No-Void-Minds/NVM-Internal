package whiteboard

import (
	"bytes"
	"encoding/json"
	"math"

	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/apperr"
)

// Gaya kartu, grup, dan kunci posisi (revisi 50). Gaya hanya nama dari daftar putih; klien memetakannya ke kelas CSS,
// jadi tidak ada CSS dari pengguna yang sampai ke halaman. Nilai kosong berarti tampilan bawaan dan tidak disimpan.

// CardStyle: bg "transparan" menghapus latar kartu (warna kartu tetap tersimpan di kolom color). Border warna memakai
// warna pena (minimal 3:1 terhadap kanvas). Bayangan bawaan halus; "tanpa" dan "tegas" pilihan lain, dipakai hanya
// untuk hierarki (kartu yang perlu menonjol), bukan dekorasi. Opacity 25 sampai 100 persen, kelipatan 5; 0 berarti bawaan (100).
type CardStyle struct {
	Bg          string `json:"bg,omitempty"`
	Border      string `json:"border,omitempty"`
	BorderColor string `json:"border_color,omitempty"`
	BorderStyle string `json:"border_style,omitempty"`
	Radius      string `json:"radius,omitempty"`
	Shadow      string `json:"shadow,omitempty"`
	Opacity     int    `json:"opacity,omitempty"`
	Padding     string `json:"padding,omitempty"`
}

var (
	styleBg          = map[string]bool{"": true, "transparan": true}
	styleBorder      = map[string]bool{"": true, "tanpa": true, "tipis": true, "sedang": true, "tebal": true}
	styleBorderStyle = map[string]bool{"": true, "putus": true, "titik": true}
	styleRadius      = map[string]bool{"": true, "siku": true, "kecil": true, "sedang": true, "bulat": true}
	styleShadow      = map[string]bool{"": true, "tanpa": true, "halus": true, "tegas": true}
	stylePadding     = map[string]bool{"": true, "rapat": true, "normal": true, "lega": true}
)

// MaxGroupDepth: grup bersarang paling dalam delapan tingkat (CHECK cardinality di migrasi 000026).
const MaxGroupDepth = 8

// CleanStyle memvalidasi gaya kartu. Kunci di luar daftar dan nilai di luar daftar ditolak, bukan dibuang diam-diam.
// Hasilnya selalu objek JSON (paling sedikit {}), dengan nilai bawaan dibuang.
func CleanStyle(raw json.RawMessage) ([]byte, error) {
	if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return []byte(`{}`), nil
	}
	var in CardStyle
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return nil, apperr.Invalid("Gaya kartu tidak terbaca atau memuat pengaturan yang tidak dikenal.")
	}
	switch {
	case !styleBg[in.Bg]:
		return nil, apperr.Invalid("Latar kartu tidak dikenal.")
	case !styleBorder[in.Border]:
		return nil, apperr.Invalid("Tebal garis tepi tidak dikenal.")
	case in.BorderColor != "" && !shapeStrokes[in.BorderColor]:
		return nil, apperr.Invalid("Warna garis tepi tidak dikenal.")
	case !styleBorderStyle[in.BorderStyle]:
		return nil, apperr.Invalid("Gaya garis tepi tidak dikenal.")
	case !styleRadius[in.Radius]:
		return nil, apperr.Invalid("Sudut kartu tidak dikenal.")
	case !styleShadow[in.Shadow]:
		return nil, apperr.Invalid("Bayangan kartu tidak dikenal.")
	case !stylePadding[in.Padding]:
		return nil, apperr.Invalid("Jarak dalam kartu tidak dikenal.")
	case in.Opacity != 0 && (in.Opacity < 25 || in.Opacity > 100 || in.Opacity%5 != 0):
		return nil, apperr.Invalid("Transparansi kartu harus 25 sampai 100 persen, kelipatan 5.")
	}
	// Nilai yang sama dengan bawaan tidak disimpan, supaya gaya yang sama selalu tersimpan dengan isi yang sama.
	if in.Opacity == 100 {
		in.Opacity = 0
	}
	if in.Radius == "kecil" {
		in.Radius = ""
	}
	if in.Border == "tipis" {
		in.Border = ""
	}
	if in.Shadow == "halus" {
		in.Shadow = ""
	}
	if in.Padding == "normal" {
		in.Padding = ""
	}
	if in.Border == "tanpa" {
		in.BorderColor, in.BorderStyle = "", ""
	}
	return json.Marshal(in)
}

// cleanGroups memvalidasi jalur grup kartu (dari terluar ke terdalam). Id yang sama dua kali berarti grup berisi
// dirinya sendiri (siklus).
func cleanGroups(ids []string) ([]pgtype.UUID, error) {
	if len(ids) > MaxGroupDepth {
		return nil, apperr.Invalid("Grup bersarang paling dalam 8 tingkat.")
	}
	out := make([]pgtype.UUID, 0, len(ids))
	seen := map[[16]byte]bool{}
	for _, t := range ids {
		id, err := ParseID(t)
		if err != nil {
			return nil, apperr.Invalid("Id grup tidak sah.")
		}
		if seen[id.Bytes] {
			return nil, apperr.Invalid("Grup tidak boleh berisi dirinya sendiri.")
		}
		seen[id.Bytes] = true
		out = append(out, id)
	}
	return out, nil
}

func groupStrings(ids []pgtype.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = idString(id)
	}
	return out
}

// Crop: bagian gambar asli yang ditampilkan kartu gambar, dalam pecahan 0 sampai 1 dari lebar dan tinggi berkas.
// Berkas aslinya tidak diubah; potongan dan posisi zoom di bingkai hanya parameter tampilan dan ekspor.
type Crop struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

func cleanCrop(c *Crop) (*Crop, error) {
	if c == nil {
		return nil, nil
	}
	for _, v := range []float64{c.X, c.Y, c.W, c.H} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, apperr.Invalid("Potongan gambar tidak sah.")
		}
	}
	const eps = 1e-6
	if c.X < 0 || c.Y < 0 || c.W < 0.02 || c.H < 0.02 || c.X+c.W > 1+eps || c.Y+c.H > 1+eps {
		return nil, apperr.Invalid("Potongan gambar harus di dalam gambar dan minimal 2% dari sisinya.")
	}
	if c.X < eps && c.Y < eps && c.W > 1-eps && c.H > 1-eps {
		return nil, nil
	}
	return &Crop{X: c.X, Y: c.Y, W: math.Min(c.W, 1-c.X), H: math.Min(c.H, 1-c.Y)}, nil
}
