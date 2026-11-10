package task

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/db/sqlc"
)

const (
	ArchivePageSize = 25
	// Di bawah tiga huruf indeks trigram tidak terpakai dan hasilnya terlalu luas untuk berguna.
	MinKeywordLen = 3
	maxKeywordLen = 100
)

const (
	SortNewest  = "newest"
	SortOldest  = "oldest"
	SortUrgency = "urgency"
)

// ArchiveFilter mengikuti query string Arsip. Nilai kosong berarti filter itu tidak dipakai.
type ArchiveFilter struct {
	PICIDs       []int64
	PICUsernames []string
	DivisionID   *int64
	LabelID      *int64
	Urgency      string
	Keyword      string
	// SolvedFrom dan SolvedTo berupa tanggal WIB (YYYY-MM-DD), keduanya inklusif.
	SolvedFrom string
	SolvedTo   string
	Sort       string
	Page       int
}

type ArchivePage struct {
	Tasks []Summary
	Total int
	// Page adalah halaman yang benar-benar dikembalikan. Berbeda dari yang diminta bila halaman itu sudah
	// melewati akhir daftar (mis. hasil menyusut); halaman pertama dikembalikan, bukan daftar kosong.
	Page int
}

// Karakter khusus LIKE di-escape supaya "100%" atau "a_b" dicari apa adanya, bukan sebagai pola.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (s *Service) Archive(ctx context.Context, f ArchiveFilter) (ArchivePage, error) {
	params, err := f.params()
	if err != nil {
		return ArchivePage{}, err
	}
	q := sqlc.New(s.pool)

	page := max(f.Page, 1)
	rows, err := fetchArchive(ctx, q, params, page)
	if err != nil {
		return ArchivePage{}, err
	}
	if len(rows) == 0 && page > 1 {
		page = 1
		if rows, err = fetchArchive(ctx, q, params, page); err != nil {
			return ArchivePage{}, err
		}
	}

	total := 0
	list := make([]sqlc.ListTasksRow, len(rows))
	for i, r := range rows {
		list[i] = sqlc.ListTasksRow{
			ID: r.ID, Title: r.Title, Status: r.Status, Urgency: r.Urgency,
			DivisionID: r.DivisionID, DivisionName: r.DivisionName,
			StartDate: r.StartDate, DueDate: r.DueDate, SolvedDate: r.SolvedDate, Version: r.Version,
		}
	}
	if len(rows) > 0 {
		total = int(rows[0].TotalCount)
	}
	summaries, err := attachPeopleAndLabels(ctx, q, list)
	if err != nil {
		return ArchivePage{}, err
	}
	return ArchivePage{Tasks: summaries, Total: total, Page: page}, nil
}

func fetchArchive(ctx context.Context, q *sqlc.Queries, p sqlc.ListArchiveTasksParams, page int) ([]sqlc.ListArchiveTasksRow, error) {
	p.RowLimit = ArchivePageSize
	p.RowOffset = int32((page - 1) * ArchivePageSize)
	return q.ListArchiveTasks(ctx, p)
}

func (f ArchiveFilter) params() (sqlc.ListArchiveTasksParams, error) {
	p := sqlc.ListArchiveTasksParams{
		PicIds: append([]int64{}, f.PICIDs...),
		// Slice nil terkirim sebagai NULL, dan cardinality(NULL) bukan 0, sehingga semua task akan tersaring habis.
		PicUsernames: []string{},
		Sort:         SortNewest,
	}
	for _, u := range f.PICUsernames {
		if u = strings.ToLower(strings.TrimSpace(u)); u != "" {
			p.PicUsernames = append(p.PicUsernames, u)
		}
	}

	switch f.Urgency {
	case "":
	case "low", "medium", "high", "urgent":
		p.Urgency = sqlc.NullTaskUrgency{TaskUrgency: sqlc.TaskUrgency(f.Urgency), Valid: true}
	default:
		return p, apperr.Invalid("Filter urgency tidak dikenal.")
	}

	switch f.Sort {
	case "":
	case SortNewest, SortOldest, SortUrgency:
		p.Sort = f.Sort
	default:
		return p, apperr.Invalid("Urutan tidak dikenal.")
	}

	if f.DivisionID != nil {
		p.DivisionID = pgtype.Int8{Int64: *f.DivisionID, Valid: true}
	}
	if f.LabelID != nil {
		p.LabelID = pgtype.Int8{Int64: *f.LabelID, Valid: true}
	}

	pattern, err := keywordPattern(f.Keyword)
	if err != nil {
		return p, err
	}
	p.Pattern = pattern

	from, err := parseDay(f.SolvedFrom)
	if err != nil {
		return p, err
	}
	to, err := parseDay(f.SolvedTo)
	if err != nil {
		return p, err
	}
	if from != nil && to != nil && from.After(*to) {
		return p, apperr.Invalid("Tanggal awal tidak boleh setelah tanggal akhir.")
	}
	if from != nil {
		p.SolvedFrom = pgtype.Timestamptz{Time: *from, Valid: true}
	}
	if to != nil {
		// Batas atas eksklusif di tengah malam berikutnya, supaya seluruh hari terakhir ikut.
		p.SolvedTo = pgtype.Timestamptz{Time: to.AddDate(0, 0, 1), Valid: true}
	}
	return p, nil
}

// keywordPattern dipakai Arsip dan papan, supaya satu kata kunci selalu menemukan hal yang sama di keduanya.
func keywordPattern(raw string) (pgtype.Text, error) {
	kw := strings.TrimSpace(raw)
	if kw == "" {
		return pgtype.Text{}, nil
	}
	n := utf8.RuneCountInString(kw)
	if n < MinKeywordLen {
		return pgtype.Text{}, apperr.Invalid("Kata kunci minimal 3 huruf.")
	}
	if n > maxKeywordLen {
		return pgtype.Text{}, apperr.Invalid("Kata kunci terlalu panjang.")
	}
	return pgtype.Text{String: "%" + likeEscaper.Replace(kw) + "%", Valid: true}, nil
}

// parseDay membaca tanggal WIB; solved_date disimpan dengan zona waktu, jadi batas harinya harus WIB, bukan UTC.
func parseDay(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02", v, officeTZ)
	if err != nil {
		return nil, apperr.Invalid("Tanggal harus berformat YYYY-MM-DD.")
	}
	return &t, nil
}
