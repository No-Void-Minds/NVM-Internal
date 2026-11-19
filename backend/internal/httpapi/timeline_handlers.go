package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/timeline"
)

// Timeline (revisi 30). Semua yang login membaca; hak menulis (hanya Manajer) diperiksa di timeline.Service,
// jadi rutenya cukup di grup app dan jawaban 403 datang dari satu tempat.

func timelineActor(c *gin.Context) timeline.Actor {
	u := currentUser(c)
	return timeline.Actor{ID: u.ID, Role: u.Role, IP: clientAddr(c)}
}

func (s *Server) listTimelineEvents(c *gin.Context) {
	actor := timelineActor(c)
	events, err := s.timeline.Events(c.Request.Context(), actor, c.Query("month"))
	if err != nil {
		s.writeTimelineError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events, "can_write": timeline.CanWrite(actor)})
}

func (s *Server) createTimelineEvent(c *gin.Context) {
	var in timeline.EventInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	id, err := s.timeline.CreateEvent(c.Request.Context(), timelineActor(c), in)
	if err != nil {
		s.writeTimelineError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (s *Server) updateTimelineEvent(c *gin.Context) {
	id, ok := timelineID(c)
	if !ok {
		return
	}
	var in timeline.EventInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	if err := s.timeline.UpdateEvent(c.Request.Context(), timelineActor(c), id, in); err != nil {
		s.writeTimelineError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) deleteTimelineEvent(c *gin.Context) {
	id, ok := timelineID(c)
	if !ok {
		return
	}
	if err := s.timeline.DeleteEvent(c.Request.Context(), timelineActor(c), id); err != nil {
		s.writeTimelineError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) listTimelineNotes(c *gin.Context) {
	notes, err := s.timeline.Notes(c.Request.Context())
	if err != nil {
		s.writeTimelineError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"notes": notes, "can_write": timeline.CanWrite(timelineActor(c))})
}

func (s *Server) createTimelineNote(c *gin.Context) {
	var in timeline.NoteInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	n, err := s.timeline.CreateNote(c.Request.Context(), timelineActor(c), in)
	if err != nil {
		s.writeTimelineError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"note": n})
}

func (s *Server) updateTimelineNote(c *gin.Context) {
	id, ok := timelineID(c)
	if !ok {
		return
	}
	var in timeline.NoteInput
	if err := c.ShouldBindJSON(&in); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	if err := s.timeline.UpdateNote(c.Request.Context(), timelineActor(c), id, in); err != nil {
		s.writeTimelineError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) deleteTimelineNote(c *gin.Context) {
	id, ok := timelineID(c)
	if !ok {
		return
	}
	if err := s.timeline.DeleteNote(c.Request.Context(), timelineActor(c), id); err != nil {
		s.writeTimelineError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) reorderTimelineNotes(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	if req.IDs == nil {
		req.IDs = []int64{}
	}
	if err := s.timeline.ReorderNotes(c.Request.Context(), timelineActor(c), req.IDs); err != nil {
		s.writeTimelineError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func timelineID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(c, http.StatusNotFound, CodeNotFound, "Isi Timeline tidak ditemukan.")
		return 0, false
	}
	return id, true
}

func (s *Server) writeTimelineError(c *gin.Context, err error) {
	var ve *apperr.Validation
	switch {
	case errors.As(err, &ve):
		writeError(c, http.StatusBadRequest, CodeBadRequest, ve.Message)
	case errors.Is(err, timeline.ErrForbidden):
		writeError(c, http.StatusForbidden, CodeForbidden, "Hanya Manajer yang bisa mengubah Timeline.")
	case errors.Is(err, timeline.ErrNotFound):
		writeError(c, http.StatusNotFound, CodeNotFound, "Isi Timeline tidak ditemukan. Mungkin sudah dihapus.")
	default:
		s.internalError(c, err)
	}
}
