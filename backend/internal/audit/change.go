package audit

import (
	"context"
	"net/netip"

	"nvm-internal/internal/db/sqlc"
)

// Actor adalah pelaku aksi pengelolaan, dicatat di kolom user_id.
type Actor struct {
	ID int64
	IP *netip.Addr
}

// Change mencatat aksi pelaku terhadap satu entitas (akun, divisi, label, task).
// before dan after cukup berisi kolom yang berubah; nil berarti tidak ada nilai untuk dicatat.
func Change(ctx context.Context, q *sqlc.Queries, actor Actor, action, entity, entityID string, before, after map[string]any) error {
	e := Entry{ActorID: &actor.ID, Action: action, Entity: entity, EntityID: entityID, IP: actor.IP}
	if before != nil {
		e.Old = before
	}
	if after != nil {
		e.New = after
	}
	return Record(ctx, q, e)
}
