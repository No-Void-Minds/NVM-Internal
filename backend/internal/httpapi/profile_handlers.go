package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/account"
	"nvm-internal/internal/task"
)

type profileRequest struct {
	Name     string `json:"name"`
	JobTitle string `json:"job_title"`
}

// updateMyProfile tidak menerima id: sasarannya selalu akun yang login (S10-04).
func (s *Server) updateMyProfile(c *gin.Context) {
	var req profileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	err := s.accounts.UpdateOwnProfile(c.Request.Context(), s.actor(c), account.ProfileInput{
		Name: req.Name, JobTitle: req.JobTitle,
	})
	if err != nil {
		s.writeAccountError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// avatarBodyLimit memberi ruang untuk header multipart di atas batas 2 MB isi foto.
const avatarBodyLimit = account.MaxAvatarSize + 64<<10

func (s *Server) uploadMyAvatar(c *gin.Context) {
	// Content-Length yang sudah jelas terlalu besar ditolak sebelum isi dibaca, supaya browser cepat mendapat jawaban.
	if c.Request.ContentLength > avatarBodyLimit {
		s.writeAvatarError(c, account.ErrAvatarTooLarge)
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, avatarBodyLimit)
	mr, err := c.Request.MultipartReader()
	if err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Unggahan harus berupa multipart/form-data.")
		return
	}
	part, err := filePart(mr)
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		s.writeAvatarError(c, err)
		return
	}
	if err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Tidak ada berkas foto di unggahan ini.")
		return
	}
	defer func() { _ = part.Close() }()

	jpeg, err := account.ProcessAvatar(part)
	if err != nil {
		s.writeAvatarError(c, err)
		return
	}
	id, err := s.avatars.Set(c.Request.Context(), s.actor(c), jpeg)
	if err != nil {
		s.writeAvatarError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"avatar_id": id})
}

func (s *Server) deleteMyAvatar(c *gin.Context) {
	if err := s.avatars.Delete(c.Request.Context(), s.actor(c)); err != nil {
		s.writeAvatarError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// userAvatar mengirim foto akun mana pun ke pengguna yang sudah login. ETag berisi avatar_id, yang berganti di
// setiap unggahan. Cache selamanya hanya bila URL membawa ?v= yang cocok, supaya URL tanpa versi tidak
// menyimpan foto lama di browser.
func (s *Server) userAvatar(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	avatar, f, err := s.avatars.Open(c.Request.Context(), id)
	if err != nil {
		s.writeAvatarError(c, err)
		return
	}
	defer func() { _ = f.Close() }()

	h := c.Writer.Header()
	h.Set("Content-Type", "image/jpeg")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("ETag", `"`+avatar+`"`)
	if c.Query("v") == avatar {
		h.Set("Cache-Control", "private, max-age=31536000, immutable")
	} else {
		h.Set("Cache-Control", "private, no-cache")
	}
	// ServeContent membalas 304 sendiri bila If-None-Match cocok dengan ETag di atas.
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}

// myTaskCounts memakai aturan papan scope=mine tanpa filter lain, jadi angkanya tidak ikut berubah saat filter
// papan diganti. Completed hanya menghitung task CompletedDays hari terakhir, sama seperti kolom papan.
func (s *Server) myTaskCounts(c *gin.Context) {
	cols, err := s.tasks.Board(c.Request.Context(), task.BoardFilter{PICIDs: []int64{currentUser(c).ID}}, time.Now())
	if err != nil {
		s.writeTaskError(c, err)
		return
	}
	counts := make(map[string]int, len(cols))
	for _, col := range cols {
		counts[string(col.Status)] = len(col.Tasks)
	}
	c.JSON(http.StatusOK, gin.H{"counts": counts, "completed_days": task.CompletedDays})
}

func (s *Server) writeAvatarError(c *gin.Context, err error) {
	var rejected *account.AvatarRejected
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &rejected):
		writeError(c, http.StatusUnsupportedMediaType, CodeUnsupportedFile, rejected.Message)
	case errors.Is(err, account.ErrAvatarTooLarge), errors.As(err, &tooBig):
		writeError(c, http.StatusRequestEntityTooLarge, CodeFileTooLarge, "Foto melebihi batas 2 MB. Pilih foto yang lebih kecil.")
	case errors.Is(err, account.ErrNoAvatar):
		writeError(c, http.StatusNotFound, CodeNotFound, "Akun ini belum punya foto profil.")
	default:
		s.writeAccountError(c, err)
	}
}

// avatarID mengirim null untuk akun tanpa foto, supaya frontend cukup memeriksa satu nilai.
func avatarID(id pgtype.UUID) *string {
	if !id.Valid {
		return nil
	}
	v := id.String()
	return &v
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}
