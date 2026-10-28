// Package account berisi pengelolaan akun oleh Admin dan Manajer (FR-AUTH-04).
package account

import (
	"context"
	"errors"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/audit"
	"nvm-internal/internal/auth"
	"nvm-internal/internal/db/sqlc"
)

var (
	ErrNotFound       = errors.New("akun tidak ditemukan")
	ErrUsernameTaken  = errors.New("username sudah dipakai")
	ErrSelfDeactivate = errors.New("tidak bisa menonaktifkan akun sendiri")
	ErrLastManager    = errors.New("harus ada minimal satu Admin atau Manajer aktif")
)

// ValidationError membawa pesan yang aman ditampilkan langsung ke pengguna.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func invalid(msg string) error { return &ValidationError{Message: msg} }

const (
	minPasswordLen = 8
	maxPasswordLen = 72
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{3,50}$`)

// Actor adalah pengelola yang melakukan aksi, dicatat sebagai pelaku di audit log.
type Actor struct {
	ID int64
	IP *netip.Addr
}

type CreateInput struct {
	Name       string
	Username   string
	Role       sqlc.UserRole
	DivisionID int64
	Password   string
}

type UpdateInput struct {
	Name       string
	Role       sqlc.UserRole
	DivisionID int64
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) List(ctx context.Context) ([]sqlc.ListUsersRow, error) {
	return sqlc.New(s.pool).ListUsers(ctx)
}

func (s *Service) Create(ctx context.Context, actor Actor, in CreateInput) (int64, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Username = strings.TrimSpace(in.Username)
	if err := validateName(in.Name); err != nil {
		return 0, err
	}
	if !usernamePattern.MatchString(in.Username) {
		return 0, invalid("Username 3 sampai 50 karakter: huruf, angka, titik, garis bawah, atau tanda hubung, tanpa spasi.")
	}
	if !validRole(in.Role) {
		return 0, invalid("Peran tidak dikenal.")
	}
	if err := validatePassword(in.Password); err != nil {
		return 0, err
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return 0, err
	}

	var id int64
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := requireActiveDivision(ctx, q, in.DivisionID); err != nil {
			return err
		}
		id, err = q.CreateUser(ctx, sqlc.CreateUserParams{
			Name:         in.Name,
			Username:     in.Username,
			PasswordHash: hash,
			Role:         in.Role,
			DivisionID:   in.DivisionID,
		})
		if isUniqueViolation(err) {
			return ErrUsernameTaken
		}
		if err != nil {
			return err
		}
		return record(ctx, q, actor, "user_created", id, nil, map[string]any{
			"name": in.Name, "username": in.Username, "role": in.Role, "division_id": in.DivisionID,
		})
	})
	return id, err
}

// Update mengubah nama, peran, dan divisi. Username sengaja tidak bisa diubah karena dipakai
// untuk login dan untuk mencocokkan data hasil migrasi.
func (s *Service) Update(ctx context.Context, actor Actor, id int64, in UpdateInput) error {
	in.Name = strings.TrimSpace(in.Name)
	if err := validateName(in.Name); err != nil {
		return err
	}
	if !validRole(in.Role) {
		return invalid("Peran tidak dikenal.")
	}

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		managers, err := q.LockActiveManagers(ctx)
		if err != nil {
			return err
		}
		u, err := getForUpdate(ctx, q, id)
		if err != nil {
			return err
		}
		if err := ensureManagerRemains(u, in.Role, u.IsActive, len(managers)); err != nil {
			return err
		}
		// Divisi lama yang sudah nonaktif boleh tetap dipakai; hanya pindah ke divisi lain yang harus aktif.
		if in.DivisionID != u.DivisionID {
			if err := requireActiveDivision(ctx, q, in.DivisionID); err != nil {
				return err
			}
		}

		before, after := map[string]any{}, map[string]any{}
		if u.Name != in.Name {
			before["name"], after["name"] = u.Name, in.Name
		}
		if u.Role != in.Role {
			before["role"], after["role"] = u.Role, in.Role
		}
		if u.DivisionID != in.DivisionID {
			before["division_id"], after["division_id"] = u.DivisionID, in.DivisionID
		}
		if len(after) == 0 {
			return nil
		}

		if err := q.UpdateUserProfile(ctx, sqlc.UpdateUserProfileParams{
			ID: id, Name: in.Name, Role: in.Role, DivisionID: in.DivisionID,
		}); err != nil {
			return err
		}
		return record(ctx, q, actor, "user_updated", id, before, after)
	})
}

func (s *Service) SetActive(ctx context.Context, actor Actor, id int64, active bool) error {
	if !active && id == actor.ID {
		return ErrSelfDeactivate
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		managers, err := q.LockActiveManagers(ctx)
		if err != nil {
			return err
		}
		u, err := getForUpdate(ctx, q, id)
		if err != nil {
			return err
		}
		if u.IsActive == active {
			return nil
		}
		if err := ensureManagerRemains(u, u.Role, active, len(managers)); err != nil {
			return err
		}
		if err := q.SetUserActive(ctx, sqlc.SetUserActiveParams{ID: id, IsActive: active}); err != nil {
			return err
		}
		action := "user_activated"
		if !active {
			action = "user_deactivated"
		}
		return record(ctx, q, actor, action, id, nil, nil)
	})
}

// ResetPassword memasang password sementara, membuka kunci, dan mengakhiri semua sesi akun itu.
// Password tidak pernah ikut dicatat di audit log.
func (s *Service) ResetPassword(ctx context.Context, actor Actor, id int64, password string) error {
	if err := validatePassword(password); err != nil {
		return err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if _, err := getForUpdate(ctx, q, id); err != nil {
			return err
		}
		if err := q.ResetUserPassword(ctx, sqlc.ResetUserPasswordParams{ID: id, PasswordHash: hash}); err != nil {
			return err
		}
		return record(ctx, q, actor, "password_reset", id, nil, nil)
	})
}

func (s *Service) Unlock(ctx context.Context, actor Actor, id int64) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if _, err := getForUpdate(ctx, q, id); err != nil {
			return err
		}
		if err := q.UnlockUser(ctx, id); err != nil {
			return err
		}
		return record(ctx, q, actor, "user_unlocked", id, nil, nil)
	})
}

func getForUpdate(ctx context.Context, q *sqlc.Queries, id int64) (sqlc.GetUserForUpdateRow, error) {
	u, err := q.GetUserForUpdate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

// ensureManagerRemains menolak perubahan yang membuat Admin dan Manajer aktif habis (S2-03).
// activeManagers adalah jumlah dari LockActiveManagers, yang sudah mengunci baris-baris itu.
func ensureManagerRemains(u sqlc.GetUserForUpdateRow, nextRole sqlc.UserRole, nextActive bool, activeManagers int) error {
	wasManager := u.IsActive && isManager(u.Role)
	stillManager := nextActive && isManager(nextRole)
	if wasManager && !stillManager && activeManagers <= 1 {
		return ErrLastManager
	}
	return nil
}

func isManager(r sqlc.UserRole) bool {
	return r == sqlc.UserRoleAdmin || r == sqlc.UserRoleManager
}

func validRole(r sqlc.UserRole) bool {
	return r == sqlc.UserRoleAdmin || r == sqlc.UserRoleManager || r == sqlc.UserRoleStaff
}

func validateName(name string) error {
	if name == "" {
		return invalid("Nama wajib diisi.")
	}
	if utf8.RuneCountInString(name) > 100 {
		return invalid("Nama maksimal 100 karakter.")
	}
	return nil
}

func validatePassword(p string) error {
	if utf8.RuneCountInString(p) < minPasswordLen {
		return invalid("Password minimal 8 karakter.")
	}
	if len(p) > maxPasswordLen {
		return invalid("Password terlalu panjang, maksimal 72 karakter.")
	}
	return nil
}

func requireActiveDivision(ctx context.Context, q *sqlc.Queries, id int64) error {
	_, err := q.GetActiveDivision(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return invalid("Divisi tidak ditemukan atau sudah nonaktif.")
	}
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func record(ctx context.Context, q *sqlc.Queries, actor Actor, action string, target int64, before, after map[string]any) error {
	e := audit.Entry{
		ActorID:  &actor.ID,
		Action:   action,
		Entity:   "user",
		EntityID: strconv.FormatInt(target, 10),
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
