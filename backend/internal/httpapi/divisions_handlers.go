package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type divisionItem struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	IsActive    bool   `json:"is_active"`
	ActiveUsers int32  `json:"active_users"`
}

// listDivisions terbuka untuk semua akun yang sudah masuk, karena daftar divisi dipakai
// juga sebagai filter di Papan Task dan Arsip (Sprint 4 dan 6).
func (s *Server) listDivisions(c *gin.Context) {
	rows, err := s.accounts.ListDivisions(c.Request.Context())
	if err != nil {
		s.internalError(c, err)
		return
	}
	items := make([]divisionItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, divisionItem{ID: r.ID, Name: r.Name, IsActive: r.IsActive, ActiveUsers: r.ActiveUsers})
	}
	c.JSON(http.StatusOK, gin.H{"divisions": items})
}

type divisionRequest struct {
	Name string `json:"name"`
}

func (s *Server) createDivision(c *gin.Context) {
	var req divisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	id, err := s.accounts.CreateDivision(c.Request.Context(), s.actor(c), req.Name)
	if err != nil {
		s.writeAccountError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (s *Server) renameDivision(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req divisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	if err := s.accounts.RenameDivision(c.Request.Context(), s.actor(c), id, req.Name); err != nil {
		s.writeAccountError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) setDivisionActive(active bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathID(c)
		if !ok {
			return
		}
		if err := s.accounts.SetDivisionActive(c.Request.Context(), s.actor(c), id, active); err != nil {
			s.writeAccountError(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}
