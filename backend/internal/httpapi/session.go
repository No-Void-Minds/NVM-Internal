package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

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

		now := s.now()
		claims, userID, err := s.tokens.Parse(raw, now)
		if err != nil {
			s.expireSession(c)
			return
		}

		u, err := sqlc.New(s.pool).GetSessionUser(c.Request.Context(), userID)
		if errors.Is(err, pgx.ErrNoRows) {
			s.expireSession(c)
			return
		}
		if err != nil {
			s.internalError(c, err)
			return
		}
		if !u.IsActive || u.TokenVersion != claims.TokenVersion {
			s.expireSession(c)
			return
		}

		// Idle timeout bergeser: setiap request yang lolos memperpanjang sesi 30 menit dari sekarang (FR-AUTH-05).
		token, expires, err := s.tokens.Issue(u.ID, string(u.Role), u.TokenVersion, now)
		if err != nil {
			s.internalError(c, err)
			return
		}
		s.setSessionCookie(c, token, int(expires.Sub(now).Seconds()))

		c.Set(sessionUserKey, u)
		c.Next()
	}
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

func (s *Server) clearSessionCookie(c *gin.Context) {
	s.setSessionCookie(c, "", -1)
}
