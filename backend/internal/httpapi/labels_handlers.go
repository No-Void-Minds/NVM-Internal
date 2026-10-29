package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/label"
)

type labelItem struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Color        string `json:"color"`
	DivisionID   *int64 `json:"division_id"`
	DivisionName string `json:"division_name,omitempty"`
	IsActive     bool   `json:"is_active"`
	TaskCount    int32  `json:"task_count"`
}

// listLabels terbuka untuk semua akun yang sudah masuk: Staf butuh daftar ini untuk memasang label
// dan untuk filter papan. Penyaringan label yang boleh dipasang Staf tetap dicek ulang saat memasang.
func (s *Server) listLabels(c *gin.Context) {
	rows, err := s.labels.List(c.Request.Context())
	if err != nil {
		s.internalError(c, err)
		return
	}
	items := make([]labelItem, 0, len(rows))
	for _, r := range rows {
		item := labelItem{ID: r.ID, Name: r.Name, Color: r.Color, IsActive: r.IsActive, TaskCount: r.TaskCount}
		if r.DivisionID.Valid {
			id := r.DivisionID.Int64
			item.DivisionID = &id
			item.DivisionName = r.DivisionName.String
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"labels": items, "palette": label.Palette})
}

type labelRequest struct {
	Name       string `json:"name"`
	Color      string `json:"color"`
	DivisionID *int64 `json:"division_id"`
}

func (s *Server) createLabel(c *gin.Context) {
	var req labelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	id, err := s.labels.Create(c.Request.Context(), s.actor(c), label.Input{Name: req.Name, Color: req.Color, DivisionID: req.DivisionID})
	if err != nil {
		s.writeLabelError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (s *Server) updateLabel(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req labelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	if err := s.labels.Update(c.Request.Context(), s.actor(c), id, label.Input{Name: req.Name, Color: req.Color, DivisionID: req.DivisionID}); err != nil {
		s.writeLabelError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) setLabelActive(active bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathID(c)
		if !ok {
			return
		}
		if err := s.labels.SetActive(c.Request.Context(), s.actor(c), id, active); err != nil {
			s.writeLabelError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func (s *Server) writeLabelError(c *gin.Context, err error) {
	var ve *apperr.Validation
	switch {
	case errors.As(err, &ve):
		writeError(c, http.StatusBadRequest, CodeBadRequest, ve.Message)
	case errors.Is(err, label.ErrNotFound):
		writeError(c, http.StatusNotFound, CodeNotFound, "Label tidak ditemukan.")
	case errors.Is(err, label.ErrNameTaken):
		writeError(c, http.StatusConflict, CodeConflict, "Nama label sudah dipakai.")
	default:
		s.internalError(c, err)
	}
}
