package auth

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/netip"
	"nvm-internal/internal/audit"
	"time"

	"nvm-internal/internal/db/sqlc"
)

type LoginOutcome int

const (
	LoginOK LoginOutcome = iota
	// LoginInvalid dipakai untuk username tidak ada, password salah, dan akun nonaktif,
	// supaya ketiganya tidak bisa dibedakan dari luar (AC-01).
	LoginInvalid
	LoginLocked
)

type LoginResult struct {
	Outcome           LoginOutcome
	User              sqlc.GetUserForLoginRow
	RetryAfterSeconds int
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// Login berjalan dalam satu transaksi. Transaksi tetap di-commit saat login gagal,
// karena hitungan gagal dan audit log justru harus tersimpan.
func (s *Service) Login(ctx context.Context, username, password string, ip *netip.Addr, now time.Time) (LoginResult, error) {
	var result LoginResult

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)

		u, err := q.GetUserForLogin(ctx, username)
		if errors.Is(err, pgx.ErrNoRows) {
			burnPasswordCheck(password)
			result.Outcome = LoginInvalid
			return audit.UserEvent(ctx, q, nil, "login_failed", ip, map[string]any{"username": username, "reason": "unknown_user"})
		}
		if err != nil {
			return err
		}
		result.User = u

		state := LockState{
			FailedCount: int(u.FailedLoginCount),
			Level:       int(u.LockoutLevel),
		}
		if u.LockedUntil.Valid {
			state.LockedUntil = u.LockedUntil.Time
		}

		// Selama terkunci, password benar pun ditolak dan tidak diperiksa (AC-02).
		if state.IsLocked(now) {
			result.Outcome = LoginLocked
			result.RetryAfterSeconds = state.RetryAfterSeconds(now)
			return audit.UserEvent(ctx, q, &u.ID, "login_failed", ip, map[string]any{"reason": "locked"})
		}

		passwordOK := CheckPassword(u.PasswordHash, password)

		// Akun nonaktif tidak menambah hitungan gagal; tidak ada gunanya mengunci akun yang memang tidak bisa dipakai.
		if !u.IsActive {
			result.Outcome = LoginInvalid
			return audit.UserEvent(ctx, q, &u.ID, "login_failed", ip, map[string]any{"reason": "inactive"})
		}

		if !passwordOK {
			next, locked := RegisterFailure(state, now)
			if err := q.SaveLockState(ctx, sqlc.SaveLockStateParams{
				ID:               u.ID,
				FailedLoginCount: int16(next.FailedCount),
				LockoutLevel:     int16(next.Level),
				LockedUntil:      pgtype.Timestamptz{Time: next.LockedUntil, Valid: !next.LockedUntil.IsZero()},
			}); err != nil {
				return err
			}
			if err := audit.UserEvent(ctx, q, &u.ID, "login_failed", ip, map[string]any{"reason": "wrong_password"}); err != nil {
				return err
			}
			if !locked {
				result.Outcome = LoginInvalid
				return nil
			}
			result.Outcome = LoginLocked
			result.RetryAfterSeconds = next.RetryAfterSeconds(now)
			return audit.UserEvent(ctx, q, &u.ID, "account_locked", ip, map[string]any{
				"level":        next.Level,
				"locked_until": next.LockedUntil,
			})
		}

		if err := q.RecordLoginSuccess(ctx, u.ID); err != nil {
			return err
		}
		result.Outcome = LoginOK
		return audit.UserEvent(ctx, q, &u.ID, "login_success", ip, nil)
	})

	return result, err
}
