package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/db/sqlc"
	"nvm-internal/internal/whiteboard"
)

// Whiteboard (docs/whiteboard.md). Semua yang login membaca. Rute tulis dipasang di belakang wbWriter: Staf dan
// Manajer menulis semua isi, Admin hanya membaca. can_write di respons berasal dari aturan yang sama, jadi frontend
// tidak menghitungnya sendiri.

func wbCanWrite(c *gin.Context) bool {
	r := currentUser(c).Role
	return r == sqlc.UserRoleStaff || r == sqlc.UserRoleManager
}

func wbWriter() gin.HandlerFunc {
	return func(c *gin.Context) {
		if wbCanWrite(c) {
			c.Next()
			return
		}
		writeError(c, http.StatusForbidden, CodeForbidden, "Admin hanya bisa melihat Whiteboard, tidak bisa mengubahnya.")
	}
}

func wbActor(c *gin.Context) whiteboard.Actor {
	return whiteboard.Actor{ID: currentUser(c).ID, IP: clientAddr(c), Manager: currentUser(c).Role == sqlc.UserRoleManager}
}

func bindJSON(c *gin.Context, v any) bool {
	if err := c.ShouldBindJSON(v); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return false
	}
	return true
}

func (s *Server) wbSearch(c *gin.Context) {
	out, err := s.whiteboard.Search(c.Request.Context(), c.Query("q"))
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": out})
}

func (s *Server) wbRenameBoard(c *gin.Context) {
	var in struct {
		Title   string `json:"title"`
		Version int32  `json:"version"`
	}
	if !bindJSON(c, &in) {
		return
	}
	b, err := s.whiteboard.RenameBoard(c.Request.Context(), wbActor(c), c.Param("id"), in.Title, in.Version)
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	s.hub.Renamed(b.ID, b.Title, b.Version)
	// Kartu papan di kanvas induk menampilkan judul ini.
	if b.ParentID != "" {
		s.hub.Resync(b.ParentID)
	}
	c.JSON(http.StatusOK, gin.H{"board": b})
}

func (s *Server) wbBoard(c *gin.Context) {
	b, err := s.whiteboard.Board(c.Request.Context(), wbActor(c), c.Param("id"))
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"board": b, "can_write": wbCanWrite(c)})
}

func (s *Server) wbActivity(c *gin.Context) {
	out, err := s.whiteboard.Activity(c.Request.Context(), wbActor(c), c.Param("id"))
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"activity": out})
}

func (s *Server) wbCreateCard(c *gin.Context) {
	var in whiteboard.CardInput
	if !bindJSON(c, &in) {
		return
	}
	card, err := s.whiteboard.CreateCard(c.Request.Context(), wbActor(c), c.Param("id"), in)
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	s.hub.Changed(card.BoardID, []whiteboard.Card{card}, currentUser(c).ID)
	c.JSON(http.StatusCreated, gin.H{"card": card})
}

func (s *Server) wbPatchCards(c *gin.Context) {
	var req struct {
		Cards []whiteboard.CardPatch `json:"cards"`
	}
	if !bindJSON(c, &req) {
		return
	}
	// Isi tabel dan gambar bebas hanya diubah pemegang kunci lunaknya; catatan dijaga content_version (W2).
	board := ""
	for _, p := range req.Cards {
		if len(p.Content) == 0 || string(p.Content) == "null" {
			continue
		}
		if board == "" {
			b, err := s.whiteboard.ResolveBoard(c.Request.Context(), wbActor(c), c.Param("id"))
			if err != nil {
				s.writeWbError(c, err)
				return
			}
			board = b
		}
		kind, err := s.whiteboard.CardKind(c.Request.Context(), board, p.ID)
		if err == nil && (kind == whiteboard.KindTable || kind == whiteboard.KindDrawing) && s.wbLocked(c, p.ID) {
			return
		}
	}
	cards, err := s.whiteboard.PatchCards(c.Request.Context(), wbActor(c), c.Param("id"), req.Cards)
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	if len(cards) > 0 {
		s.hub.Changed(cards[0].BoardID, cards, currentUser(c).ID)
	}
	c.JSON(http.StatusOK, gin.H{"cards": cards})
}

func (s *Server) wbChecklist(c *gin.Context) {
	var op whiteboard.ChecklistOp
	if !bindJSON(c, &op) {
		return
	}
	card, err := s.whiteboard.Checklist(c.Request.Context(), wbActor(c), c.Param("id"), c.Param("cardId"), op)
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	s.hub.Changed(card.BoardID, []whiteboard.Card{card}, currentUser(c).ID)
	c.JSON(http.StatusOK, gin.H{"card": card})
}

func (s *Server) wbDeleteCards(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if !bindJSON(c, &req) {
		return
	}
	ctx := c.Request.Context()
	board, err := s.whiteboard.ResolveBoard(ctx, wbActor(c), c.Param("id"))
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	// Draf terakhir kartu yang sedang diedit orang lain ikut tersimpan sebelum kartunya masuk Tempat sampah.
	batch, boards, err := s.whiteboard.DeleteCards(ctx, wbActor(c), board, req.IDs, s.hub.Drafts(req.IDs))
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	s.hub.Deleted(board, req.IDs, batch, currentUser(c).Name, boards)
	c.JSON(http.StatusOK, gin.H{"batch": batch})
}

func (s *Server) wbTrash(c *gin.Context) {
	out, err := s.whiteboard.Trash(c.Request.Context())
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"batches": out, "days": whiteboard.TrashDays, "can_write": wbCanWrite(c)})
}

func (s *Server) wbRestore(c *gin.Context) {
	out, err := s.whiteboard.Restore(c.Request.Context(), wbActor(c), c.Param("batch"))
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	s.hub.Resync(out.BoardID)
	c.JSON(http.StatusOK, gin.H{"restored": out})
}

func (s *Server) writeWbError(c *gin.Context, err error) {
	var ve *apperr.Validation
	var conflict *whiteboard.ConflictError
	switch {
	case errors.As(err, &ve):
		writeError(c, http.StatusBadRequest, CodeBadRequest, ve.Message)
	case errors.As(err, &conflict):
		// Kartu terbaru ikut dikirim supaya klien bisa menyimpan ketikan sebagai kartu salinan.
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"error": apiError{Code: CodeConflict, Message: "Isi kartu ini baru diubah orang lain."},
			"card":  conflict.Card,
		})
	case errors.Is(err, whiteboard.ErrForbidden):
		writeError(c, http.StatusForbidden, CodeForbidden, "Hanya penulis komentar (atau Manajer untuk menghapus) yang bisa mengubahnya.")
	case errors.Is(err, whiteboard.ErrStale):
		writeError(c, http.StatusConflict, CodeConflict, "Judul papan ini baru diubah orang lain. Muat ulang, lalu coba lagi.")
	case errors.Is(err, whiteboard.ErrNotFound):
		writeError(c, http.StatusNotFound, CodeNotFound, "Papan atau kartu tidak ditemukan. Mungkin sudah dipindah ke Tempat sampah.")
	default:
		s.internalError(c, err)
	}
}
