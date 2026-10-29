package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"nvm-internal/internal/testdb"
)

// taskTeam berisi Manajer, Admin, Staf PIC, dan Staf lain yang sudah login. Staf lain sengaja satu divisi
// dengan PIC supaya penolakan 403 terbukti datang dari aturan PIC, bukan dari beda divisi.
type taskTeam struct {
	liveServer
	manager, admin, pic, other *http.Client
	picID                      int64
}

func newTaskTeam(t *testing.T) taskTeam {
	t.Helper()
	s := startServer(t)
	testdb.User(t, s.pool, "manajer.a", "manager", s.division, true)
	testdb.User(t, s.pool, "admin.a", "admin", s.division, true)
	picID := testdb.User(t, s.pool, "staf.a", "staff", s.division, true)
	testdb.User(t, s.pool, "staf.b", "staff", s.division, true)
	return taskTeam{
		liveServer: s,
		manager:    s.login(t, "manajer.a", testdb.Password),
		admin:      s.login(t, "admin.a", testdb.Password),
		pic:        s.login(t, "staf.a", testdb.Password),
		other:      s.login(t, "staf.b", testdb.Password),
		picID:      picID,
	}
}

// sendJSON sama dengan call, tetapi ikut membaca body respons ke out. Body hanya dibaca untuk status 2xx
// karena bentuk galat sudah diuji di tempat lain.
func (s liveServer) sendJSON(t *testing.T, c *http.Client, method, path, body string, out any) int {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, s.srv.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", s.srv.URL)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if out != nil && res.StatusCode < 300 {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			t.Fatalf("%s %s: body tidak bisa dibaca: %v", method, path, err)
		}
	}
	return res.StatusCode
}

// taskView cukup untuk respons detail task maupun respons move, karena nama field-nya sama.
type taskView struct {
	Status     string  `json:"status"`
	Version    int32   `json:"version"`
	SolvedDate *string `json:"solved_date"`
	Labels     []struct {
		ID       int64 `json:"id"`
		IsActive bool  `json:"is_active"`
	} `json:"labels"`
}

func (tm taskTeam) createTask(t *testing.T, labelIDs ...int64) string {
	t.Helper()
	if labelIDs == nil {
		labelIDs = []int64{}
	}
	ids, _ := json.Marshal(labelIDs)
	body := fmt.Sprintf(`{"title":"Foto katalog","urgency":"high","pic_ids":[%d],"label_ids":%s}`, tm.picID, ids)
	var out struct {
		ID string `json:"id"`
	}
	if code := tm.sendJSON(t, tm.manager, http.MethodPost, "/api/tasks", body, &out); code != http.StatusCreated {
		t.Fatalf("Manajer membuat task: status %d, mau 201", code)
	}
	return out.ID
}

func (tm taskTeam) get(t *testing.T, c *http.Client, id string) taskView {
	t.Helper()
	var out struct {
		Task taskView `json:"task"`
	}
	if code := tm.sendJSON(t, c, http.MethodGet, "/api/tasks/"+id, "", &out); code != http.StatusOK {
		t.Fatalf("GET task %s: status %d, mau 200", id, code)
	}
	return out.Task
}

func (tm taskTeam) move(t *testing.T, c *http.Client, id string, version int32, to string) (int, taskView) {
	t.Helper()
	var out taskView
	body := fmt.Sprintf(`{"version":%d,"to":%q}`, version, to)
	return tm.sendJSON(t, c, http.MethodPost, "/api/tasks/"+id+"/move", body, &out), out
}

