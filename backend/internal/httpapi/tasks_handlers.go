package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
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
	SubtaskPICs []taskPerson `json:"subtask_pics"`
	// Revisi 19: frekuensi seri untuk penanda berulang; null bila tidak berulang.
	SeriesFreq     *string `json:"series_freq"`
	OccurrenceDate *string `json:"occurrence_date"`
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
	Description    string    `json:"description"`
	RequestedBy    idName    `json:"requested_by"`
	RequestDate    time.Time `json:"request_date"`
	DeliverableURL string    `json:"deliverable_url"`
	IsMigrated     bool      `json:"is_migrated"`
	Can            task.Can  `json:"can"`
	// RevisionSubtasks: sama dengan di papan, untuk jalur revisi PIC sub task (revisi 22).
	RevisionSubtasks []task.SubtaskRef `json:"revision_subtasks"`
	// RevisionNoteRequired: catatan revisi wajib dan pilihan sub task opsional (PIC task dan Manajer, revisi 34);
	// false untuk jalur PIC sub task, yang wajib memilih dan alasannya opsional (revisi 22).
	RevisionNoteRequired bool               `json:"revision_note_required"`
	History              []historyItem      `json:"history"`
	Attachments          []attachmentDetail `json:"attachments"`
	Subtasks             []subtaskItem      `json:"subtasks"`
	Series               *seriesItem        `json:"series"`
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
	Schedule       *scheduleRequest      `json:"schedule"`
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
		Schedule:       req.Schedule.input(),
	})
	if err != nil {
		s.writeTaskError(c, err)
		return
	}
	c.Set(liveTaskKey, id)
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
			// Revisi 19: tanpa dua baris ini detail task berulang selalu menampilkan "Kemunculan: -",
			// padahal kolomnya terisi di database (CHECK tasks_series_pair menjamin itu).
			SeriesFreq:     detailSeriesFreq(d),
			OccurrenceDate: dateString(t.OccurrenceDate),
		},
		Description:    t.Description.String,
		RequestedBy:    idName{ID: t.RequestedBy, Name: t.RequestedByName},
		RequestDate:    t.RequestDate.Time,
		DeliverableURL: t.DeliverableUrl.String,
		IsMigrated:     t.IsMigrated,
		Series:         seriesOf(d, time.Now()),
		History:        make([]historyItem, 0, len(d.History)),
	}
	out.Can, out.RevisionSubtasks = task.MoveOptions(actor.Role, d.IsPIC(actor.ID), t.RequestedBy == actor.ID,
		d.MySubtasks(actor.ID), task.Status(t.Status))
	out.RevisionNoteRequired = task.RevisionNoteRequired(actor.Role, d.IsPIC(actor.ID), len(d.MySubtasks(actor.ID)) > 0)
	if out.RevisionNoteRequired {
		out.RevisionSubtasks = task.TaskRevisionChoices(actor.Role, d.IsPIC(actor.ID), task.Status(t.Status), d.Subtasks)
	}
	out.SubtaskPICs = detailSubtaskPICs(d)
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
	checkable := task.CheckableSubtasks(actor, task.Status(t.Status), isPIC, t.RequestedBy == actor.ID, d.Subtasks)
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
		// atau Staf PIC task untuk semua sub task di task-nya (revisi 32).
		manager := actor.Role == task.RoleManager
		assigned := isAssigned(actor.ID, st)
		taskPIC := actor.Role == task.RoleStaff && d.IsPIC(actor.ID)
		item.CanEdit = task.SubtaskWindowOpen(task.Status(t.Status)) &&
			(manager || assigned || taskPIC)
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
	var fe *task.ForbiddenError
	switch {
	case errors.As(err, &ve):
		writeError(c, http.StatusBadRequest, CodeBadRequest, ve.Message)
	case errors.Is(err, task.ErrNotFound):
		writeError(c, http.StatusNotFound, CodeNotFound, "Task tidak ditemukan.")
	case errors.Is(err, pgx.ErrNoRows):
		// Jaring pengaman: query :one yang tidak menemukan baris berarti task atau datanya sudah hilang, bukan
		// galat server. Dicatat supaya akar masalahnya tetap terlihat di log.
		s.log.Warn("ErrNoRows di jalur task", "err", err, "method", c.Request.Method, "route", c.FullPath(), "request_id", requestID(c))
		writeError(c, http.StatusNotFound, CodeNotFound, "Task tidak ditemukan atau sudah dihapus. Muat ulang halaman.")
	case errors.As(err, &fe):
		writeError(c, http.StatusForbidden, CodeForbidden, fe.Message)
	case errors.Is(err, task.ErrForbidden):
		// Cadangan bila ada penolakan tanpa pesan aksi; tidak menyebut aksi tertentu supaya tidak menyesatkan.
		writeError(c, http.StatusForbidden, CodeForbidden, "Kamu tidak punya hak melakukan ini pada task ini.")
	case errors.Is(err, task.ErrVersionConflict):
		// Revisi 47: pesan menyebut siapa yang lebih dulu mengubah, dari baris audit terakhir task ini.
		by, msg := "", "Task ini sudah diubah orang lain. Muat ulang untuk melihat versi terbaru."
		if st, e := sqlc.New(s.pool).TaskLiveState(c.Request.Context(), c.Param("id")); e == nil && st.LastActor != "" {
			by = st.LastActor
			msg = "Task ini sudah diubah " + by + " lebih dulu. Muat ulang untuk melihat versi terbaru."
			if st.LastActorID == currentUser(c).ID {
				by, msg = "kamu", "Task ini sudah kamu ubah di tab lain. Muat ulang untuk melihat versi terbaru."
			}
		}
		c.AbortWithStatusJSON(http.StatusConflict, errorBody{Error: apiError{Code: CodeConflict, Message: msg, By: by}})
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

type scheduleRequest struct {
	Freq          string `json:"freq"`
	SkipWeekend   bool   `json:"skip_weekend"`
	Weekdays      []int  `json:"weekdays"`
	MonthDay      int    `json:"month_day"`
	EndDate       string `json:"end_date"`
	ApplyToSeries bool   `json:"apply_to_series"`
}

// input: nil bila klien tidak mengirim bagian Jadwal, supaya klien lama tidak mengubah seri.
func (r *scheduleRequest) input() *task.ScheduleInput {
	if r == nil {
		return nil
	}
	return &task.ScheduleInput{
		Freq: r.Freq, SkipWeekend: r.SkipWeekend, Weekdays: r.Weekdays, MonthDay: r.MonthDay,
		EndDate: r.EndDate, ApplyToSeries: r.ApplyToSeries,
	}
}

type seriesItem struct {
	ID          int64   `json:"id"`
	Freq        string  `json:"freq"`
	SkipWeekend bool    `json:"skip_weekend"`
	Weekdays    []int16 `json:"weekdays"`
	MonthDay    *int16  `json:"month_day"`
	StartDate   *string `json:"start_date"`
	EndDate     *string `json:"end_date"`
	Active      bool    `json:"active"`
	// NextDate: kemunculan berikutnya setelah hari ini dan setelah kemunculan task ini; null bila seri berhenti
	// atau sudah habis. Dihitung server dengan aturan yang sama dengan pembuat kemunculan.
	NextDate *string `json:"next_date"`
	// Tenggat kemunculan berikutnya dari templat, untuk kalimat pratinjau. null berarti tanpa tenggat.
	DueOffsetDays *int `json:"due_offset_days"`
	// TemplateSubtasks: sub task yang akan ikut di kemunculan berikutnya, dari templat seri. Sejak revisi 29
	// hanya kemunculan terbaru yang mengubahnya, jadi di kemunculan lama daftar ini bisa berbeda (revisi 19).
	TemplateSubtasks []templateSubtaskItem `json:"template_subtasks"`
	// TemplateMatches: false berarti sub task task ini berbeda dengan templat. Dihitung server, jadi
	// frontend tidak membandingkan dua daftar sendiri.
	TemplateMatches bool `json:"template_matches"`
	// LatestOccurrence: perubahan sub task di task ini langsung masuk templat bila seri aktif (revisi 29).
	LatestOccurrence bool `json:"latest_occurrence"`
}

type templateSubtaskItem struct {
	Title string `json:"title"`
	PIC   string `json:"pic"`
}

func seriesOf(d task.Detail, now time.Time) *seriesItem {
	sr := d.Series
	if sr == nil {
		return nil
	}
	out := &seriesItem{
		ID: sr.ID, Freq: string(sr.Freq), SkipWeekend: sr.SkipWeekend, Weekdays: sr.Weekdays,
		StartDate: dateString(sr.StartDate), EndDate: dateString(sr.EndDate), Active: sr.Active,
		DueOffsetDays: task.TemplateDueOffset(sr.Template),
	}
	if sr.MonthDay.Valid {
		out.MonthDay = &sr.MonthDay.Int16
	}
	if out.Weekdays == nil {
		out.Weekdays = []int16{}
	}
	if next, ok := task.NextOccurrence(*sr, d.Task.OccurrenceDate, now); ok {
		v := next.Format("2006-01-02")
		out.NextDate = &v
	}
	out.TemplateSubtasks = []templateSubtaskItem{}
	for _, st := range d.TemplateSubtasks {
		out.TemplateSubtasks = append(out.TemplateSubtasks, templateSubtaskItem{Title: st.Title, PIC: st.PIC})
	}
	out.TemplateMatches = d.TemplateMatchesTask()
	out.LatestOccurrence = d.LatestOccurrence
	return out
}

// detailSeriesFreq membaca frekuensi dari seri yang sudah dimuat Detail, bukan dari kolom task, karena
// GetTaskDetail tidak ikut membaca task_series.
func detailSeriesFreq(d task.Detail) *string {
	if d.Series == nil {
		return nil
	}
	v := string(d.Series.Freq)
	return &v
}

func seriesFreq(f sqlc.NullRecurrenceFreq) *string {
	if !f.Valid {
		return nil
	}
	v := string(f.RecurrenceFreq)
	return &v
}

// detailSubtaskPICs mengisi subtask_pics di detail dengan aturan yang sama dengan ListSubtaskPICs: unik, urut
// menurut sub task pertama yang dipegang. Sub task di Detail sudah urut posisi.
func detailSubtaskPICs(d task.Detail) []taskPerson {
	out := []taskPerson{}
	seen := map[int64]bool{}
	for _, st := range d.Subtasks {
		for _, a := range st.Assignees {
			if !seen[a.ID] {
				seen[a.ID] = true
				out = append(out, taskPerson{ID: a.ID, Name: a.Name, Username: a.Username, AvatarID: avatarID(a.AvatarID)})
			}
		}
	}
	return out
}
