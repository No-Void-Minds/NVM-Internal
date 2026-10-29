package task

import (
	"context"
	"errors"

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
func (s *Service) Move(ctx context.Context, actor Actor, id string, version int32, to Status, note string) (MoveResult, error) {
	var res MoveResult
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		cur, err := lockTask(ctx, q, id, version)
		if err != nil {
			return err
		}
		isPIC := false
		if actor.Role == RoleStaff {
			if isPIC, err = q.IsTaskAssignee(ctx, sqlc.IsTaskAssigneeParams{TaskID: id, UserID: actor.ID}); err != nil {
				return err
			}
		}

		from := Status(cur.Status)
		cleanNote, err := CheckTransition(actor.Role, isPIC, from, to, note)
		switch {
		case errors.Is(err, ErrInvalidTransition):
			return ErrInvalidMove
		case errors.Is(err, ErrTransitionForbidden):
			return ErrForbidden
		case errors.Is(err, ErrNoteRequired):
			return apperr.Invalid("Isi catatan revisi saat mengembalikan task ke On Progress.")
		case errors.Is(err, ErrNoteTooLong):
			return apperr.Invalid("Catatan revisi maksimal 500 karakter.")
		case err != nil:
			return err
		}

		row, err := q.SetTaskStatus(ctx, sqlc.SetTaskStatusParams{ID: id, Status: sqlc.TaskStatus(to)})
		if err != nil {
			return err
		}
		res = MoveResult{Version: row.Version, Status: to, Solved: row}

		after := map[string]any{"status": to}
		if cleanNote != "" {
			after["note"] = cleanNote
		}
		return audit.Change(ctx, q, actor.auditActor(), "task_status_changed", "task", id,
			map[string]any{"status": from}, after)
	})
	return res, err
}
