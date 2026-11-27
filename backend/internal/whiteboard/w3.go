package whiteboard

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/disintegration/imaging"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	_ "golang.org/x/image/webp"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/attachment"
	"nvm-internal/internal/db/sqlc"
	"nvm-internal/internal/linkpreview"
)

const (
	// MaxFileSize: batas satu unggahan Whiteboard (keputusan W0 bagian 9).
	MaxFileSize = 50 << 20
	// PreviewDays: cache pratinjau tautan dipakai ulang selama ini; baris lama yang tidak dipakai dibuang job harian.
	PreviewDays = 7
	// previewRetry: pratinjau yang gagal atau diblokir dicoba lagi paling cepat setelah ini.
	previewRetry = time.Hour
	// previewPerHour: batas pengambilan pratinjau per orang, supaya server tidak dipakai untuk memindai situs luar.
	previewPerHour = 30
	// unusedFileAge: berkas tanpa kartu dibiarkan sehari dulu (unggahan yang kartunya sedang dibuat, atau kartu yang
	// baru dipotong lalu ditempel lagi).
	unusedFileAge = 24 * time.Hour
	// Gambar beresolusi ekstrem memakan gigabyte memori saat di-decode; dicek dari header sebelum decode penuh.
	maxImagePixels = 40_000_000
	thumbSide      = 1200
	thumbQuality   = 82
	MaxLabel       = 200
	MaxComment     = 2000
)

var (
	ErrTooLarge  = apperr.Invalid("Berkas melebihi 50 MB.")
	ErrNoStorage = errors.New("whiteboard: penyimpanan berkas belum dipasang")
	ErrForbidden = errors.New("whiteboard: tidak boleh")
	errRateLimit = apperr.Invalid("Terlalu banyak pratinjau tautan dalam satu jam. Tautannya tetap tersimpan, coba pratinjau lagi nanti.")
)

// ---------------------------------------------------------------------------------------------------------------
// Batas pratinjau per orang

type previewRate struct {
	mu   sync.Mutex
	hits map[int64][]time.Time
}

func newPreviewRate() *previewRate { return &previewRate{hits: map[int64][]time.Time{}} }

func (r *previewRate) allow(user int64, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	keep := r.hits[user][:0]
	for _, t := range r.hits[user] {
		if now.Sub(t) < time.Hour {
			keep = append(keep, t)
		}
	}
	if len(keep) >= previewPerHour {
		r.hits[user] = keep
		return false
	}
	r.hits[user] = append(keep, now)
	return true
}

// ---------------------------------------------------------------------------------------------------------------
// Kolom

// columnPatch memasukkan atau mengeluarkan kartu dari kolom. Kolom tujuan harus kartu kolom hidup di papan yang
// sama; kolom tidak bisa masuk kolom lain (satu tingkat saja, supaya tumpukan tetap terbaca di ponsel).
func columnPatch(ctx context.Context, q *sqlc.Queries, boardID, id pgtype.UUID, kind string, p CardPatch, params *sqlc.PatchWbCardParams) error {
	if p.ColumnID == nil {
		if p.ColOrder != nil {
			if math.IsNaN(*p.ColOrder) || math.IsInf(*p.ColOrder, 0) || math.Abs(*p.ColOrder) > maxZ {
				return apperr.Invalid("Urutan di kolom tidak sah.")
			}
			params.ColOrder = f8(p.ColOrder)
		}
		return nil
	}
	if *p.ColumnID == "" {
		params.ClearColumn = true
		return nil
	}
	col, err := ParseID(*p.ColumnID)
	if err != nil {
		return apperr.Invalid("Kolom tujuan tidak ditemukan.")
	}
	if kind == KindColumn {
		return apperr.Invalid("Kolom tidak bisa dimasukkan ke kolom lain.")
	}
	if kind == KindDrawing {
		return apperr.Invalid("Gambar bebas tidak bisa dimasukkan ke kolom.")
	}
	if col == id {
		return apperr.Invalid("Kartu tidak bisa dimasukkan ke dirinya sendiri.")
	}
	target, err := q.GetWbCardForUpdate(ctx, sqlc.GetWbCardForUpdateParams{ID: col, BoardID: boardID})
	if err != nil || target.Kind != KindColumn {
		return apperr.Invalid("Kolom tujuan tidak ditemukan di papan ini.")
	}
	params.ColumnID = col
	if p.ColOrder != nil {
		if math.IsNaN(*p.ColOrder) || math.IsInf(*p.ColOrder, 0) || math.Abs(*p.ColOrder) > maxZ {
			return apperr.Invalid("Urutan di kolom tidak sah.")
		}
		params.ColOrder = f8(p.ColOrder)
		return nil
	}
	last, err := q.MaxWbColOrder(ctx, col)
	if err != nil {
		return err
	}
	params.ColOrder = pgtype.Float8{Float64: last + 1, Valid: true}
	return nil
}

// ---------------------------------------------------------------------------------------------------------------
// Berkas dan gambar

// upload: berkas yang sudah ditulis ke tmp/, dicek tipenya dari isi, dan (untuk gambar) dibuatkan thumbnail, tetapi
// belum tercatat di wb_files. Dipakai unggah kartu baru dan ganti gambar atau berkas (revisi 50).
type upload struct {
	tmpPath, thumbTmp, name, mime, kind, sum string
	size                                     int64
	width, height                            pgtype.Int4
}

func (u *upload) cleanup() {
	_ = os.Remove(u.tmpPath)
	if u.thumbTmp != "" {
		_ = os.Remove(u.thumbTmp)
	}
}

// prepareUpload: alur berkasnya sama dengan lampiran: tmp/ sambil dihitung SHA-256, tipe dari isi.
func (s *Service) prepareUpload(originalName string, r io.Reader) (*upload, error) {
	if s.store == nil {
		return nil, ErrNoStorage
	}
	tmp, err := s.store.CreateTemp()
	if err != nil {
		return nil, err
	}
	u := &upload{tmpPath: tmp.Name()}
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			u.cleanup()
		}
	}()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(r, MaxFileSize+1))
	if err != nil {
		return nil, err
	}
	if n > MaxFileSize {
		return nil, ErrTooLarge
	}
	if n == 0 {
		return nil, apperr.Invalid("Berkas kosong.")
	}
	u.size, u.sum = n, hex.EncodeToString(h.Sum(nil))
	u.name = attachment.CleanName(originalName)
	if u.mime, err = attachment.DetectWhiteboard(tmp, n, u.name); err != nil {
		var rej *attachment.Rejected
		if errors.As(err, &rej) {
			return nil, apperr.Invalid(rej.Message)
		}
		return nil, err
	}
	u.kind = KindFile
	if strings.HasPrefix(u.mime, "image/") {
		if _, err := tmp.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		w, h, thumb, err := makeThumb(tmp)
		if err != nil {
			return nil, err
		}
		u.kind = KindImage
		u.width, u.height = pgtype.Int4{Int32: int32(w), Valid: true}, pgtype.Int4{Int32: int32(h), Valid: true}
		if u.thumbTmp, err = s.writeTemp(thumb); err != nil {
			return nil, err
		}
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	ok = true
	return u, nil
}

