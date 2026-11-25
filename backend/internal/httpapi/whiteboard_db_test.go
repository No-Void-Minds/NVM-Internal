package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// rawSend mengirim permintaan dan mengembalikan status beserta body apa adanya, termasuk body galat.
func (s liveServer) rawSend(t *testing.T, c *http.Client, method, path, body string) (int, string) {
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
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b)
}

// Whiteboard: Staf dan Manajer menulis, Admin hanya membaca di setiap rute tulis. can_write mengikuti aturan yang
// sama.
func TestWhiteboardAdminHanyaMembaca(t *testing.T) {
	tm := newTaskTeam(t)
	var boardCard struct {
		Card struct {
			Ref struct {
				ID string `json:"id"`
			} `json:"ref"`
		} `json:"card"`
	}
	if code := tm.sendJSON(t, tm.pic, http.MethodPost, "/api/whiteboard/boards/root/cards", `{"kind":"board","title":"Moodboard","x":0,"y":0,"w":200,"h":120}`, &boardCard); code != http.StatusCreated {
		t.Fatalf("Staf membuat papan: %d", code)
	}
	board := boardCard.Card.Ref.ID
	var card struct {
		Card struct {
			ID string `json:"id"`
		} `json:"card"`
	}
	if code := tm.sendJSON(t, tm.manager, http.MethodPost, "/api/whiteboard/boards/"+board+"/cards", `{"kind":"note","x":0,"y":0,"w":240,"h":160}`, &card); code != http.StatusCreated {
		t.Fatalf("Manajer membuat kartu: %d", code)
	}

	for _, c := range []struct {
		client *http.Client
		write  bool
	}{{tm.admin, false}, {tm.pic, true}, {tm.manager, true}} {
		var got struct {
			CanWrite bool `json:"can_write"`
		}
		for _, path := range []string{"/api/whiteboard/boards/root", "/api/whiteboard/boards/" + board, "/api/whiteboard/trash"} {
			if code := tm.sendJSON(t, c.client, http.MethodGet, path, "", &got); code != http.StatusOK || got.CanWrite != c.write {
				t.Fatalf("GET %s: %d can_write %v, mau %v", path, code, got.CanWrite, c.write)
			}
		}
	}

	writes := []struct{ method, path, body string }{
		{http.MethodPut, "/api/whiteboard/boards/" + board, `{"title":"x","version":1}`},
		{http.MethodPut, "/api/whiteboard/boards/root", `{"title":"x","version":1}`},
		{http.MethodPost, "/api/whiteboard/boards/root/cards", `{"kind":"board","title":"x","x":0,"y":0,"w":200,"h":120}`},
		{http.MethodPost, "/api/whiteboard/boards/" + board + "/cards", `{"kind":"note","x":0,"y":0,"w":240,"h":160}`},
		{http.MethodPost, "/api/whiteboard/boards/" + board + "/cards/patch", `{"cards":[{"id":"` + card.Card.ID + `","x":5}]}`},
		{http.MethodPost, "/api/whiteboard/boards/" + board + "/cards/delete", `{"ids":["` + card.Card.ID + `"]}`},
		{http.MethodPost, "/api/whiteboard/boards/" + board + "/cards/" + card.Card.ID + "/checklist", `{"op":"title","text":"x"}`},
		{http.MethodPost, "/api/whiteboard/trash/" + board + "/restore", ""},
		{http.MethodPost, "/api/whiteboard/boards/" + board + "/files?x=0&y=0&w=240&h=160", ""},
		{http.MethodPost, "/api/whiteboard/preview", `{"url":"https://contoh.test/"}`},
		{http.MethodPost, "/api/whiteboard/boards/" + board + "/cards/" + card.Card.ID + "/table", `{"op":"header","on":true}`},
		{http.MethodPost, "/api/whiteboard/boards/" + board + "/move", `{"ids":["` + card.Card.ID + `"],"to":"root"}`},
		{http.MethodPost, "/api/whiteboard/boards/" + board + "/connectors", `{"from":"` + card.Card.ID + `","to":"` + card.Card.ID + `"}`},
		{http.MethodPut, "/api/whiteboard/boards/" + board + "/connectors/" + card.Card.ID, `{"label":"x"}`},
		{http.MethodDelete, "/api/whiteboard/boards/" + board + "/connectors/" + card.Card.ID, ""},
		{http.MethodPost, "/api/whiteboard/boards/" + board + "/cards/" + card.Card.ID + "/comments", `{"body":"x"}`},
		{http.MethodPut, "/api/whiteboard/comments/" + card.Card.ID, `{"body":"x"}`},
		{http.MethodDelete, "/api/whiteboard/comments/" + card.Card.ID, ""},
	}
	for _, w := range writes {
		code, body := tm.rawSend(t, tm.admin, w.method, w.path, w.body)
		if code != http.StatusForbidden || !strings.Contains(body, "Admin hanya bisa melihat Whiteboard") {
			t.Fatalf("Admin %s %s: %d %s", w.method, w.path, code, body)
		}
	}
	// Tulisan Admin yang ditolak tidak mengubah apa pun.
	var view struct {
		Board struct {
			Title string `json:"title"`
			Cards []struct {
				X float64 `json:"x"`
			} `json:"cards"`
		} `json:"board"`
	}
	if code := tm.sendJSON(t, tm.other, http.MethodGet, "/api/whiteboard/boards/"+board, "", &view); code != http.StatusOK ||
		view.Board.Title != "Moodboard" || len(view.Board.Cards) != 1 || view.Board.Cards[0].X != 0 {
		t.Fatalf("papan setelah percobaan Admin: %d %+v", code, view)
	}
}

