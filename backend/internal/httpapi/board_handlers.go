package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/task"
)

// boardTask menambahkan tujuan perpindahan yang boleh bagi pengguna ini. Dihitung dengan aturan yang sama
// dengan detail task, supaya tombol di kartu tidak pernah menawarkan perpindahan yang akan ditolak.
type boardTask struct {
	taskSummary
	MoveTo []task.Status `json:"move_to"`
}

type boardColumn struct {
	Status string      `json:"status"`
	Count  int         `json:"count"`
	Tasks  []boardTask `json:"tasks"`
}

type person struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Username     string `json:"username"`
	Role         string `json:"role"`
	DivisionName string `json:"division_name"`
}

// board menerima scope (mine/all), pic (username, dipisah koma), division, label, urgency, dan due.
// scope=mine selalu memakai akun yang login dan mengabaikan pic, supaya URL yang membawa keduanya tidak ambigu.
func (s *Server) board(c *gin.Context) {
	f := task.BoardFilter{Urgency: c.Query("urgency"), Due: c.Query("due")}
	if c.Query("scope") == "mine" {
		f.PICIDs = []int64{currentUser(c).ID}
	} else if p := c.Query("pic"); p != "" {
		f.PICUsernames = strings.Split(p, ",")
	}
	var ok bool
	if f.DivisionID, ok = queryID(c, "division"); !ok {
		return
	}
	if f.LabelID, ok = queryID(c, "label"); !ok {
		return
	}

	cols, err := s.tasks.Board(c.Request.Context(), f, time.Now())
	if err != nil {
		s.writeTaskError(c, err)
		return
	}
	actor := taskActor(c)
	out := make([]boardColumn, 0, len(cols))
	for _, col := range cols {
		items := make([]boardTask, 0, len(col.Tasks))
		for _, r := range col.Tasks {
			isPIC := false
			for _, a := range r.Assignees {
				if a.ID == actor.ID {
					isPIC = true
					break
				}
			}
			can := task.PermissionsFor(actor.Role, isPIC, task.Status(r.Task.Status))
			items = append(items, boardTask{taskSummary: summaryOf(r), MoveTo: can.MoveTo})
		}
		out = append(out, boardColumn{Status: string(col.Status), Count: len(items), Tasks: items})
	}
	c.JSON(http.StatusOK, gin.H{"columns": out, "completed_days": task.CompletedDays})
}

func (s *Server) listPeople(c *gin.Context) {
	rows, err := s.tasks.People(c.Request.Context())
	if err != nil {
		s.internalError(c, err)
		return
	}
	out := make([]person, 0, len(rows))
	for _, r := range rows {
		out = append(out, person{ID: r.ID, Name: r.Name, Username: r.Username, Role: string(r.Role), DivisionName: r.DivisionName})
	}
	c.JSON(http.StatusOK, gin.H{"people": out})
}

// queryID membaca parameter angka opsional. Nilai yang bukan angka positif dibalas 400, bukan diabaikan,
// supaya filter yang salah tidak diam-diam menampilkan semua task.
func queryID(c *gin.Context, key string) (*int64, bool) {
	raw := c.Query(key)
	if raw == "" {
		return nil, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(c, http.StatusBadRequest, CodeBadRequest, "Filter "+key+" tidak valid.")
		return nil, false
	}
	return &id, true
}

func summaryOf(r task.Summary) taskSummary {
	t := r.Task
	return taskSummary{
		ID:         t.ID,
		Title:      t.Title,
		Status:     string(t.Status),
		Urgency:    string(t.Urgency),
		Division:   idName{ID: t.DivisionID, Name: t.DivisionName},
		StartDate:  dateString(t.StartDate),
		DueDate:    dateString(t.DueDate),
		SolvedDate: timePtr(t.SolvedDate),
		Version:    t.Version,
		Assignees:  people(r.Assignees),
		Labels:     labelsOf(r.Labels),
	}
}
