package task

import (
	"context"
	"errors"
	"strings"
	"testing"

	"nvm-internal/internal/apperr"
)

func (f fixture) newTaskDue(t *testing.T, due string, subs ...SubtaskFields) (string, error) {
	t.Helper()
	return f.svc.Create(context.Background(), f.manager, CreateInput{
		Title: "Kampanye akhir tahun", Urgency: "high", PICIDs: []int64{f.stafA.ID}, DueDate: due, Subtasks: subs,
	})
}

func subtaskDue(t *testing.T, f fixture, taskID string, subID int64) string {
	t.Helper()
	d, err := f.svc.Get(context.Background(), taskID)
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range d.Subtasks {
		if st.ID == subID {
			if v, ok := dateValue(st.DueDate).(string); ok {
				return v
			}
			return ""
		}
	}
	t.Fatalf("sub task %d tidak ada", subID)
	return ""
}

// Revisi 16: tenggat sub task tidak boleh melewati tenggat task. Tenggat task yang dimajukan tetap diterima,
// dan sub task yang jadi melewatinya tetap bisa diganti namanya selama tenggatnya tidak diubah.
func TestSubtaskDueAgainstTaskDue(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	id, err := f.newTaskDue(t, "2026-12-10")
	if err != nil {
		t.Fatal(err)
	}

	var ve *apperr.Validation
	_, err = f.svc.AddSubtask(ctx, f.stafA, id, SubtaskFields{Title: "Video", DueDate: "2026-12-11"})
	if !errors.As(err, &ve) || !strings.Contains(ve.Message, "(10 Desember 2026)") {
		t.Fatalf("tenggat sub task melewati tenggat task: err %v, mau Validation menyebut 10 Desember 2026", err)
	}
	sub, err := f.svc.AddSubtask(ctx, f.stafA, id, SubtaskFields{Title: "Video", DueDate: "2026-12-10"})
	if err != nil {
		t.Fatalf("tenggat sama dengan tenggat task harus diterima: %v", err)
	}

	// Manajer memajukan tenggat task ke sebelum tenggat sub task: tetap diterima.
	d, err := f.svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Update(ctx, f.manager, id, UpdateInput{
		Version: d.Task.Version, Title: d.Task.Title, Urgency: string(d.Task.Urgency), PICIDs: []int64{f.stafA.ID},
		Subtasks: []SubtaskInput{{ID: sub, Title: "Video"}}, DueDate: "2026-12-05",
	}); err != nil {
		t.Fatalf("tenggat task dimajukan harus diterima: %v", err)
	}
	if got := subtaskDue(t, f, id, sub); got != "2026-12-10" {
		t.Fatalf("tenggat sub task berubah jadi %q, mau tetap 2026-12-10 (ditandai di detail)", got)
	}

	if err := f.svc.UpdateSubtask(ctx, f.stafA, id, sub, SubtaskFields{Title: "Video final", DueDate: "2026-12-10"}); err != nil {
		t.Fatalf("ganti nama tanpa mengubah tenggat harus lolos: %v", err)
	}
	if err := f.svc.UpdateSubtask(ctx, f.stafA, id, sub, SubtaskFields{Title: "Video final", DueDate: "2026-12-08"}); !errors.As(err, &ve) {
		t.Fatalf("mengubah tenggat ke setelah tenggat task baru: err %v, mau Validation", err)
	}
	if err := f.svc.UpdateSubtask(ctx, f.stafA, id, sub, SubtaskFields{Title: "Video final", DueDate: "2026-12-04"}); err != nil {
		t.Fatalf("memperbaiki tenggat sub task: %v", err)
	}
}

// Revisi 16: buat task dengan tiga sub task dalam satu transaksi, beserta isinya.
func TestCreateTaskWithSubtasks(t *testing.T) {
	f := setup(t)
	id, err := f.newTaskDue(t, "2026-12-20",
		SubtaskFields{Title: "Naskah", PICID: f.stafA.ID, DueDate: "2026-12-12", Reference: "https://contoh.id/brief"},
		SubtaskFields{Title: "Desain", PICID: f.stafC.ID, Description: "Ukuran 1080x1350"},
		// Tautan hasil diabaikan saat task dibuat: diisi saat pengerjaan.
		SubtaskFields{Title: "Unggah", Deliverable: "https://contoh.id/hasil"},
	)
	if err != nil {
		t.Fatal(err)
	}
	d, err := f.svc.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Subtasks) != 3 {
		t.Fatalf("jumlah sub task %d, mau 3", len(d.Subtasks))
	}
	s0, s1, s2 := d.Subtasks[0], d.Subtasks[1], d.Subtasks[2]
	if s0.Title != "Naskah" || s1.Title != "Desain" || s2.Title != "Unggah" {
		t.Fatalf("urutan sub task salah: %q, %q, %q", s0.Title, s1.Title, s2.Title)
	}
	if dateValue(s0.DueDate) != "2026-12-12" || s0.ReferenceURL != "https://contoh.id/brief" ||
		len(s0.Assignees) != 1 || s0.Assignees[0].ID != f.stafA.ID {
		t.Fatalf("isi sub task 1 tidak tersimpan utuh: %+v", s0)
	}
	if s1.Description != "Ukuran 1080x1350" || len(s1.Assignees) != 1 || s1.Assignees[0].ID != f.stafC.ID {
		t.Fatalf("isi sub task 2 tidak tersimpan utuh: %+v", s1)
	}
	if s2.DeliverableURL != "" || len(s2.Assignees) != 0 || s2.CreatedBy != f.manager.ID {
		t.Fatalf("sub task 3: tautan hasil harus kosong, tanpa PIC, pembuat Manajer, dapat %+v", s2)
	}
}

// Revisi 16: satu sub task tidak valid membatalkan task dan semua sub task, dan galatnya menyebut nomornya.
func TestCreateTaskInvalidSubtaskRollsBack(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	count := func() (tasks, subs int) {
		t.Helper()
		if err := f.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM tasks), (SELECT count(*) FROM task_subtasks)`).Scan(&tasks, &subs); err != nil {
			t.Fatal(err)
		}
		return
	}
	beforeTasks, beforeSubs := count()

	cases := map[string]struct {
		subs []SubtaskFields
		want string
	}{
		"tenggat melewati tenggat task": {
			[]SubtaskFields{{Title: "A"}, {Title: "B", DueDate: "2026-12-21"}, {Title: "C"}}, "Sub task nomor 2:",
		},
		"PIC Admin": {
			[]SubtaskFields{{Title: "A"}, {Title: "B"}, {Title: "C", PICID: f.admin.ID}}, "Sub task nomor 3:",
		},
		"nama kosong": {
			[]SubtaskFields{{Title: " "}, {Title: "B"}}, "Sub task nomor 1:",
		},
		"tautan javascript": {
			[]SubtaskFields{{Title: "A", Reference: "javascript:alert(1)"}}, "Sub task nomor 1:",
		},
	}
	for name, c := range cases {
		_, err := f.newTaskDue(t, "2026-12-20", c.subs...)
		var ve *apperr.Validation
		if !errors.As(err, &ve) || !strings.HasPrefix(ve.Message, c.want) {
			t.Errorf("%s: err %v, mau Validation berawalan %q", name, err, c.want)
		}
	}
	if tasks, subs := count(); tasks != beforeTasks || subs != beforeSubs {
		t.Fatalf("ada sisa setelah gagal: task %d -> %d, sub task %d -> %d", beforeTasks, tasks, beforeSubs, subs)
	}
}
