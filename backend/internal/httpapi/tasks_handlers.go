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
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Username string  `json:"username"`
	AvatarID *string `json:"avatar_id"`
}

type taskLabel struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	IsActive bool   `json:"is_active"`
}

type subtaskRequest struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type taskSummary struct {
	ID           string       `json:"id"`
	Title        string       `json:"title"`
	Status       string       `json:"status"`
	Urgency      string       `json:"urgency"`
	Division     idName       `json:"division"`
	StartDate    *string      `json:"start_date"`
	DueDate      *string      `json:"due_date"`
	SolvedDate   *time.Time   `json:"solved_date"`
	Version      int32        `json:"version"`
	Assignees    []taskPerson `json:"assignees"`
	Labels       []taskLabel  `json:"labels"`
	SubtaskTotal int          `json:"subtask_total"`
	SubtaskDone  int          `json:"subtask_done"`
	// PIC sub task yang membuat task ini masuk hasil filter PIC (revisi 15). Kosong tanpa filter PIC.
	SubtaskPICMatches []idName `json:"subtask_pic_matches"`
}

type subtaskItem struct {
	ID             int64  `json:"id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	ReferenceURL   string `json:"reference_url"`
	DeliverableURL string `json:"deliverable_url"`
	CreatedBy      string `json:"created_by"`
	// ID orang dikirim supaya nama bisa membuka kartu profil (revisi 14). nil bila akunnya tidak tercatat.
	CreatedByID *int64     `json:"created_by_id"`
	Done        bool       `json:"done"`
	DoneAt      *time.Time `json:"done_at"`
	DoneBy      string     `json:"done_by"`
	DueDate     *string    `json:"due_date"`
	ResultNote  string     `json:"result_note"`
	// Boleh mengunggah dan menghapus berkas sub task ini: PIC task, Manajer, dan PIC sub task (sama dengan
	// canAttachSubtask), supaya tombol berkas mengikuti server, termasuk untuk PIC sub task dari luar task.
	CanAttach   bool               `json:"can_attach"`
	CanCheck    bool               `json:"can_check"`
	CanEdit     bool               `json:"can_edit"`
	Assignees   []taskPerson       `json:"assignees"`
	Attachments []attachmentDetail `json:"attachments"`
}

type historyItem struct {
	ID         int64           `json:"id"`
	OccurredAt time.Time       `json:"occurred_at"`
	Action     string          `json:"action"`
	ActorName  string          `json:"actor_name"`
	ActorID    *int64          `json:"actor_id"`
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
	Subtasks       []subtaskItem      `json:"subtasks"`
}

type createTaskRequest struct {
	Title          string                `json:"title"`
	Description    string                `json:"description"`
	Urgency        string                `json:"urgency"`
	PICIDs         []int64               `json:"pic_ids"`
	Subtasks       []subtaskTitleRequest `json:"subtasks"`
	LabelIDs       []int64               `json:"label_ids"`
	StartDate      string                `json:"start_date"`
	DueDate        string                `json:"due_date"`
	DeliverableURL string                `json:"deliverable_url"`
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
		Subtasks:       subtaskFields(req.Subtasks),
		LabelIDs:       req.LabelIDs,
		StartDate:      req.StartDate,
		DueDate:        req.DueDate,
		DeliverableURL: req.DeliverableURL,
	})
	if err != nil {
		s.writeTaskError(c, err)
		return
	}
	// ID sub task dikirim urut sesuai form, supaya lampiran acuan tiap baris bisa diunggah ke sub task-nya.
	d, err := s.tasks.Get(c.Request.Context(), id)
	if err != nil {
		s.internalError(c, err)
		return
	}
	subIDs := make([]int64, len(d.Subtasks))
	for i, st := range d.Subtasks {
		subIDs[i] = st.ID
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "subtask_ids": subIDs})
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
		Can:            task.PermissionsFor(actor.Role, d.IsPIC(actor.ID), d.IsSubtaskPIC(actor.ID), task.Status(t.Status)),
		History:        make([]historyItem, 0, len(d.History)),
	}
	for _, h := range d.History {
		out.History = append(out.History, historyItem{
			ID:         h.ID,
			OccurredAt: h.OccurredAt.Time,
			Action:     h.Action,
			ActorName:  h.ActorName.String,
			ActorID:    int8Ptr(h.ActorID),
			OldValue:   rawJSON(h.OldValue),
			NewValue:   rawJSON(h.NewValue),
		})
	}
	files, err := s.attachments.List(c.Request.Context(), id)
	if err != nil {
		s.internalError(c, err)
		return
	}
	// Satu query untuk semua berkas task ini, lalu dipilah: yang punya subtask_id menempel di sub task-nya.
	out.Attachments = make([]attachmentDetail, 0, len(files))
	subFiles := map[int64][]attachmentDetail{}
	for _, a := range files {
		item := attachmentDetail{
			attachmentItem: attachmentItem{ID: a.ID, Kind: string(a.Kind), Name: a.OriginalName, MIME: a.MimeType, Size: a.SizeBytes},
			UploadedAt:     a.UploadedAt.Time,
			UploadedBy:     a.UploadedByName,
			UploadedByID:   a.UploadedByID,
		}
		if a.SubtaskID.Valid {
			subFiles[a.SubtaskID.Int64] = append(subFiles[a.SubtaskID.Int64], item)
			continue
		}
		out.Attachments = append(out.Attachments, item)
	}
	out.Subtasks = make([]subtaskItem, 0, len(d.Subtasks))
	doneCount := 0
	isPIC := d.IsPIC(actor.ID)
	checkable, err := s.tasks.CheckableSubtasks(c.Request.Context(), actor, task.Status(t.Status), isPIC, d.Subtasks)
	if err != nil {
		s.writeTaskError(c, err)
		return
	}
	for _, st := range d.Subtasks {
		item := subtaskItem{
			ID: st.ID, Title: st.Title, Description: st.Description,
			ReferenceURL: st.ReferenceURL, DeliverableURL: st.DeliverableURL,
			CreatedBy: st.CreatedByName, CreatedByID: nonZero(st.CreatedBy),
			Done: st.DoneAt.Valid, DoneAt: timePtr(st.DoneAt), DoneBy: st.DoneBy, DueDate: dateString(st.DueDate),
			ResultNote:  st.ResultNote,
			Assignees:   make([]taskPerson, 0, len(st.Assignees)),
			Attachments: subFiles[st.ID],
		}
		if item.Attachments == nil {
			item.Attachments = []attachmentDetail{}
		}
		// can_check berasal dari aturan yang sama dengan SetSubtaskDone, supaya UI tidak menebak.
		item.CanCheck = checkable[st.ID]
		// Sama dengan lockForSubtask + editableSubtask: Manajer, PIC sub task ini (juga dari luar task, revisi 11),
		// atau pembuatnya selama dia masih PIC task. Sebelumnya disyaratkan PIC task, sehingga PIC sub task dari
		// luar task tidak pernah melihat tombol Ubah dan kotak hasil walau backend menerima perubahannya.
		manager := actor.Role == task.RoleManager
		assigned := isAssigned(actor.ID, st)
		taskPIC := actor.Role == task.RoleStaff && d.IsPIC(actor.ID)
		item.CanEdit = task.SubtaskWindowOpen(task.Status(t.Status)) &&
			(manager || assigned || (taskPIC && st.CreatedBy == actor.ID))
		item.CanAttach = manager || taskPIC || assigned
		for _, a := range st.Assignees {
			item.Assignees = append(item.Assignees, taskPerson{ID: a.ID, Name: a.Name, Username: a.Username, AvatarID: avatarID(a.AvatarID)})
		}
		if item.Done {
			doneCount++
		}
		out.Subtasks = append(out.Subtasks, item)
	}
	out.SubtaskTotal, out.SubtaskDone = len(out.Subtasks), doneCount
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
		writeError(c, http.StatusForbidden, CodeForbidden, "Sub task ini hanya bisa dicentang PIC-nya, atau Manajer bila tidak dipegang Manajer lain.")
	case errors.Is(err, task.ErrVersionConflict):
		writeError(c, http.StatusConflict, CodeConflict, "Task ini sudah diubah orang lain. Muat ulang untuk melihat versi terbaru.")
	default:
		s.internalError(c, err)
	}
}

func people(rows []sqlc.ListTaskAssigneesRow) []taskPerson {
	out := make([]taskPerson, 0, len(rows))
	for _, r := range rows {
		out = append(out, taskPerson{ID: r.ID, Name: r.Name, Username: r.Username, AvatarID: avatarID(r.AvatarID)})
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

// isAssigned: orang ini PIC sub task ini.
func isAssigned(userID int64, st task.SubtaskView) bool {
	for _, a := range st.Assignees {
		if a.ID == userID {
			return true
		}
	}
	return false
}

func subtaskInputs(reqs []subtaskRequest) []task.SubtaskInput {
	out := make([]task.SubtaskInput, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, task.SubtaskInput{ID: r.ID, Title: r.Title})
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

func int8Ptr(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func nonZero(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}

func subtaskFields(reqs []subtaskTitleRequest) []task.SubtaskFields {
	out := make([]task.SubtaskFields, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, r.fields())
	}
	return out
}
