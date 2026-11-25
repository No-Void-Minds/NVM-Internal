package whiteboard

import (
	"context"
	"encoding/json"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/db/sqlc"
)

// Gambar bebas tanpa kartu (revisi A): objek berjenis drawing dirender tanpa bingkai di atas kartu. Klien mengirim
// titik koordinat dunia; server menghitung kotak batasnya dan menyimpan titik relatif ke pojok kotak itu.

// DrawingOp mengubah objek gambar: add menambah satu goresan (titik koordinat dunia), remove membuang goresan
// ke-Index, color mengganti warna semua goresan.
type DrawingOp struct {
	Op     string  `json:"op"`
	Stroke *Stroke `json:"stroke"`
	Index  *int    `json:"index"`
	Color  string  `json:"color"`
}

// placeDrawing menghitung kotak batas goresan berkoordinat dunia (ditambah setengah ketebalan pena), lalu mengubah
// titiknya menjadi relatif ke pojok kotak itu. Kotak paling kecil minSize supaya objek kecil tetap bisa dipilih.
func placeDrawing(strokes []Stroke) (CardInput, error) {
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, s := range strokes {
		if len(s.Points) < 2 || len(s.Points)%2 != 0 {
			return CardInput{}, apperr.Invalid("Goresan gambar tidak lengkap.")
		}
		pad := float64(s.Width)/2 + 2
		for i := 0; i+1 < len(s.Points); i += 2 {
			x, y := float64(s.Points[i]), float64(s.Points[i+1])
			if math.Abs(x) > maxCoord-DrawingSpace || math.Abs(y) > maxCoord-DrawingSpace {
				return CardInput{}, apperr.Invalid("Titik gambar di luar papan.")
			}
			minX, maxX = math.Min(minX, x-pad), math.Max(maxX, x+pad)
			minY, maxY = math.Min(minY, y-pad), math.Max(maxY, y+pad)
		}
	}
	if math.IsInf(minX, 1) {
		return CardInput{}, apperr.Invalid("Gambar bebas kosong.")
	}
	if maxX-minX > DrawingSpace || maxY-minY > DrawingSpace {
		return CardInput{}, apperr.Invalid("Gambar ini sudah terlalu lebar. Goresan berikutnya dimulai sebagai gambar baru.")
	}
	grow := func(lo, hi float64) (float64, float64) {
		if hi-lo >= minSize {
			return math.Floor(lo), math.Ceil(hi)
		}
		start := math.Floor((lo+hi)/2 - minSize/2)
		return start, start + minSize
	}
	minX, maxX = grow(minX, maxX)
	minY, maxY = grow(minY, maxY)
	rel := DrawingContent{Strokes: make([]Stroke, 0, len(strokes))}
	for _, s := range strokes {
		pts := make([]int, len(s.Points))
		for i := 0; i+1 < len(s.Points); i += 2 {
			pts[i], pts[i+1] = s.Points[i]-int(minX), s.Points[i+1]-int(minY)
		}
		rel.Strokes = append(rel.Strokes, Stroke{Color: s.Color, Width: s.Width, Points: pts})
	}
	raw, err := json.Marshal(rel)
	return CardInput{Kind: KindDrawing, X: minX, Y: minY, W: maxX - minX, H: maxY - minY, Content: raw}, err
}

func absolute(c DrawingContent, x, y float64) []Stroke {
	out := make([]Stroke, 0, len(c.Strokes))
	for _, s := range c.Strokes {
		pts := make([]int, len(s.Points))
		for i := 0; i+1 < len(s.Points); i += 2 {
			pts[i], pts[i+1] = s.Points[i]+int(x), s.Points[i+1]+int(y)
		}
		out = append(out, Stroke{Color: s.Color, Width: s.Width, Points: pts})
	}
	return out
}

// CreateDrawing membuat objek gambar dari goresan pertama sebuah sesi pena.
func (s *Service) CreateDrawing(ctx context.Context, actor Actor, boardText, id string, stroke Stroke) (Card, error) {
	in, err := placeDrawing([]Stroke{stroke})
	if err != nil {
		return Card{}, err
	}
	in.ID = id
	return s.CreateCard(ctx, actor, boardText, in)
}

// Drawing menerapkan satu operasi ke objek gambar. Kunci lunak dicek di handler lewat hub.
func (s *Service) Drawing(ctx context.Context, actor Actor, boardText, cardText string, op DrawingOp) (Card, error) {
	boardID, err := s.boardID(ctx, actor, boardText)
	if err != nil {
		return Card{}, err
	}
	cardID, err := ParseID(cardText)
	if err != nil {
		return Card{}, err
	}
	var out Card
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		seq, err := q.BumpWbBoardSeq(ctx, boardID)
		if err != nil {
			return notFound(err)
		}
		// Kunci baris dulu supaya dua goresan bersamaan tidak saling menimpa.
		if _, err := q.GetWbCardForUpdate(ctx, sqlc.GetWbCardForUpdateParams{ID: cardID, BoardID: boardID}); err != nil {
			return notFound(err)
		}
		cur, err := getCard(ctx, q, cardID)
		if err != nil {
			return err
		}
		if cur.Kind != KindDrawing {
			return apperr.Invalid("Objek ini bukan gambar bebas.")
		}
		var c DrawingContent
		if err := json.Unmarshal(cur.Content, &c); err != nil {
			return err
		}
		strokes := absolute(c, cur.X, cur.Y)
		switch op.Op {
		case "add":
			if op.Stroke == nil {
				return apperr.Invalid("Goresan kosong.")
			}
			strokes = append(strokes, *op.Stroke)
		case "remove":
			if op.Index == nil || *op.Index < 0 || *op.Index >= len(strokes) {
				return apperr.Invalid("Goresan tidak ditemukan. Muat ulang papan, lalu coba lagi.")
			}
			if len(strokes) == 1 {
				return apperr.Invalid("Ini goresan terakhir. Hapus gambarnya ke Tempat sampah.")
			}
			strokes = append(strokes[:*op.Index], strokes[*op.Index+1:]...)
		case "color":
			for i := range strokes {
				strokes[i].Color = op.Color
			}
		default:
			return apperr.Invalid("Perubahan gambar tidak dikenal.")
		}
		in, err := placeDrawing(strokes)
		if err != nil {
			return err
		}
		content, plain, err := CleanContent(KindDrawing, in.Content)
		if err != nil {
			return err
		}
		if err := q.PatchWbCard(ctx, sqlc.PatchWbCardParams{
			X: f8(&in.X), Y: f8(&in.Y), W: f8(&in.W), H: f8(&in.H), Content: content,
			Plain: pgtype.Text{String: plain, Valid: true}, Seq: seq, ActorID: actor.ID, ID: cardID, BoardID: boardID,
		}); err != nil {
			return err
		}
		if out, err = getCard(ctx, q, cardID); err != nil {
			return err
		}
		return record(ctx, q, boardID, actor.ID, "card_edited", cardID, map[string]any{"kind": KindDrawing})
	})
	return out, err
}
