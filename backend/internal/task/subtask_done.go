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
	// Pesannya mengikuti canToggle (revisi 34).
	ErrNotSubtaskPIC = forbidden("Sub task ini hanya bisa dicentang PIC sub task, PIC task, pembuat task, atau Manajer.")
)

// SetSubtaskDone mencentang atau melepas centang, hanya saat task On Progress. Yang boleh: lihat canToggle.
// Task dikunci tanpa cek versi: centang tidak menaikkan version, supaya dua orang yang mencentang sub task
// berbeda tidak saling menolak. Kunci yang sama dipakai Move, jadi centang tidak bisa menyelip saat task
// sedang dipindah ke Ready. done_by diisi pelaku, jadi "Dicentang <nama>" menyebut orang yang mencentang.
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
		isOwner, err := q.IsSubtaskAssignee(ctx, sqlc.IsSubtaskAssigneeParams{SubtaskID: subtaskID, UserID: actor.ID})
		if err != nil {
			return err
		}
		isTaskPIC, err := q.IsTaskAssignee(ctx, sqlc.IsTaskAssigneeParams{TaskID: taskID, UserID: actor.ID})
		if err != nil {
			return err
		}
		if !canToggle(actor, isOwner, isTaskPIC, cur.RequestedBy == actor.ID) {
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

// canToggle adalah satu-satunya aturan siapa yang boleh mencentang dan melepas centang, dipakai SetSubtaskDone dan
// CheckableSubtasks. Revisi 34: PIC sub task itu, PIC task mana pun, pembuat task (requested_by), dan Manajer,
// untuk sub task mana pun di task itu. Menggantikan aturan lama: pembuat sub task tidak lagi disebut (biasanya
// sudah PIC task atau Manajer), dan Manajer tidak lagi dibatasi sub task yang dipegang Manajer lain (revisi 4).
// Admin selalu ditolak, termasuk bila tercatat sebagai peminta task hasil impor Excel.
func canToggle(actor Actor, isOwner, isTaskPIC, isRequester bool) bool {
	if actor.Role != RoleStaff && actor.Role != RoleManager {
		return false
	}
	return isManager(actor) || isOwner || isTaskPIC || isRequester
}

// CheckableSubtasks menjawab sub task mana yang boleh dicentang actor, untuk kolom can_check di detail.
// Kosong bila task tidak sedang On Progress.
func CheckableSubtasks(actor Actor, status Status, isTaskPIC, isRequester bool, subs []SubtaskView) map[int64]bool {
	out := make(map[int64]bool, len(subs))
	if status != OnProgress {
		return out
	}
	for _, st := range subs {
		isOwner := slices.ContainsFunc(st.Assignees, func(a sqlc.ListSubtaskAssigneesRow) bool { return a.ID == actor.ID })
		out[st.ID] = canToggle(actor, isOwner, isTaskPIC, isRequester)
	}
	return out
}

// isManager membandingkan lewat string karena Actor.Role dan sqlc.UserRole adalah dua tipe berbeda
// untuk nilai yang sama ("manager").
func isManager(actor Actor) bool {
	return string(actor.Role) == string(sqlc.UserRoleManager)
}