func (u *upload) insert(ctx context.Context, q *sqlc.Queries, fileID, boardID pgtype.UUID, actor Actor) error {
	return q.InsertWbFile(ctx, sqlc.InsertWbFileParams{
		ID: fileID, BoardID: boardID, Kind: u.kind, OriginalName: u.name, MimeType: u.mime, SizeBytes: u.size,
		Checksum: u.sum, Width: u.width, Height: u.height, HasThumb: u.thumbTmp != "",
		UploadedBy: pgtype.Int8{Int64: actor.ID, Valid: true},
	})
}

// commit memindah thumbnail lalu berkas asli ke tempatnya, di akhir transaksi. Thumbnail lebih dulu: kalau berkas
// asli gagal, transaksi batal dan thumbnail yatim dibersihkan pembersih berkas yatim.
func (s *Service) commitUpload(u *upload, fileID pgtype.UUID) error {
	if u.thumbTmp != "" {
		dst, err := s.store.ThumbPath(idString(fileID))
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return err
		}
		if err := os.Rename(u.thumbTmp, dst); err != nil {
			return err
		}
	}
	return s.store.Commit(u.tmpPath, idString(fileID))
}

// Upload menyimpan berkas lalu membuat kartu gambar (bila bisa dibuatkan thumbnail) atau kartu berkas, dalam satu
// transaksi.
func (s *Service) Upload(ctx context.Context, actor Actor, boardText string, in CardInput, originalName string, r io.Reader) (Card, error) {
	if s.store == nil {
		return Card{}, ErrNoStorage
	}
	boardID, err := s.boardID(ctx, actor, boardText)
	if err != nil {
		return Card{}, err
	}
	for _, e := range []error{checkCoord(in.X, "Posisi"), checkCoord(in.Y, "Posisi"), checkSize(in.W), checkSize(in.H)} {
		if e != nil {
			return Card{}, e
		}
	}
	cardID, err := newCardID(in.ID)
	if err != nil {
		return Card{}, err
	}
	u, err := s.prepareUpload(originalName, r)
	if err != nil {
		return Card{}, err
	}
	defer u.cleanup()
	if u.kind == KindImage {
		// Tinggi kartu mengikuti perbandingan sisi gambar, supaya gambar tidak terpotong saat pertama muncul.
		in.H = math.Max(minSize, math.Min(maxSize, math.Round(in.W*float64(u.height.Int32)/float64(u.width.Int32))))
	}
	var fileID pgtype.UUID
	if err := fileID.Scan(newUUIDString()); err != nil {
		return Card{}, err
	}
	content, _ := json.Marshal(CaptionContent{})
	var out Card
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		seq, err := q.BumpWbBoardSeq(ctx, boardID)
		if err != nil {
			return notFound(err)
		}
		if exists, err := q.WbCardExists(ctx, cardID); err != nil || exists {
			if err == nil {
				err = apperr.Invalid("Id kartu sudah dipakai. Muat ulang papan, lalu coba lagi.")
			}
			return err
		}
		if err := u.insert(ctx, q, fileID, boardID, actor); err != nil {
			return err
		}
		z, err := q.MaxWbCardZ(ctx, boardID)
		if err != nil {
			return err
		}
		if err := q.InsertWbCard(ctx, sqlc.InsertWbCardParams{
			ID: cardID, BoardID: boardID, Kind: u.kind, FileID: fileID, X: in.X, Y: in.Y, W: in.W, H: in.H,
			Z: math.Min(z+1, maxZ), Content: content, Plain: u.name, Seq: seq, ActorID: actor.ID,
			Style: []byte(`{}`), GroupIds: []pgtype.UUID{},
		}); err != nil {
			return err
		}
		if out, err = getCard(ctx, q, cardID); err != nil {
			return err
		}
		if err := record(ctx, q, boardID, actor.ID, "card_created", cardID, map[string]any{"kind": u.kind, "excerpt": excerpt(u.name)}); err != nil {
			return err
		}
		return s.commitUpload(u, fileID)
	})
	return out, err
}

// ReplaceFile mengganti berkas kartu gambar atau berkas (revisi 50). Posisi, ukuran, gaya, keterangan, grup, kunci,
// dan garis tetap; potongan gambar dilepas karena berlaku untuk gambar lama. Baris berkas lama tidak dihapus: job
// harian (DeleteWbUnusedFiles) membersihkannya setelah tidak dipakai kartu mana pun, jadi undo bisa memakainya lagi.
// Hasilnya juga id berkas lama, untuk langkah undo di klien.
func (s *Service) ReplaceFile(ctx context.Context, actor Actor, boardText, cardText, originalName string, r io.Reader) (Card, string, error) {
	boardID, err := s.boardID(ctx, actor, boardText)
	if err != nil {
		return Card{}, "", err
	}
	cardID, err := ParseID(cardText)
	if err != nil {
		return Card{}, "", apperr.Invalid("Id kartu tidak sah.")
	}
	u, err := s.prepareUpload(originalName, r)
	if err != nil {
		return Card{}, "", err
	}
	defer u.cleanup()
	var fileID pgtype.UUID
	if err := fileID.Scan(newUUIDString()); err != nil {
		return Card{}, "", err
	}
	var out Card
	var old string
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
		frame := cur.Kind == KindShape
		if frame {
			// Bingkai (revisi 57b): hanya gambar, dan isi bentuknya (teks, gaya, cara isi gambar) tidak berubah.
			if !FrameShapes[shapeOf(cur.Content)] {
				return apperr.Invalid("Hanya bingkai ponsel, jendela browser, dan placeholder gambar yang bisa diisi gambar.")
			}
			if u.kind != KindImage {
				return apperr.Invalid("Bingkai hanya bisa diisi gambar (JPEG, PNG, WebP, atau GIF).")
			}
		} else if err := replaceKindMatches(cur.Kind, u.kind); err != nil {
			return err
		}
		if err := u.insert(ctx, q, fileID, boardID, actor); err != nil {
			return err
		}
		if frame {
			if err := q.PatchWbCard(ctx, sqlc.PatchWbCardParams{FileID: fileID, Seq: seq, ActorID: actor.ID, ID: cardID, BoardID: boardID}); err != nil {
				return err
			}
			if out, err = getCard(ctx, q, cardID); err != nil {
				return err
			}
			old = idString(cur.FileID)
			if err := record(ctx, q, boardID, actor.ID, "file_replaced", cardID, map[string]any{"kind": cur.Kind, "excerpt": excerpt(u.name)}); err != nil {
				return err
			}
			return s.commitUpload(u, fileID)
		}
		var caption CaptionContent
		_ = json.Unmarshal(cur.Content, &caption)
		content, _ := json.Marshal(CaptionContent{Caption: caption.Caption})
		plain := caption.Caption
		if plain == "" {
			plain = u.name
		}
		if err := q.PatchWbCard(ctx, sqlc.PatchWbCardParams{
			FileID: fileID, Content: content, Plain: pgtype.Text{String: plain, Valid: true}, Seq: seq, ActorID: actor.ID,
			ID: cardID, BoardID: boardID,
		}); err != nil {
			return err
		}
		if out, err = getCard(ctx, q, cardID); err != nil {
			return err
		}
		old = idString(cur.FileID)
		if err := record(ctx, q, boardID, actor.ID, "file_replaced", cardID, map[string]any{"kind": cur.Kind, "excerpt": excerpt(u.name)}); err != nil {
			return err
		}
		return s.commitUpload(u, fileID)
	})
	return out, old, err
}

