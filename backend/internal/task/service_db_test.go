package task

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/testdb"
)

type fixture struct {
	pool      *pgxpool.Pool
	svc       *Service
	marketing int64
	creative  int64
	manager   Actor
	admin     Actor
	stafA     Actor // Marketing
	stafC     Actor // Creative
}

func setup(t *testing.T) fixture {
	t.Helper()
	pool := testdb.Pool(t)
	f := fixture{pool: pool, svc: NewService(pool)}
	f.marketing = testdb.Division(t, pool, "Marketing", true)
	f.creative = testdb.Division(t, pool, "Creative", true)
	f.manager = Actor{ID: testdb.User(t, pool, "manajer.a", "manager", f.marketing, true), Role: RoleManager, DivisionID: f.marketing}
	f.admin = Actor{ID: testdb.User(t, pool, "admin.a", "admin", f.marketing, true), Role: RoleAdmin, DivisionID: f.marketing}
	f.stafA = Actor{ID: testdb.User(t, pool, "staf.a", "staff", f.marketing, true), Role: RoleStaff, DivisionID: f.marketing}
	f.stafC = Actor{ID: testdb.User(t, pool, "staf.c", "staff", f.creative, true), Role: RoleStaff, DivisionID: f.creative}
	return f
}

func (f fixture) newTask(t *testing.T, pics ...int64) string {
	t.Helper()
	id, err := f.svc.Create(context.Background(), f.manager, CreateInput{Title: "Foto katalog", Urgency: "high", PICIDs: pics})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestCreateTaskByManager(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	// PIC pertama dari Creative, jadi divisi task Creative walaupun Manajernya di Marketing.
	id := f.newTask(t, f.stafC.ID, f.stafA.ID)
	if id != "T-0001" {
		t.Fatalf("ID task pertama %q, mau T-0001", id)
	}
	d, err := f.svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if d.Task.Status != "requested" || d.Task.DivisionID != f.creative {
		t.Fatalf("task baru harus Requested di divisi PIC pertama, dapat status=%s divisi=%d", d.Task.Status, d.Task.DivisionID)
	}
	if len(d.Assignees) != 2 || d.Assignees[0].ID != f.stafC.ID {
		t.Fatalf("urutan PIC harus sesuai pilihan, dapat %+v", d.Assignees)
	}
	if !d.IsPIC(f.stafA.ID) || d.IsPIC(f.manager.ID) {
		t.Fatal("IsPIC salah")
	}
	if len(d.History) != 1 || d.History[0].Action != "task_created" {
		t.Fatalf("riwayat harus berisi task_created, dapat %+v", d.History)
	}
}

func TestOnlyManagerCreatesTask(t *testing.T) {
	f := setup(t)
	for _, actor := range []Actor{f.admin, f.stafA} {
		_, err := f.svc.Create(context.Background(), actor, CreateInput{Title: "X", Urgency: "low", PICIDs: []int64{f.stafA.ID}})
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("%s membuat task: err %v, mau ErrForbidden", actor.Role, err)
		}
	}
}

func TestCreateTaskValidation(t *testing.T) {
	f := setup(t)
	inactiveLabel := int64(0)
	if err := f.pool.QueryRow(context.Background(),
		`INSERT INTO labels (name, color, is_active) VALUES ('Lama', '#E6E1D7', false) RETURNING id`).Scan(&inactiveLabel); err != nil {
		t.Fatal(err)
	}

	cases := map[string]CreateInput{
		"tanpa judul":           {Title: " ", Urgency: "low", PICIDs: []int64{f.stafA.ID}},
		"tanpa PIC":             {Title: "X", Urgency: "low"},
		"Admin sebagai PIC":     {Title: "X", Urgency: "low", PICIDs: []int64{f.admin.ID}},
		"urgency asing":         {Title: "X", Urgency: "kritis", PICIDs: []int64{f.stafA.ID}},
		"tenggat sebelum mulai": {Title: "X", Urgency: "low", PICIDs: []int64{f.stafA.ID}, StartDate: "2026-11-10", DueDate: "2026-11-01"},
		"tautan javascript":     {Title: "X", Urgency: "low", PICIDs: []int64{f.stafA.ID}, DeliverableURL: "javascript:alert(1)"},
		"label nonaktif":        {Title: "X", Urgency: "low", PICIDs: []int64{f.stafA.ID}, LabelIDs: []int64{inactiveLabel}},
	}
	for name, in := range cases {
		_, err := f.svc.Create(context.Background(), f.manager, in)
		var v *apperr.Validation
		if !errors.As(err, &v) {
			t.Errorf("%s: err %v, mau Validation", name, err)
		}
	}
}

func TestListMineOnlyReturnsOwnTasks(t *testing.T) {
	f := setup(t)
	f.newTask(t, f.stafA.ID)
	f.newTask(t, f.stafC.ID)

	all, err := f.svc.List(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	mine, err := f.svc.List(context.Background(), &f.stafA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 || len(mine) != 1 || mine[0].Assignees[0].ID != f.stafA.ID {
		t.Fatalf("semua=%d, milik staf.a=%d; mau 2 dan 1", len(all), len(mine))
	}
}
