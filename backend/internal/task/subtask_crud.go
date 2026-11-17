package task

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
)

// Revisi 7: Manajer memilih PIC task, lalu PIC mana pun boleh menyusun sub task dan menugaskannya ke
// PIC task yang lain (mis. supervisor membagi pekerjaan ke timnya). Satu sub task satu PIC, dan yang
// mencentang adalah PIC sub task itu. Sub task yang belum ditugaskan boleh dicentang PIC task mana pun.

// ErrSubtaskLocked dipakai saat sub task tidak boleh diubah karena bukan urusan pelaku.
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

// setSubtaskPIC memasang atau melepas PIC sub task. Kandidatnya dibatasi PIC task supaya pekerjaan tidak
// bisa dilempar ke orang yang tidak dipilih Manajer di task ini.
func setSubtaskPIC(ctx context.Context, q *sqlc.Queries, actor Actor, taskID string, subtaskID int64, picID int64) error {
	if picID != 0 {
		ok, err := q.IsTaskAssignee(ctx, sqlc.IsTaskAssigneeParams{TaskID: taskID, UserID: picID})
		if err != nil {
			return err
		}
		if !ok {
			return apperr.Invalid("PIC sub task harus salah satu PIC task ini.")
		}
	}
	if err := q.ClearSubtaskAssignees(ctx, subtaskID); err != nil {
		return err
	}
	if picID == 0 {
		return nil
	}
	return q.AddSubtaskAssignee(ctx, sqlc.AddSubtaskAssigneeParams{
		SubtaskID: subtaskID, UserID: picID, AssignedBy: actor.ID,
	})
}

// AddSubtask menambah satu sub task di urutan paling bawah. picID 0 berarti belum ditugaskan.
func (s *Service) AddSubtask(ctx context.Context, actor Actor, taskID, title, desc, link string, picID int64) (int64, error) {
	clean, err := cleanSubtaskTitle(title, "ini")
	if err != nil {
		return 0, err
	}
	cleanDesc, err := cleanSubtaskDesc(desc)
	if err != nil {
		return 0, err
	}
	// Aturan tautannya sama dengan deliverable task: hanya http dan https (NFR-SEC-06).
	cleanLink, err := cleanURL(link)
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
		id, err = q.CreateSubtask(ctx, sqlc.CreateSubtaskParams{
			TaskID: taskID, Title: clean, Description: cleanDesc, DeliverableUrl: cleanLink.String,
			Position: slot.NextPosition, CreatedBy: userRef(actor.ID),
		})
		if err != nil {
			return err
		}
		if err := setSubtaskPIC(ctx, q, actor, taskID, id, picID); err != nil {
			return err
		}
		// Versi task ikut naik supaya form Manajer yang sudah terbuka tidak menyimpan daftar sub task
		// yang basi, karena daftar dari form menggantikan daftar lama secara utuh.
		if _, err := q.BumpTaskVersion(ctx, taskID); err != nil {
			return err
		}
		return audit.Change(ctx, q, actor.auditActor(), "subtask_added", "task", taskID, nil,
			map[string]any{"id": id, "title": clean, "pic_id": picID})
	})
	return id, err
}

// editableSubtask memastikan sub task ada di task ini dan boleh diubah pelaku: Manajer, yang membuatnya,
// atau PIC yang ditugaskan mengerjakannya.
func editableSubtask(ctx context.Context, q *sqlc.Queries, actor Actor, taskID string, subtaskID int64) (sqlc.GetSubtaskRow, error) {
	st, err := q.GetSubtask(ctx, sqlc.GetSubtaskParams{ID: subtaskID, TaskID: taskID})
	if errors.Is(err, pgx.ErrNoRows) {
		return st, ErrSubtaskNotFound
	}
	if err != nil {
		return st, err
	}
	if isManager(actor) || st.CreatedBy.Int64 == actor.ID {
		return st, nil
	}
	pic, err := q.IsSubtaskAssignee(ctx, sqlc.IsSubtaskAssigneeParams{SubtaskID: subtaskID, UserID: actor.ID})
	if err != nil {
		return st, err
	}
	if !pic {
		return st, ErrSubtaskLocked
	}
	return st, nil
}

// UpdateSubtask mengubah nama, deskripsi, dan PIC sub task tanpa menggeser urutannya.
// picID 0 melepas PIC-nya.
func (s *Service) UpdateSubtask(ctx context.Context, actor Actor, taskID string, subtaskID int64, title, desc, link string, picID int64) error {
	clean, err := cleanSubtaskTitle(title, "ini")
	if err != nil {
		return err
	}
	cleanDesc, err := cleanSubtaskDesc(desc)
	if err != nil {
		return err
	}
	cleanLink, err := cleanURL(link)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if _, err := lockForSubtask(ctx, q, actor, taskID); err != nil {
			return err
		}
		st, err := editableSubtask(ctx, q, actor, taskID, subtaskID)
		if err != nil {
			return err
		}
		oldPIC, err := q.ListSubtaskAssignees(ctx, []int64{subtaskID})
		if err != nil {
			return err
		}
		var oldPICID int64
		if len(oldPIC) > 0 {
			oldPICID = oldPIC[0].ID
		}
		if st.Title == clean && st.Description == cleanDesc && st.DeliverableUrl == cleanLink.String &&
			oldPICID == picID {
			return nil
		}
		if st.Title != clean || st.Description != cleanDesc || st.DeliverableUrl != cleanLink.String {
			if err := q.RenameSubtask(ctx, sqlc.RenameSubtaskParams{
				ID: subtaskID, TaskID: taskID, Title: clean, Description: cleanDesc,
				DeliverableUrl: cleanLink.String,
			}); err != nil {
				return err
			}
		}
		if oldPICID != picID {
			if err := setSubtaskPIC(ctx, q, actor, taskID, subtaskID, picID); err != nil {
				return err
			}
		}
		if _, err := q.BumpTaskVersion(ctx, taskID); err != nil {
			return err
		}
		// Satu kejadian audit untuk nama dan PIC sekalian, supaya riwayat tidak terbelah dua
		// untuk satu kali simpan.
		return audit.Change(ctx, q, actor.auditActor(), "subtask_renamed", "task", taskID,
			map[string]any{"id": st.ID, "title": st.Title, "pic_id": oldPICID},
			map[string]any{"id": st.ID, "title": clean, "pic_id": picID})
	})
}

// DeleteSubtask menghapus satu sub task. Posisi sisanya dibiarkan berlubang; urutan hanya dibaca terurut.
func (s *Service) DeleteSubtask(ctx context.Context, actor Actor, taskID string, subtaskID int64) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if _, err := lockForSubtask(ctx, q, actor, taskID); err != nil {
			return err
		}
		st, err := editableSubtask(ctx, q, actor, taskID, subtaskID)
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
