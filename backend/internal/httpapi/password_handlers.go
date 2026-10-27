package httpapi

import (
	"net/http"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"nvm-internal/internal/audit"
	"nvm-internal/internal/auth"
	"nvm-internal/internal/db/sqlc"
)

const minPasswordLen = 8

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// changePassword mengganti password sendiri (FR-AUTH-07). token_version naik sehingga perangkat
// lain keluar, lalu perangkat ini langsung diberi token baru supaya tidak ikut terlempar.
func (s *Server) changePassword(c *gin.Context) {
	u := currentUser(c)

	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	switch {
	case req.CurrentPassword == "" || req.NewPassword == "":
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Password lama dan password baru wajib diisi.")
		return
	case utf8.RuneCountInString(req.NewPassword) < minPasswordLen:
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Password baru minimal 8 karakter.")
		return
	case len(req.NewPassword) > maxPasswordLen:
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Password baru terlalu panjang, maksimal 72 karakter.")
		return
	case req.NewPassword == req.CurrentPassword:
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Password baru harus berbeda dari password lama.")
		return
	}

	ctx := c.Request.Context()
	now := s.now()
	var newVersion int32
	wrongCurrent := false

	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		hash, err := q.GetPasswordHash(ctx, u.ID)
		if err != nil {
			return err
		}
		if !auth.CheckPassword(hash, req.CurrentPassword) {
			wrongCurrent = true
			return audit.UserEvent(ctx, q, &u.ID, "password_change_failed", clientAddr(c),
				map[string]any{"reason": "wrong_current_password"})
		}

		newHash, err := auth.HashPassword(req.NewPassword)
		if err != nil {
			return err
		}
		newVersion, err = q.ChangePassword(ctx, sqlc.ChangePasswordParams{ID: u.ID, PasswordHash: newHash})
		if err != nil {
			return err
		}
		return audit.UserEvent(ctx, q, &u.ID, "password_changed", clientAddr(c), nil)
	})
	if err != nil {
		s.internalError(c, err)
		return
	}
	// 400, bukan 401: sesinya tetap sah, hanya isian password lamanya yang salah.
	if wrongCurrent {
		writeError(c, http.StatusBadRequest, CodeInvalidCurrentPassword, "Password lama salah.")
		return
	}

	token, expires, err := s.tokens.Issue(u.ID, string(u.Role), newVersion, now)
	if err != nil {
		s.internalError(c, err)
		return
	}
	s.setSessionCookie(c, token, int(expires.Sub(now).Seconds()))
	c.Status(http.StatusNoContent)
}
