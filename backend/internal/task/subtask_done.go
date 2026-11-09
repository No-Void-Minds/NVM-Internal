package task

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
)

var (
	ErrSubtaskNotFound = errors.New("sub task tidak ditemukan")
	ErrNotSubtaskPIC   = errors.New("bukan PIC sub task ini")
)

// SetSubtaskDone mencentang atau melepas centang. Hanya PIC sub task itu, dan hanya saat task On Progress.
// Task dikunci tanpa cek versi: centang tidak menaikkan version, supaya dua PIC yang mencentang sub task
// berbeda tidak saling menolak. Kunci yang sama dipakai Move, jadi centang tidak bisa menyelip saat task
// sedang dipindah ke Ready.
func (s *Service) SetSubtaskDone(ctx context.Context, actor Actor, taskID string, subtaskID int64, done bool) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		cur, err := q.GetTaskForUpdate(ctx, taskID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		st, err := q.GetSubtask(ctx, sqlc.GetSubtaskParams{ID: subtaskID, TaskID: taskID})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSubtaskNotFound
		}
		if err != nil {
			return err
		}
		isPIC, err := q.IsSubtaskAssignee(ctx, sqlc.IsSubtaskAssigneeParams{SubtaskID: subtaskID, UserID: actor.ID})
		if err != nil {
			return err
		}
		if !isPIC {
			return ErrNotSubtaskPIC
		}
		if Status(cur.Status) != OnProgress {
			return apperr.Invalid("Sub task hanya bisa dicentang atau dilepas saat task berstatus On Progress.")
		}
		if st.DoneAt.Valid == done {
			return nil
		}

		action := "subtask_unchecked"
		if done {
			action = "subtask_checked"
			err = q.CheckSubtask(ctx, sqlc.CheckSubtaskParams{
				UserID: pgtype.Int8{Int64: actor.ID, Valid: true},
				ID:     subtaskID,
				TaskID: taskID,
			})
		} else {
			err = q.UncheckSubtask(ctx, sqlc.UncheckSubtaskParams{ID: subtaskID, TaskID: taskID})
		}
		if err != nil {
			return err
		}
		return audit.Change(ctx, q, actor.auditActor(), action, "task", taskID, nil,
			map[string]any{"id": st.ID, "title": st.Title})
	})
}
