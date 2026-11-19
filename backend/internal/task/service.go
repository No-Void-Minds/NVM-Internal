package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
)

var (
	ErrNotFound        = errors.New("task tidak ditemukan")
	ErrForbidden       = errors.New("tidak punya hak atas task ini")
	ErrVersionConflict = errors.New("task sudah diubah orang lain")
)

var idPattern = regexp.MustCompile(`^T-[0-9]{4,8}$`)

// ValidID dipakai handler untuk menolak ID yang bentuknya salah sebelum menyentuh database.
func ValidID(id string) bool { return idPattern.MatchString(id) }

// Actor adalah akun yang sedang memakai aplikasi. DivisionID dipakai untuk membatasi label yang boleh dipasang Staf.
type Actor struct {
	ID         int64
	Role       Role
	DivisionID int64
	IP         *netip.Addr
}

func (a Actor) auditActor() audit.Actor { return audit.Actor{ID: a.ID, IP: a.IP} }

type CreateInput struct {
	Title       string
	Description string
	Urgency     string
	PICIDs      []int64
	// Revisi 16: sub task dari form Buat task membawa isi lengkap, sama dengan Tambah sub task di detail.
	Subtasks       []SubtaskFields
	LabelIDs       []int64
	StartDate      string
	DueDate        string
	DeliverableURL string
	// Schedule nil atau Freq kosong berarti tidak berulang (revisi 19).
	Schedule *ScheduleInput
}

type Service struct {
	pool *pgxpool.Pool
	// now bisa diganti di test supaya "hari ini" (officeTZ) tidak bergantung pada jam sistem (revisi 19).
	now func() time.Time
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, now: time.Now}
}

func (s *Service) clock() time.Time {
	if s.now == nil {
		return time.Now()
	}
	return s.now()
}

// Create membuat task baru berstatus Requested. Staf dan Manajer boleh, dengan aturan yang sama persis
// (revisi 27); Admin tidak, karena Admin mengelola sistem dan tidak mengerjakan task. Pembuatnya tersimpan di
// requested_by. Sejak revisi 6 PIC task wajib dipilih, dan sub task boleh dikosongkan karena PIC yang akan
// mengisinya sendiri. Divisi task mengikuti PIC pertama.
func (s *Service) Create(ctx context.Context, actor Actor, in CreateInput) (string, error) {
	if !CanCreateTask(actor.Role) {
		return "", errCreateForbidden
	}
	f, err := cleanFields(in.Title, in.Description, in.Urgency, in.StartDate, in.DueDate, in.DeliverableURL)
	if err != nil {
		return "", err
	}
	picIDs := uniqueIDs(in.PICIDs)
	if len(picIDs) == 0 {
		return "", apperr.Invalid("Pilih minimal satu PIC task.")
	}
	if len(in.Subtasks) > maxSubtasks {
		return "", apperr.Invalid(fmt.Sprintf("Satu task paling banyak punya %d sub task.", maxSubtasks))
	}
	subs := make([]cleanSubtaskFields, len(in.Subtasks))
	for i, st := range in.Subtasks {
		if subs[i], err = cleanFieldsOf(st); err != nil {
			return "", numbered(i, err)
		}
		// Tautan hasil diisi saat pengerjaan, bukan saat task dibuat.
		subs[i].deliverable = ""
	}
	labels := uniqueIDs(in.LabelIDs)

	var id string
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)

		divisionOf, err := eligiblePICs(ctx, q, picIDs)
		if err != nil {
			return err
		}
		if err := requireActiveLabels(ctx, q, labels); err != nil {
			return err
		}

		division := divisionOf[picIDs[0]]

		id, err = q.CreateTask(ctx, sqlc.CreateTaskParams{
			Title:          f.title,
			Description:    f.description,
			DivisionID:     division,
			RequestedBy:    actor.ID,
			Urgency:        f.urgency,
			StartDate:      f.startDate,
			DueDate:        f.dueDate,
			DeliverableUrl: f.deliverableURL,
		})
		if err != nil {
			return err
		}
		if err := addTaskPICs(ctx, q, actor.ID, id, picIDs); err != nil {
			return err
		}
		snaps, err := createSubtasks(ctx, q, actor, id, f.dueDate, subs, in.Subtasks)
		if err != nil {
			return err
		}
		for _, l := range labels {
			if err := q.AddTaskLabel(ctx, sqlc.AddTaskLabelParams{TaskID: id, LabelID: l}); err != nil {
				return err
			}
		}
		if err := audit.Change(ctx, q, actor.auditActor(), "task_created", "task", id, nil, map[string]any{
			"title": f.title, "urgency": f.urgency, "pic_ids": picIDs, "subtasks": snaps, "label_ids": labels,
		}); err != nil {
			return err
		}
		// Task dari form adalah kemunculan pertama seri (revisi 19). Dibuat di transaksi yang sama, jadi aturan
		// yang tidak sah membatalkan task-nya juga.
		if in.Schedule.recurring() {
			return s.startSeries(ctx, q, actor, id, in.Schedule)
		}
		return nil
	})
	return id, err
}

