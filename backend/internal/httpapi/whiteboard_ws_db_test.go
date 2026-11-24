package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// dialWb membuka WebSocket Whiteboard dengan cookie klien itu. origin kosong berarti tanpa header Origin.
func (s liveServer) dialWb(t *testing.T, c *http.Client, origin string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	h := http.Header{}
	if origin != "" {
		h.Set("Origin", origin)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, res, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(s.srv.URL, "http")+"/api/whiteboard/ws",
		&websocket.DialOptions{HTTPClient: c, HTTPHeader: h})
	if conn != nil {
		t.Cleanup(func() { conn.CloseNow() })
	}
	return conn, res, err
}

func wsSend(t *testing.T, conn *websocket.Conn, v any) {
	t.Helper()
	b, _ := json.Marshal(v)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, b); err != nil {
		t.Fatal(err)
	}
}

// wsWait membaca pesan sampai menemukan jenis t, atau gagal setelah batas waktu.
func wsWait(t *testing.T, conn *websocket.Conn, typ string) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		_, b, err := conn.Read(ctx)
		if err != nil {
			t.Fatalf("menunggu %s: %v", typ, err)
		}
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		if m["t"] == typ {
			return m
		}
	}
}

// Handshake: Origin asing dan tanpa Origin ditolak 403, tanpa cookie 401; koneksi sah menerima kehadiran.
func TestWhiteboardWSHandshake(t *testing.T) {
	tm := newTaskTeam(t)
	if _, res, err := tm.dialWb(t, tm.pic, "https://situs-lain.test"); err == nil || res == nil || res.StatusCode != http.StatusForbidden {
		t.Fatalf("Origin asing: %v %v", res, err)
	}
	if _, res, err := tm.dialWb(t, tm.pic, ""); err == nil || res == nil || res.StatusCode != http.StatusForbidden {
		t.Fatalf("tanpa Origin: %v %v", res, err)
	}
	if _, res, err := tm.dialWb(t, &http.Client{}, tm.srv.URL); err == nil || res == nil || res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("tanpa cookie: %v %v", res, err)
	}
	conn, _, err := tm.dialWb(t, tm.pic, tm.srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	wsSend(t, conn, map[string]any{"t": "join", "board": "root", "tab_id": "a"})
	if p := wsWait(t, conn, "presence"); len(p["people"].([]any)) != 1 {
		t.Fatalf("kehadiran %v", p)
	}
	// Admin terhubung tetapi tulisannya ditolak.
	adm, _, err := tm.dialWb(t, tm.admin, tm.srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	wsSend(t, adm, map[string]any{"t": "join", "board": "root"})
	wsSend(t, adm, map[string]any{"t": "select", "ids": []string{}})
	if d := wsWait(t, adm, "denied"); d["for"] != "select" {
		t.Fatalf("penolakan Admin %v", d)
	}
}

// Perubahan lewat REST disiarkan ke koneksi lain setelah commit; perubahan yang ditolak tidak disiarkan.
func TestWhiteboardWSSiaranSetelahCommit(t *testing.T) {
	tm := newTaskTeam(t)
	watch, _, err := tm.dialWb(t, tm.other, tm.srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	wsSend(t, watch, map[string]any{"t": "join", "board": "root"})
	wsWait(t, watch, "joined")
	var out struct {
		Card struct {
			ID string `json:"id"`
		} `json:"card"`
	}
	if code := tm.sendJSON(t, tm.pic, http.MethodPost, "/api/whiteboard/boards/root/cards", `{"kind":"heading","x":0,"y":0,"w":320,"h":64,"content":{"text":"Siaran","level":1}}`, &out); code != http.StatusCreated {
		t.Fatalf("buat kartu %d", code)
	}
	ch := wsWait(t, watch, "changed")
	if !strings.Contains(mustJSON(ch["cards"]), out.Card.ID) {
		t.Fatalf("siaran %v", ch)
	}
	// Ditolak 409 (versi isi basi) dan 403 (Admin): tidak ada siaran. Lalu satu perubahan sah sebagai penanda.
	tm.rawSend(t, tm.pic, http.MethodPost, "/api/whiteboard/boards/root/cards/patch", `{"cards":[{"id":"`+out.Card.ID+`","content":{"text":"x","level":1},"base_content_version":99}]}`)
	tm.rawSend(t, tm.admin, http.MethodPost, "/api/whiteboard/boards/root/cards/patch", `{"cards":[{"id":"`+out.Card.ID+`","x":5}]}`)
	tm.rawSend(t, tm.pic, http.MethodPost, "/api/whiteboard/boards/root/cards/patch", `{"cards":[{"id":"`+out.Card.ID+`","x":7}]}`)
	next := wsWait(t, watch, "changed")
	if !strings.Contains(mustJSON(next["cards"]), `"x":7`) {
		t.Fatalf("siaran berikutnya bukan perubahan sah: %v", next)
	}
	// Hapus disiarkan dengan nama penghapus dan batch.
	tm.rawSend(t, tm.pic, http.MethodPost, "/api/whiteboard/boards/root/cards/delete", `{"ids":["`+out.Card.ID+`"]}`)
	if d := wsWait(t, watch, "deleted"); d["by"] != "staf.a" || d["batch"] == "" {
		t.Fatalf("siaran hapus %v", d)
	}
}

// Sesi yang dicabut (logout) menutup koneksi dengan kode 4401 pada pemeriksaan ulang berikutnya.
func TestWhiteboardWSSesiDicabut(t *testing.T) {
	old := wsRecheckEvery
	wsRecheckEvery = 200 * time.Millisecond
	defer func() { wsRecheckEvery = old }()
	tm := newTaskTeam(t)
	conn, _, err := tm.dialWb(t, tm.pic, tm.srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	wsSend(t, conn, map[string]any{"t": "join", "board": "root"})
	wsWait(t, conn, "joined")
	if code := tm.call(t, tm.pic, http.MethodPost, "/api/auth/logout", ""); code >= 300 {
		t.Fatalf("logout %d", code)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		_, _, err := conn.Read(ctx)
		if err == nil {
			continue
		}
		var ce websocket.CloseError
		if !errors.As(err, &ce) || ce.Code != CloseSession {
			t.Fatalf("koneksi berakhir dengan %v, mau 4401", err)
		}
		return
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
