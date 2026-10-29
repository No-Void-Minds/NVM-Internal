package task

import (
	"context"
	"errors"
	"slices"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
)

// ErrLabelOutOfScope berarti Staf mencoba memasang label milik divisi lain (FR-TASK-21).
var ErrLabelOutOfScope = errors.New("label bukan untuk divisi pelaku")

type UpdateInput struct {
	Version        int32
	Title          string
	Description    string
	Urgency        string
	PICIDs         []int64
	StartDate      string
	DueDate        string
	DeliverableURL string
}

// lockTask mengunci baris task sampai transaksi selesai dan memastikan versi yang dibaca klien masih
// versi terbaru (optimistic locking, S3-06). Versi yang beda berarti orang lain sudah mengubahnya duluan.
func lockTask(ctx context.Context, q *sqlc.Queries, id string, version int32) (sqlc.GetTaskForUpdateRow, error) {
	t, err := q.GetTaskForUpdate(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	if err != nil {
		return t, err
	}
	if t.Version != version {
		return t, ErrVersionConflict
	}
	return t, nil
}

// canContribute: Manajer atau Staf yang menjadi PIC. Dipakai untuk label dan deliverable (BR-02).
func canContribute(ctx context.Context, q *sqlc.Queries, actor Actor, id string) (bool, error) {
	switch actor.Role {
	case RoleManager:
		return true, nil
	case RoleStaff:
		return q.IsTaskAssignee(ctx, sqlc.IsTaskAssigneeParams{TaskID: id, UserID: actor.ID})
	default:
		return false, nil
	}
}

// Update mengubah atribut task dan daftar PIC. Hanya Manajer; divisi ikut PIC pertama yang baru.
func (s *Service) Update(ctx context.Context, actor Actor, id string, in UpdateInput) (int32, error) {
	if actor.Role != RoleManager {
		return 0, ErrForbidden
	}
	f, err := cleanFields(in.Title, in.Description, in.Urgency, in.StartDate, in.DueDate, in.DeliverableURL)
	if err != nil {
		return 0, err
	}
	pics := uniqueIDs(in.PICIDs)
	if len(pics) == 0 {
		return 0, apperr.Invalid("Pilih minimal satu PIC.")
	}

	version := in.Version
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		cur, err := lockTask(ctx, q, id, in.Version)
		if err != nil {
			return err
		}
		divisionOf, err := eligiblePICs(ctx, q, pics)
		if err != nil {
			return err
		}
		oldPICs, err := q.ListAssigneeIDs(ctx, id)
		if err != nil {
			return err
		}

		before, after := map[string]any{}, map[string]any{}
		diff := func(key string, old, next any) {
			if old != next {
				before[key], after[key] = old, next
			}
		}
		diff("title", cur.Title, f.title)
		diff("description", cur.Description.String, f.description.String)
		diff("urgency", string(cur.Urgency), string(f.urgency))
		diff("start_date", dateOf(cur.StartDate), dateOf(f.startDate))
		diff("due_date", dateOf(cur.DueDate), dateOf(f.dueDate))
		diff("deliverable_url", cur.DeliverableUrl.String, f.deliverableURL.String)
		diff("division_id", cur.DivisionID, divisionOf[pics[0]])
		picsChanged := !slices.Equal(oldPICs, pics)
		if picsChanged {
			before["assignee_ids"], after["assignee_ids"] = oldPICs, pics
		}
		if len(after) == 0 {
			return nil
		}

		if picsChanged {
			if err := q.DeleteTaskAssignees(ctx, id); err != nil {
				return err
			}
			for _, pic := range pics {
				if err := q.AddTaskAssignee(ctx, sqlc.AddTaskAssigneeParams{TaskID: id, UserID: pic, AssignedBy: actor.ID}); err != nil {
					return err
				}
			}
		}
		version, err = q.UpdateTaskFields(ctx, sqlc.UpdateTaskFieldsParams{
			ID:             id,
			Title:          f.title,
			Description:    f.description,
			Urgency:        f.urgency,
			DivisionID:     divisionOf[pics[0]],
			StartDate:      f.startDate,
			DueDate:        f.dueDate,
			DeliverableUrl: f.deliverableURL,
		})
		if err != nil {
			return err
		}
		return audit.Change(ctx, q, actor.auditActor(), "task_updated", "task", id, before, after)
	})
	return version, err
}

