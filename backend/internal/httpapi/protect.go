package httpapi

import (
	"math"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
)

// sameOrigin menolak request pengubah data yang tidak berasal dari halaman aplikasi ini (NFR-SEC-06).
// Cookie SameSite=Lax sudah menahan sebagian besar serangan CSRF; cek Origin menutup sisanya,
// termasuk form dari situs lain yang memakai metode POST.
func sameOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		source := c.GetHeader("Origin")
		if source == "" {
			source = c.GetHeader("Referer")
		}
		u, err := url.Parse(source)
		if source == "" || err != nil || u.Host != c.Request.Host {
			writeError(c, http.StatusForbidden, CodeForbidden, "Permintaan ditolak karena tidak berasal dari aplikasi ini.")
			return
		}
		c.Next()
	}
}

func (s *Server) limitLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		wait := s.loginLimiter.allow(c.ClientIP(), s.now())
		if wait > 0 {
			c.Header("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
			writeError(c, http.StatusTooManyRequests, CodeTooManyRequests,
				"Terlalu banyak percobaan masuk dari perangkat ini. Tunggu sebentar lalu coba lagi.")
			return
		}
		c.Next()
	}
}
