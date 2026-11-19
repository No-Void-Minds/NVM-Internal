package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/todo"
)

// To Do pribadi (revisi 28). Semua endpoint memakai akun yang login, tidak pernah id akun dari URL, jadi
// tidak ada jalan membaca atau mengubah To Do orang lain.

func (s *Server) listTodos(c *gin.Context) {
	items, err := s.todos.List(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		s.writeTodoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"todos": items, "max": todo.MaxItems})
}

type todoTitleRequest struct {
	Title string `json:"title"`
}

func (s *Server) addTodo(c *gin.Context) {
	var req todoTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	item, err := s.todos.Add(c.Request.Context(), currentUser(c).ID, req.Title)
	if err != nil {
		s.writeTodoError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"todo": item})
}

func (s *Server) renameTodo(c *gin.Context) {
	id, ok := todoID(c)
	if !ok {
		return
	}
	var req todoTitleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	if err := s.todos.Rename(c.Request.Context(), currentUser(c).ID, id, req.Title); err != nil {
		s.writeTodoError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type todoDoneRequest struct {
	Done bool `json:"done"`
}

func (s *Server) setTodoDone(c *gin.Context) {
	id, ok := todoID(c)
	if !ok {
		return
	}
	var req todoDoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	if err := s.todos.SetDone(c.Request.Context(), currentUser(c).ID, id, req.Done); err != nil {
		s.writeTodoError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) deleteTodo(c *gin.Context) {
	id, ok := todoID(c)
	if !ok {
		return
	}
	if err := s.todos.Delete(c.Request.Context(), currentUser(c).ID, id); err != nil {
		s.writeTodoError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type todoReorderRequest struct {
	IDs []int64 `json:"ids"`
}

func (s *Server) reorderTodos(c *gin.Context) {
	var req todoReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	if err := s.todos.Reorder(c.Request.Context(), currentUser(c).ID, req.IDs); err != nil {
		s.writeTodoError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) clearDoneTodos(c *gin.Context) {
	n, err := s.todos.ClearDone(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		s.writeTodoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": n})
}

// todoID membaca :todoId. Bukan angka diperlakukan sama dengan item yang tidak ada.
func todoID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("todoId"), 10, 64)
	if err != nil || id <= 0 {
		writeError(c, http.StatusNotFound, CodeNotFound, "To do tidak ditemukan.")
		return 0, false
	}
	return id, true
}

func (s *Server) writeTodoError(c *gin.Context, err error) {
	var ve *apperr.Validation
	switch {
	case errors.As(err, &ve):
		writeError(c, http.StatusBadRequest, CodeBadRequest, ve.Message)
	// Item milik orang lain juga sampai ke sini: 404, bukan 403, supaya keberadaannya tidak bocor.
	case errors.Is(err, todo.ErrNotFound):
		writeError(c, http.StatusNotFound, CodeNotFound, "To do tidak ditemukan.")
	default:
		s.internalError(c, err)
	}
}
