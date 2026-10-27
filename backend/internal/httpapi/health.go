package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := s.pool.Ping(ctx); err != nil {
		s.log.Warn("health: database tidak bisa dihubungi", "err", err, "request_id", requestID(c))
		writeError(c, http.StatusServiceUnavailable, CodeUnavailable, "Database tidak dapat dihubungi.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "database": "ok"})
}
