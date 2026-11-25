package httpapi

import (
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/whiteboard"
)

// Whiteboard W3: berkas, pratinjau tautan, tabel, garis penghubung, komentar, pindah papan.

// wbLocked menolak perubahan kartu yang kuncinya dipegang akun lain (tabel dan gambar bebas, serta pindah papan).
func (s *Server) wbLocked(c *gin.Context, ids ...string) bool {
	for _, id := range ids {
		if name := s.hub.LockedByOther(id, currentUser(c).ID); name != "" {
			writeError(c, http.StatusConflict, CodeConflict, name+" sedang mengubah kartu ini. Tunggu sampai selesai, lalu coba lagi.")
			return true
		}
	}
	return false
}

func queryFloat(c *gin.Context, key string) float64 {
	v, _ := strconv.ParseFloat(c.Query(key), 64)
	return v
}

func (s *Server) wbUpload(c *gin.Context) {
	in := whiteboard.CardInput{ID: c.Query("card_id"), X: queryFloat(c, "x"), Y: queryFloat(c, "y"), W: queryFloat(c, "w"), H: queryFloat(c, "h")}
	_ = http.NewResponseController(c.Writer).SetReadDeadline(time.Now().Add(uploadTimeout))
	// Sedikit di atas 50 MB untuk ruang header multipart. Batas isi berkas yang tepat dicek di service.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, whiteboard.MaxFileSize+(1<<20))
	mr, err := c.Request.MultipartReader()
	if err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Unggahan harus berupa multipart/form-data.")
		return
	}
	part, err := filePart(mr)
	if err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Tidak ada berkas di unggahan ini.")
		return
	}
	defer func() { _ = part.Close() }()
	card, err := s.whiteboard.Upload(c.Request.Context(), wbActor(c), c.Param("id"), in, part.FileName(), part)
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		err = whiteboard.ErrTooLarge
	}
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	s.hub.Changed(card.BoardID, []whiteboard.Card{card}, currentUser(c).ID)
	c.JSON(http.StatusCreated, gin.H{"card": card})
}

// wbFile menyajikan berkas asli (gambar dan PDF tampil di tab, sisanya diunduh) atau thumbnail JPEG. Id berkas
// tidak pernah dipakai ulang untuk isi lain, jadi boleh di-cache peramban.
func (s *Server) wbFile(c *gin.Context) {
	thumb := c.Query("thumb") == "1"
	row, f, err := s.whiteboard.OpenFile(c.Request.Context(), c.Param("fileId"), thumb)
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	defer func() { _ = f.Close() }()
	kind, ctype := "attachment", row.MimeType
	if thumb {
		ctype = "image/jpeg"
	}
	if thumb || strings.HasPrefix(ctype, "image/") || ctype == "application/pdf" {
		kind = "inline"
	}
	disposition := mime.FormatMediaType(kind, map[string]string{"filename": row.OriginalName})
	if disposition == "" {
		disposition = kind
	}
	h := c.Writer.Header()
	h.Set("Content-Type", ctype)
	h.Set("Content-Disposition", disposition)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", "sandbox")
	h.Set("Cache-Control", "private, max-age=86400")
	http.ServeContent(c.Writer, c.Request, row.OriginalName, row.CreatedAt.Time, f)
}

func (s *Server) wbPreview(c *gin.Context) {
	var in struct {
		URL string `json:"url"`
	}
	if !bindJSON(c, &in) {
		return
	}
	out, err := s.whiteboard.Preview(c.Request.Context(), wbActor(c), in.URL)
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"preview": out})
}

func (s *Server) wbTable(c *gin.Context) {
	var op whiteboard.TableOp
	if !bindJSON(c, &op) || s.wbLocked(c, c.Param("cardId")) {
		return
	}
	card, err := s.whiteboard.Table(c.Request.Context(), wbActor(c), c.Param("id"), c.Param("cardId"), op)
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	s.hub.Changed(card.BoardID, []whiteboard.Card{card}, currentUser(c).ID)
	c.JSON(http.StatusOK, gin.H{"card": card})
}

func (s *Server) wbCreateDrawing(c *gin.Context) {
	var in struct {
		ID     string            `json:"id"`
		Stroke whiteboard.Stroke `json:"stroke"`
	}
	if !bindJSON(c, &in) {
		return
	}
	card, err := s.whiteboard.CreateDrawing(c.Request.Context(), wbActor(c), c.Param("id"), in.ID, in.Stroke)
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	s.hub.Changed(card.BoardID, []whiteboard.Card{card}, currentUser(c).ID)
	c.JSON(http.StatusCreated, gin.H{"card": card})
}