// Isi basi dijawab 409 beserta kartu terbaru; id acak dan bukan UUID dijawab 404, bukan 500.
func TestWhiteboardKonflikDanTidakAda(t *testing.T) {
	tm := newTaskTeam(t)
	board := "root"
	var card struct {
		Card struct {
			ID string `json:"id"`
		} `json:"card"`
	}
	tm.sendJSON(t, tm.pic, http.MethodPost, "/api/whiteboard/boards/"+board+"/cards", `{"kind":"heading","x":0,"y":0,"w":320,"h":64,"content":{"text":"Awal","level":1}}`, &card)
	patch := func(c *http.Client, text string) (int, string) {
		return tm.rawSend(t, c, http.MethodPost, "/api/whiteboard/boards/"+board+"/cards/patch",
			`{"cards":[{"id":"`+card.Card.ID+`","content":{"text":"`+text+`","level":1},"base_content_version":1}]}`)
	}
	if code, body := patch(tm.pic, "Versi staf"); code != http.StatusOK {
		t.Fatalf("edit pertama: %d %s", code, body)
	}
	code, body := patch(tm.other, "Versi lain")
	var conflict struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
		Card struct {
			ContentVersion int             `json:"content_version"`
			Content        json.RawMessage `json:"content"`
		} `json:"card"`
	}
	if err := json.Unmarshal([]byte(body), &conflict); err != nil || code != http.StatusConflict || conflict.Error.Code != CodeConflict ||
		conflict.Card.ContentVersion != 2 || !strings.Contains(string(conflict.Card.Content), "Versi staf") {
		t.Fatalf("konflik: %d %s", code, body)
	}

	for _, path := range []string{"/api/whiteboard/boards/bukan-uuid", "/api/whiteboard/boards/00000000-0000-4000-8000-000000000000",
		"/api/whiteboard/boards/" + board + "x"} {
		if code, body := tm.rawSend(t, tm.pic, http.MethodGet, path, ""); code != http.StatusNotFound {
			t.Fatalf("GET %s: %d %s", path, code, body)
		}
	}
	if code, body := tm.rawSend(t, tm.pic, http.MethodPost, "/api/whiteboard/trash/bukan-uuid/restore", ""); code != http.StatusNotFound {
		t.Fatalf("pulihkan batch acak: %d %s", code, body)
	}
}
