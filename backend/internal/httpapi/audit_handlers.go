package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
)

type auditRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type auditTarget struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type auditItem struct {
	ID         int64           `json:"id"`
	OccurredAt time.Time       `json:"occurred_at"`
	Action     string          `json:"action"`
	Actor      *auditRef       `json:"actor"`
	Target     *auditTarget    `json:"target"`
	OldValue   json.RawMessage `json:"old_value"`
	NewValue   json.RawMessage `json:"new_value"`
	IP         string          `json:"ip"`
}

// listAuditLogs menerima from dan to dalam format RFC 3339. Batas hari dihitung di browser,
// supaya zona waktu kantor tidak perlu ditebak di server.
func (s *Server) listAuditLogs(c *gin.Context) {
	f := audit.Filter{Category: c.Query("category")}

	var err error
	if f.From, err = optionalTime(c.Query("from")); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Format tanggal awal tidak dikenal.")
		return
	}
	if f.To, err = optionalTime(c.Query("to")); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Format tanggal akhir tidak dikenal.")
		return
	}
	if v := c.Query("user_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeError(c, http.StatusBadRequest, CodeBadRequest, "Filter pengguna tidak dikenal.")
			return
		}
		f.ActorID = &id
	}
	if v := c.Query("page"); v != "" {
		if f.Page, err = strconv.Atoi(v); err != nil || f.Page < 1 {
			writeError(c, http.StatusBadRequest, CodeBadRequest, "Nomor halaman tidak dikenal.")
			return
		}
	}

	rows, hasMore, err := audit.List(c.Request.Context(), sqlc.New(s.pool), f)
	if errors.Is(err, audit.ErrUnknownCategory) {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Jenis aksi tidak dikenal.")
		return
	}
	if err != nil {
		s.internalError(c, err)
		return
	}

	items := make([]auditItem, 0, len(rows))
	for _, r := range rows {
		item := auditItem{
			ID:         r.ID,
			OccurredAt: r.OccurredAt.Time,
			Action:     r.Action,
			OldValue:   rawJSON(r.OldValue),
			NewValue:   rawJSON(r.NewValue),
		}
		if r.IpAddress != nil {
			item.IP = r.IpAddress.String()
		}
		if r.ActorID.Valid {
			item.Actor = &auditRef{ID: strconv.FormatInt(r.ActorID.Int64, 10), Name: r.ActorName.String}
		}
		if r.Entity.Valid && r.EntityID.Valid {
			item.Target = &auditTarget{Type: r.Entity.String, ID: r.EntityID.String}
			switch {
			case r.TargetUserName.Valid:
				item.Target.Name = r.TargetUserName.String
			case r.TargetDivisionName.Valid:
				item.Target.Name = r.TargetDivisionName.String
			case r.TargetLabelName.Valid:
				item.Target.Name = r.TargetLabelName.String
			}
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"entries": items, "has_more": hasMore, "page": max(f.Page, 1)})
}

func optionalTime(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// rawJSON meneruskan isi jsonb apa adanya; kolom kosong dikirim sebagai null.
func rawJSON(b []byte) json.RawMessage {
	if len(b) == 0 {
		return nil
	}
	return json.RawMessage(b)
}
