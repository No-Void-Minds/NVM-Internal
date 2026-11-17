package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/task"
)

type subtaskTitleRequest struct {
	Title string `json:"title"`
}

// subtaskPath membaca :id dan :subtaskId sekaligus; keduanya salah bentuk diperlakukan sebagai tidak ada.
func subtaskPath(c *gin.Context) (string, int64, bool) {
	id, ok := taskID(c)
	if !ok {
		return "", 0, false
	}
	sid, err := strconv.ParseInt(c.Param("subtaskId"), 10, 64)
	if err != nil || sid <= 0 {
		writeError(c, http.StatusNotFound, CodeNotFound, "Sub task tidak ditemukan.")
		return "", 0, false
	}
	return id, sid, true
}

// writeSubtaskError memakai pesan yang sesuai konteks menyusun sub task, bukan pesan centang.
func (s *Server) writeSubtaskError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, task.ErrSubtaskNotFound):
		writeError(c, http.StatusNotFound, CodeNotFound, "Sub task tidak ditemukan.")
	case errors.Is(err, task.ErrSubtaskLocked):
		writeError(c, http.StatusForbidden, CodeForbidden, "Sub task ini dibuat orang lain. Hanya pembuatnya atau Manajer yang bisa mengubahnya.")
	case errors.Is(err, task.ErrForbidden):
		writeError(c, http.StatusForbidden, CodeForbidden, "Hanya PIC task ini atau Manajer yang bisa menyusun sub task.")
	default:
		s.writeTaskError(c, err)
	}
}

func (s *Server) addSubtask(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	var req subtaskTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	sid, err := s.tasks.AddSubtask(c.Request.Context(), taskActor(c), id, req.Title)
	if err != nil {
		s.writeSubtaskError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": sid})
}

func (s *Server) renameSubtask(c *gin.Context) {
	id, sid, ok := subtaskPath(c)
	if !ok {
		return
	}
	var req subtaskTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	if err := s.tasks.RenameSubtask(c.Request.Context(), taskActor(c), id, sid, req.Title); err != nil {
		s.writeSubtaskError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) deleteSubtask(c *gin.Context) {
	id, sid, ok := subtaskPath(c)
	if !ok {
		return
	}
	if err := s.tasks.DeleteSubtask(c.Request.Context(), taskActor(c), id, sid); err != nil {
		s.writeSubtaskError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

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
