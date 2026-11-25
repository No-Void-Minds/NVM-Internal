package httpapi

import (
	"fmt"
	"net/http"
	"testing"
)

// Revisi 45: "Task saya" (papan scope=mine dan hitungan kartu profil) juga memuat task yang diminta pengguna,
// tanpa mengubah hak aksesnya.
func TestTaskSayaMemuatPeminta(t *testing.T) {
	tm := newTaskTeam(t)
	var otherID int64
	if err := tm.pool.QueryRow(t.Context(), `SELECT id FROM users WHERE username = 'staf.b'`).Scan(&otherID); err != nil {
		t.Fatal(err)
	}
	create := func(c *http.Client, title string, pics ...int64) string {
		t.Helper()
		ids := ""
		for i, p := range pics {
			if i > 0 {
				ids += ","
			}
			ids += fmt.Sprint(p)
		}
		var out struct {
			ID string `json:"id"`
		}
		body := fmt.Sprintf(`{"title":%q,"urgency":"medium","pic_ids":[%s]}`, title, ids)
		if code := tm.sendJSON(t, c, http.MethodPost, "/api/tasks", body, &out); code != http.StatusCreated {
			t.Fatalf("membuat %s: status %d", title, code)
		}
		return out.ID
	}
	// staf.b (peminta) membuat task dengan PIC staf.a; Manajer membuat task yang tidak melibatkan staf.b; staf.b
	// membuat task yang dia pegang sendiri juga.
	asked := create(tm.other, "Desain banner", tm.picID)
	unrelated := create(tm.manager, "Foto katalog", tm.picID)
	both := create(tm.other, "Live Shopee", otherID, tm.picID)
	byManager := create(tm.manager, "Brief Ramadan", tm.picID)

	type card struct {
		ID            string   `json:"id"`
		MoveTo        []string `json:"move_to"`
		RequesterOnly bool     `json:"requester_only"`
	}
	mine := func(c *http.Client) map[string][]card {
		t.Helper()
		var out struct {
			Columns []struct {
				Tasks []card `json:"tasks"`
			} `json:"columns"`
		}
		if code := tm.sendJSON(t, c, http.MethodGet, "/api/board?scope=mine", "", &out); code != http.StatusOK {
			t.Fatalf("scope=mine: %d", code)
		}
		got := map[string][]card{}
		for _, col := range out.Columns {
			for _, k := range col.Tasks {
				got[k.ID] = append(got[k.ID], k)
			}
		}
		return got
	}

	got := mine(tm.other)
	if len(got[asked]) != 1 || !got[asked][0].RequesterOnly || len(got[asked][0].MoveTo) != 0 {
		t.Fatalf("task yang diminta staf.b: %+v", got[asked])
	}
	if _, ok := got[unrelated]; ok {
		t.Fatal("task yang tidak melibatkan staf.b ikut muncul")
	}
	if len(got[both]) != 1 || got[both][0].RequesterOnly {
		t.Fatalf("task yang diminta sekaligus dipegang staf.b: %+v", got[both])
	}
	if len(got) != 2 {
		t.Fatalf("Task saya staf.b berisi %d task, mau 2", len(got))
	}

	// Hak akses tidak berubah: peminta Staf tetap tidak bisa memindahkan task yang bukan miliknya.
	if code, _ := tm.move(t, tm.other, asked, 1, "on_progress"); code != http.StatusForbidden {
		t.Fatalf("peminta Staf memindahkan task: %d, mau 403", code)
	}

	var counts struct {
		Counts map[string]int `json:"counts"`
	}
	if code := tm.sendJSON(t, tm.other, http.MethodGet, "/api/me/task-counts", "", &counts); code != http.StatusOK || counts.Counts["requested"] != 2 {
		t.Fatalf("hitungan profil staf.b: %d %+v, mau requested 2", code, counts.Counts)
	}

	// Manajer juga: task yang dia minta muncul di Task saya miliknya dengan penanda peminta.
	gm := mine(tm.manager)
	if len(gm[byManager]) != 1 || !gm[byManager][0].RequesterOnly {
		t.Fatalf("Task saya Manajer: %+v", gm[byManager])
	}
	if _, ok := gm[asked]; ok {
		t.Fatal("task staf.b ikut muncul di Task saya Manajer")
	}

	// PIC task tidak mendapat penanda peminta; filter Orang tidak memuat task yang hanya diminta.
	if k := mine(tm.pic)[asked]; len(k) != 1 || k[0].RequesterOnly {
		t.Fatalf("PIC task: %+v", k)
	}
	if _, ok := tm.board(t, tm.manager, "?pic=staf.b")[asked]; ok {
		t.Fatal("filter Orang staf.b memuat task yang hanya dia minta")
	}
}
