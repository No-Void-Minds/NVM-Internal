// Seed mengisi divisi awal dan akun Admin pertama. Aman dijalankan berulang:
// data yang sudah ada tidak diubah, termasuk password Admin.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"nvm-internal/internal/config"
	"nvm-internal/internal/db"
)

var initialDivisions = []string{"Marketing", "Creative"}

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("seed gagal", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	username := strings.TrimSpace(os.Getenv("SEED_ADMIN_USERNAME"))
	password := os.Getenv("SEED_ADMIN_PASSWORD")
	if username == "" {
		return errors.New("SEED_ADMIN_USERNAME wajib diisi")
	}
	if password == "" || password == "GANTI" {
		return errors.New("SEED_ADMIN_PASSWORD belum diisi di .env")
	}

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		for _, name := range initialDivisions {
			tag, err := tx.Exec(ctx,
				`INSERT INTO divisions (name) VALUES ($1) ON CONFLICT (lower(name)) DO NOTHING`, name)
			if err != nil {
				return fmt.Errorf("divisi %s: %w", name, err)
			}
			slog.Info("divisi", "name", name, "dibuat", tag.RowsAffected() == 1)
		}

		var exists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM users WHERE lower(username) = lower($1))`, username,
		).Scan(&exists); err != nil {
			return err
		}
		if exists {
			slog.Info("akun admin sudah ada, dilewati", "username", username)
			return nil
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		// Admin ditaruh di Marketing sementara; bisa diubah dari halaman Pengguna (Sprint 2).
		var userID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO users (name, username, password_hash, role, division_id, must_change_password)
			VALUES ('Admin', $1, $2, 'admin', (SELECT id FROM divisions WHERE lower(name) = 'marketing'), true)
			RETURNING id`, username, string(hash),
		).Scan(&userID); err != nil {
			return fmt.Errorf("akun admin: %w", err)
		}

		detail, _ := json.Marshal(map[string]string{"username": username, "role": "admin", "source": "seed"})
		if _, err := tx.Exec(ctx, `
			INSERT INTO audit_logs (action, entity, entity_id, new_value)
			VALUES ('user_created', 'user', $1, $2)`, fmt.Sprint(userID), string(detail),
		); err != nil {
			return err
		}

		slog.Info("akun admin dibuat, wajib ganti password saat login pertama", "username", username)
		return nil
	})
}
