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
	items := make([]boardTask, 0, len(res.Tasks))
	for _, r := range res.Tasks {
		items = append(items, withMoveTo(actor, r))
	}
	c.JSON(http.StatusOK, gin.H{
		"tasks": items, "total": res.Total, "page": res.Page, "page_size": task.ArchivePageSize,
	})
}

// withMoveTo memakai aturan hak yang sama dengan papan, supaya tombol "Buka kembali" di Arsip hanya
// muncul untuk yang memang boleh.
func withMoveTo(actor task.Actor, r task.Summary) boardTask {
	isPIC := false
	for _, a := range r.Assignees {
		if a.ID == actor.ID {
			isPIC = true
			break
		}
	}
	can := task.PermissionsFor(actor.Role, isPIC, task.Status(r.Task.Status))
	return boardTask{taskSummary: summaryOf(r), MoveTo: can.MoveTo}
}
