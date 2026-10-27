package auth

import (
	"context"
	"net/netip"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Test ini memakai database sungguhan dan hanya jalan bila TEST_DATABASE_URL diisi.
// Jangan arahkan ke database dev: semua tabel dikosongkan di awal setiap test.
func testPool(t *testing.T) *pgxpool.Pool {
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

	if _, err := pool.Exec(ctx, `TRUNCATE audit_logs, task_attachments, task_labels, task_assignees, tasks, labels, users, divisions RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}
	return pool
}

const testPassword = "password-uji-benar"

func createUser(t *testing.T, pool *pgxpool.Pool, username string, active bool) int64 {
	t.Helper()
	ctx := context.Background()
	hash, err := HashPassword(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	var divisionID, userID int64
	if err := pool.QueryRow(ctx, `INSERT INTO divisions (name) VALUES ('Uji') ON CONFLICT (lower(name)) DO UPDATE SET name = EXCLUDED.name RETURNING id`).Scan(&divisionID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (name, username, password_hash, role, division_id, is_active)
		VALUES ($1, $1, $2, 'staff', $3, $4) RETURNING id`, username, hash, divisionID, active,
	).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	return userID
}

func countAudit(t *testing.T, pool *pgxpool.Pool, action string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_logs WHERE action = $1`, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

var testIP = func() *netip.Addr { a := netip.MustParseAddr("192.168.1.10"); return &a }()

func TestLoginProgressiveLockoutWithDatabase(t *testing.T) {
	pool := testPool(t)
	svc := NewService(pool)
	ctx := context.Background()
	createUser(t, pool, "staf.a", true)

	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	stages := []int{300, 900, 1800, 1800}

	for i, wantRetry := range stages {
		for attempt := 1; attempt <= MaxFailedAttempts; attempt++ {
			res, err := svc.Login(ctx, "staf.a", "salah", testIP, now)
			if err != nil {
				t.Fatal(err)
			}
			want := LoginInvalid
			if attempt == MaxFailedAttempts {
				want = LoginLocked
			}
			if res.Outcome != want {
				t.Fatalf("tahap %d percobaan %d: outcome %v, mau %v", i+1, attempt, res.Outcome, want)
			}
			if attempt == MaxFailedAttempts && res.RetryAfterSeconds != wantRetry {
				t.Fatalf("tahap %d: kunci %d detik, mau %d", i+1, res.RetryAfterSeconds, wantRetry)
			}
		}

		res, err := svc.Login(ctx, "staf.a", testPassword, testIP, now.Add(time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if res.Outcome != LoginLocked {
			t.Fatalf("tahap %d: password benar saat terkunci harus ditolak", i+1)
		}
		now = now.Add(time.Duration(wantRetry) * time.Second)
	}

	if got := countAudit(t, pool, "account_locked"); got != len(stages) {
		t.Fatalf("account_locked tercatat %d kali, mau %d", got, len(stages))
	}

	res, err := svc.Login(ctx, "staf.a", testPassword, testIP, now)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != LoginOK {
		t.Fatalf("setelah kunci habis, password benar harus diterima, dapat %v", res.Outcome)
	}

	var failed, level int
	var lockedUntil *time.Time
	if err := pool.QueryRow(ctx, `SELECT failed_login_count, lockout_level, locked_until FROM users WHERE username = 'staf.a'`).
		Scan(&failed, &level, &lockedUntil); err != nil {
		t.Fatal(err)
	}
	if failed != 0 || level != 0 || lockedUntil != nil {
		t.Fatalf("login berhasil harus mengosongkan kuncian, dapat failed=%d level=%d locked_until=%v", failed, level, lockedUntil)
	}
}

func TestInactiveAccountLooksLikeWrongPassword(t *testing.T) {
	pool := testPool(t)
	svc := NewService(pool)
	createUser(t, pool, "staf.nonaktif", false)
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)

	res, err := svc.Login(context.Background(), "staf.nonaktif", testPassword, testIP, now)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != LoginInvalid {
		t.Fatalf("akun nonaktif dengan password benar: outcome %v, mau LoginInvalid", res.Outcome)
	}

	var failed int
	if err := pool.QueryRow(context.Background(), `SELECT failed_login_count FROM users WHERE username = 'staf.nonaktif'`).Scan(&failed); err != nil {
		t.Fatal(err)
	}
	if failed != 0 {
		t.Fatalf("akun nonaktif tidak boleh menambah hitungan gagal, dapat %d", failed)
	}
}

func TestUnknownUsernameAuditedWithoutUser(t *testing.T) {
	pool := testPool(t)
	svc := NewService(pool)
	now := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)

	res, err := svc.Login(context.Background(), "tidak.ada", "apa-saja", testIP, now)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != LoginInvalid {
		t.Fatalf("username tidak ada: outcome %v, mau LoginInvalid", res.Outcome)
	}

	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_logs WHERE action = 'login_failed' AND user_id IS NULL AND new_value->>'username' = 'tidak.ada'`,
	).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("login_failed tanpa user_id tercatat %d kali, mau 1", n)
	}
}

func TestCaseInsensitiveUsername(t *testing.T) {
	pool := testPool(t)
	svc := NewService(pool)
	createUser(t, pool, "staf.b", true)

	res, err := svc.Login(context.Background(), "STAF.B", testPassword, testIP, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome != LoginOK {
		t.Fatalf("username huruf besar harus tetap cocok, dapat %v", res.Outcome)
	}
}
