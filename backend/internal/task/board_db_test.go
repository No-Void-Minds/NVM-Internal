package task

import (
	"context"
	"errors"
	"nvm-internal/internal/apperr"
	"nvm-internal/internal/testdb"
	"slices"
	"testing"
	"time"
)

func (f fixture) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

// complete membawa task dari Requested sampai Completed. Staf A harus menjadi PIC task itu.
func (f fixture) complete(t *testing.T, id string) {
	t.Helper()
	v := int32(1)
	steps := []struct {
		actor Actor
		to    Status
	}{{f.stafA, OnProgress}, {f.stafA, Ready}, {f.manager, Completed}}
	for _, st := range steps {
		r, err := f.svc.Move(context.Background(), st.actor, id, v, st.to, "")
		if err != nil {
			t.Fatalf("pindah %s ke %s: %v", id, st.to, err)
		}
		v = r.Version
	}
}

func boardIDs(cols []BoardColumn) []string {
	var ids []string
	for _, c := range cols {
		for _, s := range c.Tasks {
			ids = append(ids, s.Task.ID)
		}
	}
	return ids
}

func TestBoardFilters(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	// Jam dibuat tetap supaya filter tenggat tidak bergantung pada kapan test dijalankan.
	now := time.Date(2026, 10, 29, 10, 0, 0, 0, officeTZ)

	mine := f.newTask(t, f.stafA.ID)
	theirs := f.newTask(t, f.stafC.ID)
	shared := f.newTask(t, f.stafA.ID, f.stafC.ID)
	urgent, err := f.svc.Create(ctx, f.manager, CreateInput{Title: "Live Shopee", Urgency: "urgent", PICIDs: []int64{f.stafC.ID}})
	if err != nil {
		t.Fatal(err)
	}
	ads := f.label(t, "Ads", nil, true)
	f.exec(t, `INSERT INTO task_labels (task_id, label_id) VALUES ($1, $2)`, mine, ads)
	f.exec(t, `UPDATE tasks SET due_date = '2026-10-28' WHERE id = $1`, mine)
	f.exec(t, `UPDATE tasks SET due_date = '2026-11-02' WHERE id = $1`, theirs)

	cases := []struct {
		name   string
		filter BoardFilter
		want   []string
	}{
		{"tanpa filter, urgent paling atas lalu tenggat terdekat", BoardFilter{}, []string{urgent, mine, theirs, shared}},
		{"task saya Staf A", BoardFilter{PICIDs: []int64{f.stafA.ID}}, []string{mine, shared}},
		{"PIC lewat username, huruf besar diabaikan", BoardFilter{PICUsernames: []string{" Staf.C "}}, []string{urgent, theirs, shared}},
		{"divisi Creative mengikuti PIC pertama", BoardFilter{DivisionID: &f.creative}, []string{urgent, theirs}},
		{"label Ads", BoardFilter{LabelID: &ads}, []string{mine}},
		{"urgency urgent", BoardFilter{Urgency: "urgent"}, []string{urgent}},
		{"terlambat", BoardFilter{Due: DueOverdue}, []string{mine}},
		{"tenggat 7 hari ke depan", BoardFilter{Due: DueWeek}, []string{theirs}},
		{"tanpa tenggat", BoardFilter{Due: DueNone}, []string{urgent, shared}},
		{"gabungan filter", BoardFilter{PICUsernames: []string{"staf.c"}, DivisionID: &f.marketing}, []string{shared}},
	}
	for _, c := range cases {
		cols, err := f.svc.Board(ctx, c.filter, now)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := boardIDs(cols); !slices.Equal(got, c.want) {
			t.Errorf("%s: dapat %v, mau %v", c.name, got, c.want)
		}
	}

	for _, bad := range []BoardFilter{{Urgency: "penting"}, {Due: "besok"}} {
		var ve *apperr.Validation
		if _, err := f.svc.Board(ctx, bad, now); !errors.As(err, &ve) {
			t.Errorf("filter %+v: err %v, mau Validation", bad, err)
		}
	}
}

// AC-08 bagian papan: Completed hanya tampil selama CompletedDays hari, terbaru di atas.
func TestBoardCompletedWindow(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 29, 10, 0, 0, 0, officeTZ)

	active := f.newTask(t, f.stafA.ID)
	recent := f.newTask(t, f.stafA.ID)
	older := f.newTask(t, f.stafA.ID)
	expired := f.newTask(t, f.stafA.ID)
	for _, id := range []string{recent, older, expired} {
		f.complete(t, id)
	}
	// request_date ikut dimundurkan supaya tanggal selesai buatan tidak lebih awal dari tanggal permintaan.
	setSolved := func(id string, at time.Time) {
		f.exec(t, `UPDATE tasks SET request_date = $2::timestamptz - interval '30 days', solved_date = $2 WHERE id = $1`, id, at)
	}
	setSolved(recent, now.Add(-time.Hour))
	setSolved(older, time.Date(2026, 10, 23, 0, 0, 0, 0, officeTZ))     // hari ke-7, masih tampil
	setSolved(expired, time.Date(2026, 10, 22, 23, 59, 0, 0, officeTZ)) // sudah lewat, hanya di Arsip

	cols, err := f.svc.Board(ctx, BoardFilter{}, now)
	if err != nil {
		t.Fatal(err)
	}
	var statuses []Status
	for _, c := range cols {
		statuses = append(statuses, c.Status)
	}
	if !slices.Equal(statuses, []Status{Requested, OnProgress, Ready, Completed}) {
		t.Fatalf("urutan kolom %v", statuses)
	}
	if got := boardIDs(cols[:1]); !slices.Equal(got, []string{active}) {
		t.Errorf("kolom Requested: %v, mau %v", got, []string{active})
	}
	if got := boardIDs(cols[3:]); !slices.Equal(got, []string{recent, older}) {
		t.Errorf("kolom Completed: %v, mau %v", got, []string{recent, older})
	}
}

func TestPeopleOnlyActiveStaffAndManagers(t *testing.T) {
	f := setup(t)
	testdb.User(t, f.pool, "staf.lama", "staff", f.marketing, false)

	rows, err := f.svc.People(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range rows {
		got = append(got, r.Username)
	}
	slices.Sort(got)
	// Admin dan akun nonaktif tidak boleh muncul sebagai calon PIC.
	if want := []string{"manajer.a", "staf.a", "staf.c"}; !slices.Equal(got, want) {
		t.Fatalf("calon PIC %v, mau %v", got, want)
	}
}
