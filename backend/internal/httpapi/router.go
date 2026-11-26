package httpapi

import (
	"context"
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
	"nvm-internal/internal/timeline"
	"nvm-internal/internal/todo"
	"nvm-internal/internal/whiteboard"
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
	todos        *todo.Service
	timeline     *timeline.Service
	whiteboard   *whiteboard.Service
	hub          *whiteboard.Hub
	// live menyiarkan perubahan task ke papan yang terbuka (revisi 47).
	live *taskLive
	// wsRecheck: jeda pemeriksaan ulang sesi koneksi WebSocket; dipendekkan di test.
	wsRecheck   time.Duration
	tasks       *task.Service
	files       *storage.Store
	avatars     *account.Avatars
	attachments *task.Attachments
	// now bisa diganti di test supaya waktu kuncian dan kedaluwarsa token bisa diuji.
	now func() time.Time
}

// wsRecheckEvery: jeda pemeriksaan ulang sesi koneksi WebSocket Whiteboard. Variabel supaya test bisa
// memendekkannya.
var wsRecheckEvery = 60 * time.Second

func NewRouter(log *slog.Logger, pool *pgxpool.Pool, cfg config.Config) (*gin.Engine, error) {
	r, _, err := New(log, pool, cfg)
	return r, err
}

// New sama dengan NewRouter, ditambah Server supaya main bisa menjalankan ticker kemunculan yang menyiarkan task
// baru lewat hub papan yang sama (revisi 47).
func New(log *slog.Logger, pool *pgxpool.Pool, cfg config.Config) (*gin.Engine, *Server, error) {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	tokens, err := auth.NewTokenIssuer(cfg.JWTSecret)
	if err != nil {
		return nil, nil, err
	}
	files, err := storage.Open(cfg.StorageDir)
	if err != nil {
		return nil, nil, err
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
		activities:   activity.NewService(pool, files),
		labels:       label.NewService(pool),
		todos:        todo.NewService(pool),
		timeline:     timeline.NewService(pool),
		whiteboard:   whiteboard.NewService(pool),
		wsRecheck:    wsRecheckEvery,
		tasks:        task.NewService(pool),
		files:        files,
		attachments:  task.NewAttachments(pool, files),
		avatars:      account.NewAvatars(pool, files, log),
		live:         newTaskLive(),
	}

	s.whiteboard.UseStorage(files)
	s.hub = whiteboard.NewHub(s.whiteboard)
	// Hub hidup selama proses; Sweep melepas kunci kartu yang kehilangan detak.
	go s.hub.Run(context.Background())

	r := gin.New()
	// ClientIP dipakai limiter login dan kolom ip_address audit log. Header X-Forwarded-For dan X-Real-IP hanya
	// dipercaya dari TRUSTED_PROXIES (subnet jaringan Nginx di produksi, yang mengisinya dari CF-Connecting-IP).
	// Kosong di dev: tidak ada header yang dipercaya, jadi IP selalu alamat koneksi langsung.
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, nil, err
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
	// Hasil salinan Drive hanya untuk Admin, sama dengan halaman Status (revisi 23 dan 25).
	app.GET("/drive-export/status", requireRole(sqlc.UserRoleAdmin), s.driveExportStatus)
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
	// Revisi 47: koneksi realtime papan task. Event disiarkan middleware taskEvent di rute yang mengubah task.
	app.GET("/tasks/ws", s.taskSocket)
	app.GET("/board", s.board)
	app.GET("/archive", s.archive)
	app.GET("/people", s.listPeople)
	app.GET("/people/:id", s.getPerson)
	app.GET("/tasks/:id", s.getTask)
	// Tanpa middleware peran: Staf dan Manajer boleh membuat task (revisi 27), dan pembuat task boleh
	// mengubah serta menghapus task buatannya selama masih Requested. Keduanya dicek di service, yang
	// memegang status dan pembuat task.
	ubah, pindah := s.taskEvent(aksiUbah), s.taskEvent(aksiPindah)
	app.POST("/tasks", s.taskEvent(aksiBaru), s.createTask)
	managerOnly := requireRole(sqlc.UserRoleManager)
	app.PUT("/tasks/:id", ubah, s.updateTask)
	app.DELETE("/tasks/:id", s.taskEvent(aksiHapus), s.deleteTask)
	app.POST("/tasks/:id/stop-series", managerOnly, ubah, s.stopSeries)
	app.PUT("/tasks/:id/labels", ubah, s.setTaskLabels)
	app.PUT("/tasks/:id/deliverable", ubah, s.setTaskDeliverable)
	app.POST("/tasks/:id/move", pindah, s.moveTask)
	app.POST("/tasks/:id/attachments", ubah, s.uploadAttachment)
	app.GET("/tasks/:id/attachments/:attachmentId", s.downloadAttachment)
	app.DELETE("/tasks/:id/attachments/:attachmentId", ubah, s.deleteAttachment)
	app.PUT("/tasks/:id/subtasks/:subtaskId/done", s.taskEvent(aksiCentang), s.setSubtaskDone)
	// Tanpa managerOnly: PIC task menyusun sub task dan membaginya ke PIC lain (revisi 7). Haknya dicek di service.
	app.POST("/tasks/:id/subtasks", ubah, s.addSubtask)
	app.PUT("/tasks/:id/subtasks/:subtaskId", ubah, s.updateSubtask)
	app.DELETE("/tasks/:id/subtasks/:subtaskId", ubah, s.deleteSubtask)
	app.POST("/tasks/:id/subtasks/:subtaskId/attachments", ubah, s.uploadSubtaskAttachment)
	// Aktivitas Harian (revisi 21). Tabel harian bisa dibaca semua peran; hak mengubah dicek di service.
	app.GET("/activities", s.listActivities)
	app.GET("/activities/me", s.myActivities)
	app.PUT("/activities", s.saveActivity)
	app.PUT("/activities/link", s.linkActivity)
	app.GET("/activities/export", managerOnly, s.exportActivities)
	// To Do pribadi (revisi 28). Semua rute memakai akun yang login; tidak ada rute untuk To Do orang lain,
	// dan id milik orang lain dijawab 404 supaya keberadaannya tidak bocor.
	app.GET("/todos", s.listTodos)
	app.POST("/todos", s.addTodo)
	app.PUT("/todos/order", s.reorderTodos)
	app.POST("/todos/clear-done", s.clearDoneTodos)
	app.PUT("/todos/:todoId", s.renameTodo)
	app.PUT("/todos/:todoId/done", s.setTodoDone)
	app.DELETE("/todos/:todoId", s.deleteTodo)
	// Timeline (revisi 30). Semua yang login membaca; hanya Manajer menulis, diperiksa di timeline.Service.
	app.GET("/timeline/events", s.listTimelineEvents)
	app.POST("/timeline/events", s.createTimelineEvent)
	app.PUT("/timeline/events/:id", s.updateTimelineEvent)
	app.DELETE("/timeline/events/:id", s.deleteTimelineEvent)
	app.GET("/timeline/notes", s.listTimelineNotes)
	app.POST("/timeline/notes", s.createTimelineNote)
	app.PUT("/timeline/notes/order", s.reorderTimelineNotes)
	app.PUT("/timeline/notes/:id", s.updateTimelineNote)
	app.DELETE("/timeline/notes/:id", s.deleteTimelineNote)
	// Lampiran per jam (revisi 24). Haknya sama dengan isian aktivitas dan dicek di service.
	app.POST("/activities/files", s.uploadActivityFile)
	app.GET("/activities/files/:fileId", s.downloadActivityFile)
	app.DELETE("/activities/files/:fileId", s.deleteActivityFile)

	// Whiteboard (docs/whiteboard.md): baca untuk semua, tulis hanya Staf dan Manajer.
	wb := app.Group("/whiteboard")
	wbWrite := wb.Group("", wbWriter())
	// :id boleh "root" untuk papan akar.
	wb.GET("/search", s.wbSearch)
	wb.GET("/boards/:id", s.wbBoard)
	wbWrite.PUT("/boards/:id", s.wbRenameBoard)
	wb.GET("/boards/:id/activity", s.wbActivity)
	wb.GET("/ws", s.wbSocket)
	wbWrite.POST("/boards/:id/cards", s.wbCreateCard)
	wbWrite.POST("/boards/:id/cards/patch", s.wbPatchCards)
	wbWrite.POST("/boards/:id/cards/delete", s.wbDeleteCards)
	wbWrite.POST("/boards/:id/cards/:cardId/checklist", s.wbChecklist)
	wb.GET("/trash", s.wbTrash)
	wbWrite.POST("/trash/:batch/restore", s.wbRestore)
	wb.GET("/boards", s.wbBoards)
	wb.GET("/people", s.wbPeople)
	wb.GET("/files/:fileId", s.wbFile)
	wbWrite.POST("/boards/:id/files", s.wbUpload)
	wbWrite.POST("/boards/:id/cards/:cardId/file", s.wbReplaceFile)
	wbWrite.POST("/preview", s.wbPreview)
	wbWrite.POST("/boards/:id/cards/:cardId/table", s.wbTable)
	wbWrite.POST("/boards/:id/drawings", s.wbCreateDrawing)
	wbWrite.POST("/boards/:id/cards/:cardId/drawing", s.wbDrawing)
	wbWrite.POST("/boards/:id/move", s.wbMove)
	wbWrite.POST("/boards/:id/connectors", s.wbCreateConnector)
	wbWrite.PUT("/boards/:id/connectors/:connectorId", s.wbUpdateConnector)
	wbWrite.DELETE("/boards/:id/connectors/:connectorId", s.wbDeleteConnector)
	wb.GET("/boards/:id/cards/:cardId/comments", s.wbComments)
	wbWrite.POST("/boards/:id/cards/:cardId/comments", s.wbAddComment)
	wbWrite.PUT("/comments/:commentId", s.wbEditComment)
	wbWrite.DELETE("/comments/:commentId", s.wbDeleteComment)
	wb.GET("/templates", s.wbTemplates)
	wbWrite.PUT("/boards/:id/template", s.wbSetTemplate)
	wbWrite.POST("/boards/:id/from-template", s.wbFromTemplate)

	return r, s, nil
}
