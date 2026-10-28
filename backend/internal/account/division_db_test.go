package account

import (
	"context"
	"errors"
	"testing"
)

func TestDivisionNameUniqueCaseInsensitive(t *testing.T) {
	f := setup(t)
	_, err := f.svc.CreateDivision(context.Background(), f.actor(), "  MARKETING ")
	if !errors.Is(err, ErrDivisionNameTaken) {
		t.Fatalf("nama divisi kembar beda huruf: err %v, mau ErrDivisionNameTaken", err)
	}
}

func TestRenameDivisionAudited(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	id, err := f.svc.CreateDivision(ctx, f.actor(), "Creative")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.RenameDivision(ctx, f.actor(), id, "Kreatif"); err != nil {
		t.Fatal(err)
	}
	got := scalar[string](t, f.pool,
		`SELECT old_value->>'name' || '>' || (new_value->>'name') FROM audit_logs WHERE action = 'division_renamed'`)
	if got != "Creative>Kreatif" {
		t.Fatalf("audit division_renamed berisi %q, mau Creative>Kreatif", got)
	}
}

func TestDeactivatedDivisionKeepsUsers(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if err := f.svc.SetDivisionActive(ctx, f.actor(), f.marketing, false); err != nil {
		t.Fatal(err)
	}
	// Admin dari setup tetap di divisi Marketing walaupun divisinya nonaktif.
	got := scalar[int64](t, f.pool, `SELECT division_id FROM users WHERE id = $1`, f.admin)
	if got != f.marketing {
		t.Fatal("akun lama harus tetap di divisinya saat divisi dinonaktifkan")
	}
	rows, err := f.svc.ListDivisions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].IsActive || rows[0].ActiveUsers != 1 {
		t.Fatalf("daftar divisi tidak sesuai: %+v", rows)
	}
}
