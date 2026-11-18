package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/task"
)

// archive menerima q (kata kunci), scope (mine/all), pic, division, label, urgency, from, to (tanggal selesai,
// YYYY-MM-DD), sort (newest/oldest/urgency), dan page. Seperti papan, scope=mine mengabaikan pic.
func (s *Server) archive(c *gin.Context) {
	f := task.ArchiveFilter{
		Keyword: c.Query("q"), Urgency: c.Query("urgency"), Sort: c.Query("sort"),
		SolvedFrom: c.Query("from"), SolvedTo: c.Query("to"), Page: 1,
	}
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
	if raw := c.Query("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		// Batas atas menjaga offset tetap muat di int32.
		if err != nil || n < 1 || n > 100000 {
			writeError(c, http.StatusBadRequest, CodeBadRequest, "Nomor halaman tidak valid.")
			return
		}
		f.Page = n
	}

	res, err := s.tasks.Archive(c.Request.Context(), f)
	if err != nil {
		s.writeTaskError(c, err)
		return
	}
	actor := taskActor(c)
	subPIC, err := s.subtaskPICTasks(c, actor, res.Tasks)
	if err != nil {
		s.internalError(c, err)
		return
	}
	items := make([]boardTask, 0, len(res.Tasks))
	for _, r := range res.Tasks {
		items = append(items, withMoveTo(actor, r, subPIC[r.Task.ID]))
	}
	c.JSON(http.StatusOK, gin.H{
		"tasks": items, "total": res.Total, "page": res.Page, "page_size": task.ArchivePageSize,
	})
}

// withMoveTo memakai aturan hak yang sama dengan papan, supaya tombol "Buka kembali" di Arsip hanya
// muncul untuk yang memang boleh.
func withMoveTo(actor task.Actor, r task.Summary, isSubtaskPIC bool) boardTask {
	isPIC := false
	for _, a := range r.Assignees {
		if a.ID == actor.ID {
			isPIC = true
			break
		}
	}
	can := task.PermissionsFor(actor.Role, isPIC, isSubtaskPIC, task.Status(r.Task.Status))
	return boardTask{taskSummary: summaryOf(r), MoveTo: can.MoveTo}
}

// subtaskPICTasks: hanya Staf yang butuh jawaban ini (Manajer sudah boleh semua, Admin tidak boleh apa pun),
// jadi peran lain tidak membayar query tambahan.
func (s *Server) subtaskPICTasks(c *gin.Context, actor task.Actor, list []task.Summary) (map[string]bool, error) {
	if actor.Role != task.RoleStaff {
		return map[string]bool{}, nil
	}
	ids := make([]string, len(list))
	for i, r := range list {
		ids[i] = r.Task.ID
	}
	return s.tasks.SubtaskPICTasks(c.Request.Context(), actor.ID, ids)
}
