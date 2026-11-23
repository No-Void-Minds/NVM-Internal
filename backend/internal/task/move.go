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
// subtaskIDs: sub task yang dibuka lagi saat Ready ke On Progress. Wajib di jalur revisi PIC sub task (revisi 22,
// hanya sub task miliknya), opsional di jalur revisi PIC task dan Manajer (revisi 34, sub task mana pun).
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
			return moveForbidden(to)
		case errors.Is(err, ErrNoteRequired):
			return apperr.Invalid("Isi catatan revisi saat mengembalikan task ke On Progress.")
		case errors.Is(err, ErrNoteTooLong) && subtaskRevision:
			return apperr.Invalid("Alasan revisi maksimal 500 karakter.")
		case errors.Is(err, ErrNoteTooLong):
			return apperr.Invalid("Catatan revisi maksimal 500 karakter.")
		case err != nil:
			return err
		}
		// Jalur revisi biasa (PIC task dan Manajer; CheckTransition sudah memastikan haknya). Revisi 34 memberi
		// mereka pilihan membuka lagi sub task tercentang mana pun, karena sekarang mereka juga boleh mencentangnya.
		taskRevision := !subtaskRevision && from == Ready && to == OnProgress
		if !subtaskRevision && !taskRevision && len(subtaskIDs) > 0 {
			return apperr.Invalid("Pilihan sub task hanya dipakai saat mengembalikan task dari Ready ke On Progress.")
		}
		var reopened []SubtaskRef
		switch {
		case subtaskRevision:
			if len(subtaskIDs) == 0 {
				return apperr.Invalid("Pilih minimal satu sub task yang direvisi.")
			}
			cands := []SubtaskRef{}
			for _, m := range mine {
				if m.Done {
					cands = append(cands, SubtaskRef{ID: m.ID, Title: m.Title})
				}
			}
			if reopened, err = pickRevisionSubtasks(cands, subtaskIDs, "hanya sub task milikmu yang sedang tercentang yang bisa dipilih"); err != nil {
				return err
			}
		case taskRevision && len(subtaskIDs) > 0:
			subs, err := loadSubtaskViews(ctx, q, id)
			if err != nil {
				return err
			}
			if reopened, err = pickRevisionSubtasks(doneRefs(subs), subtaskIDs, "hanya sub task yang sedang tercentang yang bisa dipilih"); err != nil {
				return err
			}
		}
		if len(reopened) > 0 {
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
			if len(reopened) > 0 {
				after["reopened_subtasks"] = reopened
			}
		}
		return audit.Change(ctx, q, actor.auditActor(), "task_status_changed", "task", id,
			map[string]any{"status": from}, after)
	})
	return res, err
}

// pickRevisionSubtasks memeriksa pilihan sub task yang dibuka lagi: setiap id harus ada di cands (sub task
// tercentang yang boleh dipilih pelaku). Urutannya mengikuti posisi sub task, bukan urutan kiriman. Pesan galat
// menyebut id, bukan judul, supaya judul sub task task lain tidak bocor (revisi 22).
func pickRevisionSubtasks(cands []SubtaskRef, ids []int64, why string) ([]SubtaskRef, error) {
	want := make(map[int64]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	out := []SubtaskRef{}
	for _, c := range cands {
		if want[c.ID] {
			out = append(out, c)
			delete(want, c.ID)
		}
	}
	for id := range want {
		return nil, apperr.Invalid(fmt.Sprintf("Sub task %d tidak bisa dibuka lagi: %s.", id, why))
	}
	return out, nil
}

func doneRefs(subs []SubtaskView) []SubtaskRef {
	out := []SubtaskRef{}
	for _, s := range subs {
		if s.DoneAt.Valid {
			out = append(out, SubtaskRef{ID: s.ID, Title: s.Title})
		}
	}
	return out
}

// TaskRevisionChoices: sub task tercentang yang boleh dibuka lagi PIC task atau Manajer saat mengembalikan Ready
// ke On Progress (revisi 34). Kosong untuk yang lain; jalur PIC sub task memakai pilihan dari MoveOptions.
func TaskRevisionChoices(role Role, isPIC bool, status Status, subs []SubtaskView) []SubtaskRef {
	if status != Ready || !(role == RoleManager || (role == RoleStaff && isPIC)) {
		return []SubtaskRef{}
	}
	return doneRefs(subs)
}

// RevisionNoteRequired: catatan revisi wajib di semua jalur Ready ke On Progress kecuali jalur PIC sub task.
func RevisionNoteRequired(role Role, isPIC, isSubtaskPIC bool) bool {
	return !IsSubtaskRevision(role, isPIC, isSubtaskPIC, Ready, OnProgress)
}
