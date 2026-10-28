package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/db/sqlc"
)

// routerAs membuat router kecil yang menganggap request datang dari akun dengan peran tertentu,
// supaya requireRole bisa diuji tanpa database.
func routerAs(role sqlc.UserRole) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(sessionUserKey, sqlc.GetSessionUserRow{ID: 1, Role: role, IsActive: true})
		c.Next()
	})
	r.GET("/kelola", requireRole(sqlc.UserRoleAdmin, sqlc.UserRoleManager), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestRequireRole(t *testing.T) {
	cases := []struct {
		role sqlc.UserRole
		want int
	}{
		{sqlc.UserRoleAdmin, http.StatusOK},
		{sqlc.UserRoleManager, http.StatusOK},
		{sqlc.UserRoleStaff, http.StatusForbidden},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		routerAs(tc.role).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/kelola", nil))
		if w.Code != tc.want {
			t.Errorf("peran %s: status %d, mau %d", tc.role, w.Code, tc.want)
		}
	}
}