type Detail struct {
	Task sqlc.GetTaskDetailRow
	// Series terisi bila task ini kemunculan sebuah seri (revisi 19).
	Series    *sqlc.GetSeriesRow
	Assignees []sqlc.ListTaskAssigneesRow
	Subtasks  []SubtaskView
	Labels    []sqlc.ListTaskLabelsRow
	History   []sqlc.ListTaskHistoryRow
	// TemplateSubtasks: sub task di templat seri, nama PIC-nya sudah dicarikan. Kosong bila task ini tidak
	// berulang (revisi 19).
	TemplateSubtasks []TemplateSubtask
	// LatestOccurrence: task ini kemunculan terbaru serinya, jadi perubahan sub task-nya langsung masuk templat
	// selama seri aktif (revisi 29).
	LatestOccurrence bool
}

// TemplateSubtask adalah satu sub task di templat seri, siap ditampilkan.
type TemplateSubtask struct {
	Title string
	PIC   string
}

// TemplateMatchesTask membandingkan sub task task ini dengan templat: judul dan PIC, berurutan. Dipakai
// detail untuk memberi tahu bahwa perubahan sub task belum masuk templat (revisi 19).
func (d Detail) TemplateMatchesTask() bool {
	if d.Series == nil {
		return true
	}
	if len(d.Subtasks) != len(d.TemplateSubtasks) {
		return false
	}
	for i, st := range d.Subtasks {
		pic := ""
		if len(st.Assignees) > 0 {
			pic = st.Assignees[0].Name
		}
		if st.Title != d.TemplateSubtasks[i].Title || pic != d.TemplateSubtasks[i].PIC {
			return false
		}
	}
	return true
}

// IsPIC menjawab apakah akun ini salah satu PIC task.
func (d Detail) IsPIC(userID int64) bool {
	for _, a := range d.Assignees {
		if a.ID == userID {
			return true
		}
	}
	return false
}

// Get bisa dipanggil semua peran: Staf boleh membaca semua task (NFR-SEC-03), Admin hanya membaca (FR-TASK-19).
func (s *Service) Get(ctx context.Context, id string) (Detail, error) {
	q := sqlc.New(s.pool)
	t, err := q.GetTaskDetail(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, err
	}
	d := Detail{Task: t}
	ids := []string{id}
	if d.Assignees, err = q.ListTaskAssignees(ctx, ids); err != nil {
		return Detail{}, err
	}
	if d.Subtasks, err = loadSubtaskViews(ctx, q, id); err != nil {
		return Detail{}, err
	}
	if d.Labels, err = q.ListTaskLabels(ctx, ids); err != nil {
		return Detail{}, err
	}
	if d.History, err = q.ListTaskHistory(ctx, id); err != nil {
		return Detail{}, err
	}
	if d.Task.SeriesID.Valid {
		sr, err := q.GetSeries(ctx, d.Task.SeriesID.Int64)
		if err != nil {
			return d, err
		}
		d.Series = &sr
		if d.TemplateSubtasks, err = templateSubtasks(ctx, q, sr.Template); err != nil {
			return d, err
		}
		latest, err := q.IsLatestOccurrence(ctx, sqlc.IsLatestOccurrenceParams{
			SeriesID: d.Task.SeriesID, OccurrenceDate: d.Task.OccurrenceDate,
		})
		if err != nil {
			return d, err
		}
		d.LatestOccurrence = latest.Bool
	}
	return d, nil
}

