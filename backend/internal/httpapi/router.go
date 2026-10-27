package httpapi

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	log  *slog.Logger
	pool *pgxpool.Pool
}

func NewRouter(log *slog.Logger, pool *pgxpool.Pool, appEnv string) (*gin.Engine, error) {
	if appEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	s := &Server{log: log, pool: pool}

	r := gin.New()
	// Belum ada reverse proxy, jadi header X-Forwarded-For tidak dipercaya. ClientIP dipakai
	// untuk audit log dan rate limit login; di produksi isi dengan IP Nginx (Sprint 7).
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	r.Use(withRequestID(), s.accessLog(), s.recoverer())

	r.NoRoute(func(c *gin.Context) {
		writeError(c, 404, CodeNotFound, "Alamat tidak ditemukan.")
	})

	api := r.Group("/api")
	api.GET("/health", s.health)

	return r, nil
}