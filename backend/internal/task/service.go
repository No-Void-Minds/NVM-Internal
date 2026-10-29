package task

import (
	"context"
	"errors"
	"net/netip"
	"regexp"

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
	Title          string
	Description    string
	Urgency        string
	PICIDs         []int64
	LabelIDs       []int64
	StartDate      string
	DueDate        string
	DeliverableURL string
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// Create membuat task baru berstatus Requested. Hanya Manajer (BR-01); divisi task mengikuti PIC pertama.
func (s *Service) Create(ctx context.Context, actor Actor, in CreateInput) (string, error) {
	if actor.Role != RoleManager {
		return "", ErrForbidden
	}
	f, err := cleanFields(in.Title, in.Description, in.Urgency, in.StartDate, in.DueDate, in.DeliverableURL)
	if err != nil {
		return "", err
	}
	pics := uniqueIDs(in.PICIDs)
	if len(pics) == 0 {
		return "", apperr.Invalid("Pilih minimal satu PIC.")
	}
	labels := uniqueIDs(in.LabelIDs)

	var id string
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)

		divisionOf, err := eligiblePICs(ctx, q, pics)
		if err != nil {
			return err
		}
		if err := requireActiveLabels(ctx, q, labels); err != nil {
			return err
		}

		id, err = q.CreateTask(ctx, sqlc.CreateTaskParams{
			Title:          f.title,
			Description:    f.description,
			DivisionID:     divisionOf[pics[0]],
			RequestedBy:    actor.ID,
			Urgency:        f.urgency,
			StartDate:      f.startDate,
			DueDate:        f.dueDate,
			DeliverableUrl: f.deliverableURL,
		})
		if err != nil {
			return err
		}
		for _, pic := range pics {
			if err := q.AddTaskAssignee(ctx, sqlc.AddTaskAssigneeParams{TaskID: id, UserID: pic, AssignedBy: actor.ID}); err != nil {
				return err
			}
		}
		for _, l := range labels {
			if err := q.AddTaskLabel(ctx, sqlc.AddTaskLabelParams{TaskID: id, LabelID: l}); err != nil {
				return err
			}
		}
		return audit.Change(ctx, q, actor.auditActor(), "task_created", "task", id, nil, map[string]any{
			"title": f.title, "urgency": f.urgency, "assignee_ids": pics, "label_ids": labels,
		})
	})
	return id, err
}

type Detail struct {
	Task      sqlc.GetTaskDetailRow
	Assignees []sqlc.ListTaskAssigneesRow
	Labels    []sqlc.ListTaskLabelsRow
	History   []sqlc.ListTaskHistoryRow
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
	if d.Labels, err = q.ListTaskLabels(ctx, ids); err != nil {
		return Detail{}, err
	}
	if d.History, err = q.ListTaskHistory(ctx, id); err != nil {
		return Detail{}, err
	}
	return d, nil
}

type Summary struct {
	Task      sqlc.ListTasksRow
	Assignees []sqlc.ListTaskAssigneesRow
	Labels    []sqlc.ListTaskLabelsRow
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
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	// PIC dan label diambil sekaligus untuk semua task, bukan satu query per task.
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
		out[i] = Summary{Task: r, Assignees: []sqlc.ListTaskAssigneesRow{}, Labels: []sqlc.ListTaskLabelsRow{}}
		byTask[r.ID] = &out[i]
	}
	for _, a := range assignees {
		byTask[a.TaskID].Assignees = append(byTask[a.TaskID].Assignees, a)
	}
	for _, l := range labels {
		byTask[l.TaskID].Labels = append(byTask[l.TaskID].Labels, l)
	}
	return out, nil
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