// replaceKindMatches: kartu gambar hanya menerima gambar; kartu berkas menerima berkas yang bukan gambar, supaya
// jenis kartu selalu sama dengan jenis berkasnya (salin tempel memeriksa itu).
func replaceKindMatches(cardKind, fileKind string) error {
	switch {
	case cardKind != KindImage && cardKind != KindFile:
		return apperr.Invalid("Hanya kartu gambar dan berkas yang bisa diganti berkasnya.")
	case cardKind == KindImage && fileKind != KindImage:
		return apperr.Invalid("Kartu gambar hanya bisa diganti dengan gambar (JPEG, PNG, WebP, atau GIF).")
	case cardKind == KindFile && fileKind == KindImage:
		return apperr.Invalid("Gambar ditaruh sebagai kartu gambar. Kartu berkas diganti dengan berkas yang bukan gambar.")
	}
	return nil
}

// replaceFilePatch: patch file_id (undo dan redo ganti gambar) memakai berkas yang sudah ada dengan jenis yang sama.
// Bingkai (revisi 57b) diisi berkas gambar, dan "" melepas gambarnya.
func replaceFilePatch(ctx context.Context, q *sqlc.Queries, cardKind, shape, fileText string, params *sqlc.PatchWbCardParams) error {
	if cardKind == KindShape {
		if !FrameShapes[shape] {
			return apperr.Invalid("Hanya bingkai ponsel, jendela browser, dan placeholder gambar yang bisa diisi gambar.")
		}
		if fileText == "" {
			params.ClearFile = true
			return nil
		}
	}
	fileID, err := ParseID(fileText)
	if err != nil {
		return apperr.Invalid("Id berkas tidak sah.")
	}
	f, err := q.GetWbFile(ctx, fileID)
	if err != nil {
		return apperr.Invalid("Berkas itu sudah tidak ada. Unggah ulang berkasnya.")
	}
	if cardKind == KindShape {
		if f.Kind != KindImage {
			return apperr.Invalid("Bingkai hanya bisa diisi gambar (JPEG, PNG, WebP, atau GIF).")
		}
	} else if err := replaceKindMatches(cardKind, f.Kind); err != nil {
		return err
	}
	params.FileID = fileID
	return nil
}

// newCardID memakai id dari klien bila ada (kartu sudah tampil sebelum server menjawab), atau membuat yang baru.
func newCardID(text string) (pgtype.UUID, error) {
	if text != "" {
		id, err := ParseID(text)
		if err != nil {
			return id, apperr.Invalid("Id kartu tidak sah.")
		}
		return id, nil
	}
	var id pgtype.UUID
	err := id.Scan(newUUIDString())
	return id, err
}

func (s *Service) writeTemp(data []byte) (string, error) {
	f, err := s.store.CreateTemp()
	if err != nil {
		return "", err
	}
	_, werr := f.Write(data)
	cerr := f.Close()
	if werr != nil || cerr != nil {
		_ = os.Remove(f.Name())
		return "", errors.Join(werr, cerr)
	}
	return f.Name(), nil
}

// makeThumb membuat JPEG paling besar 1200 px per sisi. Disimpan ulang dari piksel, jadi metadata EXIF (termasuk
// lokasi foto ponsel) tidak ikut; arah putar EXIF diterapkan lebih dulu. GIF dan WebP animasi memakai bingkai pertama.
func makeThumb(r io.ReadSeeker) (int, int, []byte, error) {
	cfg, _, err := image.DecodeConfig(r)
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return 0, 0, nil, apperr.Invalid("Berkas gambar rusak atau isinya tidak sesuai dengan tipenya.")
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
		return 0, 0, nil, apperr.Invalid("Resolusi gambar terlalu besar, maksimal 40 megapiksel. Kecilkan dulu gambarnya.")
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return 0, 0, nil, err
	}
	img, err := imaging.Decode(r, imaging.AutoOrientation(true))
	if err != nil {
		return 0, 0, nil, apperr.Invalid("Berkas gambar rusak atau isinya tidak sesuai dengan tipenya.")
	}
	img = imaging.Fit(img, thumbSide, thumbSide, imaging.Lanczos)
	b := img.Bounds()
	// JPEG tidak punya transparansi; tanpa latar putih, area transparan PNG menjadi hitam.
	flat := imaging.Overlay(imaging.New(b.Dx(), b.Dy(), color.White), img, image.Pt(0, 0), 1)
	var buf bytes.Buffer
	if err := imaging.Encode(&buf, flat, imaging.JPEG, imaging.JPEGQuality(thumbQuality)); err != nil {
		return 0, 0, nil, err
	}
	w, h := b.Dx(), b.Dy()
	return w, h, buf.Bytes(), nil
}

// OpenFile membuka berkas asli atau thumbnail-nya. Izinnya sama dengan membaca Whiteboard (semua yang login),
// karena berkas hanya bisa dijangkau lewat kartu di papan yang memang terbuka untuk semua.
func (s *Service) OpenFile(ctx context.Context, idText string, thumb bool) (sqlc.GetWbFileRow, *os.File, error) {
	if s.store == nil {
		return sqlc.GetWbFileRow{}, nil, ErrNoStorage
	}
	id, err := ParseID(idText)
	if err != nil {
		return sqlc.GetWbFileRow{}, nil, err
	}
	row, err := sqlc.New(s.pool).GetWbFile(ctx, id)
	if err != nil {
		return row, nil, notFound(err)
	}
	if thumb && !row.HasThumb {
		return row, nil, ErrNotFound
	}
	p, err := s.store.PathFor(idString(id))
	if thumb {
		p, err = s.store.ThumbPath(idString(id))
	}
	if err != nil {
		return row, nil, err
	}
	f, err := os.Open(p)
	if err != nil {
		return row, nil, fmt.Errorf("berkas Whiteboard %s tidak terbaca dari disk: %w", idString(id), err)
	}
	return row, f, nil
}

