// Package whiteboard menyimpan papan dan kartu Whiteboard (docs/whiteboard.md). Semua adalah papan: satu papan akar
// bersama, dan papan lain tampil sebagai kartu jenis board di kanvas induknya. Semua Staf dan Manajer boleh menulis
// semua isi; Admin hanya membaca, dan itu dijaga di router (grup tulis) karena aturannya sama untuk setiap rute.
// Hapus selalu ke Tempat sampah; hapus permanen hanya oleh job harian setelah 30 hari.
package whiteboard

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
	"nvm-internal/internal/linkpreview"
	"nvm-internal/internal/storage"
)

const (
	KindNote      = "note"
	KindHeading   = "heading"
	KindChecklist = "checklist"
	KindBoard     = "board"

	// RootID adalah nama papan akar di URL API, supaya klien tidak perlu tahu UUID-nya.
	RootID   = "root"
	MaxTitle = 120
	// MaxDepth: papan akar berkedalaman 0. Lebih dalam dari ini membuat breadcrumb tidak terbaca di ponsel.
	MaxDepth = 8
	// TrashDays: lama isi Tempat sampah sebelum dihapus permanen job harian.
	TrashDays = 30
	// mergeWindow: riwayat seret dan edit beruntun dari orang yang sama digabung ke satu baris.
	mergeWindow = 5 * time.Minute
	// MaxBatch membatasi satu permintaan patch atau hapus; seret 500 kartu sekaligus tetap muat.
	MaxBatch = 500

	minSize, maxSize = 40, 4000
	maxCoord         = 1_000_000
	maxZ             = 1_000_000_000
)

// Colors adalah nama warna kartu yang diizinkan; frontend memetakannya ke token di style.css.
var Colors = map[string]bool{"abu": true, "hijau": true, "kuning": true, "biru": true, "krem": true, "merah": true}

var (
	ErrNotFound = errors.New("whiteboard: tidak ditemukan")
	ErrStale    = errors.New("whiteboard: versi basi")
)

// ConflictError: isi kartu sudah diubah orang lain sejak versi yang dipegang klien. Card berisi keadaan terbaru,
// supaya klien bisa menyimpan ketikannya sebagai kartu salinan tanpa memuat ulang papan.
type ConflictError struct {
	Card Card
}

func (e *ConflictError) Error() string { return "whiteboard: isi kartu sudah diubah" }

type Actor struct {
	ID int64
	IP *netip.Addr
	// Manager: Manajer boleh menghapus komentar siapa pun (keputusan bawaan W3); aturan lain sama untuk Staf.
	Manager bool
}

func (a Actor) audit() audit.Actor { return audit.Actor{ID: a.ID, IP: a.IP} }

type Service struct {
	pool    *pgxpool.Pool
	store   *storage.Store
	preview *linkpreview.Fetcher
	rate    *previewRate
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, preview: linkpreview.New(), rate: newPreviewRate()}
}

// UseStorage memasang penyimpanan berkas untuk kartu gambar dan berkas (W3).
func (s *Service) UseStorage(st *storage.Store) { s.store = st }

// ParseID mengubah id teks menjadi UUID. Teks yang bukan UUID diperlakukan sebagai tidak ditemukan, jadi URL acak
// tidak pernah menjadi galat 500.
func ParseID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(strings.ToLower(strings.TrimSpace(s))); err != nil || !u.Valid {
		return u, ErrNotFound
	}
	return u, nil
}

func idString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return u.String()
}

func optID(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := u.String()
	return &s
}

func ts(t pgtype.Timestamptz) string { return t.Time.UTC().Format(time.RFC3339) }

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func cleanTitle(v string) (string, error) {
	v = strings.Join(strings.Fields(stripControl(v, false)), " ")
	if v == "" {
		return "", apperr.Invalid("Judul papan wajib diisi.")
	}
	if utf8.RuneCountInString(v) > MaxTitle {
		return "", apperr.Invalid(fmt.Sprintf("Judul papan maksimal %d karakter.", MaxTitle))
	}
	return v, nil
}

// newUUIDString membuat UUID versi 4 untuk batch Tempat sampah, kartu, dan butir checklist.
func newUUIDString() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func newBatch() (pgtype.UUID, error) {
	var b pgtype.UUID
	err := b.Scan(newUUIDString())
	return b, err
}

// ---------------------------------------------------------------------------------------------------------------
// Papan

// rootID mengembalikan papan akar. Migrasi membuatnya bila sudah ada akun; database tanpa papan akar (test)
// membuatnya di sini atas nama pembaca pertama.
func (s *Service) rootID(ctx context.Context, actor Actor) (pgtype.UUID, error) {
	q := sqlc.New(s.pool)
	id, err := q.GetWbRoot(ctx)
	if err == nil || !errors.Is(err, pgx.ErrNoRows) {
		return id, err
	}
	if _, err := q.InsertWbRoot(ctx, actor.ID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return id, err
	}
	return q.GetWbRoot(ctx)
}

// boardID menerima "root" atau UUID papan.
func (s *Service) boardID(ctx context.Context, actor Actor, text string) (pgtype.UUID, error) {
	if text == RootID {
		return s.rootID(ctx, actor)
	}
	return ParseID(text)
}

type Crumb struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func ancestors(ctx context.Context, q *sqlc.Queries, id pgtype.UUID) ([]Crumb, error) {
	rows, err := q.ListWbAncestors(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Crumb, 0, len(rows))
	for _, r := range rows {
		out = append(out, Crumb{ID: idString(r.ID), Title: r.Title})
	}
	return out, nil
}

// RefSummary: papan anak yang ditunjuk kartu jenis board.
type RefSummary struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Version int32  `json:"version"`
	Boards  int64  `json:"boards"`
	Cards   int64  `json:"cards"`
}

type Card struct {
	ID             string          `json:"id"`
	BoardID        string          `json:"board_id"`
	Kind           string          `json:"kind"`
	X              float64         `json:"x"`
	Y              float64         `json:"y"`
	W              float64         `json:"w"`
	H              float64         `json:"h"`
	Z              float64         `json:"z"`
	Color          *string         `json:"color"`
	Content        json.RawMessage `json:"content"`
	Version        int32           `json:"version"`
	ContentVersion int32           `json:"content_version"`
	Seq            int64           `json:"seq"`
	UpdatedBy      int64           `json:"updated_by"`
	UpdatedAt      string          `json:"updated_at"`
	Ref            *RefSummary     `json:"ref,omitempty"`
	File           *FileInfo       `json:"file,omitempty"`
	Preview        *PreviewInfo    `json:"preview,omitempty"`
	ColumnID       *string         `json:"column_id"`
	ColOrder       *float64        `json:"col_order"`
	Comments       int64           `json:"comments"`
	CreatedByName  string          `json:"created_by_name"`
}

