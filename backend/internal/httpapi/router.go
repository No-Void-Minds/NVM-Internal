package httpapi

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/account"
	"nvm-internal/internal/activity"
	"nvm-internal/internal/auth"
	"nvm-internal/internal/config"
	"nvm-internal/internal/db/sqlc"
	"nvm-internal/internal/label"
	"nvm-internal/internal/storage"
	"nvm-internal/internal/task"
)

type Server struct {
	log          *slog.Logger
	pool         *pgxpool.Pool
	auth         *auth.Service
	tokens       auth.TokenIssuer
	cookieSecure bool
	cookieDomain string
	loginLimiter *ipLimiter
	accounts     *account.Service
	activities   *activity.Service
	labels       *label.Service
	tasks        *task.Service
	files        *storage.Store
	avatars      *account.Avatars
	attachments  *task.Attachments
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
	files, err := storage.Open(cfg.StorageDir)
	if err != nil {
		return nil, err
	}
	log.Info("penyimpanan berkas siap", "dir", files.Root())

	s := &Server{
		log:          log,
		pool:         pool,
		auth:         auth.NewService(pool),
		tokens:       tokens,
		cookieSecure: cfg.CookieSecure,
		cookieDomain: cfg.CookieDomain,
		now:          time.Now,
		loginLimiter: newIPLimiter(30*time.Second, 10),
		accounts:     account.NewService(pool),
		activities:   activity.NewService(pool),
		labels:       label.NewService(pool),
		tasks:        task.NewService(pool),
		files:        files,
		attachments:  task.NewAttachments(pool, files),
		avatars:      account.NewAvatars(pool, files, log),
	}

	r := gin.New()
	// ClientIP dipakai limiter login dan kolom ip_address audit log. Header X-Forwarded-For dan X-Real-IP hanya
	// dipercaya dari TRUSTED_PROXIES (subnet jaringan Nginx di produksi, yang mengisinya dari CF-Connecting-IP).
	// Kosong di dev: tidak ada header yang dipercaya, jadi IP selalu alamat koneksi langsung.
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(withRequestID(), s.accessLog(), s.recoverer())

	r.NoRoute(func(c *gin.Context) {
		writeError(c, 404, CodeNotFound, "Alamat tidak ditemukan.")
	})

	// no-store untuk semua jawaban API (S12-12): data pengelola dan sesi tidak boleh tertahan cache Cloudflare,
	// Nginx, atau browser. Foto profil menimpanya dengan cache sendiri karena URL-nya berversi (?v=).
	api := r.Group("/api", sameOrigin(), noStore())
	api.GET("/health", s.health)
	api.POST("/auth/login", s.limitLogin(), s.login)
	// Untuk forward_auth Caddy: dipanggil di setiap permintaan dashboard, jadi tidak memakai rate limit login.
	api.GET("/auth/verify", s.requireSession(), requirePasswordChanged(), s.verify)
	authed := api.Group("", s.requireSession())
	authed.GET("/auth/me", s.me)
	authed.POST("/auth/logout", s.logout)
	authed.POST("/auth/change-password", s.limitLogin(), s.changePassword)
	app := authed.Group("", requirePasswordChanged())
	// Rute /me selalu memakai akun yang login, tidak pernah id dari URL (S10-04).
	app.PUT("/me/profile", s.updateMyProfile)
	app.PUT("/me/avatar", s.uploadMyAvatar)
	app.DELETE("/me/avatar", s.deleteMyAvatar)
	app.GET("/me/task-counts", s.myTaskCounts)
	app.GET("/users/:id/avatar", s.userAvatar)
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
	app.GET("/board", s.board)
	app.GET("/archive", s.archive)
	app.GET("/people", s.listPeople)
	app.GET("/people/:id", s.getPerson)
	app.GET("/tasks/:id", s.getTask)
	app.POST("/tasks", requireRole(sqlc.UserRoleManager), s.createTask)
	managerOnly := requireRole(sqlc.UserRoleManager)
	app.PUT("/tasks/:id", managerOnly, s.updateTask)
	app.DELETE("/tasks/:id", managerOnly, s.deleteTask)
	app.POST("/tasks/:id/stop-series", managerOnly, s.stopSeries)
	app.PUT("/tasks/:id/labels", s.setTaskLabels)
	app.PUT("/tasks/:id/deliverable", s.setTaskDeliverable)
	app.POST("/tasks/:id/move", s.moveTask)
	app.POST("/tasks/:id/attachments", s.uploadAttachment)
	app.GET("/tasks/:id/attachments/:attachmentId", s.downloadAttachment)
	app.DELETE("/tasks/:id/attachments/:attachmentId", s.deleteAttachment)
	app.PUT("/tasks/:id/subtasks/:subtaskId/done", s.setSubtaskDone)
	// Tanpa managerOnly: PIC task menyusun sub task dan membaginya ke PIC lain (revisi 7). Haknya dicek di service.
	app.POST("/tasks/:id/subtasks", s.addSubtask)
	app.PUT("/tasks/:id/subtasks/:subtaskId", s.updateSubtask)
	app.DELETE("/tasks/:id/subtasks/:subtaskId", s.deleteSubtask)
	app.POST("/tasks/:id/subtasks/:subtaskId/attachments", s.uploadSubtaskAttachment)
	// Aktivitas Harian (revisi 21). Tabel harian bisa dibaca semua peran; hak mengubah dicek di service.
	app.GET("/activities", s.listActivities)
	app.GET("/activities/me", s.myActivities)
	app.PUT("/activities", s.saveActivity)
	app.GET("/activities/export", managerOnly, s.exportActivities)

	return r, nil
}
