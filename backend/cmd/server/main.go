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
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config tidak valid", "err", err)
		os.Exit(1)
	}

	log := newLogger(cfg.AppEnv)
	if err := run(cfg, log); err != nil {
		log.Error("server berhenti karena galat", "err", err)
		os.Exit(1)
	}
}

// Teks lebih enak dibaca saat dev; JSON di produksi supaya log Docker bisa disaring per field.
func newLogger(appEnv string) *slog.Logger {
	if appEnv == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
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
