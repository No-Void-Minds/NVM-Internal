package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/activity"
	"nvm-internal/internal/apperr"
	"nvm-internal/internal/db/sqlc"
)

func activityActor(c *gin.Context) activity.Actor {
	u := currentUser(c)
	return activity.Actor{ID: u.ID, Role: u.Role, IP: clientAddr(c)}
}

// myActivities adalah tabel di papan task: isian sendiri untuk hari ini. Hanya Staf yang punya tabel
// aktivitas, jadi peran lain ditolak di sini, bukan hanya disembunyikan di frontend (keputusan revisi 21).
func (s *Server) myActivities(c *gin.Context) {
	u := currentUser(c)
	if u.Role != sqlc.UserRoleStaff {
		writeError(c, http.StatusForbidden, CodeForbidden, "Tabel aktivitas harian hanya untuk akun Staf.")
		return
	}
	out, err := s.activities.Mine(c.Request.Context(), u.ID, s.activities.Today())
	if err != nil {
		s.writeActivityError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// listActivities bisa dibaca semua pengguna yang login; yang membatasi hanya hak mengubah. Rentangnya bisa
// diminta tiga cara: week (Senin sampai Jumat minggu tanggal itu), from dan to, atau date untuk satu hari.
// Satu panggilan sudah memuat seluruh minggu, jadi halaman mingguan tidak memanggil API per hari.
func (s *Server) listActivities(c *gin.Context) {
	from, to, err := s.activityRange(c)
	if err != nil {
		s.writeActivityError(c, err)
		return
	}
	out, err := s.activities.Range(c.Request.Context(), from, to)
	if err != nil {
		s.writeActivityError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"from":   out.From,
		"to":     out.To,
		"dates":  out.Dates,
		"today":  s.activities.Today(),
		"slots":  out.Slots,
		"people": out.People,
		// Manajer boleh mengubah isian siapa pun di tanggal mana pun; peran lain hanya membaca di halaman ini.
		"can_edit": currentUser(c).Role == sqlc.UserRoleManager,
	})
}

func (s *Server) activityRange(c *gin.Context) (string, string, error) {
	if week := c.Query("week"); week != "" {
		day, err := s.activities.ParseDay(week)
		if err != nil {
			return "", "", err
		}
		return activity.WeekRange(day)
	}
	if from := c.Query("from"); from != "" {
		start, err := s.activities.ParseDay(from)
		if err != nil {
			return "", "", err
		}
		end, err := s.activities.ParseDay(c.Query("to"))
		if err != nil {
			return "", "", err
		}
		return start, end, nil
	}
	// Tanpa parameter: satu hari, hari ini.
	day, err := s.activities.ParseDay(c.Query("date"))
	return day, day, err
}

type saveActivityRequest struct {
	UserID   int64  `json:"user_id"`
	Date     string `json:"date"`
	Slot     int    `json:"slot"`
	Activity string `json:"activity"`
	Output   string `json:"output"`
}

func (s *Server) saveActivity(c *gin.Context) {
	var req saveActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	actor := activityActor(c)
	// user_id kosong berarti isian sendiri, supaya frontend papan task tidak perlu mengirim id akun.
	if req.UserID == 0 {
		req.UserID = actor.ID
	}
	err := s.activities.Save(c.Request.Context(), actor, activity.SaveInput{
		UserID: req.UserID, Date: req.Date, Slot: req.Slot, Activity: req.Activity, Output: req.Output,
	})
	if err != nil {
		s.writeActivityError(c, err)
		return
	}
	// Dibaca ulang supaya frontend langsung tahu teks yang tersimpan dan siapa yang terakhir mengubahnya.
	date, err := s.activities.ParseDay(req.Date)
	if err != nil {
		s.writeActivityError(c, err)
		return
	}
	out, err := s.activities.Mine(c.Request.Context(), req.UserID, date)
	if err != nil {
		s.writeActivityError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// exportActivities mengirim .xlsx untuk rentang tanggal. Hanya Manajer (dibatasi di router).
func (s *Server) exportActivities(c *gin.Context) {
	from, err := s.activities.ParseDay(c.Query("from"))
	if err != nil {
		s.writeActivityError(c, err)
		return
	}
	to, err := s.activities.ParseDay(c.Query("to"))
	if err != nil {
		s.writeActivityError(c, err)
		return
	}
	data, name, err := s.activities.Export(c.Request.Context(), from, to)
	if err != nil {
		s.writeActivityError(c, err)
		return
	}
	h := c.Writer.Header()
	h.Set("Content-Disposition", `attachment; filename="`+name+`"`)
	h.Set("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func (s *Server) writeActivityError(c *gin.Context, err error) {
	var ve *apperr.Validation
	switch {
	case errors.As(err, &ve):
		writeError(c, http.StatusBadRequest, CodeBadRequest, ve.Message)
	case errors.Is(err, activity.ErrForbidden):
		writeError(c, http.StatusForbidden, CodeForbidden,
			"Kamu hanya bisa mengisi aktivitasmu sendiri untuk hari ini.")
	default:
		s.internalError(c, err)
	}
}
