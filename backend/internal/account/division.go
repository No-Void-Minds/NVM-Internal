package account

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
)

var (
	ErrDivisionNotFound  = errors.New("divisi tidak ditemukan")
	ErrDivisionNameTaken = errors.New("nama divisi sudah dipakai")
)

func (s *Service) ListDivisions(ctx context.Context) ([]sqlc.ListDivisionsRow, error) {
	return sqlc.New(s.pool).ListDivisions(ctx)
}

func (s *Service) CreateDivision(ctx context.Context, actor Actor, name string) (int64, error) {
	name, err := cleanDivisionName(name)
	if err != nil {
		return 0, err
	}
	var id int64
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		id, err = q.CreateDivision(ctx, name)
		if isUniqueViolation(err) {
			return ErrDivisionNameTaken
		}
		if err != nil {
			return err
		}
		return recordDivision(ctx, q, actor, "division_created", id, nil, map[string]any{"name": name})
	})
	return id, err
}

func (s *Service) RenameDivision(ctx context.Context, actor Actor, id int64, name string) error {
	name, err := cleanDivisionName(name)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		d, err := getDivisionForUpdate(ctx, q, id)
		if err != nil {
			return err
		}
		if d.Name == name {
			return nil
		}
		err = q.RenameDivision(ctx, sqlc.RenameDivisionParams{ID: id, Name: name})
		if isUniqueViolation(err) {
			return ErrDivisionNameTaken
		}
		if err != nil {
			return err
		}
		return recordDivision(ctx, q, actor, "division_renamed", id,
			map[string]any{"name": d.Name}, map[string]any{"name": name})
	})
}

// SetDivisionActive tidak menyentuh akun di divisi itu: akun lama tetap di divisinya,
// hanya divisi nonaktif yang tidak bisa dipilih untuk akun baru atau perpindahan divisi.
func (s *Service) SetDivisionActive(ctx context.Context, actor Actor, id int64, active bool) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		d, err := getDivisionForUpdate(ctx, q, id)
		if err != nil {
			return err
		}
		if d.IsActive == active {
			return nil
		}
		if err := q.SetDivisionActive(ctx, sqlc.SetDivisionActiveParams{ID: id, IsActive: active}); err != nil {
			return err
		}
		action := "division_activated"
		if !active {
			action = "division_deactivated"
		}
		return recordDivision(ctx, q, actor, action, id, nil, nil)
	})
}

func cleanDivisionName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", invalid("Nama divisi wajib diisi.")
	}
	if utf8.RuneCountInString(name) > 50 {
		return "", invalid("Nama divisi maksimal 50 karakter.")
	}
	return name, nil
}

func getDivisionForUpdate(ctx context.Context, q *sqlc.Queries, id int64) (sqlc.GetDivisionForUpdateRow, error) {
	d, err := q.GetDivisionForUpdate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrDivisionNotFound
	}
	return d, err
}

func recordDivision(ctx context.Context, q *sqlc.Queries, actor Actor, action string, id int64, before, after map[string]any) error {
	e := audit.Entry{
		ActorID:  &actor.ID,
		Action:   action,
		Entity:   "division",
		EntityID: strconv.FormatInt(id, 10),
		IP:       actor.IP,
	}
	if before != nil {
		e.Old = before
	}
	if after != nil {
		e.New = after
	}
	return audit.Record(ctx, q, e)
}
