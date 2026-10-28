package audit

import (
	"context"
	"encoding/json"
	"net/netip"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/db/sqlc"
)

// Entry adalah satu baris audit log. ActorID adalah orang yang melakukan aksi (kolom user_id);
// nil berarti sistem atau login dengan username yang tidak terdaftar.
// Entity dan EntityID menunjuk yang dikenai aksi, mis. "user" dan id akun yang dinonaktifkan.
type Entry struct {
	ActorID  *int64
	Action   string
	Entity   string
	EntityID string
	Old      any
	New      any
	IP       *netip.Addr
}

func Record(ctx context.Context, q *sqlc.Queries, e Entry) error {
	p := sqlc.InsertAuditLogParams{Action: e.Action, IpAddress: e.IP}
	if e.ActorID != nil {
		p.UserID = pgtype.Int8{Int64: *e.ActorID, Valid: true}
	}
	if e.Entity != "" {
		p.Entity = pgtype.Text{String: e.Entity, Valid: true}
	}
	if e.EntityID != "" {
		p.EntityID = pgtype.Text{String: e.EntityID, Valid: true}
	}

	var err error
	if p.OldValue, err = toJSON(e.Old); err != nil {
		return err
	}
	if p.NewValue, err = toJSON(e.New); err != nil {
		return err
	}
	return q.InsertAuditLog(ctx, p)
}

// UserEvent dipakai untuk kejadian yang pelaku dan sasarannya akun yang sama: login, kuncian, logout, ganti password.
func UserEvent(ctx context.Context, q *sqlc.Queries, userID *int64, action string, ip *netip.Addr, detail map[string]any) error {
	e := Entry{ActorID: userID, Action: action, Entity: "user", IP: ip}
	if userID != nil {
		e.EntityID = strconv.FormatInt(*userID, 10)
	}
	// Map nil yang dibungkus interface tidak sama dengan nil; tanpa cek ini kolomnya terisi JSON null.
	if detail != nil {
		e.New = detail
	}
	return Record(ctx, q, e)
}

func toJSON(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}
