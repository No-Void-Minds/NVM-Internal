package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/account"
	"nvm-internal/internal/db/sqlc"
)

type userItem struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Username     string     `json:"username"`
	Role         string     `json:"role"`
	DivisionID   int64      `json:"division_id"`
	DivisionName string     `json:"division_name"`
	Status       string     `json:"status"`
	LockedUntil  *time.Time `json:"locked_until"`
	LastLoginAt  *time.Time `json:"last_login_at"`
}

// listUsers mengembalikan semua akun sekaligus. Dengan sekitar 10 pengguna, filter peran, divisi,
// dan status cukup dilakukan di frontend tanpa pagination.
func (s *Server) listUsers(c *gin.Context) {
	rows, err := s.accounts.List(c.Request.Context())
	if err != nil {
		s.internalError(c, err)
		return
	}
	now := s.now()
	items := make([]userItem, 0, len(rows))
	for _, r := range rows {
		item := userItem{
			ID:           r.ID,
			Name:         r.Name,
			Username:     r.Username,
			Role:         string(r.Role),
			DivisionID:   r.DivisionID,
			DivisionName: r.DivisionName,
			Status:       "aktif",
		}
		if r.LockedUntil.Valid && r.LockedUntil.Time.After(now) {
			item.Status = "terkunci"
			item.LockedUntil = &r.LockedUntil.Time
		}
		if !r.IsActive {
			item.Status = "nonaktif"
		}
		if r.LastLoginAt.Valid {
			item.LastLoginAt = &r.LastLoginAt.Time
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"users": items})
}

type createUserRequest struct {
	Name       string `json:"name"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	DivisionID int64  `json:"division_id"`
	Password   string `json:"password"`
}

func (s *Server) createUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	id, err := s.accounts.Create(c.Request.Context(), s.actor(c), account.CreateInput{
		Name:       req.Name,
		Username:   req.Username,
		Role:       sqlc.UserRole(req.Role),
		DivisionID: req.DivisionID,
		Password:   req.Password,
	})
	if err != nil {
		s.writeAccountError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

type updateUserRequest struct {
	Name       string `json:"name"`
	Role       string `json:"role"`
	DivisionID int64  `json:"division_id"`
}

func (s *Server) updateUser(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	err := s.accounts.Update(c.Request.Context(), s.actor(c), id, account.UpdateInput{
		Name:       req.Name,
		Role:       sqlc.UserRole(req.Role),
		DivisionID: req.DivisionID,
	})
	if err != nil {
		s.writeAccountError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) setUserActive(active bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathID(c)
		if !ok {
			return
		}
		if err := s.accounts.SetActive(c.Request.Context(), s.actor(c), id, active); err != nil {
			s.writeAccountError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

func (s *Server) resetUserPassword(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	if err := s.accounts.ResetPassword(c.Request.Context(), s.actor(c), id, req.Password); err != nil {
		s.writeAccountError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) unlockUser(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := s.accounts.Unlock(c.Request.Context(), s.actor(c), id); err != nil {
		s.writeAccountError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) actor(c *gin.Context) account.Actor {
	return account.Actor{ID: currentUser(c).ID, IP: clientAddr(c)}
}

// pathID membaca :id dari URL. ID yang bukan angka diperlakukan sama dengan ID yang tidak ada.
func pathID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(c, http.StatusNotFound, CodeNotFound, "Data tidak ditemukan.")
		return 0, false
	}
	return id, true
}

func (s *Server) writeAccountError(c *gin.Context, err error) {
	var ve *account.ValidationError
	switch {
	case errors.As(err, &ve):
		writeError(c, http.StatusBadRequest, CodeBadRequest, ve.Message)
	case errors.Is(err, account.ErrNotFound):
		writeError(c, http.StatusNotFound, CodeNotFound, "Akun tidak ditemukan.")
	case errors.Is(err, account.ErrUsernameTaken):
		writeError(c, http.StatusConflict, CodeConflict, "Username sudah dipakai akun lain.")
	case errors.Is(err, account.ErrSelfDeactivate):
		writeError(c, http.StatusConflict, CodeConflict, "Kamu tidak bisa menonaktifkan akunmu sendiri.")
	case errors.Is(err, account.ErrLastManager):
		writeError(c, http.StatusConflict, CodeConflict, "Harus ada minimal satu Admin atau Manajer yang aktif.")
	case errors.Is(err, account.ErrDivisionNotFound):
		writeError(c, http.StatusNotFound, CodeNotFound, "Divisi tidak ditemukan.")
	case errors.Is(err, account.ErrDivisionNameTaken):
		writeError(c, http.StatusConflict, CodeConflict, "Nama divisi sudah dipakai.")
	default:
		s.internalError(c, err)
	}
}