// FileInfo: berkas kartu gambar atau berkas. Thumb true berarti ada thumbnail JPEG untuk ditampilkan di kanvas.
type FileInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	MIME   string `json:"mime"`
	Size   int64  `json:"size"`
	Thumb  bool   `json:"thumb"`
	Width  int32  `json:"width"`
	Height int32  `json:"height"`
}

// PreviewInfo: pratinjau kartu tautan. Status ok, blocked, atau failed; Image id berkas gambar pratinjau.
type PreviewInfo struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Site        string  `json:"site"`
	FinalURL    string  `json:"final_url"`
	Status      string  `json:"status"`
	Image       *string `json:"image"`
}

func cardOf(r sqlc.ListWbCardsRow) Card {
	c := Card{
		ID: idString(r.ID), BoardID: idString(r.BoardID), Kind: r.Kind, X: r.X, Y: r.Y, W: r.W, H: r.H, Z: r.Z,
		Content: json.RawMessage(r.Content), Version: r.Version, ContentVersion: r.ContentVersion, Seq: r.Seq,
		UpdatedBy: r.UpdatedBy, UpdatedAt: ts(r.UpdatedAt),
	}
	if r.Color.Valid {
		v := r.Color.String
		c.Color = &v
	}
	if r.RefBoardID.Valid {
		c.Ref = &RefSummary{ID: idString(r.RefBoardID), Title: r.RefTitle, Version: r.RefVersion, Boards: r.RefBoards, Cards: r.RefCards}
	}
	if r.FileID.Valid {
		c.File = &FileInfo{ID: idString(r.FileID), Name: r.FileName, MIME: r.FileMime, Size: r.FileSize, Thumb: r.FileThumb,
			Width: r.FileWidth, Height: r.FileHeight}
	}
	if r.PreviewID.Valid {
		c.Preview = &PreviewInfo{ID: idString(r.PreviewID), Title: r.PreviewTitle, Description: r.PreviewDescription,
			Site: r.PreviewSite, FinalURL: r.PreviewFinalUrl, Status: r.PreviewStatus, Image: optID(r.PreviewImage)}
	}
	c.ColumnID = optID(r.ColumnID)
	if r.ColOrder.Valid {
		v := r.ColOrder.Float64
		c.ColOrder = &v
	}
	c.Comments = r.CommentCount
	c.CreatedByName = r.CreatedByName
	return c
}

func getCard(ctx context.Context, q *sqlc.Queries, id pgtype.UUID) (Card, error) {
	r, err := q.GetWbCard(ctx, id)
	if err != nil {
		return Card{}, notFound(err)
	}
	return cardOf(sqlc.ListWbCardsRow(r)), nil
}

type BoardView struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	IsRoot     bool        `json:"is_root"`
	IsTemplate bool        `json:"is_template"`
	Version    int32       `json:"version"`
	Seq        int64       `json:"seq"`
	Crumbs     []Crumb     `json:"crumbs"`
	Cards      []Card      `json:"cards"`
	Connectors []Connector `json:"connectors"`
}

// Board memuat satu papan ("root" untuk papan akar) beserta breadcrumb dan semua kartunya.
func (s *Service) Board(ctx context.Context, actor Actor, idText string) (BoardView, error) {
	id, err := s.boardID(ctx, actor, idText)
	if err != nil {
		return BoardView{}, err
	}
	q := sqlc.New(s.pool)
	n, err := q.GetWbBoard(ctx, id)
	if err != nil {
		return BoardView{}, notFound(err)
	}
	crumbs, err := ancestors(ctx, q, id)
	if err != nil {
		return BoardView{}, err
	}
	rows, err := q.ListWbCards(ctx, id)
	if err != nil {
		return BoardView{}, err
	}
	cards := make([]Card, 0, len(rows))
	for _, r := range rows {
		cards = append(cards, cardOf(r))
	}
	conns, err := q.ListWbConnectors(ctx, id)
	if err != nil {
		return BoardView{}, err
	}
	connectors := make([]Connector, 0, len(conns))
	for _, r := range conns {
		connectors = append(connectors, connectorOf(sqlc.InsertWbConnectorRow(r)))
	}
	return BoardView{ID: idString(n.ID), Title: n.Title, IsRoot: n.IsRoot, IsTemplate: n.IsTemplate, Version: n.Version, Seq: n.Seq, Crumbs: crumbs,
		Cards: cards, Connectors: connectors}, nil
}

type BoardTitle struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Version int32  `json:"version"`
	// ParentID: papan induk yang kartunya menampilkan judul ini (kosong untuk papan akar), untuk siaran hub.
	ParentID string `json:"-"`
}

// RenameBoard mengganti judul papan, termasuk papan akar. Versi basi ditolak 409, supaya judul yang baru diganti
// orang lain tidak tertimpa diam-diam.
func (s *Service) RenameBoard(ctx context.Context, actor Actor, idText, title string, version int32) (BoardTitle, error) {
	id, err := s.boardID(ctx, actor, idText)
	if err != nil {
		return BoardTitle{}, err
	}
	title, err = cleanTitle(title)
	if err != nil {
		return BoardTitle{}, err
	}
	var out BoardTitle
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		old, err := q.GetWbBoardForUpdate(ctx, id)
		if err != nil {
			return notFound(err)
		}
		if old.Version != version {
			return ErrStale
		}
		r, err := q.RenameWbBoard(ctx, sqlc.RenameWbBoardParams{Title: title, ActorID: actor.ID, ID: id, Version: version})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStale
		}
		if err != nil {
			return err
		}
		out = BoardTitle{ID: idString(r.ID), Title: r.Title, Version: r.Version, ParentID: idString(old.ParentID)}
		if old.Title == r.Title {
			return nil
		}
		// Kartu papan di kanvas induk menampilkan judul ini, jadi papan induk ikut mendapat seq baru (W2).
		if old.ParentID.Valid {
			if _, err := q.BumpWbBoardSeq(ctx, old.ParentID); err != nil {
				return notFound(err)
			}
		}
		return record(ctx, q, id, actor.ID, "board_renamed", pgtype.UUID{}, map[string]any{"from": old.Title, "to": r.Title})
	})
	return out, err
}