// ---------------------------------------------------------------------------------------------------------------
// Pratinjau tautan

func previewOf(r sqlc.UpsertWbPreviewRow) PreviewInfo {
	return PreviewInfo{ID: idString(r.ID), Title: r.Title, Description: r.Description, Site: r.SiteName,
		FinalURL: r.FinalUrl, Status: r.Status, Image: optID(r.ImageFileID)}
}

// PreviewResult: Message berisi alasan bila status blocked atau failed, aman ditampilkan.
type PreviewResult struct {
	PreviewInfo
	Message string `json:"message"`
}

// Preview mengambil (atau memakai cache) pratinjau sebuah URL. Hasil diblokir dan gagal juga disimpan, supaya kartu
// tautan bisa menunjukkan alasannya dan URL yang sama tidak diambil berulang. Kartu tautan tetap bisa dibuat tanpa
// pratinjau.
func (s *Service) Preview(ctx context.Context, actor Actor, raw string) (PreviewResult, error) {
	clean, err := CleanLinkURL(raw)
	if err != nil {
		return PreviewResult{}, err
	}
	norm := linkpreview.Normalize(clean)
	sum := sha256.Sum256([]byte(norm))
	hash := hex.EncodeToString(sum[:])
	q := sqlc.New(s.pool)
	if c, err := q.GetWbPreviewByHash(ctx, hash); err == nil {
		age := time.Since(c.FetchedAt.Time)
		if (c.Status == "ok" && age < PreviewDays*24*time.Hour) || age < previewRetry {
			return PreviewResult{PreviewInfo: previewOf(sqlc.UpsertWbPreviewRow(c)), Message: c.Message}, nil
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return PreviewResult{}, err
	}
	if !s.rate.allow(actor.ID, time.Now()) {
		return PreviewResult{}, errRateLimit
	}
	params := sqlc.UpsertWbPreviewParams{UrlHash: hash, Url: norm, FinalUrl: norm, Status: "ok"}
	page, err := s.preview.Fetch(ctx, norm)
	if err != nil {
		var pe *linkpreview.Error
		if !errors.As(err, &pe) {
			return PreviewResult{}, err
		}
		params.Status, params.Message = "failed", pe.Message
		if pe.Blocked {
			params.Status = "blocked"
		}
	} else {
		params.FinalUrl, params.Title, params.Description, params.SiteName = page.FinalURL, page.Title, page.Description, page.SiteName
		if page.ImageURL != "" {
			// Gambar pratinjau opsional: gagal mengambilnya tidak menggagalkan pratinjau.
			if id, err := s.previewImage(ctx, page.ImageURL); err == nil {
				params.ImageFileID = id
			}
		}
	}
	row, err := q.UpsertWbPreview(ctx, params)
	if err != nil {
		return PreviewResult{}, err
	}
	return PreviewResult{PreviewInfo: previewOf(row), Message: row.Message}, nil
}

// previewImage mengambil og:image lewat pemeriksaan SSRF yang sama, lalu menyimpannya ulang sebagai JPEG kecil,
// jadi peramban pengguna tidak pernah memuat gambar dari situs luar (CSP img-src tetap 'self').
func (s *Service) previewImage(ctx context.Context, raw string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if s.store == nil {
		return id, ErrNoStorage
	}
	data, err := s.preview.FetchImage(ctx, raw)
	if err != nil {
		return id, err
	}
	w, h, thumb, err := makeThumb(bytes.NewReader(data))
	if err != nil {
		return id, err
	}
	path, err := s.writeTemp(thumb)
	if err != nil {
		return id, err
	}
	defer os.Remove(path)
	if err := id.Scan(newUUIDString()); err != nil {
		return id, err
	}
	sum := sha256.Sum256(thumb)
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := sqlc.New(tx).InsertWbFile(ctx, sqlc.InsertWbFileParams{
			ID: id, Kind: "preview", OriginalName: "pratinjau.jpg", MimeType: "image/jpeg", SizeBytes: int64(len(thumb)),
			Checksum: hex.EncodeToString(sum[:]), Width: pgtype.Int4{Int32: int32(w), Valid: true},
			Height: pgtype.Int4{Int32: int32(h), Valid: true},
		}); err != nil {
			return err
		}
		return s.store.Commit(path, idString(id))
	})
	return id, err
}

// ---------------------------------------------------------------------------------------------------------------
// Tabel

// Table mengubah satu sel, baris, atau kolom tabel secara atomik, sama dengan checklist. Kunci lunak dicek di
// handler lewat hub sebelum fungsi ini dipanggil.
func (s *Service) Table(ctx context.Context, actor Actor, boardText, cardText string, op TableOp) (Card, error) {
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
		if cur.Kind != KindTable {
			return apperr.Invalid("Kartu ini bukan tabel.")
		}
		var t TableContent
		if err := json.Unmarshal(cur.Content, &t); err != nil {
			return err
		}
		if err := applyTableOp(&t, op); err != nil {
			return err
		}
		raw, err := json.Marshal(t)
		if err != nil {
			return err
		}
		content, plain, err := CleanContent(KindTable, raw)
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
		return record(ctx, q, boardID, actor.ID, "card_edited", cardID, map[string]any{"kind": KindTable, "excerpt": excerpt(plain)})
	})
	return out, err
}

// ---------------------------------------------------------------------------------------------------------------
// Garis penghubung

// ConnectorStyle: jenis ujung awal dan akhir, gaya garis, warna (nama token; kosong berarti tinta).
type ConnectorStyle struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Line  string `json:"line"`
	Color string `json:"color"`
}

// ConnectorEnd: Port tempat garis menempel di kartu atau bentuk ("" atau auto berarti sisi terdekat). X dan Y hanya
// untuk ujung bebas (tidak menempel ke kartu), dalam koordinat dunia.
type ConnectorEnd struct {
	Port string `json:"port"`
	// Row (revisi 57a): id baris tabel ERD tempat ujung menempel, dengan Port left atau right. Baris yang sudah dihapus
	// tidak membuat garis tidak sah; klien lalu menempelkan ujungnya ke kotak tabel.
	Row string   `json:"row,omitempty"`
	X   *float64 `json:"x,omitempty"`
	Y   *float64 `json:"y,omitempty"`
}

// ConnectorGeometry: jalur straight (lurus), orthogonal (siku), atau curved (melengkung), dan titik belok yang
// diseret pengguna (koordinat dunia). Model ini sudah memuat semua yang dipakai diagram W5.
type ConnectorGeometry struct {
	From   ConnectorEnd `json:"from"`
	To     ConnectorEnd `json:"to"`
	Route  string       `json:"route"`
	Points [][2]float64 `json:"points"`
	// Anchor (revisi 53): pusat ujung awal dan akhir saat titik belok disimpan. Klien memetakan titik belok dari
	// anchor ke pusat ujung sekarang, jadi garis ikut elastis saat kartunya digeser tanpa menulis ulang garis.
	Anchor *[2][2]float64 `json:"anchor,omitempty"`
}

