package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"nvm-internal/internal/db/sqlc"
)

// driveStatus adalah hasil salinan Drive terakhir untuk halaman Status (revisi 25). Dibaca dari tabel
// drive_export_runs yang ditulis job, bukan dari berkas log, supaya backend tidak perlu membaca disk host.
type driveStatus struct {
	// Never true berarti job-nya belum pernah jalan; frontend menampilkan kalimat yang berbeda.
	Never      bool       `json:"never"`
	FinishedAt *time.Time `json:"finished_at"`
	OK         bool       `json:"ok"`
	Files      int32      `json:"files"`
	Bytes      int64      `json:"bytes"`
	Message    string     `json:"message"`
}

// driveExportStatus hanya untuk Admin (dibatasi di router). Peran lain dapat 403, yang belum masuk 401.
func (s *Server) driveExportStatus(c *gin.Context) {
	row, err := sqlc.New(s.pool).GetLastDriveExportRun(c.Request.Context())
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusOK, driveStatus{Never: true})
		return
	}
	if err != nil {
		s.internalError(c, err)
		return
	}
	out := driveStatus{OK: row.Ok, Files: row.Files, Bytes: row.Bytes}
	if row.FinishedAt.Valid {
		out.FinishedAt = &row.FinishedAt.Time
	}
	if row.Message.Valid {
		out.Message = row.Message.String
	}
	c.JSON(http.StatusOK, out)
}