type SearchResult struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// Path: judul papan induk dari papan akar, dipisah " / ".
	Path string `json:"path"`
}

// Search mencari judul papan; isi kartu dicari SearchCards (W4).
func (s *Service) Search(ctx context.Context, term string) ([]SearchResult, error) {
	esc, err := searchTerm(term)
	if err != nil {
		return nil, err
	}
	q := sqlc.New(s.pool)
	rows, err := q.SearchWbBoards(ctx, esc)
	if err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(rows))
	for _, r := range rows {
		path, err := crumbPath(ctx, q, r.ParentID)
		if err != nil {
			return nil, err
		}
		out = append(out, SearchResult{ID: idString(r.ID), Title: r.Title, Path: path})
	}
	return out, nil
}

// ---------------------------------------------------------------------------------------------------------------
// Kartu

func checkCoord(v float64, label string) error {
	if math.IsNaN(v) || v < -maxCoord || v > maxCoord {
		return apperr.Invalid(label + " kartu di luar kanvas.")
	}
	return nil
}

func checkSize(v float64) error {
	if math.IsNaN(v) || v < minSize || v > maxSize {
		return apperr.Invalid(fmt.Sprintf("Ukuran kartu harus %d sampai %d piksel.", minSize, maxSize))
	}
	return nil
}

func checkZ(v float64) error {
	if math.IsNaN(v) || v < -maxZ || v > maxZ {
		return apperr.Invalid("Urutan tumpuk kartu tidak sah.")
	}
	return nil
}

func checkColor(v string) error {
	if v != "" && !Colors[v] {
		return apperr.Invalid("Warna kartu tidak dikenal.")
	}
	return nil
}

func excerpt(plain string) string {
	plain = strings.Join(strings.Fields(plain), " ")
	if utf8.RuneCountInString(plain) <= 80 {
		return plain
	}
	r := []rune(plain)
	return string(r[:80]) + "..."
}

type CardInput struct {
	ID      string          `json:"id"`
	Kind    string          `json:"kind"`
	X       float64         `json:"x"`
	Y       float64         `json:"y"`
	W       float64         `json:"w"`
	H       float64         `json:"h"`
	Color   string          `json:"color"`
	Content json.RawMessage `json:"content"`
	// Title: judul papan baru, hanya untuk kartu jenis board.
	Title string `json:"title"`
	// FileID: berkas yang sudah diunggah, untuk kartu gambar atau berkas hasil salin tempel.
	FileID string `json:"file_id"`
}

// CreateCard menaruh kartu baru di atas tumpukan. Kartu jenis board sekaligus membuat papan anak di papan ini.
// Id boleh dibuat klien (supaya kartu bisa langsung dipakai sebelum server menjawab), tetapi tidak boleh sudah
// dipakai, termasuk oleh kartu di Tempat sampah.
func (s *Service) CreateCard(ctx context.Context, actor Actor, boardText string, in CardInput) (Card, error) {
	boardID, err := s.boardID(ctx, actor, boardText)
	if err != nil {
		return Card{}, err
	}
	for _, e := range []error{checkCoord(in.X, "Posisi"), checkCoord(in.Y, "Posisi"), checkSize(in.W), checkSize(in.H), checkColor(in.Color)} {
		if e != nil {
			return Card{}, e
		}
	}
	var content []byte
	var plain, title string
	switch in.Kind {
	case KindBoard:
		if title, err = cleanTitle(in.Title); err != nil {
			return Card{}, err
		}
		content = []byte(`{}`)
	case KindNote, KindHeading, KindChecklist, KindLink, KindColumn, KindDrawing, KindTable, KindImage, KindFile, KindShape:
		raw := in.Content
		if len(raw) == 0 || string(raw) == "null" {
			raw = DefaultContent(in.Kind)
		}
		if content, plain, err = CleanContent(in.Kind, raw); err != nil {
			return Card{}, err
		}
	default:
		return Card{}, apperr.Invalid("Jenis kartu tidak dikenal.")
	}
	var cardID pgtype.UUID
	if in.ID != "" {
		if cardID, err = ParseID(in.ID); err != nil {
			return Card{}, apperr.Invalid("Id kartu tidak sah.")
		}
	} else if err := cardID.Scan(newUUIDString()); err != nil {
		return Card{}, err
	}
	var fileID pgtype.UUID
	if in.Kind == KindImage || in.Kind == KindFile {
		if fileID, err = ParseID(in.FileID); err != nil {
			return Card{}, apperr.Invalid("Kartu gambar dan berkas dibuat lewat unggahan.")
		}
	}
	var out Card
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		seq, err := q.BumpWbBoardSeq(ctx, boardID)
		if err != nil {
			return notFound(err)
		}
		exists, err := q.WbCardExists(ctx, cardID)
		if err != nil {
			return err
		}
		if exists {
			return apperr.Invalid("Id kartu sudah dipakai. Muat ulang papan, lalu coba lagi.")
		}
		if fileID.Valid {
			// Salin tempel memakai berkas yang sama; jenis kartu harus cocok dengan jenis berkasnya.
			f, err := q.GetWbFile(ctx, fileID)
			if err != nil || f.Kind != in.Kind {
				return apperr.Invalid("Berkas kartu ini tidak ditemukan. Unggah ulang berkasnya.")
			}
			if plain == "" {
				plain = f.OriginalName
			}
		}
		var ref pgtype.UUID
		if in.Kind == KindBoard {
			crumbs, err := ancestors(ctx, q, boardID)
			if err != nil {
				return err
			}
			// crumbs memuat papan akar dan papan ini, jadi papan baru berkedalaman len(crumbs).
			if len(crumbs) > MaxDepth {
				return apperr.Invalid(fmt.Sprintf("Papan bertingkat paling dalam %d tingkat.", MaxDepth))
			}
			if ref, err = q.InsertWbBoard(ctx, sqlc.InsertWbBoardParams{ParentID: boardID, Title: title, ActorID: actor.ID}); err != nil {
				return err
			}
			plain = title
		}
		z, err := q.MaxWbCardZ(ctx, boardID)
		if err != nil {
			return err
		}
		if err := q.InsertWbCard(ctx, sqlc.InsertWbCardParams{
			ID: cardID, BoardID: boardID, Kind: in.Kind, RefBoardID: ref, FileID: fileID, X: in.X, Y: in.Y, W: in.W, H: in.H, Z: math.Min(z+1, maxZ),
			Color: pgtype.Text{String: in.Color, Valid: in.Color != ""}, Content: content, Plain: plain, Seq: seq, ActorID: actor.ID,
		}); err != nil {
			return err
		}
		if out, err = getCard(ctx, q, cardID); err != nil {
			return err
		}
		return record(ctx, q, boardID, actor.ID, "card_created", cardID, map[string]any{"kind": in.Kind, "excerpt": excerpt(plain)})
	})
	return out, err
}

