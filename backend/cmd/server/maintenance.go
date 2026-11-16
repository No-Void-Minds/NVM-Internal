package main

import (
	"context"
	"flag"
	"log/slog"
	"time"

	"nvm-internal/internal/config"
	"nvm-internal/internal/db"
	"nvm-internal/internal/maintenance"
	"nvm-internal/internal/storage"
)

func runMaintenance(cfg config.Config, log *slog.Logger, args []string) error {
	fs := flag.NewFlagSet("maintenance", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "hanya melaporkan, tidak menghapus apa pun")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	store, err := storage.Open(cfg.StorageDir)
	if err != nil {
		return err
	}

	r, err := maintenance.Run(ctx, pool, store, time.Now(), *dryRun)
	// Laporan tetap ditulis walaupun ada langkah yang gagal, supaya terlihat sampai mana job berjalan.
	log.Info("maintenance",
		"dry_run", r.DryRun,
		"sesi_kedaluwarsa", r.ExpiredSessions,
		"audit_login_lama", r.OldLoginAuditLogs,
		"lampiran_yatim", len(r.OrphanAttachments),
		"foto_yatim", len(r.OrphanAvatars),
		"unggahan_terputus", len(r.StaleUploads),
		"berkas_tak_dikenal", len(r.Unknown),
	)
	for _, p := range r.OrphanAttachments {
		log.Info("lampiran yatim", "path", p, "dihapus", !r.DryRun)
	}
	for _, p := range r.OrphanAvatars {
		log.Info("foto yatim", "path", p, "dihapus", !r.DryRun)
	}
	for _, p := range r.StaleUploads {
		log.Info("unggahan terputus", "path", p, "dihapus", !r.DryRun)
	}
	for _, p := range r.Unknown {
		log.Warn("berkas tak dikenal, dibiarkan", "path", p)
	}
	return err
}
