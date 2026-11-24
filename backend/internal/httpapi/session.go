package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/auth"
	"nvm-internal/internal/db/sqlc"
)

const sessionUserKey = "session_user"

// requireSession memeriksa cookie sesi pada setiap request dan membaca ulang akun dari database,
// sehingga akun yang dinonaktifkan atau token_version yang naik langsung mengakhiri sesi (FR-AUTH-08).
func (s *Server) requireSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := c.Cookie(sessionCookie)
		if err != nil || raw == "" {
			writeError(c, http.StatusUnauthorized, CodeUnauthorized, "Silakan masuk terlebih dahulu.")
			return
		}

		u, ok, err := s.sessionFor(c.Request.Context(), raw)
		if err != nil {
			s.internalError(c, err)
			return
		}
		if !ok {
			s.expireSession(c)
			return
		}
		c.Set(sessionUserKey, u)
		c.Next()
	}
}

// sessionFor memeriksa token sesi dan membaca ulang akun dari database. ok false berarti sesi sudah tidak berlaku
// (token rusak atau kedaluwarsa, akun nonaktif, token_version naik, baris sesi hilang atau lewat umur mutlak).
// Dipakai requireSession dan pemeriksaan ulang koneksi WebSocket Whiteboard, supaya aturannya satu.
func (s *Server) sessionFor(ctx context.Context, raw string) (sqlc.GetSessionUserRow, bool, error) {
	now := s.now()
	claims, userID, err := s.tokens.Parse(raw, now)
	if err != nil {
		return sqlc.GetSessionUserRow{}, false, nil
	}
	u, err := sqlc.New(s.pool).GetSessionUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, false, nil
	}
	if err != nil {
		return u, false, err
	}
	if !u.IsActive || u.TokenVersion != claims.TokenVersion {
		return u, false, nil
	}
	alive, err := s.touchSession(ctx, u.ID, claims.SessionID, now)
	if err != nil || !alive {
		return u, false, err
	}
	return u, true, nil
}

// sessionTouchEvery membatasi penulisan last_seen_at: Caddy memanggil verify untuk setiap aset dashboard,
// jadi tanpa batas ini satu halaman dashboard bisa menulis puluhan kali ke database.
const sessionTouchEvery = 60 * time.Second

// touchSession memastikan baris sesinya masih ada dan belum melewati umur mutlak, lalu mencatat aktivitas
// terakhir. Cookie tidak diterbitkan ulang di sini, sehingga verify untuk Caddy (yang tidak meneruskan
// Set-Cookie) tidak membuat token berganti di setiap request.
func (s *Server) touchSession(ctx context.Context, userID int64, sid string, now time.Time) (bool, error) {
	var id pgtype.UUID
	if err := id.Scan(sid); err != nil || !id.Valid {
		return false, nil
	}
	q := sqlc.New(s.pool)
	row, err := q.GetUserSession(ctx, sqlc.GetUserSessionParams{ID: id, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// Batas mutlak juga dicek dari database, bukan hanya dari exp di JWT, supaya tetap berlaku bila umur token
	// di kode suatu saat diubah tanpa mengubah aturan sesi.
	if now.Sub(row.CreatedAt.Time) > auth.SessionMaxAge {
		return false, nil
	}
	idle := now.Sub(row.LastSeenAt.Time)
	if idle >= sessionTouchEvery {
		err = q.TouchUserSession(ctx, sqlc.TouchUserSessionParams{
			ID: id, Now: pgtype.Timestamptz{Time: now, Valid: true},
		})
	}
	return true, err
}

// startSession membuat baris sesi baru dan memasang cookie. Dipakai saat login dan setelah ganti password, satu-
// satunya saat cookie diterbitkan. Umur cookie sama dengan umur mutlak token (auth.SessionMaxAge).
func (s *Server) startSession(c *gin.Context, userID int64, role sqlc.UserRole, tokenVersion int32, now time.Time) error {
	sid, err := sqlc.New(s.pool).CreateUserSession(c.Request.Context(), sqlc.CreateUserSessionParams{
		UserID: userID, Now: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		return err
	}
	token, expires, err := s.tokens.Issue(userID, string(role), tokenVersion, sid.String(), now)
	if err != nil {
		return err
	}
	s.setSessionCookie(c, token, int(expires.Sub(now).Seconds()))
	return nil
}

// requirePasswordChanged menutup endpoint fitur sampai password sementara diganti (FR-AUTH-07).
// Endpoint /auth/me, /auth/logout, dan ganti password sengaja tidak memakai middleware ini.
func requirePasswordChanged() gin.HandlerFunc {
	return func(c *gin.Context) {
		if currentUser(c).MustChangePassword {
			writeError(c, http.StatusForbidden, CodePasswordChangeRequired, "Ganti password sementara terlebih dahulu.")
			return
		}
		c.Next()
	}
}

func currentUser(c *gin.Context) sqlc.GetSessionUserRow {
	return c.MustGet(sessionUserKey).(sqlc.GetSessionUserRow)
}

func (s *Server) expireSession(c *gin.Context) {
	s.clearSessionCookie(c)
	writeError(c, http.StatusUnauthorized, CodeSessionExpired, "Sesi kamu sudah berakhir. Silakan masuk lagi.")
}

// clearSessionCookie memakai setSessionCookie yang sama, jadi Domain, Path, dan Secure selalu sama dengan saat
// cookie dibuat. Kalau berbeda, browser menganggapnya cookie lain dan cookie lama tertinggal.
func (s *Server) clearSessionCookie(c *gin.Context) {
	s.setSessionCookie(c, "", -1)
}
