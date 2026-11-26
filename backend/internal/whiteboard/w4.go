package whiteboard

import (
	"context"
	"fmt"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/db/sqlc"
)

// W4 (docs/whiteboard.md bagian 9): template papan dan pencarian isi kartu.

// MaxTemplateCards membatasi satu salinan template, supaya satu permintaan tidak menahan transaksi terlalu lama.
const MaxTemplateCards = 2000

type TemplateFlag struct {
	ID         string `json:"id"`
	IsTemplate bool   `json:"is_template"`
}

// SetTemplate menandai atau melepas papan sebagai template. Papan akar ditolak.
func (s *Service) SetTemplate(ctx context.Context, actor Actor, idText string, on bool) (TemplateFlag, error) {
	id, err := s.boardID(ctx, actor, idText)
	if err != nil {
		return TemplateFlag{}, err
	}
	var out TemplateFlag
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		b, err := q.GetWbBoardForUpdate(ctx, id)
		if err != nil {
			return notFound(err)
		}
		if b.IsRoot {
			return apperr.Invalid("Papan akar tidak bisa dijadikan template. Buat papan di dalamnya, lalu jadikan template.")
		}
		r, err := q.SetWbTemplate(ctx, sqlc.SetWbTemplateParams{IsTemplate: on, ActorID: actor.ID, ID: id})
		if err != nil {
			return notFound(err)
		}
		out = TemplateFlag{ID: idString(r.ID), IsTemplate: r.IsTemplate}
		return record(ctx, q, id, actor.ID, "board_template", pgtype.UUID{}, map[string]any{"on": on})
	})
	return out, err
}

type TemplateInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Path  string `json:"path"`
	Cards int64  `json:"cards"`
}

func (s *Service) Templates(ctx context.Context) ([]TemplateInfo, error) {
	q := sqlc.New(s.pool)
	rows, err := q.ListWbTemplates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]TemplateInfo, 0, len(rows))
	for _, r := range rows {
		path, err := crumbPath(ctx, q, r.ParentID)
		if err != nil {
			return nil, err
		}
		out = append(out, TemplateInfo{ID: idString(r.ID), Title: r.Title, Path: path, Cards: r.Cards})
	}
	return out, nil
}

func crumbPath(ctx context.Context, q *sqlc.Queries, id pgtype.UUID) (string, error) {
	crumbs, err := ancestors(ctx, q, id)
	if err != nil {
		return "", err
	}
	names := make([]string, len(crumbs))
	for i, c := range crumbs {
		names[i] = c.Title
	}
	return strings.Join(names, " / "), nil
}

