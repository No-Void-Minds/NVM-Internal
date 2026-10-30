package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/db/sqlc"
	"nvm-internal/internal/task"
)

type idName struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type taskPerson struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type taskLabel struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	IsActive bool   `json:"is_active"`
}

type taskSummary struct {
	ID         string       `json:"id"`
	Title      string       `json:"title"`
	Status     string       `json:"status"`
	Urgency    string       `json:"urgency"`
	Division   idName       `json:"division"`
	StartDate  *string      `json:"start_date"`
	DueDate    *string      `json:"due_date"`
	SolvedDate *time.Time   `json:"solved_date"`
	Version    int32        `json:"version"`
	Assignees  []taskPerson `json:"assignees"`
	Labels     []taskLabel  `json:"labels"`
}

type historyItem struct {
	ID         int64           `json:"id"`
	OccurredAt time.Time       `json:"occurred_at"`
	Action     string          `json:"action"`
	ActorName  string          `json:"actor_name"`
	OldValue   json.RawMessage `json:"old_value"`
	NewValue   json.RawMessage `json:"new_value"`
}

type taskDetail struct {
	taskSummary
	Description    string             `json:"description"`
	RequestedBy    idName             `json:"requested_by"`
	RequestDate    time.Time          `json:"request_date"`
	DeliverableURL string             `json:"deliverable_url"`
	IsMigrated     bool               `json:"is_migrated"`
	Can            task.Can           `json:"can"`
	History        []historyItem      `json:"history"`
	Attachments    []attachmentDetail `json:"attachments"`
}

type createTaskRequest struct {
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	Urgency        string  `json:"urgency"`
	PICIDs         []int64 `json:"pic_ids"`
	LabelIDs       []int64 `json:"label_ids"`
	StartDate      string  `json:"start_date"`
	DueDate        string  `json:"due_date"`
	DeliverableURL string  `json:"deliverable_url"`
}

func (s *Server) createTask(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Isian tidak terbaca.")
		return
	}
	id, err := s.tasks.Create(c.Request.Context(), taskActor(c), task.CreateInput{
		Title:          req.Title,
		Description:    req.Description,
		Urgency:        req.Urgency,
		PICIDs:         req.PICIDs,
		LabelIDs:       req.LabelIDs,
		StartDate:      req.StartDate,
		DueDate:        req.DueDate,
		DeliverableURL: req.DeliverableURL,
	})
	if err != nil {
		s.writeTaskError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (s *Server) getTask(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	d, err := s.tasks.Get(c.Request.Context(), id)
	if err != nil {
		s.writeTaskError(c, err)
		return
	}
	actor := taskActor(c)
	t := d.Task
	out := taskDetail{
		taskSummary: taskSummary{
			ID:         t.ID,
			Title:      t.Title,
			Status:     string(t.Status),
			Urgency:    string(t.Urgency),
			Division:   idName{ID: t.DivisionID, Name: t.DivisionName},
			StartDate:  dateString(t.StartDate),
			DueDate:    dateString(t.DueDate),
			SolvedDate: timePtr(t.SolvedDate),
			Version:    t.Version,
			Assignees:  people(d.Assignees),
			Labels:     labelsOf(d.Labels),
		},
		Description:    t.Description.String,
		RequestedBy:    idName{ID: t.RequestedBy, Name: t.RequestedByName},
		RequestDate:    t.RequestDate.Time,
		DeliverableURL: t.DeliverableUrl.String,
		IsMigrated:     t.IsMigrated,
		Can:            task.PermissionsFor(actor.Role, d.IsPIC(actor.ID), task.Status(t.Status)),
		History:        make([]historyItem, 0, len(d.History)),
	}
	for _, h := range d.History {
		out.History = append(out.History, historyItem{
			ID:         h.ID,
			OccurredAt: h.OccurredAt.Time,
			Action:     h.Action,
			ActorName:  h.ActorName.String,
			OldValue:   rawJSON(h.OldValue),
			NewValue:   rawJSON(h.NewValue),
		})
	}
	files, err := s.attachments.List(c.Request.Context(), id)
	if err != nil {
		s.internalError(c, err)
		return
	}
	out.Attachments = make([]attachmentDetail, 0, len(files))
	for _, a := range files {
		out.Attachments = append(out.Attachments, attachmentDetail{
			attachmentItem: attachmentItem{ID: a.ID, Kind: string(a.Kind), Name: a.OriginalName, MIME: a.MimeType, Size: a.SizeBytes},
			UploadedAt:     a.UploadedAt.Time,
			UploadedBy:     a.UploadedByName,
		})
	}
	c.JSON(http.StatusOK, gin.H{"task": out})
}

// listTasks menerima ?scope=mine untuk tampilan "Task saya"; tanpa itu semua task dikirim (Staf boleh membaca semua).
func (s *Server) listTasks(c *gin.Context) {
	var assignee *int64
	if c.Query("scope") == "mine" {
		id := currentUser(c).ID
		assignee = &id
	}
	rows, err := s.tasks.List(c.Request.Context(), assignee)
	if err != nil {
		s.internalError(c, err)
		return
	}
	items := make([]taskSummary, 0, len(rows))
	for _, r := range rows {
		items = append(items, summaryOf(r))
	}
	c.JSON(http.StatusOK, gin.H{"tasks": items})
}

func taskActor(c *gin.Context) task.Actor {
	u := currentUser(c)
	return task.Actor{ID: u.ID, Role: task.Role(u.Role), DivisionID: u.DivisionID, IP: clientAddr(c)}
}

// taskID membaca :id berbentuk T-nnnn. Bentuk lain diperlakukan sama dengan task yang tidak ada.
func taskID(c *gin.Context) (string, bool) {
	id := c.Param("id")
	if !task.ValidID(id) {
		writeError(c, http.StatusNotFound, CodeNotFound, "Task tidak ditemukan.")
		return "", false
	}
	return id, true
}

func (s *Server) writeTaskError(c *gin.Context, err error) {
	var ve *apperr.Validation
	switch {
	case errors.As(err, &ve):
		writeError(c, http.StatusBadRequest, CodeBadRequest, ve.Message)
	case errors.Is(err, task.ErrNotFound):
		writeError(c, http.StatusNotFound, CodeNotFound, "Task tidak ditemukan.")
	case errors.Is(err, task.ErrForbidden):
		writeError(c, http.StatusForbidden, CodeForbidden, "Kamu tidak punya hak untuk mengubah task ini.")
	case errors.Is(err, task.ErrVersionConflict):
		writeError(c, http.StatusConflict, CodeConflict, "Task ini sudah diubah orang lain. Muat ulang untuk melihat versi terbaru.")
	default:
		s.internalError(c, err)
	}
}

func people(rows []sqlc.ListTaskAssigneesRow) []taskPerson {
	out := make([]taskPerson, 0, len(rows))
	for _, r := range rows {
		out = append(out, taskPerson{ID: r.ID, Name: r.Name, Username: r.Username})
	}
	return out
}

func labelsOf(rows []sqlc.ListTaskLabelsRow) []taskLabel {
	out := make([]taskLabel, 0, len(rows))
	for _, r := range rows {
		out = append(out, taskLabel{ID: r.ID, Name: r.Name, Color: r.Color, IsActive: r.IsActive})
	}
	return out
}

// dateString mengirim tanggal tanpa jam ("2026-10-29"), supaya tidak bergeser karena zona waktu di browser.
func dateString(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := d.Time.Format("2006-01-02")
	return &s
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
