package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
)

func (s *Server) me(c *gin.Context) {
	u := currentUser(c)
	c.JSON(http.StatusOK, gin.H{"user": sessionUser{
		ID:                 u.ID,
		Name:               u.Name,
		Username:           u.Username,
		Role:               string(u.Role),
		DivisionID:         u.DivisionID,
		DivisionName:       u.DivisionName,
		JobTitle:           textPtr(u.JobTitle),
		AvatarID:           avatarID(u.AvatarID),
		MustChangePassword: u.MustChangePassword,
	}})
}

// logout menaikkan token_version, jadi sesi di semua perangkat berakhir (keputusan Sprint 1).
func (s *Server) logout(c *gin.Context) {
	u := currentUser(c)
	ctx := c.Request.Context()

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if _, err := q.BumpTokenVersion(ctx, u.ID); err != nil {
			return err
		}
		return audit.UserEvent(ctx, q, &u.ID, "logout", clientAddr(c), nil)
	})
	if err != nil {
		s.internalError(c, err)
		return
	}

	s.clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}

// noStore dipasang sebelum pengecekan sesi, supaya jawaban 401 dan 403 juga tidak disimpan cache mana pun.
func noStore() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}

// verify dipanggil Caddy (forward_auth) untuk dashboard dan studio di PC tim (revisi 5). 401 dan 403 sudah dijawab
// requireSession dan requirePasswordChanged; sampai di sini berarti sesi sah dan password sudah diganti.
func (s *Server) verify(c *gin.Context) {
	// Username untuk log Caddy; bukan rahasia dan sudah terlihat di aplikasi.
	c.Header("X-NVM-User", currentUser(c).Username)
	c.Status(http.StatusNoContent)
}