// wbDrawing: penghapus atau goresan baru di objek yang dikunci orang lain ditolak dengan nama pemegangnya.
func (s *Server) wbDrawing(c *gin.Context) {
	var op whiteboard.DrawingOp
	if !bindJSON(c, &op) || s.wbLocked(c, c.Param("cardId")) {
		return
	}
	card, err := s.whiteboard.Drawing(c.Request.Context(), wbActor(c), c.Param("id"), c.Param("cardId"), op)
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	s.hub.Changed(card.BoardID, []whiteboard.Card{card}, currentUser(c).ID)
	c.JSON(http.StatusOK, gin.H{"card": card})
}

func (s *Server) wbBoards(c *gin.Context) {
	out, err := s.whiteboard.Boards(c.Request.Context())
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"boards": out})
}

func (s *Server) wbMove(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"`
		To  string   `json:"to"`
	}
	if !bindJSON(c, &req) || s.wbLocked(c, req.IDs...) {
		return
	}
	out, err := s.whiteboard.MoveCards(c.Request.Context(), wbActor(c), c.Param("id"), req.IDs, req.To)
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	s.hub.Moved(out, currentUser(c).Name)
	c.JSON(http.StatusOK, gin.H{"moved": out})
}

func (s *Server) wbCreateConnector(c *gin.Context) {
	var in whiteboard.ConnectorInput
	if !bindJSON(c, &in) {
		return
	}
	out, err := s.whiteboard.CreateConnector(c.Request.Context(), wbActor(c), c.Param("id"), in)
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	s.hub.Connectors(out.BoardID, []whiteboard.Connector{out})
	c.JSON(http.StatusCreated, gin.H{"connector": out})
}

func (s *Server) wbUpdateConnector(c *gin.Context) {
	var in whiteboard.ConnectorPatch
	if !bindJSON(c, &in) {
		return
	}
	out, err := s.whiteboard.UpdateConnector(c.Request.Context(), wbActor(c), c.Param("id"), c.Param("connectorId"), in)
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	s.hub.Connectors(out.BoardID, []whiteboard.Connector{out})
	c.JSON(http.StatusOK, gin.H{"connector": out})
}

func (s *Server) wbDeleteConnector(c *gin.Context) {
	ctx := c.Request.Context()
	board, err := s.whiteboard.ResolveBoard(ctx, wbActor(c), c.Param("id"))
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	batch, err := s.whiteboard.DeleteConnector(ctx, wbActor(c), board, c.Param("connectorId"))
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	s.hub.ConnectorsGone(board, []string{c.Param("connectorId")}, batch)
	c.JSON(http.StatusOK, gin.H{"batch": batch})
}

func (s *Server) wbComments(c *gin.Context) {
	out, err := s.whiteboard.Comments(c.Request.Context(), wbActor(c), c.Param("id"), c.Param("cardId"))
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"comments": out, "me": currentUser(c).ID, "can_moderate": wbActor(c).Manager})
}

func (s *Server) wbAddComment(c *gin.Context) {
	var in struct {
		Body string `json:"body"`
	}
	if !bindJSON(c, &in) {
		return
	}
	card, err := s.whiteboard.AddComment(c.Request.Context(), wbActor(c), c.Param("id"), c.Param("cardId"), in.Body)
	s.wbCommentDone(c, card, err, http.StatusCreated)
}

func (s *Server) wbEditComment(c *gin.Context) {
	var in struct {
		Body string `json:"body"`
	}
	if !bindJSON(c, &in) {
		return
	}
	card, err := s.whiteboard.EditComment(c.Request.Context(), wbActor(c), c.Param("commentId"), in.Body)
	s.wbCommentDone(c, card, err, http.StatusOK)
}

func (s *Server) wbDeleteComment(c *gin.Context) {
	card, err := s.whiteboard.DeleteComment(c.Request.Context(), wbActor(c), c.Param("commentId"))
	s.wbCommentDone(c, card, err, http.StatusOK)
}

// wbCommentDone menyiarkan kartu (jumlah komentar berubah) supaya panel komentar orang lain ikut memuat ulang.
func (s *Server) wbCommentDone(c *gin.Context, card whiteboard.Card, err error, status int) {
	if err != nil {
		s.writeWbError(c, err)
		return
	}
	s.hub.Changed(card.BoardID, []whiteboard.Card{card}, currentUser(c).ID)
	c.JSON(status, gin.H{"card": card})
}
