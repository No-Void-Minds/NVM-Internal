package task

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
)

// ErrInvalidMove berarti perpindahan itu tidak ada di alur status (mis. Requested langsung ke Completed).
var ErrInvalidMove = errors.New("perpindahan status tidak dikenal")

type MoveResult struct {
	Version int32
	Status  Status
	Solved  sqlc.SetTaskStatusRow
}

// Move memindahkan status task menurut aturan di transition.go (S3-04). solved_date diatur oleh query,
// dan setiap perpindahan tercatat sebagai task_status_changed untuk riwayat task (FR-TASK-11).
// subtaskIDs hanya untuk jalur revisi PIC sub task (revisi 22): sub task miliknya yang dibuka lagi.
func (s *Service) Move(ctx context.Context, actor Actor, id string, version int32, to Status, note string, subtaskIDs []int64) (MoveResult, error) {
	var res MoveResult
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		cur, err := lockTask(ctx, q, id, version)
		if err != nil {
			return err
		}
		isPIC := false
		mine := []sqlc.ListMySubtasksRow{}
		if actor.Role == RoleStaff {
			if isPIC, err = q.IsTaskAssignee(ctx, sqlc.IsTaskAssigneeParams{TaskID: id, UserID: actor.ID}); err != nil {
				return err
			}
			// Revisi 17: dihitung dari subtask_assignees saat itu juga, bukan dari data yang dikirim frontend.
			if mine, err = q.ListMySubtasks(ctx, sqlc.ListMySubtasksParams{TaskIds: []string{id}, UserID: actor.ID}); err != nil {
				return err
			}
		}

		from := Status(cur.Status)
		subtaskRevision := IsSubtaskRevision(actor.Role, isPIC, len(mine) > 0, from, to)
		cleanNote, err := CheckTransition(actor.Role, isPIC, len(mine) > 0, from, to, note)
		switch {
		case errors.Is(err, ErrInvalidTransition):
			return ErrInvalidMove
		case errors.Is(err, ErrTransitionForbidden):
			return ErrForbidden
		case errors.Is(err, ErrNoteRequired):
			return apperr.Invalid("Isi catatan revisi saat mengembalikan task ke On Progress.")
		case errors.Is(err, ErrNoteTooLong) && subtaskRevision:
			return apperr.Invalid("Alasan revisi maksimal 500 karakter.")
		case errors.Is(err, ErrNoteTooLong):
			return apperr.Invalid("Catatan revisi maksimal 500 karakter.")
		case err != nil:
			return err
		}
		if !subtaskRevision && len(subtaskIDs) > 0 {
			return apperr.Invalid("Pilihan sub task hanya dipakai PIC sub task yang mengembalikan task ke On Progress.")
		}
		var reopened []SubtaskRef
		if subtaskRevision {
			if reopened, err = pickRevisionSubtasks(mine, subtaskIDs); err != nil {
				return err
			}
			ids := make([]int64, len(reopened))
			for i, r := range reopened {
				ids[i] = r.ID
			}
			if _, err := q.ReopenSubtasks(ctx, sqlc.ReopenSubtasksParams{TaskID: id, Ids: ids}); err != nil {
				return err
			}
		}
		// Semua sub task harus tercentang sebelum Ready. Dicek di dalam transaksi yang sudah mengunci task,
		// jadi hitungannya tidak bisa berubah di antara pengecekan dan perpindahan.
		if from == OnProgress && to == Ready {
			open, err := q.CountOpenSubtasks(ctx, id)
			if err != nil {
				return err
			}
			if open > 0 {
				return apperr.Invalid(fmt.Sprintf("Masih %d sub task yang belum dicentang. Centang semuanya dulu sebelum menandai Ready.", open))
			}
		}

		row, err := q.SetTaskStatus(ctx, sqlc.SetTaskStatusParams{ID: id, Status: sqlc.TaskStatus(to)})
		if err != nil {
			return err
		}
		res = MoveResult{Version: row.Version, Status: to, Solved: row}

		after := map[string]any{"status": to}
		switch {
		case subtaskRevision:
			// Kunci reason, bukan note: riwayat menulisnya di dalam kalimat revisi sub task, bukan sebagai
			// baris "Catatan" terpisah.
			after["reopened_subtasks"] = reopened
			if cleanNote != "" {
				after["reason"] = cleanNote
			}
		case cleanNote != "":
			after["note"] = cleanNote
		}
		return audit.Change(ctx, q, actor.auditActor(), "task_status_changed", "task", id,
			map[string]any{"status": from}, after)
	})
	return res, err
}

// pickRevisionSubtasks memeriksa pilihan PIC sub task (revisi 22): minimal satu, dan setiap id harus sub task
// miliknya di task ini yang sedang tercentang. Urutannya mengikuti posisi sub task, bukan urutan kiriman.
func pickRevisionSubtasks(mine []sqlc.ListMySubtasksRow, ids []int64) ([]SubtaskRef, error) {
	if len(ids) == 0 {
		return nil, apperr.Invalid("Pilih minimal satu sub task yang direvisi.")
	}
	want := make(map[int64]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	out := []SubtaskRef{}
	for _, m := range mine {
		if want[m.ID] && m.Done {
			out = append(out, SubtaskRef{ID: m.ID, Title: m.Title})
			delete(want, m.ID)
		}
	}
	for id := range want {
		return nil, apperr.Invalid(fmt.Sprintf("Sub task %d tidak bisa dibuka lagi: hanya sub task milikmu yang sedang tercentang yang bisa dipilih.", id))
	}
	return out, nil
}
