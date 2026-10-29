package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/task"
)

type updateTaskRequest struct {
	Version        int32   `json:"version"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	Urgency        string  `json:"urgency"`
	PICIDs         []int64 `json:"pic_ids"`
	StartDate      string  `json:"start_date"`
	DueDate        string  `json:"due_date"`
	DeliverableURL string  `json:"deliverable_url"`
}

func (s *Server) updateTask(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	var req updateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	v, err := s.tasks.Update(c.Request.Context(), taskActor(c), id, task.UpdateInput{
		Version:        req.Version,
		Title:          req.Title,
		Description:    req.Description,
		Urgency:        req.Urgency,
		PICIDs:         req.PICIDs,
		StartDate:      req.StartDate,
		DueDate:        req.DueDate,
		DeliverableURL: req.DeliverableURL,
	})
	if err != nil {
		s.writeTaskError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"version": v})
}

type taskLabelsRequest struct {
	Version  int32   `json:"version"`
	LabelIDs []int64 `json:"label_ids"`
}

func (s *Server) setTaskLabels(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	var req taskLabelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	v, err := s.tasks.SetLabels(c.Request.Context(), taskActor(c), id, req.Version, req.LabelIDs)
	if errors.Is(err, task.ErrLabelOutOfScope) {
		writeError(c, http.StatusForbidden, CodeForbidden,
			"Kamu hanya bisa memasang label divisimu sendiri atau label untuk semua divisi.")
		return
	}
	if err != nil {
		s.writeTaskError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"version": v})
}

type deliverableRequest struct {
	Version        int32  `json:"version"`
	DeliverableURL string `json:"deliverable_url"`
}

func (s *Server) setTaskDeliverable(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	var req deliverableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	v, err := s.tasks.SetDeliverable(c.Request.Context(), taskActor(c), id, req.Version, req.DeliverableURL)
	if err != nil {
		s.writeTaskError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"version": v})
}

// deleteTask membaca versi dari query string (?version=3), karena sebagian klien membuang body pada DELETE.
func (s *Server) deleteTask(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	v, err := strconv.ParseInt(c.Query("version"), 10, 32)
	if err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Versi task wajib dikirim.")
		return
	}
	if err := s.tasks.Delete(c.Request.Context(), taskActor(c), id, int32(v)); err != nil {
		s.writeTaskError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