// CardPatch: field nil berarti tidak diubah. Color "" mengosongkan warna. Content wajib disertai
// BaseContentVersion; isi yang diubah orang lain sejak versi itu ditolak dengan ConflictError.
type CardPatch struct {
	ID                 string          `json:"id"`
	X                  *float64        `json:"x"`
	Y                  *float64        `json:"y"`
	W                  *float64        `json:"w"`
	H                  *float64        `json:"h"`
	Z                  *float64        `json:"z"`
	Color              *string         `json:"color"`
	Content            json.RawMessage `json:"content"`
	BaseContentVersion int32           `json:"base_content_version"`
	// ColumnID: id kartu kolom tujuan, atau "" untuk keluar dari kolom. ColOrder urutan di dalam kolom.
	ColumnID *string  `json:"column_id"`
	ColOrder *float64 `json:"col_order"`
}

func f8(v *float64) pgtype.Float8 {
	if v == nil {
		return pgtype.Float8{}
	}
	return pgtype.Float8{Float64: *v, Valid: true}
}

func (p CardPatch) check() error {
	for _, v := range []*float64{p.X, p.Y} {
		if v != nil {
			if err := checkCoord(*v, "Posisi"); err != nil {
				return err
			}
		}
	}
	for _, v := range []*float64{p.W, p.H} {
		if v != nil {
			if err := checkSize(*v); err != nil {
				return err
			}
		}
	}
	if p.Z != nil {
		if err := checkZ(*p.Z); err != nil {
			return err
		}
	}
	if p.Color != nil {
		return checkColor(*p.Color)
	}
	return nil
}

func (p CardPatch) hasContent() bool { return len(p.Content) > 0 && string(p.Content) != "null" }

// PatchCards menerapkan perubahan beberapa kartu di satu papan dalam satu transaksi. Posisi, ukuran, urutan, dan
// warna digabung per field (yang terakhir sampai menang); isi dijaga content_version.
func (s *Service) PatchCards(ctx context.Context, actor Actor, boardText string, patches []CardPatch) ([]Card, error) {
	boardID, err := s.boardID(ctx, actor, boardText)
	if err != nil {
		return nil, err
	}
	if len(patches) == 0 {
		return []Card{}, nil
	}
	if len(patches) > MaxBatch {
		return nil, apperr.Invalid(fmt.Sprintf("Maksimal %d kartu sekaligus.", MaxBatch))
	}
	seen := map[string]bool{}
	for _, p := range patches {
		if seen[p.ID] {
			return nil, apperr.Invalid("Kartu yang sama dikirim dua kali.")
		}
		seen[p.ID] = true
		if err := p.check(); err != nil {
			return nil, err
		}
	}
	out := make([]Card, 0, len(patches))
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		seq, err := q.BumpWbBoardSeq(ctx, boardID)
		if err != nil {
			return notFound(err)
		}
		layoutIDs := []string{}
		for _, p := range patches {
			id, err := ParseID(p.ID)
			if err != nil {
				return err
			}
			cur, err := q.GetWbCardForUpdate(ctx, sqlc.GetWbCardForUpdateParams{ID: id, BoardID: boardID})
			if err != nil {
				return notFound(err)
			}
			params := sqlc.PatchWbCardParams{
				X: f8(p.X), Y: f8(p.Y), W: f8(p.W), H: f8(p.H), Z: f8(p.Z), Seq: seq, ActorID: actor.ID, ID: id, BoardID: boardID,
			}
			if p.Color != nil {
				params.ClearColor = *p.Color == ""
				params.Color = pgtype.Text{String: *p.Color, Valid: *p.Color != ""}
			}
			if err := columnPatch(ctx, q, boardID, id, cur.Kind, p, &params); err != nil {
				return err
			}
			if cur.Kind == KindDrawing && (p.hasContent() || p.W != nil || p.H != nil) {
				return apperr.Invalid("Gambar bebas diubah lewat pena, tidak lewat isi atau ukuran.")
			}
			var plain string
			if p.hasContent() {
				if cur.Kind == KindBoard {
					return apperr.Invalid("Judul papan diganti dari dalam papan itu.")
				}
				if p.BaseContentVersion != cur.ContentVersion {
					latest, err := getCard(ctx, q, id)
					if err != nil {
						return err
					}
					return &ConflictError{Card: latest}
				}
				content, pl, err := CleanContent(cur.Kind, p.Content)
				if err != nil {
					return err
				}
				params.Content = content
				params.Plain = pgtype.Text{String: pl, Valid: true}
				plain = pl
			}
			if err := q.PatchWbCard(ctx, params); err != nil {
				return err
			}
			c, err := getCard(ctx, q, id)
			if err != nil {
				return err
			}
			out = append(out, c)
			if p.hasContent() {
				if err := record(ctx, q, boardID, actor.ID, "card_edited", id, map[string]any{"kind": cur.Kind, "excerpt": excerpt(plain)}); err != nil {
					return err
				}
			}
			if p.X != nil || p.Y != nil || p.W != nil || p.H != nil || p.Z != nil || p.Color != nil || p.ColumnID != nil || p.ColOrder != nil {
				layoutIDs = append(layoutIDs, idString(id))
			}
		}
		if len(layoutIDs) > 0 {
			return recordLayout(ctx, q, boardID, actor.ID, layoutIDs)
		}
		return nil
	})
	return out, err
}

// ChecklistOp mengubah satu butir atau judul checklist secara atomik di server, jadi dua orang yang mencentang
// butir berbeda tidak saling menimpa dan centang tidak pernah bentrok dengan versi isi.
type ChecklistOp struct {
	Op     string  `json:"op"`
	ItemID string  `json:"item_id"`
	Text   *string `json:"text"`
	Done   *bool   `json:"done"`
	// Index: posisi butir baru atau tujuan pindah, dihitung dari 0. Di luar rentang berarti paling akhir.
	Index *int `json:"index"`
}

