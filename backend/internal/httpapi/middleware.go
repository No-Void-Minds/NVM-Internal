package httpapi

import (
	"crypto/rand"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDKey = "request_id"

// withRequestID memberi setiap request ID yang sama di log dan di header respons,
// supaya keluhan pengguna bisa dicocokkan dengan baris log.
func withRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := rand.Text()
		c.Set(requestIDKey, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func requestID(c *gin.Context) string {
	return c.GetString(requestIDKey)
}

func (s *Server) accessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		// Path tanpa query string, supaya isi pencarian atau token tidak ikut tercatat.
		s.log.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
			"request_id", requestID(c),
		)
	}
}

func (s *Server) recoverer() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		s.log.Error("panic",
			"recovered", recovered,
			"method", c.Request.Method,
			"route", c.FullPath(),
			"request_id", requestID(c),
		)
		writeError(c, 500, CodeInternal, "Terjadi kesalahan di server. Coba lagi beberapa saat.")
	})
}