type Connector struct {
	ID         string            `json:"id"`
	BoardID    string            `json:"board_id"`
	From       *string           `json:"from"`
	To         *string           `json:"to"`
	Geometry   ConnectorGeometry `json:"geometry"`
	Style      ConnectorStyle    `json:"style"`
	Label      string            `json:"label"`
	LabelStart string            `json:"label_start"`
	LabelEnd   string            `json:"label_end"`
	Version    int32             `json:"version"`
	Seq        int64             `json:"seq"`
	// Z: urutan tumpuk manual (revisi 53); null berarti otomatis, tepat di atas kartu ujung yang paling depan.
	Z *float64 `json:"z"`
}

const maxConnectorPoints = 50

var (
	// cf_*: ujung crow's foot ERD (revisi 54e): satu, banyak, nol atau satu, satu atau banyak, nol atau banyak.
	connectorEnds = map[string]bool{"none": true, "arrow": true, "open_arrow": true, "triangle": true, "diamond": true,
		"diamond_open": true, "circle": true, "cf_one": true, "cf_many": true, "cf_zero_one": true, "cf_one_many": true,
		"cf_zero_many": true}
	connectorLine = map[string]bool{"solid": true, "dashed": true, "dotted": true}
	// bebas (revisi 53): ruas lurus antar titik belok dengan sudut apa pun.
	// message (revisi 54d): pesan sequence, mendatar di antara garis tengah dua lifeline (digambar klien); satu-satunya
	// jalur yang boleh menghubungkan kartu ke dirinya sendiri (pesan ke diri sendiri, CHECK di migrasi 000029).
	connectorRoutes = map[string]bool{"straight": true, "orthogonal": true, "curved": true, "bebas": true, "message": true}
	connectorPorts  = map[string]bool{"": true, "auto": true, "top": true, "right": true, "bottom": true, "left": true,
		"top_left": true, "top_right": true, "bottom_left": true, "bottom_right": true}
)

func connectorOf(r sqlc.InsertWbConnectorRow) Connector {
	var st ConnectorStyle
	var g ConnectorGeometry
	_ = json.Unmarshal(r.Style, &st)
	_ = json.Unmarshal(r.Geometry, &g)
	if g.Route == "" {
		g.Route = "straight"
	}
	if g.Points == nil {
		g.Points = [][2]float64{}
	}
	var z *float64
	if r.Z.Valid {
		z = &r.Z.Float64
	}
	return Connector{ID: idString(r.ID), BoardID: idString(r.BoardID), From: optID(r.FromCardID), To: optID(r.ToCardID),
		Geometry: g, Style: st, Label: r.Label, LabelStart: r.LabelStart, LabelEnd: r.LabelEnd, Version: r.Version, Seq: r.Seq,
		Z: z}
}

// connectorZ: nil tetap otomatis; nilai diperiksa dengan batas yang sama dengan z kartu.
func connectorZ(z *float64) (pgtype.Float8, error) {
	if z == nil {
		return pgtype.Float8{}, nil
	}
	if !finite(*z) || math.Abs(*z) > maxZ {
		return pgtype.Float8{}, apperr.Invalid("Urutan tumpuk garis tidak sah.")
	}
	return pgtype.Float8{Float64: *z, Valid: true}, nil
}

func cleanStyle(st ConnectorStyle) (ConnectorStyle, error) {
	if st.Start == "" {
		st.Start = "none"
	}
	if st.End == "" {
		st.End = "arrow"
	}
	if st.Line == "" {
		st.Line = "solid"
	}
	if !connectorEnds[st.Start] || !connectorEnds[st.End] || !connectorLine[st.Line] {
		return st, apperr.Invalid("Gaya garis tidak dikenal.")
	}
	if st.Color != "" && !Colors[st.Color] {
		return st, apperr.Invalid("Warna garis tidak dikenal.")
	}
	return st, nil
}

// cleanGeometry memeriksa jalur dan ujung. attachedFrom dan attachedTo: ujung itu menempel ke kartu, jadi titik
// bebasnya dibuang; ujung yang tidak menempel wajib punya titik.
func cleanGeometry(g ConnectorGeometry, attachedFrom, attachedTo bool) (ConnectorGeometry, error) {
	if g.Route == "" {
		g.Route = "straight"
	}
	if !connectorRoutes[g.Route] {
		return g, apperr.Invalid("Jalur garis tidak dikenal.")
	}
	if len(g.Points) > maxConnectorPoints {
		return g, apperr.Invalid(fmt.Sprintf("Garis maksimal %d titik belok.", maxConnectorPoints))
	}
	for _, p := range g.Points {
		if !finite(p[0]) || !finite(p[1]) || math.Abs(p[0]) > maxCoord || math.Abs(p[1]) > maxCoord {
			return g, apperr.Invalid("Titik belok di luar papan.")
		}
	}
	if g.Points == nil {
		g.Points = [][2]float64{}
	}
	if g.Anchor != nil {
		for _, p := range g.Anchor {
			if !finite(p[0]) || !finite(p[1]) || math.Abs(p[0]) > maxCoord || math.Abs(p[1]) > maxCoord {
				return g, apperr.Invalid("Titik acuan garis di luar papan.")
			}
		}
	}
	for _, e := range []struct {
		end      *ConnectorEnd
		attached bool
	}{{&g.From, attachedFrom}, {&g.To, attachedTo}} {
		if !connectorPorts[e.end.Port] {
			return g, apperr.Invalid("Titik sambung garis tidak dikenal.")
		}
		if e.end.Row != "" && (!e.attached || !ValidRowID(e.end.Row) || (e.end.Port != "left" && e.end.Port != "right")) {
			return g, apperr.Invalid("Ujung garis ke kolom tabel harus di sisi kiri atau kanan kolom itu.")
		}
		if e.attached {
			e.end.X, e.end.Y = nil, nil
			continue
		}
		e.end.Port = ""
		if e.end.X == nil || e.end.Y == nil || math.Abs(*e.end.X) > maxCoord || math.Abs(*e.end.Y) > maxCoord ||
			!finite(*e.end.X) || !finite(*e.end.Y) {
			return g, apperr.Invalid("Ujung garis yang tidak menempel butuh titik di papan.")
		}
	}
	return g, nil
}

func cleanLabel(v string) (string, error) {
	v = strings.Join(strings.Fields(stripControl(v, false)), " ")
	if utf8.RuneCountInString(v) > MaxLabel {
		return "", apperr.Invalid(fmt.Sprintf("Label garis maksimal %d karakter.", MaxLabel))
	}
	return v, nil
}

