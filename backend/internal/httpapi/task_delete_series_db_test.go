package httpapi

import (
	"fmt"
	"net/http"
	"testing"
)

// Revisi 31 dan 32 lewat HTTP: Staf yang mengirim stop_series ditolak 403, Manajer berhasil dan seri berhenti;
// detail task mengirim stop_series di can, dan PIC task mendapat can_edit untuk sub task buatan orang lain.
func TestDeleteStopSeriesHTTP(t *testing.T) {
	tm := newTaskTeam(t)
	body := fmt.Sprintf(`{"title":"Unggah harian","urgency":"medium","pic_ids":[%d],"label_ids":[],
		"subtasks":[{"title":"Potong video","pic_id":0}],
		"schedule":{"freq":"daily","skip_weekend":false,"weekdays":[],"month_day":0,"end_date":""}}`, tm.picID)
	var created struct {
		ID string `json:"id"`
	}
	if code := tm.sendJSON(t, tm.manager, http.MethodPost, "/api/tasks", body, &created); code != http.StatusCreated {
		t.Fatalf("Manajer membuat task berulang: %d", code)
	}
	id := created.ID

	type detail struct {
		Task struct {
			Version int32 `json:"version"`
			Series  *struct {
				Active bool `json:"active"`
			} `json:"series"`
			Can struct {
				StopSeries bool `json:"stop_series"`
			} `json:"can"`
			Subtasks []struct {
				CanEdit bool `json:"can_edit"`
			} `json:"subtasks"`
		} `json:"task"`
	}
	get := func(c *http.Client) detail {
		var d detail
		if code := tm.sendJSON(t, c, http.MethodGet, "/api/tasks/"+id, "", &d); code != http.StatusOK {
			t.Fatalf("GET task: %d", code)
		}
		return d
	}
	pic := get(tm.pic)
	if pic.Task.Can.StopSeries || !get(tm.manager).Task.Can.StopSeries {
		t.Fatal("can.stop_series harus true hanya untuk Manajer")
	}
	if len(pic.Task.Subtasks) != 1 || !pic.Task.Subtasks[0].CanEdit {
		t.Fatalf("PIC task harus mendapat can_edit untuk sub task buatan Manajer: %+v", pic.Task.Subtasks)
	}
	if o := get(tm.other); len(o.Task.Subtasks) != 1 || o.Task.Subtasks[0].CanEdit {
		t.Fatal("Staf yang bukan PIC tidak boleh mendapat can_edit")
	}

	path := fmt.Sprintf("/api/tasks/%s?version=%d&stop_series=true", id, pic.Task.Version)
	for name, c := range map[string]*http.Client{"PIC task": tm.pic, "Staf lain": tm.other} {
		if code := tm.sendJSON(t, c, http.MethodDelete, path, "", nil); code != http.StatusForbidden {
			t.Fatalf("%s mengirim stop_series: %d, mau 403", name, code)
		}
	}
	if code := tm.sendJSON(t, tm.manager, http.MethodDelete, path, "", nil); code != http.StatusNoContent {
		t.Fatalf("Manajer hapus dan hentikan seri: %d, mau 204", code)
	}
	var active bool
	if err := tm.pool.QueryRow(t.Context(), `SELECT s.active FROM task_series s JOIN tasks t ON t.series_id = s.id WHERE t.id = $1`, id).Scan(&active); err != nil || active {
		t.Fatalf("seri harus berhenti: active=%v err %v", active, err)
	}
}
