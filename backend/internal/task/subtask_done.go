package task

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"slices"

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
		otherManager := false
		if !isPIC && isManager(actor) {
			others, qerr := q.ListSubtasksWithOtherManagerPIC(ctx, sqlc.ListSubtasksWithOtherManagerPICParams{
				SubtaskIds: []int64{subtaskID}, UserID: actor.ID,
			})
			if qerr != nil {
				return qerr
			}
			otherManager = len(others) > 0
		}
		if !canToggle(actor, isPIC, otherManager) {
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

// canToggle adalah satu-satunya aturan siapa yang boleh mencentang: PIC sub task itu, atau Manajer
// untuk sub task yang tidak dipegang Manajer lain. Dipakai SetSubtaskDone dan CheckableSubtasks.
func canToggle(actor Actor, isPIC, otherManagerPIC bool) bool {
	if isPIC {
		return true
	}
	return isManager(actor) && !otherManagerPIC
}

// CheckableSubtasks menjawab sub task mana yang boleh dicentang actor, untuk kolom can_check di detail.
// Satu query untuk semua sub task, dan kosong bila task tidak sedang On Progress.
func (s *Service) CheckableSubtasks(ctx context.Context, actor Actor, status Status, subs []SubtaskView) (map[int64]bool, error) {
	out := make(map[int64]bool, len(subs))
	if status != OnProgress || len(subs) == 0 {
		return out, nil
	}
	otherManager := map[int64]bool{}
	if isManager(actor) {
		ids := make([]int64, len(subs))
		for i, st := range subs {
			ids[i] = st.ID
		}
		rows, err := sqlc.New(s.pool).ListSubtasksWithOtherManagerPIC(ctx, sqlc.ListSubtasksWithOtherManagerPICParams{
			SubtaskIds: ids, UserID: actor.ID,
		})
		if err != nil {
			return nil, err
		}
		for _, id := range rows {
			otherManager[id] = true
		}
	}
	for _, st := range subs {
		isPIC := slices.ContainsFunc(st.Assignees, func(a sqlc.ListSubtaskAssigneesRow) bool { return a.ID == actor.ID })
		out[st.ID] = canToggle(actor, isPIC, otherManager[st.ID])
	}
	return out, nil
}

// isManager membandingkan lewat string karena Actor.Role dan sqlc.UserRole adalah dua tipe berbeda
// untuk nilai yang sama ("manager").
func isManager(actor Actor) bool {
	return string(actor.Role) == string(sqlc.UserRoleManager)
}