// templateSubtasks membaca sub task dari templat seri dan mencarikan nama PIC-nya, supaya detail bisa
// menampilkan sub task mana yang ikut diulang (revisi 19).
func templateSubtasks(ctx context.Context, q *sqlc.Queries, raw []byte) ([]TemplateSubtask, error) {
	var tmpl SeriesTemplate
	if len(raw) == 0 {
		return nil, nil
	}
	if err := json.Unmarshal(raw, &tmpl); err != nil {
		// Templat yang tidak terbaca tidak boleh membuat detail task gagal dimuat; daftarnya saja yang kosong.
		return nil, nil
	}
	ids := []int64{}
	for _, st := range tmpl.Subtasks {
		if st.PICID != 0 {
			ids = append(ids, st.PICID)
		}
	}
	names := map[int64]string{}
	if len(ids) > 0 {
		rows, err := q.ListUserNames(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			names[r.ID] = r.Name
		}
	}
	out := make([]TemplateSubtask, 0, len(tmpl.Subtasks))
	for _, st := range tmpl.Subtasks {
		out = append(out, TemplateSubtask{Title: st.Title, PIC: names[st.PICID]})
	}
	return out, nil
}

type Summary struct {
	Task         sqlc.ListTasksRow
	Assignees    []sqlc.ListTaskAssigneesRow
	Labels       []sqlc.ListTaskLabelsRow
	SubtaskTotal int
	SubtaskDone  int
	// SubtaskPICs: semua PIC sub task, unik, urut menurut sub task pertama yang dipegang (revisi 22).
	SubtaskPICs []sqlc.ListSubtaskPICsRow
}

// List mengambil task yang belum dihapus. assigneeID diisi untuk tampilan "Task saya".
func (s *Service) List(ctx context.Context, assigneeID *int64) ([]Summary, error) {
	q := sqlc.New(s.pool)
	var assignee pgtype.Int8
	if assigneeID != nil {
		assignee = pgtype.Int8{Int64: *assigneeID, Valid: true}
	}
	rows, err := q.ListTasks(ctx, assignee)
	if err != nil {
		return nil, err
	}
	return attachPeopleAndLabels(ctx, q, rows)
}

