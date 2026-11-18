package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/task"
)

type moveTaskRequest struct {
	Version int32  `json:"version"`
	To      string `json:"to"`
	Note    string `json:"note"`
	// SubtaskIDs: sub task yang dibuka lagi saat PIC sub task mengembalikan task ke On Progress (revisi 22).
	SubtaskIDs []int64 `json:"subtask_ids"`
}

func (s *Server) moveTask(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	var req moveTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	res, err := s.tasks.Move(c.Request.Context(), taskActor(c), id, req.Version, task.Status(req.To), req.Note, req.SubtaskIDs)
	if errors.Is(err, task.ErrInvalidMove) {
		writeError(c, http.StatusConflict, CodeConflict, "Perpindahan status ini tidak ada di alur task.")
		return
	}
	if err != nil {
		s.writeTaskError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"version":     res.Version,
		"status":      res.Status,
		"solved_date": timePtr(res.Solved.SolvedDate),
	})
}
