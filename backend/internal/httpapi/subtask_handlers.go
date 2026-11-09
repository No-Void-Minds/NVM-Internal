package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/task"
)

type subtaskDoneRequest struct {
	// Pointer supaya body kosong ditolak, bukan dianggap "lepas centang".
	Done *bool `json:"done"`
}

func (s *Server) setSubtaskDone(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	subtaskID, err := strconv.ParseInt(c.Param("subtaskId"), 10, 64)
	if err != nil || subtaskID <= 0 {
		writeError(c, http.StatusNotFound, CodeNotFound, "Sub task tidak ditemukan.")
		return
	}
	var req subtaskDoneRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Done == nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	err = s.tasks.SetSubtaskDone(c.Request.Context(), taskActor(c), id, subtaskID, *req.Done)
	switch {
	case errors.Is(err, task.ErrSubtaskNotFound):
		writeError(c, http.StatusNotFound, CodeNotFound, "Sub task tidak ditemukan.")
	case errors.Is(err, task.ErrNotSubtaskPIC):
		writeError(c, http.StatusForbidden, CodeForbidden, "Hanya PIC sub task ini yang bisa mencentangnya.")
	case err != nil:
		s.writeTaskError(c, err)
	default:
		c.Status(http.StatusNoContent)
	}
}