func (s *Service) Checklist(ctx context.Context, actor Actor, boardText, cardText string, op ChecklistOp) (Card, error) {
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
		cur, err := q.GetWbCardForUpdate(ctx, sqlc.GetWbCardForUpdateParams{ID: cardID, BoardID: boardID})
		if err != nil {
			return notFound(err)
		}
		if cur.Kind != KindChecklist {
			return apperr.Invalid("Kartu ini bukan checklist.")
		}
		var c ChecklistContent
		if err := json.Unmarshal(cur.Content, &c); err != nil {
			return err
		}
		if err := applyChecklistOp(&c, op); err != nil {
			return err
		}
		raw, err := json.Marshal(c)
		if err != nil {
			return err
		}
		content, plain, err := CleanContent(KindChecklist, raw)
		if err != nil {
			return err
		}
		if err := q.PatchWbCard(ctx, sqlc.PatchWbCardParams{
			Content: content, Plain: pgtype.Text{String: plain, Valid: true}, Seq: seq, ActorID: actor.ID, ID: cardID, BoardID: boardID,
		}); err != nil {
			return err
		}
		if out, err = getCard(ctx, q, cardID); err != nil {
			return err
		}
		return record(ctx, q, boardID, actor.ID, "card_edited", cardID, map[string]any{"kind": KindChecklist, "excerpt": excerpt(plain)})
	})
	return out, err
}

func applyChecklistOp(c *ChecklistContent, op ChecklistOp) error {
	find := func() (int, error) {
		id := strings.ToLower(strings.TrimSpace(op.ItemID))
		for i, it := range c.Items {
			if it.ID == id {
				return i, nil
			}
		}
		return -1, apperr.Invalid("Butir checklist tidak ditemukan. Mungkin sudah dihapus orang lain.")
	}
	at := func(n int) int {
		if op.Index == nil || *op.Index < 0 || *op.Index > n {
			return n
		}
		return *op.Index
	}
	switch op.Op {
	case "title":
		if op.Text == nil {
			return apperr.Invalid("Judul checklist kosong.")
		}
		c.Title = *op.Text
	case "add":
		if len(c.Items) >= MaxChecklist {
			return apperr.Invalid(fmt.Sprintf("Checklist maksimal %d butir.", MaxChecklist))
		}
		id := strings.ToLower(strings.TrimSpace(op.ItemID))
		if id == "" {
			id = newUUIDString()
		}
		for _, it := range c.Items {
			if it.ID == id {
				return apperr.Invalid("Butir checklist ini sudah ada.")
			}
		}
		text := ""
		if op.Text != nil {
			text = *op.Text
		}
		i := at(len(c.Items))
		c.Items = append(c.Items[:i], append([]ChecklistItem{{ID: id, Text: text, Done: op.Done != nil && *op.Done}}, c.Items[i:]...)...)
	case "set":
		i, err := find()
		if err != nil {
			return err
		}
		if op.Text != nil {
			c.Items[i].Text = *op.Text
		}
		if op.Done != nil {
			c.Items[i].Done = *op.Done
		}
	case "remove":
		i, err := find()
		if err != nil {
			return err
		}
		c.Items = append(c.Items[:i], c.Items[i+1:]...)
	case "move":
		i, err := find()
		if err != nil {
			return err
		}
		it := c.Items[i]
		rest := append(append([]ChecklistItem{}, c.Items[:i]...), c.Items[i+1:]...)
		j := at(len(rest))
		c.Items = append(rest[:j], append([]ChecklistItem{it}, rest[j:]...)...)
	default:
		return apperr.Invalid("Perubahan checklist tidak dikenal.")
	}
	return nil
}

// DeleteCards memindahkan kartu terpilih ke Tempat sampah dalam satu batch. Kartu papan membawa papan anaknya dan
// seluruh turunannya ke batch yang sama, jadi pulihkan mengembalikan semuanya sekaligus. drafts berisi draf terakhir
// kartu yang sedang diedit orang lain (dari hub); draf yang sah disimpan sebagai isi kartu di transaksi yang sama,
// jadi ketikan terakhir ikut masuk Tempat sampah dan kembali saat dipulihkan.
// Papan yang ikut terhapus dikembalikan (id) supaya hub bisa memberi tahu siapa pun yang sedang membukanya.
func (s *Service) DeleteCards(ctx context.Context, actor Actor, boardText string, ids []string, drafts map[string]json.RawMessage) (string, []string, error) {
	boardID, err := s.boardID(ctx, actor, boardText)
	if err != nil {
		return "", nil, err
	}
	if len(ids) == 0 {
		return "", nil, apperr.Invalid("Pilih kartu yang mau dihapus.")
	}
	if len(ids) > MaxBatch {
		return "", nil, apperr.Invalid(fmt.Sprintf("Maksimal %d kartu sekaligus.", MaxBatch))
	}
	uuids := make([]pgtype.UUID, 0, len(ids))
	for _, t := range ids {
		u, err := ParseID(t)
		if err != nil {
			return "", nil, err
		}
		uuids = append(uuids, u)
	}
	var batch pgtype.UUID
	boardIDs := []string{}
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		seq, err := q.BumpWbBoardSeq(ctx, boardID)
		if err != nil {
			return notFound(err)
		}
		if batch, err = newBatch(); err != nil {
			return err
		}
		for _, u := range uuids {
			draft, ok := drafts[idString(u)]
			if !ok {
				continue
			}
			cur, err := q.GetWbCardForUpdate(ctx, sqlc.GetWbCardForUpdateParams{ID: u, BoardID: boardID})
			if err != nil {
				continue
			}
			content, plain, err := CleanContent(cur.Kind, draft)
			if err != nil {
				continue
			}
			if err := q.PatchWbCard(ctx, sqlc.PatchWbCardParams{
				Content: content, Plain: pgtype.Text{String: plain, Valid: true}, Seq: seq, ActorID: actor.ID, ID: u, BoardID: boardID,
			}); err != nil {
				return err
			}
		}
		// Isi kolom ikut terhapus bersama kolomnya, dan garis yang menempel di kartu mana pun ikut ke batch yang sama.
		children, err := q.ListWbColumnChildren(ctx, uuids)
		if err != nil {
			return err
		}
		uuids = append(uuids, children...)
		rows, err := q.TrashWbCards(ctx, sqlc.TrashWbCardsParams{ActorID: actor.ID, Batch: batch, Seq: seq, BoardID: boardID, Ids: uuids})
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return ErrNotFound
		}
		if _, err := q.TrashWbConnectors(ctx, sqlc.TrashWbConnectorsParams{ActorID: actor.ID, Batch: batch, Seq: seq,
			BoardID: boardID, Ids: []pgtype.UUID{}, Cards: uuids}); err != nil {
			return err
		}
		refs := []pgtype.UUID{}
		excerpts := make([]string, 0, len(rows))
		for _, r := range rows {
			if r.RefBoardID.Valid {
				refs = append(refs, r.RefBoardID)
			}
			excerpts = append(excerpts, excerpt(r.Plain))
		}
		boards := []string{}
		if len(refs) > 0 {
			sub, err := q.TrashWbSubtree(ctx, sqlc.TrashWbSubtreeParams{Ids: refs, ActorID: actor.ID, Batch: batch})
			if err != nil {
				return err
			}
			for _, b := range sub {
				boards = append(boards, b.Title)
				boardIDs = append(boardIDs, idString(b.ID))
			}
		}
		board, err := q.GetWbBoard(ctx, boardID)
		if err != nil {
			return err
		}
		if err := record(ctx, q, boardID, actor.ID, "cards_deleted", pgtype.UUID{}, map[string]any{"count": len(rows), "boards": len(boards), "batch": idString(batch)}); err != nil {
			return err
		}
		return audit.Change(ctx, q, actor.audit(), "wb_deleted", "wb_batch", idString(batch), nil,
			map[string]any{"board": board.Title, "cards": len(rows), "boards": boards, "excerpts": excerpts})
	})
	return idString(batch), boardIDs, err
}

