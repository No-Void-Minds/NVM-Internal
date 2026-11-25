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

// Gambar bebas dan tabel disiarkan seperti kartu lain. Tabel dan gambar bebas yang dikunci orang lain ditolak 409
// dengan nama pemegang kunci, baik lewat op tabel maupun patch isi, dan pindah papan juga ditolak.
func TestWhiteboardWSTabelDanGambarBebas(t *testing.T) {
	tm := newTaskTeam(t)
	watch, _, err := tm.dialWb(t, tm.other, tm.srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	wsSend(t, watch, map[string]any{"t": "join", "board": "root"})
	wsWait(t, watch, "joined")
	var table, drawing struct {
		Card struct {
			ID             string `json:"id"`
			ContentVersion int32  `json:"content_version"`
		} `json:"card"`
	}
	if code := tm.sendJSON(t, tm.pic, http.MethodPost, "/api/whiteboard/boards/root/cards", `{"kind":"table","x":0,"y":0,"w":320,"h":160}`, &table); code != http.StatusCreated {
		t.Fatalf("buat tabel %d", code)
	}
	wsWait(t, watch, "changed")
	// Gambar bebas (revisi A): objek dibuat dari goresan pertama berkoordinat dunia, kotak batasnya dihitung server.
	if code, body := tm.rawSend(t, tm.pic, http.MethodPost, "/api/whiteboard/boards/root/drawings",
		`{"stroke":{"color":"tinta","width":4,"points":[100,300,200,400]}}`); code != http.StatusCreated {
		t.Fatalf("buat gambar %d %s", code, body)
	} else {
		_ = json.Unmarshal([]byte(body), &drawing)
	}
	if ch := wsWait(t, watch, "changed"); !strings.Contains(mustJSON(ch["cards"]), `"x":96`) {
		t.Fatalf("siaran gambar %v", ch)
	}
	drawOp := "/api/whiteboard/boards/root/cards/" + drawing.Card.ID + "/drawing"
	if code, body := tm.rawSend(t, tm.pic, http.MethodPost, drawOp, `{"op":"add","stroke":{"color":"tinta","width":5,"points":[0,0,1,1]}}`); code != http.StatusBadRequest || !strings.Contains(body, "Ketebalan") {
		t.Fatalf("goresan tidak sah %d %s", code, body)
	}
	if code, body := tm.rawSend(t, tm.pic, http.MethodPost, "/api/whiteboard/boards/root/cards/patch",
		`{"cards":[{"id":"`+drawing.Card.ID+`","content":{"strokes":[]},"base_content_version":1}]}`); code != http.StatusBadRequest {
		t.Fatalf("patch isi gambar %d %s", code, body)
	}
	// Goresan yang sedang digambar disiarkan langsung ke orang lain.
	pen, _, err := tm.dialWb(t, tm.pic, tm.srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	wsSend(t, pen, map[string]any{"t": "join", "board": "root"})
	wsWait(t, pen, "joined")
	wsSend(t, pen, map[string]any{"t": "ink", "points": []int{10, 10, 20, 25}, "color": "biru", "width": 4})
	if m := wsWait(t, watch, "ink"); mustJSON(m["points"]) != "[10,10,20,25]" {
		t.Fatalf("siaran goresan langsung %v", m)
	}

	for _, id := range []string{table.Card.ID, drawing.Card.ID} {
		wsSend(t, watch, map[string]any{"t": "lock", "card": id})
		wsWait(t, watch, "lock_granted")
	}
	tableOp := "/api/whiteboard/boards/root/cards/" + table.Card.ID + "/table"
	for _, req := range []struct{ path, body string }{
		{tableOp, `{"op":"set","row":0,"col":0,"text":"Harga"}`},
		{drawOp, `{"op":"color","color":"merah"}`},
		{"/api/whiteboard/boards/root/move", `{"ids":["` + table.Card.ID + `"],"to":"root"}`},
	} {
		if code, body := tm.rawSend(t, tm.pic, http.MethodPost, req.path, req.body); code != http.StatusConflict || !strings.Contains(body, "staf.b sedang mengubah") {
			t.Fatalf("%s saat dikunci orang lain: %d %s", req.path, code, body)
		}
	}
	// Pemegang kunci tetap bisa mengubah, dan hasilnya disiarkan.
	if code, body := tm.rawSend(t, tm.other, http.MethodPost, tableOp, `{"op":"set","row":0,"col":0,"text":"Harga"}`); code != http.StatusOK {
		t.Fatalf("pemegang kunci %d %s", code, body)
	}
	if ch := wsWait(t, watch, "changed"); !strings.Contains(mustJSON(ch["cards"]), "Harga") {
		t.Fatalf("siaran tabel %v", ch)
	}
}

// Garis dan pindah papan disiarkan: papan asal menerima moved (kartu dan garis yang terputus dibuang), papan
// tujuan diminta memuat ulang.
func TestWhiteboardWSGarisDanPindah(t *testing.T) {
	tm := newTaskTeam(t)
	type created struct {
		Card struct {
			ID  string `json:"id"`
			Ref struct {
				ID string `json:"id"`
			} `json:"ref"`
		} `json:"card"`
	}
	var child, a, b created
	tm.sendJSON(t, tm.pic, http.MethodPost, "/api/whiteboard/boards/root/cards", `{"kind":"board","title":"Anak","x":0,"y":0,"w":200,"h":120}`, &child)
	tm.sendJSON(t, tm.pic, http.MethodPost, "/api/whiteboard/boards/root/cards", `{"kind":"note","x":300,"y":0,"w":240,"h":160}`, &a)
	tm.sendJSON(t, tm.pic, http.MethodPost, "/api/whiteboard/boards/root/cards", `{"kind":"note","x":600,"y":0,"w":240,"h":160}`, &b)

	src, _, err := tm.dialWb(t, tm.other, tm.srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	wsSend(t, src, map[string]any{"t": "join", "board": "root"})
	wsWait(t, src, "joined")
	dst, _, err := tm.dialWb(t, tm.manager, tm.srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	wsSend(t, dst, map[string]any{"t": "join", "board": child.Card.Ref.ID})
	wsWait(t, dst, "joined")

	if code, body := tm.rawSend(t, tm.pic, http.MethodPost, "/api/whiteboard/boards/root/connectors", `{"from":"`+a.Card.ID+`","to":"`+b.Card.ID+`","label":"lalu"}`); code != http.StatusCreated {
		t.Fatalf("buat garis %d %s", code, body)
	}
	if m := wsWait(t, src, "connectors"); !strings.Contains(mustJSON(m["items"]), `"label":"lalu"`) {
		t.Fatalf("siaran garis %v", m)
	}
	if code, body := tm.rawSend(t, tm.pic, http.MethodPost, "/api/whiteboard/boards/root/move", `{"ids":["`+a.Card.ID+`"],"to":"`+child.Card.Ref.ID+`"}`); code != http.StatusOK {
		t.Fatalf("pindah %d %s", code, body)
	}
	m := wsWait(t, src, "moved")
	if mustJSON(m["ids"]) != `["`+a.Card.ID+`"]` || len(m["connectors_gone"].([]any)) != 1 {
		t.Fatalf("siaran pindah %v", m)
	}
	wsWait(t, dst, "resync")
}