type TemplateInput struct {
	TemplateID string  `json:"template_id"`
	ID         string  `json:"id"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	W          float64 `json:"w"`
	H          float64 `json:"h"`
}

// FromTemplate membuat kartu papan baru di papan ini berisi salinan template beserta semua papan turunannya:
// kartu, isi kolom, dan garis ikut; komentar dan riwayat tidak. Kartu gambar dan berkas memakai berkas yang sama
// (file_id), seperti salin tempel, jadi tidak ada byte yang disalin; berkas baru dibuang job harian setelah tidak
// ada kartu yang memakainya. Salinan bukan template.
func (s *Service) FromTemplate(ctx context.Context, actor Actor, boardText string, in TemplateInput) (Card, error) {
	boardID, err := s.boardID(ctx, actor, boardText)
	if err != nil {
		return Card{}, err
	}
	tplID, err := ParseID(in.TemplateID)
	if err != nil {
		return Card{}, apperr.Invalid("Template tidak sah.")
	}
	for _, e := range []error{checkCoord(in.X, "Posisi"), checkCoord(in.Y, "Posisi"), checkSize(in.W), checkSize(in.H)} {
		if e != nil {
			return Card{}, e
		}
	}
	var cardID pgtype.UUID
	if in.ID != "" {
		if cardID, err = ParseID(in.ID); err != nil {
			return Card{}, apperr.Invalid("Id kartu tidak sah.")
		}
	} else if err := cardID.Scan(newUUIDString()); err != nil {
		return Card{}, err
	}
	var out Card
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		seq, err := q.BumpWbBoardSeq(ctx, boardID)
		if err != nil {
			return notFound(err)
		}
		if exists, err := q.WbCardExists(ctx, cardID); err != nil {
			return err
		} else if exists {
			return apperr.Invalid("Id kartu sudah dipakai. Muat ulang papan, lalu coba lagi.")
		}
		boards, err := q.ListWbSubtreeBoards(ctx, tplID)
		if err != nil {
			return err
		}
		if len(boards) == 0 {
			return apperr.Invalid("Template itu sudah dihapus. Pilih template lain.")
		}
		if !boards[0].IsTemplate {
			return apperr.Invalid("Papan itu bukan template lagi. Pilih template lain.")
		}
		crumbs, err := ancestors(ctx, q, boardID)
		if err != nil {
			return err
		}
		// crumbs memuat papan akar dan papan ini, jadi salinan berkedalaman len(crumbs) dan turunan terdalamnya
		// len(crumbs) + depth terbesar.
		if len(crumbs)+int(boards[len(boards)-1].Depth) > MaxDepth {
			return apperr.Invalid(fmt.Sprintf("Template ini terlalu bertingkat untuk ditaruh di sini (paling dalam %d tingkat).", MaxDepth))
		}
		ids := make([]pgtype.UUID, len(boards))
		for i, b := range boards {
			ids[i] = b.ID
		}
		cards, err := q.ListWbCardsForCopy(ctx, ids)
		if err != nil {
			return err
		}
		if len(cards) > MaxTemplateCards {
			return apperr.Invalid(fmt.Sprintf("Template ini berisi lebih dari %d kartu, terlalu besar untuk disalin sekaligus.", MaxTemplateCards))
		}

		boardMap := make(map[[16]byte]pgtype.UUID, len(boards))
		for _, b := range boards {
			parent := boardID
			if b.Depth > 0 {
				parent = boardMap[b.ParentID.Bytes]
			}
			nb, err := q.InsertWbBoard(ctx, sqlc.InsertWbBoardParams{ParentID: parent, Title: b.Title, ActorID: actor.ID})
			if err != nil {
				return err
			}
			boardMap[b.ID.Bytes] = nb
		}
		cardMap := make(map[[16]byte]pgtype.UUID, len(cards))
		for _, c := range cards {
			var nid pgtype.UUID
			if err := nid.Scan(newUUIDString()); err != nil {
				return err
			}
			cardMap[c.ID.Bytes] = nid
		}
		// Grup salinan memakai id baru, supaya template yang ditaruh dua kali di papan yang sama tidak menggabungkan grupnya.
		groupMap := map[[16]byte]pgtype.UUID{}
		for _, c := range cards {
			groups := make([]pgtype.UUID, len(c.GroupIds))
			for i, g := range c.GroupIds {
				ng, ok := groupMap[g.Bytes]
				if !ok {
					if err := ng.Scan(newUUIDString()); err != nil {
						return err
					}
					groupMap[g.Bytes] = ng
				}
				groups[i] = ng
			}
			p := sqlc.InsertWbCardCopyParams{
				ID: cardMap[c.ID.Bytes], BoardID: boardMap[c.BoardID.Bytes], Kind: c.Kind, FileID: c.FileID,
				X: c.X, Y: c.Y, W: c.W, H: c.H, Z: c.Z, Color: c.Color, Content: c.Content, Plain: c.Plain, ActorID: actor.ID,
				Style: c.Style, Locked: c.Locked, GroupIds: groups,
			}
			if c.RefBoardID.Valid {
				p.RefBoardID = boardMap[c.RefBoardID.Bytes]
			}
			// Kolom yang tidak ikut tersalin (seharusnya tidak terjadi, karena kolom dan isinya dihapus bersama)
			// membuat kartunya berdiri sendiri.
			if col, ok := cardMap[c.ColumnID.Bytes]; c.ColumnID.Valid && ok {
				p.ColumnID, p.ColOrder = col, c.ColOrder
			}
			if err := q.InsertWbCardCopy(ctx, p); err != nil {
				return err
			}
		}
		conns, err := q.ListWbConnectorsForCopy(ctx, ids)
		if err != nil {
			return err
		}
		for _, n := range conns {
			from, okFrom := mapEnd(cardMap, n.FromCardID)
			to, okTo := mapEnd(cardMap, n.ToCardID)
			// Ujung yang kartunya di Tempat sampah: garisnya tidak tampil di template, jadi tidak ikut tersalin.
			if !okFrom || !okTo {
				continue
			}
			if _, err := q.InsertWbConnector(ctx, sqlc.InsertWbConnectorParams{
				BoardID: boardMap[n.BoardID.Bytes], FromCardID: from, ToCardID: to, Geometry: n.Geometry, Style: n.Style,
				Label: n.Label, LabelStart: n.LabelStart, LabelEnd: n.LabelEnd, ActorID: actor.ID,
			}); err != nil {
				return err
			}
		}

		title := boards[0].Title
		z, err := q.MaxWbCardZ(ctx, boardID)
		if err != nil {
			return err
		}
		if err := q.InsertWbCard(ctx, sqlc.InsertWbCardParams{
			ID: cardID, BoardID: boardID, Kind: KindBoard, RefBoardID: boardMap[tplID.Bytes], X: in.X, Y: in.Y, W: in.W, H: in.H,
			Z: math.Min(z+1, maxZ), Content: []byte(`{}`), Plain: title, Seq: seq, ActorID: actor.ID,
			Style: []byte(`{}`), GroupIds: []pgtype.UUID{},
		}); err != nil {
			return err
		}
		if out, err = getCard(ctx, q, cardID); err != nil {
			return err
		}
		if err := record(ctx, q, boardMap[tplID.Bytes], actor.ID, "board_from_template", pgtype.UUID{}, map[string]any{"template": title, "cards": len(cards)}); err != nil {
			return err
		}
		return record(ctx, q, boardID, actor.ID, "card_created", cardID, map[string]any{"kind": KindBoard, "excerpt": excerpt(title), "template": title})
	})
	return out, err
}

// mapEnd memetakan ujung garis ke kartu salinannya; ujung bebas (NULL) tetap bebas.
func mapEnd(m map[[16]byte]pgtype.UUID, id pgtype.UUID) (pgtype.UUID, bool) {
	if !id.Valid {
		return id, true
	}
	n, ok := m[id.Bytes]
	return n, ok
}

// searchTerm memeriksa kata kunci dan meloloskan % dan _ supaya dicari apa adanya, bukan sebagai wildcard.
func searchTerm(term string) (string, error) {
	term = strings.TrimSpace(term)
	if utf8.RuneCountInString(term) < 2 {
		return "", apperr.Invalid("Ketik minimal 2 huruf untuk mencari.")
	}
	if utf8.RuneCountInString(term) > 100 {
		return "", apperr.Invalid("Kata kunci maksimal 100 karakter.")
	}
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(term), nil
}

type CardHit struct {
	ID      string `json:"id"`
	BoardID string `json:"board_id"`
	// Board: jalur papan tempat kartu ini, dari papan akar, dipisah " / ".
	Board   string `json:"board"`
	Kind    string `json:"kind"`
	Snippet string `json:"snippet"`
}

// SearchCards mencari isi kartu di semua papan hidup, paling baru diubah lebih dulu, paling banyak 50.
func (s *Service) SearchCards(ctx context.Context, term string) ([]CardHit, error) {
	esc, err := searchTerm(term)
	if err != nil {
		return nil, err
	}
	q := sqlc.New(s.pool)
	rows, err := q.SearchWbCards(ctx, esc)
	if err != nil {
		return nil, err
	}
	out := make([]CardHit, 0, len(rows))
	paths := map[[16]byte]string{}
	for _, r := range rows {
		path, ok := paths[r.BoardID.Bytes]
		if !ok {
			if path, err = crumbPath(ctx, q, r.BoardID); err != nil {
				return nil, err
			}
			paths[r.BoardID.Bytes] = path
		}
		text := r.Plain
		if r.PreviewTitle != "" {
			text = r.PreviewTitle + " " + r.Plain
		}
		out = append(out, CardHit{ID: idString(r.ID), BoardID: idString(r.BoardID), Board: path, Kind: r.Kind,
			Snippet: snippet(text, strings.TrimSpace(term))})
	}
	return out, nil
}

// snippet memotong teks sekitar 80 karakter di sekitar kemunculan pertama kata kunci (tanpa membedakan huruf
// besar), supaya hasil pencarian menunjukkan kenapa kartu itu cocok.
func snippet(text, term string) string {
	r := []rune(strings.Join(strings.Fields(text), " "))
	const width = 80
	if len(r) <= width {
		return string(r)
	}
	t := []rune(strings.Join(strings.Fields(term), " "))
	at := -1
	for i := 0; i+len(t) <= len(r) && len(t) > 0; i++ {
		match := true
		for j, c := range t {
			if unicode.ToLower(r[i+j]) != unicode.ToLower(c) {
				match = false
				break
			}
		}
		if match {
			at = i
			break
		}
	}
	start := 0
	if at > 30 {
		start = at - 30
	}
	end := start + width
	if end > len(r) {
		end = len(r)
		start = max(0, end-width)
	}
	out := string(r[start:end])
	if start > 0 {
		out = "..." + out
	}
	if end < len(r) {
		out += "..."
	}
	return out
}