// attachPeopleAndLabels melengkapi task dengan PIC dan label. Keduanya diambil sekaligus untuk semua task,
// bukan satu query per task.
func attachPeopleAndLabels(ctx context.Context, q *sqlc.Queries, rows []sqlc.ListTasksRow) ([]Summary, error) {
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	assignees, err := q.ListTaskAssignees(ctx, ids)
	if err != nil {
		return nil, err
	}
	labels, err := q.ListTaskLabels(ctx, ids)
	if err != nil {
		return nil, err
	}
	byTask := make(map[string]*Summary, len(rows))
	out := make([]Summary, len(rows))
	for i, r := range rows {
		out[i] = Summary{
			Task: r, Assignees: []sqlc.ListTaskAssigneesRow{}, Labels: []sqlc.ListTaskLabelsRow{},
			SubtaskPICs: []sqlc.ListSubtaskPICsRow{},
		}
		byTask[r.ID] = &out[i]
	}
	for _, a := range assignees {
		byTask[a.TaskID].Assignees = append(byTask[a.TaskID].Assignees, a)
	}
	for _, l := range labels {
		byTask[l.TaskID].Labels = append(byTask[l.TaskID].Labels, l)
	}
	counts, err := q.CountSubtasksByTask(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, c := range counts {
		byTask[c.TaskID].SubtaskTotal = int(c.Total)
		byTask[c.TaskID].SubtaskDone = int(c.Done)
	}
	subPICs, err := q.ListSubtaskPICs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, p := range subPICs {
		byTask[p.TaskID].SubtaskPICs = append(byTask[p.TaskID].SubtaskPICs, p)
	}
	return out, nil
}

// addTaskPICs memasang PIC satu per satu supaya urutan pemasangan (clock_timestamp) mengikuti urutan pilihan
// Manajer: PIC pertama menentukan divisi task.
func addTaskPICs(ctx context.Context, q *sqlc.Queries, actorID int64, taskID string, ids []int64) error {
	for _, uid := range ids {
		if err := q.AddTaskAssignee(ctx, sqlc.AddTaskAssigneeParams{TaskID: taskID, UserID: uid, AssignedBy: actorID}); err != nil {
			return err
		}
	}
	return nil
}

// eligiblePICs memastikan semua PIC adalah Staf atau Manajer aktif, lalu mengembalikan divisi masing-masing.
func eligiblePICs(ctx context.Context, q *sqlc.Queries, ids []int64) (map[int64]int64, error) {
	rows, err := q.GetEligiblePICs(ctx, ids)
	if err != nil {
		return nil, err
	}
	divisionOf := make(map[int64]int64, len(rows))
	for _, r := range rows {
		divisionOf[r.ID] = r.DivisionID
	}
	for _, id := range ids {
		if _, ok := divisionOf[id]; !ok {
			return nil, apperr.Invalid("PIC harus akun Staf atau Manajer yang aktif.")
		}
	}
	return divisionOf, nil
}

func requireActiveLabels(ctx context.Context, q *sqlc.Queries, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := q.GetLabelsByIDs(ctx, ids)
	if err != nil {
		return err
	}
	active := make(map[int64]bool, len(rows))
	for _, r := range rows {
		active[r.ID] = r.IsActive
	}
	for _, id := range ids {
		if !active[id] {
			return apperr.Invalid("Label tidak ditemukan atau sudah nonaktif.")
		}
	}
	return nil
}

// createSubtasks menulis sub task dari form Buat task dalam transaksi pembuatan task, dengan aturan yang sama
// dengan AddSubtask: tenggat tidak melewati tenggat task dan PIC harus calon yang sah. Galat pertama
// membatalkan seluruh transaksi, jadi task tidak tersimpan setengah jadi.
func createSubtasks(ctx context.Context, q *sqlc.Queries, actor Actor, taskID string, taskDue pgtype.Date,
	subs []cleanSubtaskFields, raw []SubtaskFields) ([]map[string]any, error) {
	snaps := make([]map[string]any, 0, len(subs))
	for i, f := range subs {
		if err := checkSubtaskDue(f.due, taskDue); err != nil {
			return nil, numbered(i, err)
		}
		sid, err := q.CreateSubtask(ctx, sqlc.CreateSubtaskParams{
			TaskID: taskID, Title: f.title, Description: f.description,
			ReferenceUrl: f.reference, DeliverableUrl: f.deliverable,
			Position: int32(i), CreatedBy: userRef(actor.ID), DueDate: f.due,
		})
		if err != nil {
			return nil, err
		}
		if err := setSubtaskPIC(ctx, q, actor, taskID, sid, raw[i].PICID); err != nil {
			return nil, numbered(i, err)
		}
		snaps = append(snaps, map[string]any{
			"id": sid, "title": f.title, "pic_id": raw[i].PICID, "due_date": dateValue(f.due),
		})
	}
	return snaps, nil
}

// numbered menyebut nomor sub task di galat validasi, supaya Manajer tahu baris mana yang perlu diperbaiki.
func numbered(i int, err error) error {
	var ve *apperr.Validation
	if errors.As(err, &ve) {
		return apperr.Invalid(fmt.Sprintf("Sub task nomor %d: %s", i+1, ve.Message))
	}
	return err
}

// MySubtasks: sub task di detail ini yang dipegang userID, dalam bentuk yang sama dengan ListMySubtasks
// supaya MoveOptions di detail dan di papan memakai hitungan yang sama.
func (d Detail) MySubtasks(userID int64) []sqlc.ListMySubtasksRow {
	out := []sqlc.ListMySubtasksRow{}
	for _, st := range d.Subtasks {
		for _, a := range st.Assignees {
			if a.ID == userID {
				out = append(out, sqlc.ListMySubtasksRow{TaskID: d.Task.ID, ID: st.ID, Title: st.Title, Done: st.DoneAt.Valid})
				break
			}
		}
	}
	return out
}

// IsSubtaskPIC menjawab apakah akun ini PIC salah satu sub task task ini (revisi 17).
func (d Detail) IsSubtaskPIC(userID int64) bool {
	for _, st := range d.Subtasks {
		for _, a := range st.Assignees {
			if a.ID == userID {
				return true
			}
		}
	}
	return false
}
