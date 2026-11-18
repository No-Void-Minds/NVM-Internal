package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nvm-internal/internal/config"
	"nvm-internal/internal/db"
	"nvm-internal/internal/httpapi"
	"nvm-internal/internal/task"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config tidak valid", "err", err)
		os.Exit(1)
	}

	log := newLogger(cfg.AppEnv)
	// `server maintenance [--dry-run]`: job harian S12-07 dari cron host, memakai image dan env yang sama.
	if len(os.Args) > 1 && os.Args[1] == "maintenance" {
		if err := runMaintenance(cfg, log, os.Args[2:]); err != nil {
			log.Error("maintenance gagal", "err", err)
			os.Exit(1)
		}
		return
	}
	// `server occurrences [--date YYYY-MM-DD]`: pembuat kemunculan task berulang sekali jalan (revisi 19).
	if len(os.Args) > 1 && os.Args[1] == "occurrences" {
		if err := runOccurrences(cfg, log, os.Args[2:]); err != nil {
			log.Error("pembuat kemunculan gagal", "err", err)
			os.Exit(1)
		}
		return
	}
	// `server drive-export stage|record|fail` (revisi 25): salinan berkas ke Google Drive, dipanggil
	// deploy/drive-export.sh dari cron host.
	if len(os.Args) > 1 && os.Args[1] == "drive-export" {
		if err := runDriveExport(cfg, log, os.Args[2:]); err != nil {
			log.Error("salinan Drive gagal", "err", err)
			os.Exit(1)
		}
		return
	}
	if err := run(cfg, log); err != nil {
		log.Error("server berhenti karena galat", "err", err)
		os.Exit(1)
	}
}

// Teks lebih enak dibaca saat dev; JSON di produksi supaya log Docker bisa disaring per field.
// Log selalu ke stderr, bukan stdout. Subperintah seperti `drive-export stage` memakai stdout untuk
// keluaran yang dibaca skrip (daftar path), jadi satu baris log di stdout akan merusaknya, dan galatnya
// malah hilang dari stderr yang dipakai skrip sebagai pesan kegagalan.
func newLogger(appEnv string) *slog.Logger {
	if appEnv == "production" {
		return slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

func run(cfg config.Config, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	router, err := httpapi.NewRouter(log, pool, cfg)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: router,
		// Batasi waktu membaca header agar koneksi lambat tidak menahan server. ReadTimeout
		// sengaja tidak dipasang karena unggahan 50 MB bisa butuh waktu lebih lama.
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Revisi 19: kemunculan task berulang dibuat di dalam proses ini setiap 5 menit, bukan cron host, supaya
	// tidak ada langkah deploy tambahan. Aman bila ada dua instance: pembuatnya idempoten.
	go task.NewService(pool).RunOccurrenceTicker(ctx, log, 5*time.Minute)

	serveErr := make(chan error, 1)
	go func() {
		log.Info("server jalan", "addr", cfg.HTTPAddr, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	// Beri request yang sedang berjalan waktu untuk selesai sebelum koneksi database ditutup.
	log.Info("mematikan server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