// ---------------------------------------------------------------------------------------------------------------
// Tempat sampah

type TrashBatch struct {
	Batch         string `json:"batch"`
	DeletedAt     string `json:"deleted_at"`
	PurgeAt       string `json:"purge_at"`
	DeletedByName string `json:"deleted_by_name"`
	// BoardTitle: papan asal kartu yang dihapus (tempat kartu kembali saat dipulihkan).
	BoardTitle string `json:"board_title"`
	// Boards: judul papan yang ikut terhapus (papan teratas lebih dulu). Cards: kartu selain kartu papan.
	Boards     []string `json:"boards"`
	Cards      int64    `json:"cards"`
	Connectors int      `json:"connectors"`
	Excerpts   []string `json:"excerpts"`
	BlockedBy  string   `json:"blocked_by,omitempty"`
}

type trashData struct {
	boards     []sqlc.ListWbTrashBoardsRow
	cards      []sqlc.ListWbTrashCardsRow
	connectors []sqlc.ListWbTrashConnectorsRow
}

func loadTrash(ctx context.Context, q *sqlc.Queries) (trashData, error) {
	b, err := q.ListWbTrashBoards(ctx)
	if err != nil {
		return trashData{}, err
	}
	c, err := q.ListWbTrashCards(ctx)
	if err != nil {
		return trashData{}, err
	}
	n, err := q.ListWbTrashConnectors(ctx)
	return trashData{boards: b, cards: c, connectors: n}, err
}

// batches menyusun satu baris per batch dari kartu yang dihapus dan papan yang ikut terhapus. BlockedBy terisi
// bila papan asal kartunya juga di Tempat sampah (dari aksi hapus lain).
func (d trashData) batches() []TrashBatch {
	byID := map[string]*TrashBatch{}
	order := []string{}
	get := func(b pgtype.UUID, at pgtype.Timestamptz, by string) *TrashBatch {
		k := idString(b)
		if t, ok := byID[k]; ok {
			return t
		}
		t := &TrashBatch{Batch: k, DeletedAt: ts(at), PurgeAt: at.Time.UTC().AddDate(0, 0, TrashDays).Format(time.RFC3339),
			DeletedByName: by, Boards: []string{}, Excerpts: []string{}}
		byID[k] = t
		order = append(order, k)
		return t
	}
	for _, c := range d.cards {
		t := get(c.DeletedBatch, c.DeletedAt, c.DeletedByName)
		t.BoardTitle = c.BoardTitle
		if c.BoardInTrash {
			t.BlockedBy = c.BoardTitle
		}
		if c.Kind == KindBoard {
			continue
		}
		t.Cards++
		if len(t.Excerpts) < 3 && c.Plain != "" {
			t.Excerpts = append(t.Excerpts, excerpt(c.Plain))
		}
	}
	inBatch := map[string]string{}
	for _, b := range d.boards {
		inBatch[idString(b.ID)] = idString(b.DeletedBatch)
	}
	for _, b := range d.boards {
		t := get(b.DeletedBatch, b.DeletedAt, b.DeletedByName)
		t.Cards += b.CardCount
		top := inBatch[idString(b.ParentID)] != idString(b.DeletedBatch)
		if top {
			t.Boards = append([]string{b.Title}, t.Boards...)
		} else {
			t.Boards = append(t.Boards, b.Title)
		}
	}
	for _, n := range d.connectors {
		t := get(n.DeletedBatch, n.DeletedAt, n.DeletedByName)
		t.Connectors++
		if t.BoardTitle == "" {
			t.BoardTitle = n.BoardTitle
		}
		if n.BoardInTrash && t.Cards == 0 && len(t.Boards) == 0 {
			t.BlockedBy = n.BoardTitle
		}
	}
	out := make([]TrashBatch, 0, len(order))
	for _, k := range order {
		out = append(out, *byID[k])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].DeletedAt > out[j].DeletedAt })
	return out
}

func (s *Service) Trash(ctx context.Context) ([]TrashBatch, error) {
	d, err := loadTrash(ctx, sqlc.New(s.pool))
	if err != nil {
		return nil, err
	}
	return d.batches(), nil
}

type RestoreResult struct {
	// BoardID: papan tempat kartu kembali, untuk dibuka klien setelah pulih.
	BoardID string `json:"board_id"`
	Title   string `json:"title"`
}