func cleanLabels(vals ...*string) error {
	for _, v := range vals {
		if v == nil {
			continue
		}
		c, err := cleanLabel(*v)
		if err != nil {
			return err
		}
		*v = c
	}
	return nil
}

// ConnectorInput: From atau To kosong berarti ujung itu bebas (titiknya di Geometry).
type ConnectorInput struct {
	From       string            `json:"from"`
	To         string            `json:"to"`
	Geometry   ConnectorGeometry `json:"geometry"`
	Style      ConnectorStyle    `json:"style"`
	Label      string            `json:"label"`
	LabelStart string            `json:"label_start"`
	LabelEnd   string            `json:"label_end"`
	Z          *float64          `json:"z"`
}

func optCard(text string) (pgtype.UUID, error) {
	if text == "" {
		return pgtype.UUID{}, nil
	}
	id, err := ParseID(text)
	if err != nil {
		return id, apperr.Invalid("Kartu yang dihubungkan tidak ada di papan ini.")
	}
	return id, nil
}

// CreateConnector menghubungkan dua kartu hidup yang berbeda di papan yang sama, atau kartu dengan titik bebas.
func (s *Service) CreateConnector(ctx context.Context, actor Actor, boardText string, in ConnectorInput) (Connector, error) {
	boardID, err := s.boardID(ctx, actor, boardText)
	if err != nil {
		return Connector{}, err
	}
	from, err := optCard(in.From)
	if err != nil {
		return Connector{}, err
	}
	to, err := optCard(in.To)
	if err != nil {
		return Connector{}, err
	}
	if from.Valid && from == to && in.Geometry.Route != "message" {
		return Connector{}, apperr.Invalid("Garis harus menghubungkan dua kartu yang berbeda.")
	}
	st, err := cleanStyle(in.Style)
	if err != nil {
		return Connector{}, err
	}
	geo, err := cleanGeometry(in.Geometry, from.Valid, to.Valid)
	if err != nil {
		return Connector{}, err
	}
	if err := cleanLabels(&in.Label, &in.LabelStart, &in.LabelEnd); err != nil {
		return Connector{}, err
	}
	z, err := connectorZ(in.Z)
	if err != nil {
		return Connector{}, err
	}
	style, _ := json.Marshal(st)
	geometry, _ := json.Marshal(geo)
	var out Connector
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		seq, err := q.BumpWbBoardSeq(ctx, boardID)
		if err != nil {
			return notFound(err)
		}
		for _, id := range []pgtype.UUID{from, to} {
			if !id.Valid {
				continue
			}
			if _, err := q.GetWbCardForUpdate(ctx, sqlc.GetWbCardForUpdateParams{ID: id, BoardID: boardID}); err != nil {
				return apperr.Invalid("Kartu yang dihubungkan tidak ada di papan ini.")
			}
		}
		row, err := q.InsertWbConnector(ctx, sqlc.InsertWbConnectorParams{BoardID: boardID, FromCardID: from, ToCardID: to,
			Geometry: geometry, Style: style, Label: in.Label, LabelStart: in.LabelStart, LabelEnd: in.LabelEnd, Seq: seq,
			ActorID: actor.ID, Z: z})
		if err != nil {
			return err
		}
		out = connectorOf(row)
		return record(ctx, q, boardID, actor.ID, "connector_created", row.ID, map[string]any{"label": in.Label})
	})
	return out, err
}

type ConnectorPatch struct {
	Style      *ConnectorStyle    `json:"style"`
	Geometry   *ConnectorGeometry `json:"geometry"`
	Label      *string            `json:"label"`
	LabelStart *string            `json:"label_start"`
	LabelEnd   *string            `json:"label_end"`
	// Z mengatur urutan manual; ZAuto mengembalikannya ke otomatis (dipakai undo dari otomatis ke manual).
	Z     *float64 `json:"z"`
	ZAuto bool     `json:"z_auto"`
}

// UpdateConnector mengganti gaya, jalur, atau label; yang terakhir sampai menang, sama dengan tata letak kartu.
func (s *Service) UpdateConnector(ctx context.Context, actor Actor, boardText, idText string, p ConnectorPatch) (Connector, error) {
	boardID, err := s.boardID(ctx, actor, boardText)
	if err != nil {
		return Connector{}, err
	}
	id, err := ParseID(idText)
	if err != nil {
		return Connector{}, err
	}
	params := sqlc.UpdateWbConnectorParams{ActorID: actor.ID, ID: id, BoardID: boardID}
	if p.Style != nil {
		st, err := cleanStyle(*p.Style)
		if err != nil {
			return Connector{}, err
		}
		params.Style, _ = json.Marshal(st)
	}
	if err := cleanLabels(p.Label, p.LabelStart, p.LabelEnd); err != nil {
		return Connector{}, err
	}
	text := func(v *string) pgtype.Text {
		if v == nil {
			return pgtype.Text{}
		}
		return pgtype.Text{String: *v, Valid: true}
	}
	params.Label, params.LabelStart, params.LabelEnd = text(p.Label), text(p.LabelStart), text(p.LabelEnd)
	if p.Z != nil && p.ZAuto {
		return Connector{}, apperr.Invalid("Pilih urutan manual atau otomatis, bukan keduanya.")
	}
	if p.Z != nil || p.ZAuto {
		params.SetZ = true
		if params.Z, err = connectorZ(p.Z); err != nil {
			return Connector{}, err
		}
	}
	var out Connector
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if params.Seq, err = q.BumpWbBoardSeq(ctx, boardID); err != nil {
			return notFound(err)
		}
		if p.Geometry != nil {
			ends, err := q.GetWbConnectorEnds(ctx, sqlc.GetWbConnectorEndsParams{ID: id, BoardID: boardID})
			if err != nil {
				return notFound(err)
			}
			geo, err := cleanGeometry(*p.Geometry, ends.FromCardID.Valid, ends.ToCardID.Valid)
			if err != nil {
				return err
			}
			if ends.FromCardID.Valid && ends.FromCardID == ends.ToCardID && geo.Route != "message" {
				return apperr.Invalid("Pesan ke diri sendiri harus tetap memakai jalur pesan.")
			}
			params.Geometry, _ = json.Marshal(geo)
		}
		row, err := q.UpdateWbConnector(ctx, params)
		if err != nil {
			return notFound(err)
		}
		out = connectorOf(sqlc.InsertWbConnectorRow(row))
		return nil
	})
	return out, err
}

