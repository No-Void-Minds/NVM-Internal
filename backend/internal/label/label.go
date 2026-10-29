package label

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
)

var (
	ErrNotFound  = errors.New("label tidak ditemukan")
	ErrNameTaken = errors.New("nama label sudah dipakai")
)

// Palette adalah warna latar yang boleh dipakai label. Teks label selalu warna ink, dan semua warna
// ini kontrasnya di atas 12:1 terhadap ink, jadi nama label tetap terbaca apa pun warnanya.
var Palette = []string{"#E6E1D7", "#D5E3C8", "#D3E2EC", "#E1DAEC", "#F2D8D8", "#F3E5BF", "#F2D9C2", "#DADDE0"}

type Input struct {
	Name       string
	Color      string
	DivisionID *int64 // nil berarti label berlaku untuk semua divisi
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) List(ctx context.Context) ([]sqlc.ListLabelsRow, error) {
	return sqlc.New(s.pool).ListLabels(ctx)
}

func (s *Service) Create(ctx context.Context, actor audit.Actor, in Input) (int64, error) {
	in, err := clean(in)
	if err != nil {
		return 0, err
	}
	var id int64
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := requireActiveDivision(ctx, q, in.DivisionID); err != nil {
			return err
		}
		id, err = q.CreateLabel(ctx, sqlc.CreateLabelParams{Name: in.Name, Color: in.Color, DivisionID: toInt8(in.DivisionID)})
		if isUniqueViolation(err) {
			return ErrNameTaken
		}
		if err != nil {
			return err
		}
		return audit.Change(ctx, q, actor, "label_created", "label", strconv.FormatInt(id, 10), nil,
			map[string]any{"name": in.Name, "color": in.Color, "division_id": in.DivisionID})
	})
	return id, err
}

func (s *Service) Update(ctx context.Context, actor audit.Actor, id int64, in Input) error {
	in, err := clean(in)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		cur, err := getForUpdate(ctx, q, id)
		if err != nil {
			return err
		}
		curDivision := fromInt8(cur.DivisionID)

		before, after := map[string]any{}, map[string]any{}
		if cur.Name != in.Name {
			before["name"], after["name"] = cur.Name, in.Name
		}
		if cur.Color != in.Color {
			before["color"], after["color"] = cur.Color, in.Color
		}
		if !sameDivision(curDivision, in.DivisionID) {
			// Divisi lama yang sudah nonaktif boleh tetap dipakai; hanya pindah divisi yang harus ke divisi aktif.
			if err := requireActiveDivision(ctx, q, in.DivisionID); err != nil {
				return err
			}
			before["division_id"], after["division_id"] = curDivision, in.DivisionID
		}
		if len(after) == 0 {
			return nil
		}

		err = q.UpdateLabel(ctx, sqlc.UpdateLabelParams{ID: id, Name: in.Name, Color: in.Color, DivisionID: toInt8(in.DivisionID)})
		if isUniqueViolation(err) {
			return ErrNameTaken
		}
		if err != nil {
			return err
		}
		return audit.Change(ctx, q, actor, "label_updated", "label", strconv.FormatInt(id, 10), before, after)
	})
}

// SetActive tidak melepas label dari task: label nonaktif tetap tampil di task lama,
// hanya tidak muncul lagi di pilihan (AC-14).
func (s *Service) SetActive(ctx context.Context, actor audit.Actor, id int64, active bool) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		cur, err := getForUpdate(ctx, q, id)
		if err != nil {
			return err
		}
		if cur.IsActive == active {
			return nil
		}
		if err := q.SetLabelActive(ctx, sqlc.SetLabelActiveParams{ID: id, IsActive: active}); err != nil {
			return err
		}
		action := "label_activated"
		if !active {
			action = "label_deactivated"
		}
		return audit.Change(ctx, q, actor, action, "label", strconv.FormatInt(id, 10), nil, nil)
	})
}

func clean(in Input) (Input, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Color = strings.ToUpper(strings.TrimSpace(in.Color))
	if in.Name == "" {
		return in, apperr.Invalid("Nama label wajib diisi.")
	}
	if utf8.RuneCountInString(in.Name) > 50 {
		return in, apperr.Invalid("Nama label maksimal 50 karakter.")
	}
	if !inPalette(in.Color) {
		return in, apperr.Invalid("Pilih warna dari palet yang tersedia.")
	}
	return in, nil
}

func inPalette(color string) bool {
	for _, c := range Palette {
		if c == color {
			return true
		}
	}
	return false
}

func requireActiveDivision(ctx context.Context, q *sqlc.Queries, id *int64) error {
	if id == nil {
		return nil
	}
	_, err := q.GetActiveDivision(ctx, *id)
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.Invalid("Divisi tidak ditemukan atau sudah nonaktif.")
	}
	return err
}

func getForUpdate(ctx context.Context, q *sqlc.Queries, id int64) (sqlc.GetLabelForUpdateRow, error) {
	l, err := q.GetLabelForUpdate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return l, ErrNotFound
	}
	return l, err
}

func toInt8(id *int64) pgtype.Int8 {
	if id == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *id, Valid: true}
}

func fromInt8(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func sameDivision(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
