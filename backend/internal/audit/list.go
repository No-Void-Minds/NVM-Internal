package audit

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/db/sqlc"
)

// Categories dipakai filter "Jenis aksi" di halaman Audit Log. Kategori task ditambahkan di Sprint 3.
var Categories = map[string][]string{
	"auth": {
		"login_success", "login_failed", "account_locked", "logout",
		"password_changed", "password_change_failed",
	},
	"account": {
		"user_created", "user_updated", "user_activated", "user_deactivated",
		"password_reset", "user_unlocked",
	},
	"division": {
		"division_created", "division_renamed", "division_activated", "division_deactivated",
	},
	"label": {
		"label_created", "label_updated", "label_activated", "label_deactivated",
	},
	"task": {
		"task_created", "task_updated", "task_deleted",
		"task_labels_changed", "task_deliverable_changed", "task_status_changed",
	},
}

const PageSize = 50

var ErrUnknownCategory = errors.New("kategori audit tidak dikenal")

type Filter struct {
	From     *time.Time
	To       *time.Time
	ActorID  *int64
	Category string
	Page     int // dimulai dari 1
}

// List mengambil satu halaman audit log, terbaru di atas. Satu baris lebih diminta dari ukuran
// halaman untuk mengetahui apakah masih ada halaman berikutnya tanpa query COUNT terpisah.
func List(ctx context.Context, q *sqlc.Queries, f Filter) ([]sqlc.ListAuditLogsRow, bool, error) {
	actions := []string{}
	if f.Category != "" {
		a, ok := Categories[f.Category]
		if !ok {
			return nil, false, ErrUnknownCategory
		}
		actions = a
	}
	page := max(f.Page, 1)

	p := sqlc.ListAuditLogsParams{
		Actions:   actions,
		RowLimit:  PageSize + 1,
		RowOffset: int32((page - 1) * PageSize),
	}
	if f.From != nil {
		p.FromTime = pgtype.Timestamptz{Time: *f.From, Valid: true}
	}
	if f.To != nil {
		p.ToTime = pgtype.Timestamptz{Time: *f.To, Valid: true}
	}
	if f.ActorID != nil {
		p.ActorID = pgtype.Int8{Int64: *f.ActorID, Valid: true}
	}

	rows, err := q.ListAuditLogs(ctx, p)
	if err != nil {
		return nil, false, err
	}
	if len(rows) > PageSize {
		return rows[:PageSize], true, nil
	}
	return rows, false, nil
}