// SetLabels mengganti label task. Manajer bebas memilih label aktif. Staf yang menjadi PIC hanya boleh
// mengatur label divisinya atau label semua divisi; label divisi lain yang sudah terpasang dibiarkan.
// Label yang sudah nonaktif boleh tetap menempel, tetapi tidak bisa dipasang baru (AC-14).
func (s *Service) SetLabels(ctx context.Context, actor Actor, id string, version int32, labelIDs []int64) (int32, error) {
	desired := uniqueIDs(labelIDs)
	next := version
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if _, err := lockTask(ctx, q, id, version); err != nil {
			return err
		}
		ok, err := canContribute(ctx, q, actor, id)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}

		current, err := q.ListTaskLabelScopes(ctx, id)
		if err != nil {
			return err
		}
		attached := make(map[int64]bool, len(current))
		for _, l := range current {
			attached[l.LabelID] = true
		}

		found, err := q.GetLabelsByIDs(ctx, desired)
		if err != nil {
			return err
		}
		info := make(map[int64]sqlc.GetLabelsByIDsRow, len(found))
		for _, l := range found {
			info[l.ID] = l
		}
		inScope := func(division pgtype.Int8) bool {
			return actor.Role == RoleManager || !division.Valid || division.Int64 == actor.DivisionID
		}
		final := make([]int64, 0, len(desired)+len(current))
		for _, lid := range desired {
			l, exists := info[lid]
			if !exists {
				return apperr.Invalid("Label tidak ditemukan.")
			}
			if !inScope(l.DivisionID) {
				// Label divisi lain yang memang sudah menempel boleh ikut terkirim; ia dipertahankan di bawah.
				if attached[lid] {
					continue
				}
				return ErrLabelOutOfScope
			}
			if !l.IsActive && !attached[lid] {
				return apperr.Invalid("Label yang sudah nonaktif tidak bisa dipasang.")
			}
			final = append(final, lid)
		}
		for _, l := range current {
			if !inScope(l.DivisionID) {
				final = append(final, l.LabelID)
			}
		}
		old := make([]int64, 0, len(current))
		for _, l := range current {
			old = append(old, l.LabelID)
		}
		slices.Sort(final)
		slices.Sort(old)
		if slices.Equal(final, old) {
			return nil
		}

		if err := q.DeleteTaskLabels(ctx, id); err != nil {
			return err
		}
		for _, lid := range final {
			if err := q.AddTaskLabel(ctx, sqlc.AddTaskLabelParams{TaskID: id, LabelID: lid}); err != nil {
				return err
			}
		}
		if next, err = q.BumpTaskVersion(ctx, id); err != nil {
			return err
		}
		return audit.Change(ctx, q, actor.auditActor(), "task_labels_changed", "task", id,
			map[string]any{"label_ids": old}, map[string]any{"label_ids": final})
	})
	return next, err
}

// SetDeliverable mengisi tautan hasil kerja. Boleh oleh Manajer atau Staf yang menjadi PIC (BR-02).
func (s *Service) SetDeliverable(ctx context.Context, actor Actor, id string, version int32, link string) (int32, error) {
	u, err := cleanURL(link)
	if err != nil {
		return 0, err
	}
	next := version
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		cur, err := lockTask(ctx, q, id, version)
		if err != nil {
			return err
		}
		ok, err := canContribute(ctx, q, actor, id)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if cur.DeliverableUrl.String == u.String {
			return nil
		}
		if next, err = q.SetTaskDeliverable(ctx, sqlc.SetTaskDeliverableParams{ID: id, DeliverableUrl: u}); err != nil {
			return err
		}
		return audit.Change(ctx, q, actor.auditActor(), "task_deliverable_changed", "task", id,
			map[string]any{"deliverable_url": cur.DeliverableUrl.String}, map[string]any{"deliverable_url": u.String})
	})
	return next, err
}

// Delete menandai task terhapus (FR-TASK-16). Datanya tetap ada untuk audit, tetapi hilang dari semua daftar.
func (s *Service) Delete(ctx context.Context, actor Actor, id string, version int32) error {
	if actor.Role != RoleManager {
		return ErrForbidden
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		cur, err := lockTask(ctx, q, id, version)
		if err != nil {
			return err
		}
		if err := q.SoftDeleteTask(ctx, id); err != nil {
			return err
		}
		return audit.Change(ctx, q, actor.auditActor(), "task_deleted", "task", id, map[string]any{"title": cur.Title}, nil)
	})
}

func dateOf(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format("2006-01-02")
}
