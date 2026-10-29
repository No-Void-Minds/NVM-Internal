package label

import (
	"context"
	"errors"
	"testing"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/audit"
	"nvm-internal/internal/testdb"
)

func TestLabelLifecycle(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	svc := NewService(pool)
	marketing := testdb.Division(t, pool, "Marketing", true)
	lama := testdb.Division(t, pool, "Lama", false)
	actor := audit.Actor{ID: testdb.User(t, pool, "manajer.a", "manager", marketing, true)}

	id, err := svc.Create(ctx, actor, Input{Name: "Konten", Color: "#e6e1d7"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Create(ctx, actor, Input{Name: "KONTEN", Color: Palette[1]}); !errors.Is(err, ErrNameTaken) {
		t.Fatalf("nama kembar beda huruf: err %v, mau ErrNameTaken", err)
	}
	var v *apperr.Validation
	if _, err := svc.Create(ctx, actor, Input{Name: "Ads", Color: "#FF0000"}); !errors.As(err, &v) {
		t.Fatalf("warna di luar palet: err %v, mau Validation", err)
	}
	if _, err := svc.Create(ctx, actor, Input{Name: "Ads", Color: Palette[2], DivisionID: &lama}); !errors.As(err, &v) {
		t.Fatalf("divisi nonaktif: err %v, mau Validation", err)
	}

	if err := svc.Update(ctx, actor, id, Input{Name: "Konten", Color: Palette[0], DivisionID: &marketing}); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetActive(ctx, actor, id, false); err != nil {
		t.Fatal(err)
	}

	rows, err := svc.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].IsActive || rows[0].Color != "#E6E1D7" || !rows[0].DivisionName.Valid {
		t.Fatalf("daftar label tidak sesuai: %+v", rows)
	}

	var actions []string
	r, err := pool.Query(ctx, `SELECT action FROM audit_logs WHERE entity = 'label' ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	for r.Next() {
		var a string
		if err := r.Scan(&a); err != nil {
			t.Fatal(err)
		}
		actions = append(actions, a)
	}
	want := []string{"label_created", "label_updated", "label_deactivated"}
	if len(actions) != len(want) {
		t.Fatalf("audit label %v, mau %v", actions, want)
	}
	for i := range want {
		if actions[i] != want[i] {
			t.Fatalf("audit label %v, mau %v", actions, want)
		}
	}
}
