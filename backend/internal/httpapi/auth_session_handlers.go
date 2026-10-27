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