// Restore memulihkan satu batch utuh. Batch yang papan asalnya masih di Tempat sampah ditolak dengan nama papan
// itu, supaya papan asal tidak ikut dipulihkan diam-diam.
func (s *Service) Restore(ctx context.Context, actor Actor, batchText string) (RestoreResult, error) {
	batch, err := ParseID(batchText)
	if err != nil {
		return RestoreResult{}, err
	}
	var out RestoreResult
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		d, err := loadTrash(ctx, q)
		if err != nil {
			return err
		}
		var found *TrashBatch
		for _, t := range d.batches() {
			if t.Batch == idString(batch) {
				found = &t
				break
			}
		}
		if found == nil {
			return ErrNotFound
		}
		if found.BlockedBy != "" {
			return apperr.Invalid(fmt.Sprintf("Pulihkan dulu papan \"%s\" dari Tempat sampah, karena isi ini berada di dalamnya.", found.BlockedBy))
		}
		out.Title = found.BoardTitle
		bump := map[string]pgtype.UUID{}
		for _, c := range d.cards {
			if c.DeletedBatch == batch {
				bump[idString(c.BoardID)] = c.BoardID
				out.BoardID = idString(c.BoardID)
			}
		}
		for _, n := range d.connectors {
			if n.DeletedBatch == batch {
				bump[idString(n.BoardID)] = n.BoardID
			}
		}
		for _, b := range bump {
			if _, err := q.BumpWbBoardSeq(ctx, b); err != nil {
				return notFound(err)
			}
		}
		if _, err := q.RestoreWbBoards(ctx, sqlc.RestoreWbBoardsParams{ActorID: actor.ID, Batch: batch}); err != nil {
			return err
		}
		restored, err := q.RestoreWbCards(ctx, sqlc.RestoreWbCardsParams{ActorID: actor.ID, Batch: batch})
		if err != nil {
			return err
		}
		conns, err := q.RestoreWbConnectors(ctx, sqlc.RestoreWbConnectorsParams{ActorID: actor.ID, Batch: batch})
		if err != nil {
			return err
		}
		if out.BoardID == "" && len(conns) > 0 {
			out.BoardID = idString(conns[0].BoardID)
		}
		if len(restored) > 0 {
			if err := record(ctx, q, restored[0].BoardID, actor.ID, "cards_restored", pgtype.UUID{},
				map[string]any{"count": len(restored), "boards": len(found.Boards)}); err != nil {
				return err
			}
		}
		return audit.Change(ctx, q, actor.audit(), "wb_restored", "wb_batch", idString(batch), nil,
			map[string]any{"board": found.BoardTitle, "boards": found.Boards, "cards": found.Cards})
	})
	return out, err
}

// Purge menghapus permanen batch Tempat sampah yang lebih tua dari TrashDays. Dipanggil job maintenance harian,
// yang berjalan sebagai role aplikasi. Setiap batch dicatat di audit log tanpa pelaku (sistem).
func Purge(ctx context.Context, pool *pgxpool.Pool, now time.Time, dryRun bool) (int, error) {
	before := pgtype.Timestamptz{Time: now.AddDate(0, 0, -TrashDays), Valid: true}
	var n int
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		batches, err := q.ListWbExpiredBatches(ctx, before)
		if err != nil {
			return err
		}
		n = len(batches)
		if dryRun {
			return nil
		}
		// Berkas yang tidak dipakai kartu mana pun (umur lebih dari satu hari) dan cache pratinjau lama yang tidak
		// dipakai kartu tautan: barisnya dihapus di sini, berkasnya dibersihkan pembersih berkas yatim.
		if _, err := q.DeleteWbOldPreviews(ctx, pgtype.Timestamptz{Time: now.AddDate(0, 0, -PreviewDays), Valid: true}); err != nil {
			return err
		}
		if n == 0 {
			_, err := q.DeleteWbUnusedFiles(ctx, pgtype.Timestamptz{Time: now.Add(-unusedFileAge), Valid: true})
			return err
		}
		if _, err := q.DeleteWbExpiredConnectors(ctx, before); err != nil {
			return err
		}
		if _, err := q.DeleteWbExpiredCards(ctx, before); err != nil {
			return err
		}
		if _, err := q.DeleteWbExpiredBoards(ctx, before); err != nil {
			return err
		}
		if _, err := q.DeleteWbUnusedFiles(ctx, pgtype.Timestamptz{Time: now.Add(-unusedFileAge), Valid: true}); err != nil {
			return err
		}
		for _, b := range batches {
			if err := audit.Record(ctx, q, audit.Entry{Action: "wb_purged", Entity: "wb_batch", EntityID: idString(b.Batch),
				New: map[string]any{"boards": b.Boards, "cards": b.Cards}}); err != nil {
				return err
			}
		}
		return nil
	})
	return n, err
}

// ---------------------------------------------------------------------------------------------------------------
// Riwayat aktivitas

