package audit

import (
	"context"
	"testing"
	"time"

	"nvm-internal/internal/db/sqlc"
	"nvm-internal/internal/testdb"
)

func TestListFiltersAndPaginates(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	q := sqlc.New(pool)
	division := testdb.Division(t, pool, "Marketing", true)
	admin := testdb.User(t, pool, "admin.a", "admin", division, true)
	staf := testdb.User(t, pool, "staf.a", "staff", division, true)

	for range PageSize + 5 {
		if err := UserEvent(ctx, q, &staf, "login_success", nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := Record(ctx, q, Entry{ActorID: &admin, Action: "user_deactivated", Entity: "user", EntityID: "2"}); err != nil {
		t.Fatal(err)
	}

	rows, hasMore, err := List(ctx, q, Filter{Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != PageSize || !hasMore {
		t.Fatalf("halaman 1: %d baris, hasMore=%v; mau %d baris dan masih ada halaman berikutnya", len(rows), hasMore, PageSize)
	}
	if rows[0].Action != "user_deactivated" {
		t.Fatalf("baris pertama %q, mau yang paling baru (user_deactivated)", rows[0].Action)
	}
	if !rows[0].TargetUserName.Valid || rows[0].TargetUserName.String != "staf.a" {
		t.Fatalf("nama sasaran %+v, mau staf.a", rows[0].TargetUserName)
	}

	rows, hasMore, err = List(ctx, q, Filter{Page: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 6 || hasMore {
		t.Fatalf("halaman 2: %d baris, hasMore=%v; mau 6 baris dan halaman terakhir", len(rows), hasMore)
	}

	rows, _, err = List(ctx, q, Filter{Category: "account", ActorID: &admin})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || !rows[0].ActorName.Valid || rows[0].ActorName.String != "admin.a" {
		t.Fatalf("filter kategori akun dan pelaku admin: %+v", rows)
	}

	future := time.Now().Add(time.Hour)
	rows, _, err = List(ctx, q, Filter{From: &future})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("filter tanggal di masa depan harus kosong, dapat %d baris", len(rows))
	}

	if _, _, err := List(ctx, q, Filter{Category: "ngawur"}); err != ErrUnknownCategory {
		t.Fatalf("kategori asing: err %v, mau ErrUnknownCategory", err)
	}
}
