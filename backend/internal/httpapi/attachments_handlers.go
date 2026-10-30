package httpapi

import (
	"errors"
	"mime"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/attachment"
	"nvm-internal/internal/storage"
	"nvm-internal/internal/task"
)

type attachmentItem struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Name string `json:"name"`
	MIME string `json:"mime_type"`
	Size int64  `json:"size_bytes"`
}

type attachmentDetail struct {
	attachmentItem
	UploadedAt time.Time `json:"uploaded_at"`
	UploadedBy string    `json:"uploaded_by"`
}

// attachmentID membaca :attachmentId. ID yang bukan UUID diperlakukan sama dengan lampiran yang tidak ada.
func attachmentID(c *gin.Context) (string, bool) {
	id := c.Param("attachmentId")
	if !storage.ValidID(id) {
		writeError(c, http.StatusNotFound, CodeNotFound, "Lampiran tidak ditemukan.")
		return "", false
	}
	return id, true
}

func (s *Server) downloadAttachment(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	aid, ok := attachmentID(c)
	if !ok {
		return
	}
	att, f, err := s.attachments.Open(c.Request.Context(), id, aid)
	if err != nil {
		s.writeAttachmentError(c, err)
		return
	}
	defer func() { _ = f.Close() }()

	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": att.OriginalName})
	if disposition == "" {
		disposition = "attachment"
	}
	h := c.Writer.Header()
	h.Set("Content-Type", att.MimeType)
	// Berkas selalu diunduh, tidak pernah dibuka atau dijalankan di dalam aplikasi (NFR-SEC-09 d).
	h.Set("Content-Disposition", disposition)
	h.Set("X-Content-Type-Options", "nosniff")
	// Lapis tambahan: kalau berkas terbuka di tab browser, skrip di dalamnya tetap tidak bisa jalan.
	h.Set("Content-Security-Policy", "sandbox")
	h.Set("Cache-Control", "private, no-cache")
	// ServeContent mendukung Range, jadi unduhan yang terputus bisa dilanjutkan.
	http.ServeContent(c.Writer, c.Request, att.OriginalName, att.UploadedAt.Time, f)
}

func (s *Server) deleteAttachment(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	aid, ok := attachmentID(c)
	if !ok {
		return
	}
	if err := s.attachments.Delete(c.Request.Context(), taskActor(c), id, aid); err != nil {
		s.writeAttachmentError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// uploadTimeout cukup untuk 50 MB lewat Wi-Fi kantor yang lambat. Batas waktu server yang umum (belasan detik)
// akan memutus unggahan besar di tengah jalan, jadi rute ini diberi batas sendiri.
const uploadTimeout = 10 * time.Minute

// uploadAttachment menerima multipart/form-data dengan satu bagian "file". kind dikirim lewat query string supaya
// izin bisa dicek sebelum isi berkas dibaca.
func (s *Server) uploadAttachment(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	kind, err := task.ParseKind(c.Query("kind"))
	if err != nil {
		s.writeTaskError(c, err)
		return
	}
	actor := taskActor(c)
	if err := s.attachments.CheckUpload(c.Request.Context(), actor, id, kind); err != nil {
		s.writeAttachmentError(c, err)
		return
	}

	_ = http.NewResponseController(c.Writer).SetReadDeadline(time.Now().Add(uploadTimeout))
	// Sedikit di atas 50 MB untuk ruang header multipart. Batas isi berkas yang tepat dicek di service.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, task.MaxAttachmentSize+(1<<20))
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

	saved, err := s.attachments.Upload(c.Request.Context(), actor, id, kind, part.FileName(), part)
	if err != nil {
		s.writeAttachmentError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"attachment": attachmentItem{
		ID: saved.ID, Kind: string(saved.Kind), Name: saved.Name, MIME: saved.MIME, Size: saved.Size,
	}})
}

// filePart melewati bagian form lain dan mengambil bagian "file" pertama yang berisi berkas.
func filePart(mr *multipart.Reader) (*multipart.Part, error) {
	for {
		p, err := mr.NextPart()
		if err != nil {
			return nil, err
		}
		if p.FormName() == "file" && p.FileName() != "" {
			return p, nil
		}
		_ = p.Close()
	}
}

func (s *Server) writeAttachmentError(c *gin.Context, err error) {
	var rejected *attachment.Rejected
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &rejected):
		writeError(c, http.StatusUnsupportedMediaType, CodeUnsupportedFile, rejected.Message)
	case errors.Is(err, task.ErrTooLarge), errors.As(err, &tooBig):
		writeError(c, http.StatusRequestEntityTooLarge, CodeFileTooLarge, "Berkas melebihi batas 50 MB.")
	case errors.Is(err, task.ErrAttachmentNotFound):
		writeError(c, http.StatusNotFound, CodeNotFound, "Lampiran tidak ditemukan.")
	default:
		s.writeTaskError(c, err)
	}
}
