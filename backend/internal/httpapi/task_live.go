package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"nvm-internal/internal/db/sqlc"
	"nvm-internal/internal/whiteboard"
)

// Papan task realtime (revisi 47). Server hanya menyiarkan bahwa sebuah task berubah: id, versi, aksi, dan siapa
// pelakunya, tanpa isi task. Klien yang membuka papan, Arsip, atau detail mengambil ulang lewat REST, jadi filter,
// lingkup Task saya, batas 10 kartu Completed, dan hak akses tetap dihitung server. Pengguna yang menerima event
// untuk task yang tidak boleh dia lihat tetap ditolak saat mengambilnya.

// Aksi event, dipakai klien untuk teks toast.
const (
	aksiBaru    = "baru"
	aksiUbah    = "ubah"
	aksiPindah  = "pindah"
	aksiCentang = "centang"
	aksiHapus   = "hapus"
)

// liveTaskKey: handler pembuat task menaruh id task baru di sini, karena rute POST /tasks tidak punya :id.
const liveTaskKey = "live_task_id"

// tabHeader: id acak per tab dari klien. Event membawanya kembali supaya tab pelaku tidak menampilkan
// pemberitahuan "diubah orang lain" untuk perubahannya sendiri. Bukan penanda keamanan.
const tabHeader = "X-Tab-Id"

var tabPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

type taskEvent struct {
	T        string `json:"t"`
	TaskID   string `json:"task_id"`
	Version  int32  `json:"version"`
	Aksi     string `json:"aksi"`
	Pelaku   string `json:"pelaku"`
	PelakuID int64  `json:"pelaku_id,omitempty"`
	Tab      string `json:"tab,omitempty"`
}

type taskLiveClient struct {
	userID int64
	send   chan []byte
	kick   func(string)
}

type taskLive struct {
	mu      sync.Mutex
	clients map[*taskLiveClient]bool
	conns   map[int64]int
}

func newTaskLive() *taskLive {
	return &taskLive{clients: map[*taskLiveClient]bool{}, conns: map[int64]int{}}
}

// register memakai batas tab yang sama dengan Whiteboard.
func (h *taskLive) register(c *taskLiveClient) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[c.userID] >= whiteboard.MaxConnsPerUser {
		return false
	}
	h.conns[c.userID]++
	h.clients[c] = true
	return true
}

func (h *taskLive) unregister(c *taskLiveClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.clients[c] {
		return
	}
	delete(h.clients, c)
	if h.conns[c.userID]--; h.conns[c.userID] <= 0 {
		delete(h.conns, c.userID)
	}
}

// publish tidak pernah menunggu: antrean klien yang penuh berarti koneksinya terlalu lambat, jadi ditutup. Klien
// menyambung ulang lalu memuat ulang papan sekali, jadi tidak ada event yang hilang diam-diam.
func (h *taskLive) publish(ev taskEvent) {
	ev.T = "task"
	b, _ := json.Marshal(ev)
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c.send <- b:
		default:
			if c.kick != nil {
				go c.kick("Koneksi terlalu lambat, menyambung ulang.")
			}
		}
	}
}

// taskSocket membuka koneksi realtime papan task. Satu koneksi per tab dipakai bersama papan, Arsip, dan detail.
func (s *Server) taskSocket(c *gin.Context) {
	if !wsOriginOK(c) {
		return
	}
	client := &taskLiveClient{userID: currentUser(c).ID, send: make(chan []byte, 256)}
	s.serveSocket(c, socketPeer{
		send: client.send,
		// Klien tidak mengirim apa pun selain pong; batas kecil cukup.
		readLimit:  1024,
		register:   func() bool { return s.live.register(client) },
		unregister: func() { s.live.unregister(client) },
		setKick: func(k func(string)) {
			s.live.mu.Lock()
			client.kick = k
			s.live.mu.Unlock()
		},
		tooMany: "Terlalu banyak tab papan task terbuka.",
	})
}

// taskEvent menyiarkan event setelah handler selesai dengan jawaban 2xx. Handler task baru menjawab setelah
// transaksinya commit, jadi rollback, 409 versi basi, dan penolakan hak tidak pernah menghasilkan event.
func (s *Server) taskEvent(aksi string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if st := c.Writer.Status(); st < 200 || st > 299 {
			return
		}
		id := c.Param("id")
		if v, ok := c.Get(liveTaskKey); ok {
			id, _ = v.(string)
		}
		if id == "" {
			return
		}
		u := currentUser(c)
		tab := c.GetHeader(tabHeader)
		if !tabPattern.MatchString(tab) {
			tab = ""
		}
		s.publishTask(c.Request.Context(), id, aksi, u.ID, u.Name, tab)
	}
}

func (s *Server) publishTask(ctx context.Context, id, aksi string, actorID int64, actorName, tab string) {
	ev := taskEvent{TaskID: id, Aksi: aksi, Pelaku: actorName, PelakuID: actorID, Tab: tab}
	// Versi hanya membantu klien membandingkan dengan yang dipegangnya; event tetap dikirim bila gagal dibaca.
	if st, err := sqlc.New(s.pool).TaskLiveState(ctx, id); err == nil {
		ev.Version = st.Version
	}
	s.live.publish(ev)
}

// RunOccurrences menjalankan ticker kemunculan task berulang dan menyiarkan task baru yang dibuatnya.
func (s *Server) RunOccurrences(ctx context.Context, log *slog.Logger, every time.Duration) {
	s.tasks.RunOccurrenceTicker(ctx, log, every, func(ids []string) {
		for _, id := range ids {
			s.publishTask(ctx, id, aksiBaru, 0, "Sistem", "")
		}
	})
}
