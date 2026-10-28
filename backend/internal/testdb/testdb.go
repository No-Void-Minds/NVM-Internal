// Package testdb menyiapkan database untuk test yang butuh PostgreSQL sungguhan.
package testdb

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// Password dipakai semua akun buatan helper ini.
const Password = "password-uji-benar"

// lockKey adalah nomor advisory lock bersama untuk semua test database.
const lockKey = 7_242_026

// Pool membuka TEST_DATABASE_URL dan mengosongkan semua tabel. Test dilewati bila variabel itu kosong.
// Jangan arahkan ke database dev.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL kosong, test database dilewati")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	// go test menjalankan beberapa package sekaligus, dan semuanya memakai database yang sama.
	// Kunci ini membuat test database mengantre, supaya TRUNCATE tidak mengenai transaksi test lain.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockKey); err != nil {
		conn.Release()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, lockKey)
		conn.Release()
	})

	if _, err := pool.Exec(ctx, `TRUNCATE audit_logs, task_attachments, task_labels, task_assignees, tasks, labels, users, divisions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}
	return pool
}

func Division(t *testing.T, pool *pgxpool.Pool, name string, active bool) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO divisions (name, is_active) VALUES ($1, $2) RETURNING id`, name, active,
	).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// User membuat akun dengan Password. bcrypt.MinCost dipakai supaya test cepat; kekuatan hash tidak diuji di sini.
func User(t *testing.T, pool *pgxpool.Pool, username, role string, divisionID int64, active bool) int64 {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(Password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO users (name, username, password_hash, role, division_id, is_active, must_change_password)
		VALUES ($1, $1, $2, $3, $4, $5, false) RETURNING id`, username, string(hash), role, divisionID, active,
	).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
