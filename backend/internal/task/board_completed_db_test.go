package task

import (
	"context"
	"fmt"
	"testing"
	"time"

	"nvm-internal/internal/db/sqlc"
)

func completedColumn(t *testing.T, cols []BoardColumn) BoardColumn {
	t.Helper()
	for _, c := range cols {
		if c.Status == Completed {
			return c
		}
	}
	t.Fatal("kolom Completed tidak ada")
	return BoardColumn{}
}

func completedIDs(c BoardColumn) []string {
	out := make([]string, len(c.Tasks))
	for i, s := range c.Tasks {
		out[i] = s.Task.ID
	}
	return out
}

// Revisi 40: kolom Completed paling banyak 10 task, yang paling baru masuk Completed di atas, dengan total yang
// cocok dengan filter; task yang selesai lebih dari 7 hari lalu tidak ikut; filter mengurangi total.
func TestKolomCompletedMaksimalSepuluh(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	now := time.Now()
	// 12 task selesai dalam 7 hari terakhir, task ke-i selesai i jam lalu: T-0001 paling baru.
	ids := make([]string, 12)
	for i := range ids {
		pic := f.stafA.ID
		if i%2 == 1 {
			pic = f.stafC.ID
		}
		ids[i] = f.newTask(t, pic)
		f.exec(t, `UPDATE tasks SET status = 'completed', solved_date = $2 WHERE id = $1`, ids[i], now.Add(-time.Duration(i+1)*time.Hour))
	}
	old := f.newTask(t, f.stafA.ID)
	f.exec(t, `UPDATE tasks SET status = 'completed', solved_date = $2 WHERE id = $1`, old, now.AddDate(0, 0, -10))

	cols, err := f.svc.Board(ctx, BoardFilter{}, now)
	if err != nil {
		t.Fatal(err)
	}
	c := completedColumn(t, cols)
	if c.Total != 12 || len(c.Tasks) != CompletedLimit {
		t.Fatalf("Completed: total %d tampil %d, mau total 12 tampil 10", c.Total, len(c.Tasks))
	}
	if got := fmt.Sprint(completedIDs(c)); got != fmt.Sprint(ids[:10]) {
		t.Fatalf("urutan %s, mau %s", got, ids[:10])
	}
	for _, id := range completedIDs(c) {
		if id == old {
			t.Fatal("task yang selesai lebih dari 7 hari lalu ikut tampil")
		}
	}

	// Filter PIC: hanya 6 task staf.a dalam 7 hari, jadi total 6 dan semuanya tampil.
	cols, err = f.svc.Board(ctx, BoardFilter{PICIDs: []int64{f.stafA.ID}}, now)
	if err != nil {
		t.Fatal(err)
	}
	if c := completedColumn(t, cols); c.Total != 6 || len(c.Tasks) != 6 {
		t.Fatalf("dengan filter PIC: total %d tampil %d, mau 6 dan 6", c.Total, len(c.Tasks))
	}

	// Task ke-12 (paling lama, tidak tampil) dikembalikan lalu Completed lagi lewat query aplikasi: naik ke atas.
	q := sqlc.New(f.pool)
	last := ids[11]
	if _, err := q.SetTaskStatus(ctx, sqlc.SetTaskStatusParams{ID: last, Status: sqlc.TaskStatusOnProgress}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.SetTaskStatus(ctx, sqlc.SetTaskStatusParams{ID: last, Status: sqlc.TaskStatusCompleted}); err != nil {
		t.Fatal(err)
	}
	cols, err = f.svc.Board(ctx, BoardFilter{}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	c = completedColumn(t, cols)
	if c.Tasks[0].Task.ID != last || c.Total != 12 {
		t.Fatalf("setelah Completed lagi: teratas %s total %d, mau %s dan 12", c.Tasks[0].Task.ID, c.Total, last)
	}
	if got := completedIDs(c)[CompletedLimit-1]; got != ids[8] {
		t.Fatalf("task ke-10 %s, mau %s (yang ke-11 tergeser keluar)", got, ids[8])
	}
}

// Waktu Completed yang sama dipecah dengan nomor task terbesar dulu, dibandingkan sebagai angka.
func TestSortCompletedPemecahSeri(t *testing.T) {
	at := time.Date(2026, 11, 24, 3, 0, 0, 0, time.UTC)
	rows := []sqlc.ListTasksRow{{ID: "T-9999"}, {ID: "T-10000"}, {ID: "T-0002"}}
	for i := range rows {
		rows[i].SolvedDate.Time, rows[i].SolvedDate.Valid = at, true
	}
	sortCompleted(rows)
	if got := []string{rows[0].ID, rows[1].ID, rows[2].ID}; fmt.Sprint(got) != "[T-10000 T-9999 T-0002]" {
		t.Fatalf("urutan seri %v", got)
	}
}