// record menulis satu baris riwayat. Edit isi kartu yang sama oleh orang yang sama dalam mergeWindow digabung ke
// baris sebelumnya.
func record(ctx context.Context, q *sqlc.Queries, boardID pgtype.UUID, userID int64, action string, entity pgtype.UUID, summary map[string]any) error {
	raw, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	if action == "card_edited" {
		prev, err := q.GetWbActivityToMerge(ctx, sqlc.GetWbActivityToMergeParams{
			BoardID: boardID, UserID: userID, Action: action, EntityID: entity,
			Since: pgtype.Timestamptz{Time: time.Now().Add(-mergeWindow), Valid: true},
		})
		if err == nil {
			return q.UpdateWbActivitySummary(ctx, sqlc.UpdateWbActivitySummaryParams{Summary: raw, ID: prev.ID})
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
	}
	return q.InsertWbActivity(ctx, sqlc.InsertWbActivityParams{BoardID: boardID, UserID: userID, Action: action, EntityID: entity, Summary: raw})
}

const maxLayoutIDs = 200

// recordLayout menggabungkan seret, ubah ukuran, urutan, dan warna beruntun dari orang yang sama ke satu baris per
// jendela 5 menit, dengan daftar kartu yang tersentuh, jadi tetap terlacak tanpa ratusan baris.
func recordLayout(ctx context.Context, q *sqlc.Queries, boardID pgtype.UUID, userID int64, ids []string) error {
	prev, err := q.GetWbActivityToMerge(ctx, sqlc.GetWbActivityToMergeParams{
		BoardID: boardID, UserID: userID, Action: "cards_arranged",
		Since: pgtype.Timestamptz{Time: time.Now().Add(-mergeWindow), Valid: true},
	})
	merge := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	set := map[string]bool{}
	list := []string{}
	add := func(id string) {
		if !set[id] && len(list) < maxLayoutIDs {
			set[id] = true
			list = append(list, id)
		}
	}
	if merge {
		var old struct {
			IDs []string `json:"ids"`
		}
		_ = json.Unmarshal(prev.Summary, &old)
		for _, id := range old.IDs {
			add(id)
		}
	}
	for _, id := range ids {
		add(id)
	}
	raw, err := json.Marshal(map[string]any{"count": len(list), "ids": list})
	if err != nil {
		return err
	}
	if merge {
		return q.UpdateWbActivitySummary(ctx, sqlc.UpdateWbActivitySummaryParams{Summary: raw, ID: prev.ID})
	}
	return q.InsertWbActivity(ctx, sqlc.InsertWbActivityParams{BoardID: boardID, UserID: userID, Action: "cards_arranged", Summary: raw})
}

type ActivityView struct {
	ID        int64           `json:"id"`
	Action    string          `json:"action"`
	EntityID  *string         `json:"entity_id"`
	Summary   json.RawMessage `json:"summary"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
	UserID    int64           `json:"user_id"`
	UserName  string          `json:"user_name"`
}

func (s *Service) Activity(ctx context.Context, actor Actor, boardText string) ([]ActivityView, error) {
	id, err := s.boardID(ctx, actor, boardText)
	if err != nil {
		return nil, err
	}
	q := sqlc.New(s.pool)
	if _, err := q.GetWbBoard(ctx, id); err != nil {
		return nil, notFound(err)
	}
	rows, err := q.ListWbActivity(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]ActivityView, 0, len(rows))
	for _, r := range rows {
		out = append(out, ActivityView{
			ID: r.ID, Action: r.Action, EntityID: optID(r.EntityID), Summary: json.RawMessage(r.Summary),
			CreatedAt: ts(r.CreatedAt), UpdatedAt: ts(r.UpdatedAt), UserID: r.UserID, UserName: r.UserName,
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------------------------------------------------
// Pendukung hub kolaborasi (hub.go)

// ResolveBoard mengubah "root" atau UUID menjadi id papan hidup.
func (s *Service) ResolveBoard(ctx context.Context, actor Actor, key string) (string, error) {
	id, err := s.boardID(ctx, actor, key)
	if err != nil {
		return "", err
	}
	if _, err := sqlc.New(s.pool).GetWbBoard(ctx, id); err != nil {
		return "", notFound(err)
	}
	return idString(id), nil
}

// CardKind mengembalikan jenis kartu hidup di papan itu, dipakai hub untuk memvalidasi draf.
func (s *Service) CardKind(ctx context.Context, boardText, cardText string) (string, error) {
	b, err := ParseID(boardText)
	if err != nil {
		return "", err
	}
	c, err := ParseID(cardText)
	if err != nil {
		return "", err
	}
	kind, err := sqlc.New(s.pool).GetWbCardKind(ctx, sqlc.GetWbCardKindParams{ID: c, BoardID: b})
	return kind, notFound(err)
}

// MaxSyncRows: selisih lebih dari ini membuat klien memuat ulang papan utuh, bukan menerima daftar perubahan.
const MaxSyncRows = 2000

// Sync berisi perubahan sejak seq tertentu: kartu dan penghubung yang hidup, serta id yang harus dibuang klien.
// TooMany berarti selisihnya melewati MaxSyncRows, jadi klien memuat ulang papan utuh.
type Sync struct {
	Cards          []Card      `json:"cards"`
	Deleted        []string    `json:"deleted"`
	Connectors     []Connector `json:"connectors"`
	ConnectorsGone []string    `json:"connectors_gone"`
	TooMany        bool        `json:"-"`
}

// Since mengembalikan semua yang berubah di papan sesudah seq itu, termasuk yang masuk Tempat sampah.
func (s *Service) Since(ctx context.Context, boardText string, seq int64) (Sync, error) {
	out := Sync{Cards: []Card{}, Deleted: []string{}, Connectors: []Connector{}, ConnectorsGone: []string{}}
	b, err := ParseID(boardText)
	if err != nil {
		return out, err
	}
	q := sqlc.New(s.pool)
	moved, err := q.GetWbResyncSeq(ctx, b)
	if err != nil || moved > seq {
		out.TooMany = err == nil
		return out, notFound(err)
	}
	n, err := q.CountWbCardsSince(ctx, sqlc.CountWbCardsSinceParams{BoardID: b, Seq: seq})
	if err != nil || n > MaxSyncRows {
		out.TooMany = n > MaxSyncRows
		return out, err
	}
	rows, err := q.ListWbCardsSince(ctx, sqlc.ListWbCardsSinceParams{BoardID: b, Seq: seq})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		if r.Deleted {
			out.Deleted = append(out.Deleted, idString(r.ID))
			continue
		}
		out.Cards = append(out.Cards, cardOf(sqlc.ListWbCardsRow{
			ID: r.ID, BoardID: r.BoardID, Kind: r.Kind, RefBoardID: r.RefBoardID, X: r.X, Y: r.Y, W: r.W, H: r.H, Z: r.Z,
			Color: r.Color, Content: r.Content, Version: r.Version, ContentVersion: r.ContentVersion, Seq: r.Seq,
			UpdatedBy: r.UpdatedBy, UpdatedAt: r.UpdatedAt, FileID: r.FileID, ColumnID: r.ColumnID, ColOrder: r.ColOrder,
			RefTitle: r.RefTitle, RefVersion: r.RefVersion, RefBoards: r.RefBoards, RefCards: r.RefCards,
			FileName: r.FileName, FileMime: r.FileMime, FileSize: r.FileSize, FileThumb: r.FileThumb, FileWidth: r.FileWidth,
			FileHeight: r.FileHeight, PreviewID: r.PreviewID, PreviewTitle: r.PreviewTitle, PreviewDescription: r.PreviewDescription,
			PreviewSite: r.PreviewSite, PreviewFinalUrl: r.PreviewFinalUrl, PreviewStatus: r.PreviewStatus,
			PreviewImage: r.PreviewImage, CommentCount: r.CommentCount, CreatedByName: r.CreatedByName,
		}))
	}
	conns, err := q.ListWbConnectorsSince(ctx, sqlc.ListWbConnectorsSinceParams{BoardID: b, Seq: seq})
	if err != nil {
		return out, err
	}
	for _, r := range conns {
		if r.Hidden {
			out.ConnectorsGone = append(out.ConnectorsGone, idString(r.ID))
			continue
		}
		out.Connectors = append(out.Connectors, connectorOf(sqlc.InsertWbConnectorRow{
			ID: r.ID, BoardID: r.BoardID, FromCardID: r.FromCardID, ToCardID: r.ToCardID, Geometry: r.Geometry, Style: r.Style,
			Label: r.Label, LabelStart: r.LabelStart, LabelEnd: r.LabelEnd, Version: r.Version, Seq: r.Seq,
		}))
	}
	return out, nil
}
