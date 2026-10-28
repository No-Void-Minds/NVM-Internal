package httpapi

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/db/sqlc"
)

// requireRole dipasang setelah requireSession. Peran dibaca dari database lewat sesi, bukan dari
// isi token, sehingga perubahan peran langsung berlaku. Manajer memegang seluruh hak Admin (BR-07),
// jadi endpoint pengelolaan cukup memakai requireRole(sqlc.UserRoleAdmin, sqlc.UserRoleManager).
func requireRole(roles ...sqlc.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		if slices.Contains(roles, currentUser(c).Role) {
			c.Next()
			return
		}
		writeError(c, http.StatusForbidden, CodeForbidden, "Kamu tidak punya akses ke fitur ini.")
	}
}