// AC-04 bagian task: Staf tidak bisa membuat, mengubah, menghapus, atau menutup task, dan Admin tidak
// bisa menyentuh task sama sekali, walaupun dipanggil langsung ke API.
func TestTaskActionsRejectedByRole(t *testing.T) {
	tm := newTaskTeam(t)
	id := tm.createTask(t)
	path := "/api/tasks/" + id
	create := fmt.Sprintf(`{"title":"Coba","urgency":"low","pic_ids":[%d]}`, tm.picID)
	update := fmt.Sprintf(`{"version":1,"title":"Diubah","urgency":"low","pic_ids":[%d]}`, tm.picID)
	labels := `{"version":1,"label_ids":[]}`
	deliverable := `{"version":1,"deliverable_url":"https://contoh.test/hasil"}`
	start := `{"version":1,"to":"on_progress"}`

	calls := []struct {
		who                string
		c                  *http.Client
		method, path, body string
	}{
		{"Staf PIC", tm.pic, http.MethodPost, "/api/tasks", create},
		{"Staf PIC", tm.pic, http.MethodPut, path, update},
		{"Staf PIC", tm.pic, http.MethodDelete, path + "?version=1", ""},
		{"Staf lain", tm.other, http.MethodPut, path + "/labels", labels},
		{"Staf lain", tm.other, http.MethodPut, path + "/deliverable", deliverable},
		{"Staf lain", tm.other, http.MethodPost, path + "/move", start},
		{"Admin", tm.admin, http.MethodPost, "/api/tasks", create},
		{"Admin", tm.admin, http.MethodPut, path, update},
		{"Admin", tm.admin, http.MethodDelete, path + "?version=1", ""},
		{"Admin", tm.admin, http.MethodPut, path + "/labels", labels},
		{"Admin", tm.admin, http.MethodPost, path + "/move", start},
	}
	for _, c := range calls {
		if got := tm.call(t, c.c, c.method, c.path, c.body); got != http.StatusForbidden {
			t.Errorf("%s: %s %s status %d, mau 403", c.who, c.method, c.path, got)
		}
	}

	// Staf tetap boleh membaca task orang lain (S3-07), dan semua penolakan di atas tidak mengubah apa pun.
	if v := tm.get(t, tm.other, id); v.Version != 1 || v.Status != "requested" {
		t.Fatalf("task berubah setelah ditolak: versi %d, status %s", v.Version, v.Status)
	}

	// Staf PIC boleh membawa task sampai Ready, tetapi tidak boleh menutupnya.
	v := int32(1)
	for _, to := range []string{"on_progress", "ready"} {
		code, r := tm.move(t, tm.pic, id, v, to)
		if code != http.StatusOK {
			t.Fatalf("Staf PIC ke %s: status %d, mau 200", to, code)
		}
		v = r.Version
	}
	if code, _ := tm.move(t, tm.pic, id, v, "completed"); code != http.StatusForbidden {
		t.Fatalf("Staf PIC menutup task: status %d, mau 403", code)
	}
}

// AC-08 bagian backend: Completed mengisi solved_date, dan membuka kembali mengosongkannya.
// Hilang dari papan aktif dan muncul di arsip diuji di Sprint 4 dan 6.
func TestCompletedSetsSolvedDate(t *testing.T) {
	tm := newTaskTeam(t)
	id := tm.createTask(t)
	v := int32(1)
	steps := []struct {
		c  *http.Client
		to string
	}{{tm.pic, "on_progress"}, {tm.pic, "ready"}, {tm.manager, "completed"}}
	for _, st := range steps {
		code, r := tm.move(t, st.c, id, v, st.to)
		if code != http.StatusOK {
			t.Fatalf("pindah ke %s: status %d, mau 200", st.to, code)
		}
		v = r.Version
	}

	if done := tm.get(t, tm.manager, id); done.Status != "completed" || done.SolvedDate == nil {
		t.Fatalf("setelah Completed: status %s, solved_date kosong %t", done.Status, done.SolvedDate == nil)
	}

	code, r := tm.move(t, tm.manager, id, v, "on_progress")
	if code != http.StatusOK {
		t.Fatalf("buka kembali: status %d, mau 200", code)
	}
	if r.SolvedDate != nil {
		t.Fatal("buka kembali harus mengosongkan solved_date")
	}
}

// AC-14 bagian backend: Manajer membuat label, memasang beberapa label di satu task, lalu menonaktifkan
// salah satunya. Label nonaktif tetap menempel di task lama, tetapi ditolak untuk task baru.
// Filter papan per label diuji di Sprint 4.
func TestLabelsEndToEnd(t *testing.T) {
	tm := newTaskTeam(t)
	newLabel := func(name, color string) int64 {
		t.Helper()
		var out struct {
			ID int64 `json:"id"`
		}
		body := fmt.Sprintf(`{"name":%q,"color":%q,"division_id":null}`, name, color)
		if code := tm.sendJSON(t, tm.manager, http.MethodPost, "/api/labels", body, &out); code != http.StatusCreated {
			t.Fatalf("membuat label %s: status %d, mau 201", name, code)
		}
		return out.ID
	}
	tiktok := newLabel("TikTok", "#D3E2EC")
	ads := newLabel("Ads", "#F3E5BF")

	id := tm.createTask(t, tiktok, ads)
	if got := tm.get(t, tm.manager, id).Labels; len(got) != 2 {
		t.Fatalf("task punya %d label, mau 2", len(got))
	}

	deactivate := fmt.Sprintf("/api/labels/%d/deactivate", tiktok)
	if code := tm.call(t, tm.manager, http.MethodPost, deactivate, ""); code != http.StatusNoContent {
		t.Fatalf("menonaktifkan label: status %d, mau 204", code)
	}

	old := tm.get(t, tm.manager, id).Labels
	if len(old) != 2 {
		t.Fatalf("task lama kehilangan label: tersisa %d, mau 2", len(old))
	}
	for _, l := range old {
		if l.ID == tiktok && l.IsActive {
			t.Fatal("label TikTok di task lama harus bertanda nonaktif")
		}
	}

	body := fmt.Sprintf(`{"title":"Video baru","urgency":"low","pic_ids":[%d],"label_ids":[%d]}`, tm.picID, tiktok)
	if code := tm.call(t, tm.manager, http.MethodPost, "/api/tasks", body); code != http.StatusBadRequest {
		t.Fatalf("task baru dengan label nonaktif: status %d, mau 400", code)
	}
}
