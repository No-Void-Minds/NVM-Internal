package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// dialTasks membuka WebSocket papan task dengan cookie klien itu.
func (s liveServer) dialTasks(t *testing.T, c *http.Client, origin string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	h := http.Header{}
	if origin != "" {
		h.Set("Origin", origin)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, res, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(s.srv.URL, "http")+"/api/tasks/ws",
		&websocket.DialOptions{HTTPClient: c, HTTPHeader: h})
	if conn != nil {
		t.Cleanup(func() { conn.CloseNow() })
	}
	return conn, res, err
}

// events membaca koneksi di goroutine sendiri. Read dengan batas waktu tidak dipakai langsung karena coder/websocket
// menutup koneksi saat konteks Read habis, padahal test perlu memeriksa "tidak ada event" lalu membaca lagi.
func events(t *testing.T, conn *websocket.Conn) chan map[string]any {
	t.Helper()
	ch := make(chan map[string]any, 64)
	go func() {
		for {
			_, b, err := conn.Read(context.Background())
			if err != nil {
				close(ch)
				return
			}
			var m map[string]any
			_ = json.Unmarshal(b, &m)
			ch <- m
		}
	}()
	return ch
}

// nextEvent: pesan berikutnya, atau nil bila tidak ada dalam batas waktu.
func nextEvent(ch chan map[string]any, wait time.Duration) map[string]any {
	select {
	case m := <-ch:
		return m
	case <-time.After(wait):
		return nil
	}
}

