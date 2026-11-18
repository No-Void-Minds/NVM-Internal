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
// Manajer, Staf yang menjadi PIC task, atau sejak revisi 11 orang luar yang ditugaskan di sub task itu
// sendiri (subtaskID 0 berarti menambah sub task baru, dan itu tetap hanya untuk PIC task dan Manajer).
func lockForSubtask(ctx context.Context, q *sqlc.Queries, actor Actor, taskID string, subtaskID int64) (sqlc.GetTaskForUpdateRow, error) {
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
	if !ok && subtaskID != 0 {
		ok, err = q.IsSubtaskAssignee(ctx, sqlc.IsSubtaskAssigneeParams{SubtaskID: subtaskID, UserID: actor.ID})
		if err != nil {
			return cur, err
		}
	}
	if !ok {
		return cur, ErrForbidden
	}
	if !subtaskWindowOpen(Status(cur.Status)) {
		return cur, apperr.Invalid("Sub task hanya bisa diubah saat task berstatus Requested atau On Progress.")
	}
	return cur, nil
}

// setSubtaskPIC memasang atau melepas PIC sub task. Sejak revisi 11 kandidatnya tidak lagi dibatasi PIC
// task, karena pekerjaan sering dititipkan ke orang divisi lain. Revisi 13 membatalkan revisi 12, jadi
// aturannya kembali sama dengan PIC task (GetEligiblePICs): akun aktif berperan Staf atau Manajer.
// Dipanggil hanya saat PIC berubah, jadi PIC lama yang akunnya sudah nonaktif tetap bertahan.
func setSubtaskPIC(ctx context.Context, q *sqlc.Queries, actor Actor, taskID string, subtaskID int64, picID int64) error {
	if picID != 0 {
		rows, err := q.GetEligiblePICs(ctx, []int64{picID})
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return apperr.Invalid("Sub task hanya bisa ditugaskan ke akun Staf atau Manajer yang aktif.")
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
// SubtaskFields adalah isi sub task yang bisa ditulis pengguna. Dikumpulkan dalam satu struct supaya
// menambah kolom baru tidak memanjangkan daftar parameter di setiap lapisan.
type SubtaskFields struct {
	Title       string
	Description string
	// Reference diisi yang menugaskan (acuan kerja), Deliverable diisi yang mengerjakan (hasilnya).
	Reference   string
	Deliverable string
	PICID       int64
	// DueDate berbentuk YYYY-MM-DD, kosong berarti tanpa tenggat (revisi 16).
	DueDate string
}

type cleanSubtaskFields struct {
	title       string
	description string
	reference   string
	deliverable string
	due         pgtype.Date
}

func cleanFieldsOf(in SubtaskFields) (cleanSubtaskFields, error) {
	var out cleanSubtaskFields
	var err error
	if out.title, err = cleanSubtaskTitle(in.Title, "ini"); err != nil {
		return out, err
	}
	if out.description, err = cleanSubtaskDesc(in.Description); err != nil {
		return out, err
	}
	// Aturan tautannya sama dengan deliverable task: hanya http dan https (NFR-SEC-06).
	ref, err := cleanURL(in.Reference)
	if err != nil {
		return out, err
	}
	del, err := cleanURL(in.Deliverable)
	if err != nil {
		return out, err
	}
	out.reference, out.deliverable = ref.String, del.String
	if out.due, err = parseDate(in.DueDate, "Tenggat sub task"); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Service) AddSubtask(ctx context.Context, actor Actor, taskID string, in SubtaskFields) (int64, error) {
	f, err := cleanFieldsOf(in)
	if err != nil {
		return 0, err
	}
	var id int64
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		cur, err := lockForSubtask(ctx, q, actor, taskID, 0)
		if err != nil {
			return err
		}
		if err := checkSubtaskDue(f.due, cur.DueDate); err != nil {
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
			TaskID: taskID, Title: f.title, Description: f.description,
			ReferenceUrl: f.reference, DeliverableUrl: f.deliverable,
			Position: slot.NextPosition, CreatedBy: userRef(actor.ID), DueDate: f.due,
		})
		if err != nil {
			return err
		}
		if err := setSubtaskPIC(ctx, q, actor, taskID, id, in.PICID); err != nil {
			return err
		}
		// Versi task ikut naik supaya form Manajer yang sudah terbuka tidak menyimpan daftar sub task
		// yang basi, karena daftar dari form menggantikan daftar lama secara utuh.
		if _, err := q.BumpTaskVersion(ctx, taskID); err != nil {
			return err
		}
		return audit.Change(ctx, q, actor.auditActor(), "subtask_added", "task", taskID, nil,
			map[string]any{"id": id, "title": f.title, "pic_id": in.PICID, "due_date": dateValue(f.due)})
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
func (s *Service) UpdateSubtask(ctx context.Context, actor Actor, taskID string, subtaskID int64, in SubtaskFields) error {
	f, err := cleanFieldsOf(in)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		cur, err := lockForSubtask(ctx, q, actor, taskID, subtaskID)
		if err != nil {
			return err
		}
		st, err := editableSubtask(ctx, q, actor, taskID, subtaskID)
		if err != nil {
			return err
		}
		// Hanya dicek bila tenggatnya diubah: sub task yang terlanjur melewati tenggat task (karena tenggat task
		// dimajukan) tetap bisa diganti namanya tanpa dipaksa memperbaiki tenggatnya dulu.
		sameDue := dateValue(st.DueDate) == dateValue(f.due)
		if !sameDue {
			if err := checkSubtaskDue(f.due, cur.DueDate); err != nil {
				return err
			}
		}
		oldPIC, err := q.ListSubtaskAssignees(ctx, []int64{subtaskID})
		if err != nil {
			return err
		}
		var oldPICID int64
		if len(oldPIC) > 0 {
			oldPICID = oldPIC[0].ID
		}
		sameText := st.Title == f.title && st.Description == f.description &&
			st.ReferenceUrl == f.reference && st.DeliverableUrl == f.deliverable && sameDue
		if sameText && oldPICID == in.PICID {
			return nil
		}
		if !sameText {
			if err := q.RenameSubtask(ctx, sqlc.RenameSubtaskParams{
				ID: subtaskID, TaskID: taskID, Title: f.title, Description: f.description,
				ReferenceUrl: f.reference, DeliverableUrl: f.deliverable, DueDate: f.due,
			}); err != nil {
				return err
			}
		}
		if oldPICID != in.PICID {
			if err := setSubtaskPIC(ctx, q, actor, taskID, subtaskID, in.PICID); err != nil {
				return err
			}
		}
		if _, err := q.BumpTaskVersion(ctx, taskID); err != nil {
			return err
		}
		// Satu kejadian audit untuk nama dan PIC sekalian, supaya riwayat tidak terbelah dua
		// untuk satu kali simpan.
		return audit.Change(ctx, q, actor.auditActor(), "subtask_renamed", "task", taskID,
			map[string]any{"id": st.ID, "title": st.Title, "pic_id": oldPICID, "due_date": dateValue(st.DueDate)},
			map[string]any{"id": st.ID, "title": f.title, "pic_id": in.PICID, "due_date": dateValue(f.due)})
	})
}

// DeleteSubtask menghapus satu sub task. Posisi sisanya dibiarkan berlubang; urutan hanya dibaca terurut.
func (s *Service) DeleteSubtask(ctx context.Context, actor Actor, taskID string, subtaskID int64) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if _, err := lockForSubtask(ctx, q, actor, taskID, subtaskID); err != nil {
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
