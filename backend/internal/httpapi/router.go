package httpapi

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/account"
	"nvm-internal/internal/auth"
	"nvm-internal/internal/config"
	"nvm-internal/internal/db/sqlc"
	"nvm-internal/internal/label"
	"nvm-internal/internal/task"
)

type Server struct {
	log          *slog.Logger
	pool         *pgxpool.Pool
	auth         *auth.Service
	tokens       auth.TokenIssuer
	cookieSecure bool
	loginLimiter *ipLimiter
	accounts     *account.Service
	labels       *label.Service
	tasks        *task.Service
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
		loginLimiter: newIPLimiter(30*time.Second, 10),
		accounts:     account.NewService(pool),
		labels:       label.NewService(pool),
		tasks:        task.NewService(pool),
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

	api := r.Group("/api", sameOrigin())
	api.GET("/health", s.health)
	api.POST("/auth/login", s.limitLogin(), s.login)
	authed := api.Group("", s.requireSession())
	authed.GET("/auth/me", s.me)
	authed.POST("/auth/logout", s.logout)
	authed.POST("/auth/change-password", s.limitLogin(), s.changePassword)
	app := authed.Group("", requirePasswordChanged())
	manage := app.Group("", requireRole(sqlc.UserRoleAdmin, sqlc.UserRoleManager))
	manage.GET("/users", s.listUsers)
	manage.POST("/users", s.createUser)
	manage.PUT("/users/:id", s.updateUser)
	manage.POST("/users/:id/deactivate", s.setUserActive(false))
	manage.POST("/users/:id/activate", s.setUserActive(true))
	manage.POST("/users/:id/reset-password", s.resetUserPassword)
	manage.POST("/users/:id/unlock", s.unlockUser)
	app.GET("/divisions", s.listDivisions)
	manage.POST("/divisions", s.createDivision)
	manage.PUT("/divisions/:id", s.renameDivision)
	manage.POST("/divisions/:id/deactivate", s.setDivisionActive(false))
	manage.POST("/divisions/:id/activate", s.setDivisionActive(true))
	manage.GET("/audit-logs", s.listAuditLogs)
	app.GET("/labels", s.listLabels)
	manage.POST("/labels", s.createLabel)
	manage.PUT("/labels/:id", s.updateLabel)
	manage.POST("/labels/:id/deactivate", s.setLabelActive(false))
	manage.POST("/labels/:id/activate", s.setLabelActive(true))
	app.GET("/tasks", s.listTasks)
	app.GET("/tasks/:id", s.getTask)
	app.POST("/tasks", requireRole(sqlc.UserRoleManager), s.createTask)

	return r, nil
}