// Revisi 47: event papan task dikirim ke semua klien setelah transaksi commit, tidak dikirim untuk 409 atau
// penolakan, tidak membawa isi task, dan menerimanya tidak memberi akses REST.
func TestTaskLiveEvents(t *testing.T) {
	tm := newTaskTeam(t)
	if _, res, err := tm.dialTasks(t, tm.pic, "https://situs-lain.test"); err == nil || res == nil || res.StatusCode != http.StatusForbidden {
		t.Fatalf("Origin asing: %v %v", res, err)
	}
	if _, res, err := tm.dialTasks(t, &http.Client{}, tm.srv.URL); err == nil || res == nil || res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("tanpa cookie: %v %v", res, err)
	}
	a, _, err := tm.dialTasks(t, tm.pic, tm.srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := tm.dialTasks(t, tm.other, tm.srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	m, _, err := tm.dialTasks(t, tm.manager, tm.srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	ea, eb, em := events(t, a), events(t, b), events(t, m)
	conns := map[string]chan map[string]any{"PIC": ea, "Staf lain": eb, "Manajer": em}
	// Koneksi baru terdaftar setelah handshake; tunggu sebentar supaya event pertama tidak mendahuluinya.
	time.Sleep(100 * time.Millisecond)

	expectAll := func(step, id, aksi string, version int32, pelaku string) {
		t.Helper()
		for who, c := range conns {
			ev := nextEvent(c, 3*time.Second)
			if ev == nil {
				t.Fatalf("%s: %s tidak menerima event", step, who)
			}
			if ev["t"] != "task" || ev["task_id"] != id || ev["aksi"] != aksi || ev["pelaku"] != pelaku ||
				int32(ev["version"].(float64)) != version {
				t.Fatalf("%s: %s menerima %v", step, who, ev)
			}
			// Hanya penanda perubahan, tanpa isi task.
			for k := range ev {
				switch k {
				case "t", "task_id", "version", "aksi", "pelaku", "pelaku_id", "tab":
				default:
					t.Fatalf("%s: event membawa field %q: %v", step, k, ev)
				}
			}
		}
	}
	expectNone := func(step string) {
		t.Helper()
		for who, c := range conns {
			if ev := nextEvent(c, 300*time.Millisecond); ev != nil {
				t.Fatalf("%s: %s menerima event yang tidak semestinya: %v", step, who, ev)
			}
		}
	}

	id := tm.createTask(t)
	expectAll("buat", id, aksiBaru, 1, "manajer.a")

	// Ditolak sebelum commit: 403 (Staf lain memindahkan), 409 (versi basi), 400 (isian salah).
	if code, _ := tm.move(t, tm.other, id, 1, "on_progress"); code != http.StatusForbidden {
		t.Fatalf("Staf lain memindahkan: %d", code)
	}
	if code, _ := tm.move(t, tm.pic, id, 7, "on_progress"); code != http.StatusConflict {
		t.Fatalf("versi basi: %d", code)
	}
	if code := tm.call(t, tm.manager, http.MethodPut, "/api/tasks/"+id, `{"version":1,"title":"","urgency":"low","pic_ids":[]}`); code != http.StatusBadRequest {
		t.Fatalf("isian salah: %d", code)
	}
	expectNone("ditolak")

	// Event membawa tab pengirim, supaya tab itu tidak memberi tahu dirinya sendiri.
	req, _ := http.NewRequest(http.MethodPost, tm.srv.URL+"/api/tasks/"+id+"/move", strings.NewReader(`{"version":1,"to":"on_progress"}`))
	req.Header.Set("Origin", tm.srv.URL)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(tabHeader, "tab-a1")
	res, err := tm.pic.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("PIC memindahkan: %v %v", res, err)
	}
	_ = res.Body.Close()
	ev := nextEvent(ea, 3*time.Second)
	if ev == nil || ev["aksi"] != aksiPindah || ev["tab"] != "tab-a1" || int32(ev["version"].(float64)) != 2 || ev["pelaku"] != "staf.a" {
		t.Fatalf("event pindah: %v", ev)
	}
	for _, c := range []chan map[string]any{eb, em} {
		if ev := nextEvent(c, 3*time.Second); ev == nil || ev["aksi"] != aksiPindah {
			t.Fatalf("event pindah ke klien lain: %v", ev)
		}
	}

	// 409 menyebut siapa yang lebih dulu mengubah.
	var conflict struct {
		Error struct {
			Message string `json:"message"`
			By      string `json:"by"`
		} `json:"error"`
	}
	req, _ = http.NewRequest(http.MethodPost, tm.srv.URL+"/api/tasks/"+id+"/move", strings.NewReader(`{"version":1,"to":"ready"}`))
	req.Header.Set("Origin", tm.srv.URL)
	req.Header.Set("Content-Type", "application/json")
	res, err = tm.manager.Do(req)
	if err != nil || res.StatusCode != http.StatusConflict {
		t.Fatalf("Manajer dengan versi basi: %v %v", res, err)
	}
	_ = json.NewDecoder(res.Body).Decode(&conflict)
	_ = res.Body.Close()
	if conflict.Error.By != "staf.a" || !strings.Contains(conflict.Error.Message, "sudah diubah staf.a lebih dulu") {
		t.Fatalf("pesan 409: %+v", conflict.Error)
	}
	expectNone("409 kedua")

	// Ubah dan hapus juga disiarkan; setelah dihapus, penerima event tetap tidak bisa mengambilnya.
	body := fmt.Sprintf(`{"version":2,"title":"Foto katalog baru","urgency":"low","pic_ids":[%d]}`, tm.picID)
	if code := tm.call(t, tm.manager, http.MethodPut, "/api/tasks/"+id, body); code != http.StatusOK && code != http.StatusNoContent {
		t.Fatalf("Manajer mengubah: %d", code)
	}
	expectAll("ubah", id, aksiUbah, 3, "manajer.a")
	if code := tm.call(t, tm.manager, http.MethodDelete, "/api/tasks/"+id+"?version=3", ""); code != http.StatusNoContent {
		t.Fatalf("Manajer menghapus: %d", code)
	}
	expectAll("hapus", id, aksiHapus, 4, "manajer.a")
	if code := tm.call(t, tm.other, http.MethodGet, "/api/tasks/"+id, ""); code != http.StatusNotFound {
		t.Fatalf("task terhapus diambil setelah event: %d, mau 404", code)
	}

	// Akun yang kehilangan akses (wajib ganti password) menerima event dari koneksi lamanya, tetapi REST tetap
	// menolak. Koneksi itu ditutup pada pemeriksaan sesi berikutnya.
	id2 := tm.createTask(t)
	expectAll("buat kedua", id2, aksiBaru, 1, "manajer.a")
	if _, err := tm.pool.Exec(context.Background(), `UPDATE users SET must_change_password = true WHERE username = 'staf.b'`); err != nil {
		t.Fatal(err)
	}
	if code := tm.call(t, tm.other, http.MethodGet, "/api/tasks/"+id2, ""); code != http.StatusForbidden {
		t.Fatalf("akun wajib ganti password mengambil task: %d, mau 403", code)
	}
	if code := tm.call(t, &http.Client{}, http.MethodGet, "/api/tasks/"+id2, ""); code != http.StatusUnauthorized {
		t.Fatalf("tanpa login mengambil task: %d, mau 401", code)
	}
	if _, res, err := tm.dialTasks(t, tm.other, tm.srv.URL); err == nil || res == nil || res.StatusCode != http.StatusForbidden {
		t.Fatalf("akun wajib ganti password membuka koneksi baru: %v %v", res, err)
	}
}
