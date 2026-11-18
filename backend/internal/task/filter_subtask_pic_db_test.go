package task

import (
	"context"
	"testing"
	"time"
)

// Revisi 15: filter PIC = X cocok bila X PIC task induk ATAU PIC salah satu sub task. Empat task:
// hanya PIC task, hanya PIC sub task, keduanya, dan tidak terkait. X adalah stafC.
type picCase struct {
	onlyTask, onlySub, both, none string
}

func setupPICCases(t *testing.T, f fixture) picCase {
	t.Helper()
	ctx := context.Background()
	c := picCase{
		onlyTask: f.newTask(t, f.stafC.ID),
		onlySub:  f.newTask(t, f.stafA.ID),
		both:     f.newTask(t, f.stafC.ID),
		none:     f.newTask(t, f.stafA.ID),
	}
	subs := []struct {
		actor Actor
		task  string
		pic   int64
	}{
		{f.stafA, c.onlySub, f.stafC.ID},
		{f.stafC, c.both, f.stafC.ID},
		{f.stafA, c.none, f.stafA.ID},
	}
	for _, s := range subs {
		if _, err := f.svc.AddSubtask(ctx, s.actor, s.task, SubtaskFields{Title: "Sub " + s.task, PICID: s.pic}); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

// matches mengembalikan berapa kali tiap task muncul, plus nama PIC sub task yang dikirim untuknya.
func matches(list []Summary) (map[string]int, map[string][]int64) {
	seen := map[string]int{}
	subPICs := map[string][]int64{}
	for _, s := range list {
		seen[s.Task.ID]++
		for _, p := range s.SubtaskPICs {
			subPICs[s.Task.ID] = append(subPICs[s.Task.ID], p.ID)
		}
	}
	return seen, subPICs
}

func checkPICResult(t *testing.T, where string, f fixture, c picCase, list []Summary) {
	t.Helper()
	seen, subPICs := matches(list)
	for _, id := range []string{c.onlyTask, c.onlySub, c.both} {
		if seen[id] != 1 {
			t.Errorf("%s: %s muncul %d kali, mau 1", where, id, seen[id])
		}
	}
	if seen[c.none] != 0 {
		t.Errorf("%s: task tidak terkait %s ikut muncul", where, c.none)
	}
	if got := subPICs[c.onlySub]; len(got) != 1 || got[0] != f.stafC.ID {
		t.Errorf("%s: PIC sub task untuk %s = %v, mau [%d]", where, c.onlySub, got, f.stafC.ID)
	}
	// Bila X juga PIC task induk, baris "Sub task" tidak dikirim karena namanya sudah ada di daftar PIC.
	if len(subPICs[c.both]) != 0 || len(subPICs[c.onlyTask]) != 0 {
		t.Errorf("%s: PIC sub task tidak boleh dikirim untuk PIC task induk, dapat %v", where, subPICs)
	}
}

func TestBoardPICFilterIncludesSubtaskPIC(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	c := setupPICCases(t, f)

	// Satu task pindah ke On Progress supaya penghitung tiap kolom ikut teruji.
	d, err := f.svc.Get(ctx, c.onlySub)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Move(ctx, f.stafA, c.onlySub, d.Task.Version, OnProgress, ""); err != nil {
		t.Fatal(err)
	}

	// Lewat id (dipakai "Task saya") dan lewat username (dipakai filter PIC papan).
	for name, filter := range map[string]BoardFilter{
		"pic id":       {PICIDs: []int64{f.stafC.ID}},
		"pic username": {PICUsernames: []string{"Staf.C"}},
	} {
		cols, err := f.svc.Board(ctx, filter, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		var all []Summary
		count := map[Status]int{}
		for _, col := range cols {
			all = append(all, col.Tasks...)
			count[col.Status] = len(col.Tasks)
		}
		checkPICResult(t, "papan "+name, f, c, all)
		if count[Requested] != 2 || count[OnProgress] != 1 || count[Ready] != 0 || count[Completed] != 0 {
			t.Errorf("papan %s: hitungan kolom %v, mau Requested 2 dan On Progress 1", name, count)
		}
	}

	// Tanpa filter PIC: semua task, dan baris PIC sub task kosong.
	cols, err := f.svc.Board(ctx, BoardFilter{}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, col := range cols {
		for _, s := range col.Tasks {
			if len(s.SubtaskPICs) != 0 {
				t.Errorf("tanpa filter PIC, %s tetap membawa PIC sub task %v", s.Task.ID, s.SubtaskPICs)
			}
		}
	}

	// GET /tasks?scope=mine memakai List.
	list, err := f.svc.List(ctx, &f.stafC.ID)
	if err != nil {
		t.Fatal(err)
	}
	seen, _ := matches(list)
	for _, id := range []string{c.onlyTask, c.onlySub, c.both} {
		if seen[id] != 1 {
			t.Errorf("List mine: %s muncul %d kali, mau 1", id, seen[id])
		}
	}
	if seen[c.none] != 0 {
		t.Errorf("List mine: task tidak terkait %s ikut muncul", c.none)
	}
}

func TestArchivePICFilterIncludesSubtaskPIC(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	c := setupPICCases(t, f)
	// Status ditulis langsung: yang diuji di sini filter Arsip, bukan alur status (sudah di TestLifecycleWithDatabase).
	if _, err := f.pool.Exec(ctx, `UPDATE tasks SET status = 'completed', solved_date = now() WHERE id = ANY($1)`,
		[]string{c.onlyTask, c.onlySub, c.both, c.none}); err != nil {
		t.Fatal(err)
	}
	for name, filter := range map[string]ArchiveFilter{
		"pic id":       {PICIDs: []int64{f.stafC.ID}},
		"pic username": {PICUsernames: []string{"staf.c"}},
	} {
		page, err := f.svc.Archive(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		checkPICResult(t, "arsip "+name, f, c, page.Tasks)
		if page.Total != 3 {
			t.Errorf("arsip %s: total %d, mau 3", name, page.Total)
		}
	}
}