// DeleteConnector memasukkan garis ke Tempat sampah sebagai batch sendiri, jadi bisa dipulihkan.
func (s *Service) DeleteConnector(ctx context.Context, actor Actor, boardText, idText string) (string, error) {
	boardID, err := s.boardID(ctx, actor, boardText)
	if err != nil {
		return "", err
	}
	id, err := ParseID(idText)
	if err != nil {
		return "", err
	}
	batch, err := newBatch()
	if err != nil {
		return "", err
	}
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		seq, err := q.BumpWbBoardSeq(ctx, boardID)
		if err != nil {
			return notFound(err)
		}
		rows, err := q.TrashWbConnectors(ctx, sqlc.TrashWbConnectorsParams{ActorID: actor.ID, Batch: batch, Seq: seq,
			BoardID: boardID, Ids: []pgtype.UUID{id}, Cards: []pgtype.UUID{}})
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return ErrNotFound
		}
		return record(ctx, q, boardID, actor.ID, "connector_deleted", id, map[string]any{"batch": idString(batch)})
	})
	return idString(batch), err
}

// ---------------------------------------------------------------------------------------------------------------
// Komentar

type Comment struct {
	ID        string  `json:"id"`
	CardID    string  `json:"card_id"`
	UserID    int64   `json:"user_id"`
	UserName  string  `json:"user_name"`
	AvatarID  *string `json:"avatar_id"`
	Body      string  `json:"body"`
	CreatedAt string  `json:"created_at"`
	EditedAt  *string `json:"edited_at"`
}

func cleanComment(v string) (string, error) {
	v = strings.TrimSpace(stripControl(v, true))
	if v == "" {
		return "", apperr.Invalid("Komentar tidak boleh kosong.")
	}
	if utf8.RuneCountInString(v) > MaxComment {
		return "", apperr.Invalid(fmt.Sprintf("Komentar maksimal %d karakter.", MaxComment))
	}
	return v, nil
}

// cardOnBoard memastikan kartu hidup dan berada di papan itu, supaya komentar tidak bisa dibaca atau ditulis lewat
// papan lain.
func (s *Service) cardOnBoard(ctx context.Context, q *sqlc.Queries, actor Actor, boardText, cardText string) (pgtype.UUID, pgtype.UUID, error) {
	boardID, err := s.boardID(ctx, actor, boardText)
	if err != nil {
		return boardID, pgtype.UUID{}, err
	}
	cardID, err := ParseID(cardText)
	if err != nil {
		return boardID, cardID, err
	}
	c, err := getCard(ctx, q, cardID)
	if err != nil {
		return boardID, cardID, err
	}
	if c.BoardID != idString(boardID) {
		return boardID, cardID, ErrNotFound
	}
	return boardID, cardID, nil
}

func (s *Service) Comments(ctx context.Context, actor Actor, boardText, cardText string) ([]Comment, error) {
	q := sqlc.New(s.pool)
	_, cardID, err := s.cardOnBoard(ctx, q, actor, boardText, cardText)
	if err != nil {
		return nil, err
	}
	rows, err := q.ListWbComments(ctx, cardID)
	if err != nil {
		return nil, err
	}
	out := make([]Comment, 0, len(rows))
	for _, r := range rows {
		c := Comment{ID: idString(r.ID), CardID: idString(r.CardID), UserID: r.UserID, UserName: r.UserName,
			AvatarID: optID(r.AvatarID), Body: r.Body, CreatedAt: ts(r.CreatedAt)}
		if r.EditedAt.Valid {
			e := ts(r.EditedAt)
			c.EditedAt = &e
		}
		out = append(out, c)
	}
	return out, nil
}

// touchCard menaikkan seq kartu (jumlah komentar berubah) dan mengembalikan keadaan terbarunya untuk disiarkan.
func touchCard(ctx context.Context, q *sqlc.Queries, boardID, cardID pgtype.UUID) (Card, error) {
	seq, err := q.BumpWbBoardSeq(ctx, boardID)
	if err != nil {
		return Card{}, notFound(err)
	}
	if _, err := q.TouchWbCard(ctx, sqlc.TouchWbCardParams{Seq: seq, ID: cardID}); err != nil {
		return Card{}, notFound(err)
	}
	return getCard(ctx, q, cardID)
}

// AddComment menambah komentar dan mengembalikan kartu dengan jumlah komentar terbaru.
func (s *Service) AddComment(ctx context.Context, actor Actor, boardText, cardText, body string) (Card, error) {
	body, err := cleanComment(body)
	if err != nil {
		return Card{}, err
	}
	var out Card
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		boardID, cardID, err := s.cardOnBoard(ctx, q, actor, boardText, cardText)
		if err != nil {
			return err
		}
		if _, err := q.InsertWbComment(ctx, sqlc.InsertWbCommentParams{CardID: cardID, UserID: actor.ID, Body: body}); err != nil {
			return err
		}
		if out, err = touchCard(ctx, q, boardID, cardID); err != nil {
			return err
		}
		return record(ctx, q, boardID, actor.ID, "comment_added", cardID, map[string]any{"excerpt": excerpt(body)})
	})
	return out, err
}

// EditComment: hanya penulisnya. DeleteComment: penulisnya atau Manajer (keputusan bawaan W3).
func (s *Service) EditComment(ctx context.Context, actor Actor, idText, body string) (Card, error) {
	return s.changeComment(ctx, actor, idText, &body)
}

func (s *Service) DeleteComment(ctx context.Context, actor Actor, idText string) (Card, error) {
	return s.changeComment(ctx, actor, idText, nil)
}

func (s *Service) changeComment(ctx context.Context, actor Actor, idText string, body *string) (Card, error) {
	id, err := ParseID(idText)
	if err != nil {
		return Card{}, err
	}
	if body != nil {
		if *body, err = cleanComment(*body); err != nil {
			return Card{}, err
		}
	}
	var out Card
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		cur, err := q.GetWbCommentForUpdate(ctx, id)
		if err != nil {
			return notFound(err)
		}
		mine := cur.UserID == actor.ID
		switch {
		case body != nil && !mine:
			return ErrForbidden
		case body == nil && !mine && !actor.Manager:
			return ErrForbidden
		case body != nil:
			err = q.UpdateWbComment(ctx, sqlc.UpdateWbCommentParams{Body: *body, ID: id})
		default:
			err = q.SoftDeleteWbComment(ctx, sqlc.SoftDeleteWbCommentParams{ActorID: actor.ID, ID: id})
		}
		if err != nil {
			return err
		}
		out, err = touchCard(ctx, q, cur.BoardID, cur.CardID)
		return err
	})
	return out, err
}

// ---------------------------------------------------------------------------------------------------------------
// Pindah ke papan lain

type BoardOption struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	ParentID *string `json:"parent_id"`
	IsRoot   bool    `json:"is_root"`
}

func (s *Service) Boards(ctx context.Context) ([]BoardOption, error) {
	rows, err := sqlc.New(s.pool).ListWbBoards(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]BoardOption, 0, len(rows))
	for _, r := range rows {
		out = append(out, BoardOption{ID: idString(r.ID), Title: r.Title, ParentID: optID(r.ParentID), IsRoot: r.IsRoot})
	}
	return out, nil
}

