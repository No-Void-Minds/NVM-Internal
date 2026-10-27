package httpapi

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/auth"
	"nvm-internal/internal/config"
)

type Server struct {
	log          *slog.Logger
	pool         *pgxpool.Pool
	auth         *auth.Service
	tokens       auth.TokenIssuer
	cookieSecure bool
	// now bisa diganti di test supaya waktu kuncian dan kedaluwarsa token bisa diuji.
	now func() time.Time
}

func NewRouter(log *slog.Logger, pool *pgxpool.Pool, cfg config.Config) (*gin.Engine, error) {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	tokens, err := auth.NewTokenIssuer(cfg.JWTSecret)
	if err != nil {
		return nil, err
	}

	s := &Server{
		log:          log,
		pool:         pool,
		auth:         auth.NewService(pool),
		tokens:       tokens,
		cookieSecure: cfg.CookieSecure,
		now:          time.Now,
	}

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
	api.POST("/auth/login", s.login)

	return r, nil
}
