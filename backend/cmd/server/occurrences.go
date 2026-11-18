package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"time"

	"nvm-internal/internal/config"
	"nvm-internal/internal/db"
	"nvm-internal/internal/task"
)

// runOccurrences: `server occurrences [--date YYYY-MM-DD]` menjalankan pembuat kemunculan task berulang sekali
// (revisi 19). --date mengganti "hari ini" (officeTZ) tanpa mengubah jam sistem, untuk uji di dev; tanpa --date
// memakai hari ini, sama dengan ticker di dalam server.
func runOccurrences(cfg config.Config, log *slog.Logger, args []string) error {
	fs := flag.NewFlagSet("occurrences", flag.ContinueOnError)
	date := fs.String("date", "", "tanggal kantor YYYY-MM-DD yang dianggap hari ini")
	if err := fs.Parse(args); err != nil {
		return err
	}
	now := time.Now()
	if *date != "" {
		d, err := time.ParseInLocation("2006-01-02", *date, task.OfficeTZ())
		if err != nil {
			return fmt.Errorf("--date harus berbentuk YYYY-MM-DD: %w", err)
		}
		// Tengah hari supaya tidak ada keraguan tanggal di zona mana pun.
		now = d.Add(12 * time.Hour)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	created, err := task.NewService(pool).CreateDueOccurrences(ctx, now)
	log.Info("kemunculan task berulang", "hari_ini", now.In(task.OfficeTZ()).Format("2006-01-02"), "dibuat", created)
	return err
}