// MoveResult: Moved berisi id kartu yang keluar dari papan asal (termasuk isi kolom) beserta garis yang ikut pindah
// atau masuk Tempat sampah (ConnectorsGone). Papan tujuan dimuat ulang klien lewat siaran resync.
type MoveResult struct {
	From           string   `json:"from"`
	To             string   `json:"to"`
	Moved          []string `json:"moved"`
	ConnectorsGone []string `json:"connectors_gone"`
	Batch          string   `json:"batch"`
}

// MoveCards memindah kartu (dan isi kolomnya) ke papan lain. Kartu papan membawa papan anaknya; papan tidak bisa
// dipindah ke dalam dirinya sendiri atau melewati batas kedalaman. Garis yang kedua ujungnya ikut pindah ikut
// pindah; garis yang tertinggal satu ujungnya masuk Tempat sampah (bisa dipulihkan setelah kartunya kembali).
func (s *Service) MoveCards(ctx context.Context, actor Actor, boardText string, ids []string, toText string) (MoveResult, error) {
	var out MoveResult
	from, err := s.boardID(ctx, actor, boardText)
	if err != nil {
		return out, err
	}
	to, err := s.boardID(ctx, actor, toText)
	if err != nil {
		return out, apperr.Invalid("Papan tujuan tidak ditemukan.")
	}
	if from == to {
		return out, apperr.Invalid("Kartu sudah berada di papan ini.")
	}
	if len(ids) == 0 || len(ids) > MaxBatch {
		return out, apperr.Invalid(fmt.Sprintf("Pilih 1 sampai %d kartu.", MaxBatch))
	}
	uuids := make([]pgtype.UUID, 0, len(ids))
	for _, t := range ids {
		u, err := ParseID(t)
		if err != nil {
			return out, err
		}
		uuids = append(uuids, u)
	}
	out = MoveResult{From: idString(from), To: idString(to), Moved: []string{}, ConnectorsGone: []string{}}
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		// Urutan kunci baris papan tetap (UUID lebih kecil dulu) supaya dua pindahan berlawanan arah tidak deadlock.
		first, second := from, to
		if idString(to) < idString(from) {
			first, second = to, from
		}
		seqs := map[pgtype.UUID]int64{}
		for _, b := range []pgtype.UUID{first, second} {
			seq, err := q.BumpWbBoardSeq(ctx, b)
			if err != nil {
				if b == to {
					return apperr.Invalid("Papan tujuan tidak ditemukan.")
				}
				return notFound(err)
			}
			seqs[b] = seq
		}
		children, err := q.ListWbColumnChildren(ctx, uuids)
		if err != nil {
			return err
		}
		all := append(append([]pgtype.UUID{}, uuids...), children...)
		// Kartu terkunci (termasuk isi kolom yang ikut pindah) tidak boleh pindah papan; keputusan Manajer revisi 51.
		locked, err := q.ListWbLockedCards(ctx, sqlc.ListWbLockedCardsParams{BoardID: from, Ids: all})
		if err != nil {
			return err
		}
		if len(locked) > 0 {
			return apperr.Invalid(fmt.Sprintf("%d kartu terkunci. Buka kunci dulu, lalu pindahkan lagi.", len(locked)))
		}
		crumbs, err := ancestors(ctx, q, to)
		if err != nil {
			return err
		}
		// Tata letak: rombongan kartu ditaruh di kanan isi papan tujuan, sejajar bagian atasnya.
		minX, minY := math.Inf(1), math.Inf(1)
		refs := []pgtype.UUID{}
		for _, id := range uuids {
			c, err := q.GetWbCardForUpdate(ctx, sqlc.GetWbCardForUpdateParams{ID: id, BoardID: from})
			if err != nil {
				return notFound(err)
			}
			full, err := getCard(ctx, q, id)
			if err != nil {
				return err
			}
			if full.ColumnID == nil {
				minX, minY = math.Min(minX, full.X), math.Min(minY, full.Y)
			}
			if c.RefBoardID.Valid {
				refs = append(refs, c.RefBoardID)
			}
		}
		for _, ref := range refs {
			for _, cr := range crumbs {
				if cr.ID == idString(ref) {
					return apperr.Invalid("Papan tidak bisa dipindah ke dalam dirinya sendiri.")
				}
			}
			depth, err := q.WbSubtreeDepth(ctx, ref)
			if err != nil {
				return err
			}
			if len(crumbs)+int(depth) > MaxDepth {
				return apperr.Invalid(fmt.Sprintf("Papan bertingkat paling dalam %d tingkat.", MaxDepth))
			}
		}
		bounds, err := q.ListWbCardBounds(ctx, to)
		if err != nil {
			return err
		}
		dx, dy := 0.0, 0.0
		if !math.IsInf(minX, 1) && bounds.N > 0 {
			dx, dy = bounds.RightEdge+80-minX, bounds.TopEdge-minY
		}
		moved, err := q.MoveWbCards(ctx, sqlc.MoveWbCardsParams{ToBoard: to, Seq: seqs[to], ActorID: actor.ID, Dx: dx, Dy: dy,
			Ids: all, FromBoard: from})
		if err != nil {
			return err
		}
		if len(moved) == 0 {
			return ErrNotFound
		}
		if len(refs) > 0 {
			if err := q.ReparentWbBoards(ctx, sqlc.ReparentWbBoardsParams{ParentID: to, ActorID: actor.ID, Ids: refs}); err != nil {
				return err
			}
		}
		conns, err := q.MoveWbConnectors(ctx, sqlc.MoveWbConnectorsParams{ToBoard: to, Seq: seqs[to], ActorID: actor.ID,
			FromBoard: from, Ids: all})
		if err != nil {
			return err
		}
		batch, err := newBatch()
		if err != nil {
			return err
		}
		gone, err := q.TrashWbConnectors(ctx, sqlc.TrashWbConnectorsParams{ActorID: actor.ID, Batch: batch, Seq: seqs[from],
			BoardID: from, Ids: []pgtype.UUID{}, Cards: all})
		if err != nil {
			return err
		}
		if len(gone) > 0 {
			out.Batch = idString(batch)
		}
		if err := q.MarkWbResync(ctx, from); err != nil {
			return err
		}
		for _, m := range moved {
			out.Moved = append(out.Moved, idString(m.ID))
		}
		for _, c := range append(conns, gone...) {
			out.ConnectorsGone = append(out.ConnectorsGone, idString(c))
		}
		target := crumbs[len(crumbs)-1].Title
		if err := record(ctx, q, from, actor.ID, "cards_moved_out", pgtype.UUID{}, map[string]any{"count": len(moved), "board": target}); err != nil {
			return err
		}
		return record(ctx, q, to, actor.ID, "cards_moved_in", pgtype.UUID{}, map[string]any{"count": len(moved)})
	})
	return out, err
}
