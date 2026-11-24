package httpapi

import (
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/activity"
	"nvm-internal/internal/apperr"
	"nvm-internal/internal/attachment"
	"nvm-internal/internal/db/sqlc"
)

func activityActor(c *gin.Context) activity.Actor {
	u := currentUser(c)
	return activity.Actor{ID: u.ID, Role: u.Role, IP: clientAddr(c)}
}

// myActivities adalah tabel di papan task: isian sendiri untuk hari ini, atau ?date= untuk tanggal lain yang
// dibuka lewat tombol Sebelumnya (revisi 37; hak mengubahnya tetap diperiksa CanEdit). Hanya Staf yang punya
// tabel aktivitas, jadi peran lain ditolak di sini, bukan hanya disembunyikan di frontend (keputusan revisi 21).
func (s *Server) myActivities(c *gin.Context) {
	u := currentUser(c)
	if u.Role != sqlc.UserRoleStaff {
		writeError(c, http.StatusForbidden, CodeForbidden, "Tabel aktivitas harian hanya untuk akun Staf.")
		return
	}
	date, err := s.activities.ParseDay(c.Query("date"))
	if err != nil {
		s.writeActivityError(c, err)
		return
	}
	out, err := s.activities.Mine(c.Request.Context(), u.ID, date)
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

type linkActivityRequest struct {
	UserID int64  `json:"user_id"`
	Date   string `json:"date"`
	Slot   int    `json:"slot"`
	Linked bool   `json:"linked"`
}

// linkActivity memasang atau melepas rantai sambung satu jam ke jam sebelumnya (revisi 33). Jawabannya isian
// sehari penuh, karena melepas satu jam bisa ikut melepas jam setelahnya.
func (s *Server) linkActivity(c *gin.Context) {
	var req linkActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	actor := activityActor(c)
	if req.UserID == 0 {
		req.UserID = actor.ID
	}
	err := s.activities.Link(c.Request.Context(), actor, activity.LinkInput{
		UserID: req.UserID, Date: req.Date, Slot: req.Slot, Linked: req.Linked,
	})
	if err != nil {
		s.writeActivityError(c, err)
		return
	}
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
	var rejected *attachment.Rejected
	var tooBig *http.MaxBytesError
	var conflict *activity.Conflict
	switch {
	case errors.As(err, &conflict):
		writeError(c, http.StatusConflict, CodeConflict, conflict.Message)
	case errors.As(err, &rejected):
		writeError(c, http.StatusUnsupportedMediaType, CodeUnsupportedFile, rejected.Message)
	case errors.Is(err, activity.ErrTooLarge), errors.As(err, &tooBig):
		writeError(c, http.StatusRequestEntityTooLarge, CodeFileTooLarge, "Berkas melebihi batas 95 MB.")
	case errors.Is(err, activity.ErrFileMissing):
		writeError(c, http.StatusNotFound, CodeNotFound, "Lampiran tidak ditemukan.")
	case errors.As(err, &ve):
		writeError(c, http.StatusBadRequest, CodeBadRequest, ve.Message)
	case errors.Is(err, activity.ErrForbidden):
		writeError(c, http.StatusForbidden, CodeForbidden,
			"Kamu hanya bisa mengisi aktivitasmu sendiri untuk hari ini dan 2 hari kerja sebelumnya.")
	default:
		s.internalError(c, err)
	}
}

// uploadActivityFile menerima multipart/form-data dengan satu bagian "file". Pemilik jam, tanggal, dan slot
// dikirim lewat query string supaya izin bisa dicek sebelum isi berkas dibaca.
func (s *Server) uploadActivityFile(c *gin.Context) {
	actor := activityActor(c)
	userID, ok := activityOwner(c, actor)
	if !ok {
		return
	}
	slot, err := strconv.Atoi(c.Query("slot"))
	if err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Slot jam tidak dikenal.")
		return
	}
	date := c.Query("date")
	if _, err := s.activities.CheckUpload(c.Request.Context(), actor, userID, date, slot); err != nil {
		s.writeActivityError(c, err)
		return
	}

	_ = http.NewResponseController(c.Writer).SetReadDeadline(time.Now().Add(uploadTimeout))
	// Sedikit di atas 95 MB untuk ruang header multipart. Batas isi berkas yang tepat dicek di service.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, activity.MaxFileSize+(1<<20))
	mr, err := c.Request.MultipartReader()
	if err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Unggahan harus berupa multipart/form-data.")
		return
	}
	part, err := filePart(mr)
	if err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Tidak ada berkas di unggahan ini.")
		return
	}
	defer func() { _ = part.Close() }()

	saved, err := s.activities.Upload(c.Request.Context(), actor, userID, date, slot, part.FileName(), part)
	if err != nil {
		s.writeActivityError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"file": saved})
}

// downloadActivityFile menyajikan berkas. Gambar, video, dan PDF boleh tampil di dalam halaman supaya
// pratinjaunya bisa langsung dibuka; sisanya selalu diunduh.
func (s *Server) downloadActivityFile(c *gin.Context) {
	row, f, err := s.activities.Open(c.Request.Context(), c.Param("fileId"))
	if err != nil {
		s.writeActivityError(c, err)
		return
	}
	defer func() { _ = f.Close() }()

	kind := "attachment"
	if inlineActivityMIME(row.MimeType) {
		kind = "inline"
	}
	disposition := mime.FormatMediaType(kind, map[string]string{"filename": row.OriginalName})
	if disposition == "" {
		disposition = kind
	}
	h := c.Writer.Header()
	h.Set("Content-Type", row.MimeType)
	h.Set("Content-Disposition", disposition)
	h.Set("X-Content-Type-Options", "nosniff")
	// Lapis tambahan: kalau berkas terbuka di tab browser, skrip di dalamnya tetap tidak bisa jalan.
	h.Set("Content-Security-Policy", "sandbox")
	h.Set("Cache-Control", "private, no-store")
	// ServeContent mendukung Range (206), yang dibutuhkan pemutar video untuk melompat ke tengah berkas.
	http.ServeContent(c.Writer, c.Request, row.OriginalName, row.CreatedAt.Time, f)
}

// inlineActivityMIME: hanya jenis yang memang ditampilkan browser sebagai gambar, video, atau PDF.
// ZIP dan dokumen Office tidak pernah inline, supaya tidak ada berkas yang terbuka sendiri di tab.
func inlineActivityMIME(m string) bool {
	return strings.HasPrefix(m, "image/") || strings.HasPrefix(m, "video/") || m == "application/pdf"
}

func (s *Server) deleteActivityFile(c *gin.Context) {
	if err := s.activities.DeleteFile(c.Request.Context(), activityActor(c), c.Param("fileId")); err != nil {
		s.writeActivityError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// activityOwner membaca user_id dari query. Kosong atau 0 berarti jam milik sendiri, supaya tabel di papan
// task tidak perlu mengirim id akunnya.
func activityOwner(c *gin.Context, actor activity.Actor) (int64, bool) {
	raw := c.Query("user_id")
	if raw == "" || raw == "0" {
		return actor.ID, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Pemilik isian tidak valid.")
		return 0, false
	}
	return id, true
}
