package audit

import (
	"context"
	"encoding/json"
	"net/netip"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/db/sqlc"
)

// UserEvent mencatat kejadian yang menyangkut akun: login, kuncian, logout, ganti password.
// userID nil dipakai untuk login dengan username yang tidak terdaftar.
func UserEvent(ctx context.Context, q *sqlc.Queries, userID *int64, action string, ip *netip.Addr, detail map[string]any) error {
	p := sqlc.InsertAuditLogParams{
		Action:    action,
		Entity:    pgtype.Text{String: "user", Valid: true},
		IpAddress: ip,
	}
	if userID != nil {
		p.UserID = pgtype.Int8{Int64: *userID, Valid: true}
		p.EntityID = pgtype.Text{String: strconv.FormatInt(*userID, 10), Valid: true}
	}
	if detail != nil {
		b, err := json.Marshal(detail)
		if err != nil {
			return err
		}
		p.NewValue = b
	}
	return q.InsertAuditLog(ctx, p)
}
