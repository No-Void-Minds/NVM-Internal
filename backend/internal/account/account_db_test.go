package account

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/db/sqlc"
	"nvm-internal/internal/testdb"
)

type fixture struct {
	pool      *pgxpool.Pool
	svc       *Service
	marketing int64
	admin     int64
}

func setup(t *testing.T) fixture {
	t.Helper()
	pool := testdb.Pool(t)
	marketing := testdb.Division(t, pool, "Marketing", true)
	admin := testdb.User(t, pool, "admin.a", "admin", marketing, true)
	return fixture{pool: pool, svc: NewService(pool), marketing: marketing, admin: admin}
}

func (f fixture) actor() Actor { return Actor{ID: f.admin} }

func scalar[T any](t *testing.T, pool *pgxpool.Pool, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestCreateUserAndUsernameCaseInsensitive(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	id, err := f.svc.Create(ctx, f.actor(), CreateInput{
		Name: "Staf A", Username: "staf.a", Role: sqlc.UserRoleStaff, DivisionID: f.marketing, Password: "sementara123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !scalar[bool](t, f.pool, `SELECT must_change_password FROM users WHERE id = $1`, id) {
		t.Fatal("akun baru wajib ganti password saat login pertama")
	}

	_, err = f.svc.Create(ctx, f.actor(), CreateInput{
		Name: "Staf A Lagi", Username: "STAF.A", Role: sqlc.UserRoleStaff, DivisionID: f.marketing, Password: "sementara123",
	})
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("username kembar beda huruf: err %v, mau ErrUsernameTaken", err)
	}

	n := scalar[int](t, f.pool, `SELECT count(*) FROM audit_logs WHERE action = 'user_created' AND user_id = $1 AND entity_id = $2`, f.admin, "2")
	if n != 1 {
		t.Fatalf("user_created dengan pelaku admin dan sasaran akun baru tercatat %d kali, mau 1", n)
	}
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	f := setup(t)
	inactive := testdb.Division(t, f.pool, "Lama", false)
	cases := map[string]CreateInput{
		"username berspasi": {Name: "X", Username: "staf a", Role: sqlc.UserRoleStaff, DivisionID: f.marketing, Password: "sementara123"},
		"password pendek":   {Name: "X", Username: "staf.x", Role: sqlc.UserRoleStaff, DivisionID: f.marketing, Password: "pendek"},
		"peran asing":       {Name: "X", Username: "staf.x", Role: "owner", DivisionID: f.marketing, Password: "sementara123"},
		"divisi nonaktif":   {Name: "X", Username: "staf.x", Role: sqlc.UserRoleStaff, DivisionID: inactive, Password: "sementara123"},
	}
	for name, in := range cases {
		_, err := f.svc.Create(context.Background(), f.actor(), in)
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("%s: err %v, mau ValidationError", name, err)
		}
	}
}

func TestCannotDeactivateSelf(t *testing.T) {
	f := setup(t)
	err := f.svc.SetActive(context.Background(), f.actor(), f.admin, false)
	if !errors.Is(err, ErrSelfDeactivate) {
		t.Fatalf("err %v, mau ErrSelfDeactivate", err)
	}
}

func TestLastManagerProtected(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	manager := testdb.User(t, f.pool, "manajer.a", "manager", f.marketing, true)

	// Dua pengelola aktif: Manajer boleh menurunkan Admin menjadi Staf.
	if err := f.svc.Update(ctx, Actor{ID: manager}, f.admin, UpdateInput{Name: "Admin A", Role: sqlc.UserRoleStaff, DivisionID: f.marketing}); err != nil {
		t.Fatalf("menurunkan salah satu dari dua pengelola harus boleh: %v", err)
	}
	// Sekarang Manajer satu-satunya pengelola: menurunkan dirinya sendiri harus ditolak.
	err := f.svc.Update(ctx, Actor{ID: manager}, manager, UpdateInput{Name: "Manajer A", Role: sqlc.UserRoleStaff, DivisionID: f.marketing})
	if !errors.Is(err, ErrLastManager) {
		t.Fatalf("menurunkan pengelola terakhir: err %v, mau ErrLastManager", err)
	}
}

func TestDeactivateEndsSessionsAndActivateRestores(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	staf := testdb.User(t, f.pool, "staf.b", "staff", f.marketing, true)
	before := scalar[int32](t, f.pool, `SELECT token_version FROM users WHERE id = $1`, staf)

	if err := f.svc.SetActive(ctx, f.actor(), staf, false); err != nil {
		t.Fatal(err)
	}
	after := scalar[int32](t, f.pool, `SELECT token_version FROM users WHERE id = $1`, staf)
	if after <= before {
		t.Fatal("menonaktifkan akun harus menaikkan token_version supaya sesinya berakhir")
	}
	if err := f.svc.SetActive(ctx, f.actor(), staf, true); err != nil {
		t.Fatal(err)
	}
	if !scalar[bool](t, f.pool, `SELECT is_active FROM users WHERE id = $1`, staf) {
		t.Fatal("akun harus aktif lagi")
	}
}

func TestResetPasswordUnlocksAndForcesChange(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	staf := testdb.User(t, f.pool, "staf.c", "staff", f.marketing, true)
	if _, err := f.pool.Exec(ctx, `UPDATE users SET lockout_level = 2, locked_until = now() + interval '15 minutes' WHERE id = $1`, staf); err != nil {
		t.Fatal(err)
	}

	if err := f.svc.ResetPassword(ctx, f.actor(), staf, "sementara456"); err != nil {
		t.Fatal(err)
	}
	locked := scalar[bool](t, f.pool, `SELECT locked_until IS NOT NULL OR lockout_level <> 0 FROM users WHERE id = $1`, staf)
	mustChange := scalar[bool](t, f.pool, `SELECT must_change_password FROM users WHERE id = $1`, staf)
	if locked || !mustChange {
		t.Fatalf("reset password harus membuka kunci dan mewajibkan ganti password, dapat locked=%v mustChange=%v", locked, mustChange)
	}
	leaked := scalar[int](t, f.pool, `SELECT count(*) FROM audit_logs WHERE new_value::text LIKE '%sementara456%' OR old_value::text LIKE '%sementara456%'`)
	if leaked != 0 {
		t.Fatal("password tidak boleh tercatat di audit log")
	}
}
