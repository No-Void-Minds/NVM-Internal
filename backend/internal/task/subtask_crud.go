package task

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
)

// Revisi 6: Manajer membuat task dan memilih PIC, lalu PIC menyusun sendiri sub task yang ia kerjakan.
// Manajer tetap boleh menambah sub task. Sub task yang dibuat PIC langsung menjadi miliknya, supaya
// aturan centang yang lama (pemilik sub task yang mencentang) tetap berlaku tanpa aturan baru.

// ErrSubtaskLocked dipakai saat sub task tidak boleh diubah karena status task atau karena bukan miliknya.
var ErrSubtaskLocked = errors.New("sub task tidak bisa diubah")

// subtaskWindow: menambah dan mengubah sub task hanya saat task masih disiapkan atau sedang dikerjakan.
// Ditutup mulai Ready supaya pekerjaan tidak bertambah saat Manajer sedang mereview, dan supaya syarat
// "semua sub task tercentang" tidak bisa digoyang setelah task lolos ke Ready.
func subtaskWindowOpen(status Status) bool {
	return status == Requested || status == OnProgress
}

// lockForSubtask mengunci task dan memastikan pelaku boleh menyentuh sub task-nya sama sekali:
// Manajer, atau Staf yang menjadi PIC task.
func lockForSubtask(ctx context.Context, q *sqlc.Queries, actor Actor, taskID string) (sqlc.GetTaskForUpdateRow, error) {
	cur, err := q.GetTaskForUpdate(ctx, taskID)
	if errors.Is(err, pgx.ErrNoRows) {
		return cur, ErrNotFound
	}
	if err != nil {
		return cur, err
	}
	ok, err := canContribute(ctx, q, actor, taskID)
	if err != nil {
		return cur, err
	}
	if !ok {
		return cur, ErrForbidden
	}
	if !subtaskWindowOpen(Status(cur.Status)) {
		return cur, apperr.Invalid("Sub task hanya bisa diubah saat task berstatus Requested atau On Progress.")
	}
	return cur, nil
}

// AddSubtask menambah satu sub task di urutan paling bawah. Pelaku menjadi pemiliknya, termasuk kalau
// pelakunya Manajer yang juga PIC task ini.
func (s *Service) AddSubtask(ctx context.Context, actor Actor, taskID, title string) (int64, error) {
	clean, err := cleanSubtaskTitle(title, "ini")
	if err != nil {
		return 0, err
	}
	var id int64
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if _, err := lockForSubtask(ctx, q, actor, taskID); err != nil {
			return err
		}
		slot, err := q.SubtaskSlot(ctx, taskID)
		if err != nil {
			return err
		}
		if int(slot.Total) >= maxSubtasks {
			return apperr.Invalid("Satu task paling banyak punya 50 sub task.")
		}
		id, err = q.CreateSubtask(ctx, sqlc.CreateSubtaskParams{TaskID: taskID, Title: clean, Position: slot.NextPosition})
		if err != nil {
			return err
		}
		if err := q.AddSubtaskAssignee(ctx, sqlc.AddSubtaskAssigneeParams{
			SubtaskID: id, UserID: actor.ID, AssignedBy: actor.ID,
		}); err != nil {
			return err
		}
		// Versi task ikut naik supaya form Manajer yang sudah terbuka tidak menyimpan daftar sub task
		// yang basi, karena daftar dari form menggantikan daftar lama secara utuh.
		if _, err := q.BumpTaskVersion(ctx, taskID); err != nil {
			return err
		}
		return audit.Change(ctx, q, actor.auditActor(), "subtask_added", "task", taskID, nil,
			map[string]any{"id": id, "title": clean})
	})
	return id, err
}

// ownSubtask memastikan sub task ada di task ini dan boleh diubah pelaku: pemiliknya, atau Manajer.
func ownSubtask(ctx context.Context, q *sqlc.Queries, actor Actor, taskID string, subtaskID int64) (sqlc.GetSubtaskRow, error) {
	st, err := q.GetSubtask(ctx, sqlc.GetSubtaskParams{ID: subtaskID, TaskID: taskID})
	if errors.Is(err, pgx.ErrNoRows) {
		return st, ErrSubtaskNotFound
	}
	if err != nil {
		return st, err
	}
	if isManager(actor) {
		return st, nil
	}
	owner, err := q.IsSubtaskAssignee(ctx, sqlc.IsSubtaskAssigneeParams{SubtaskID: subtaskID, UserID: actor.ID})
	if err != nil {
		return st, err
	}
	if !owner {
		return st, ErrSubtaskLocked
	}
	return st, nil
}

// RenameSubtask mengubah nama sub task tanpa menggeser urutannya.
func (s *Service) RenameSubtask(ctx context.Context, actor Actor, taskID string, subtaskID int64, title string) error {
	clean, err := cleanSubtaskTitle(title, "ini")
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if _, err := lockForSubtask(ctx, q, actor, taskID); err != nil {
			return err
		}
		st, err := ownSubtask(ctx, q, actor, taskID, subtaskID)
		if err != nil {
			return err
		}
		if st.Title == clean {
			return nil
		}
		if err := q.RenameSubtask(ctx, sqlc.RenameSubtaskParams{ID: subtaskID, TaskID: taskID, Title: clean}); err != nil {
			return err
		}
		if _, err := q.BumpTaskVersion(ctx, taskID); err != nil {
			return err
		}
		return audit.Change(ctx, q, actor.auditActor(), "subtask_renamed", "task", taskID,
			map[string]any{"id": st.ID, "title": st.Title}, map[string]any{"id": st.ID, "title": clean})
	})
}

// DeleteSubtask menghapus satu sub task. Posisi sisanya dibiarkan berlubang; urutan hanya dibaca terurut.
func (s *Service) DeleteSubtask(ctx context.Context, actor Actor, taskID string, subtaskID int64) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if _, err := lockForSubtask(ctx, q, actor, taskID); err != nil {
			return err
		}
		st, err := ownSubtask(ctx, q, actor, taskID, subtaskID)
		if err != nil {
			return err
		}
		if err := q.DeleteSubtasks(ctx, sqlc.DeleteSubtasksParams{TaskID: taskID, Ids: []int64{subtaskID}}); err != nil {
			return err
		}
		if _, err := q.BumpTaskVersion(ctx, taskID); err != nil {
			return err
		}
		return audit.Change(ctx, q, actor.auditActor(), "subtask_removed", "task", taskID,
			map[string]any{"id": st.ID, "title": st.Title}, nil)
	})
}
