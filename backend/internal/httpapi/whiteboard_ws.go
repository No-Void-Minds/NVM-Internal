package httpapi

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"

	"nvm-internal/internal/whiteboard"
)

const (
	// wsPing: Cloudflare menutup koneksi yang diam sekitar 100 detik, jadi server mengirim ping lebih sering.
	wsPing = 30 * time.Second
	// wsReadLimit: draf catatan maksimal 64 KB ditambah amplop pesan.
	wsReadLimit = whiteboard.MaxContentBytes + 4096
	// CloseSession adalah kode tutup saat sesi berakhir; klien mengarahkan ke halaman masuk, bukan menyambung ulang.
	CloseSession websocket.StatusCode = 4401
	// CloseTooMany: pengguna sudah membuka terlalu banyak koneksi.
	CloseTooMany websocket.StatusCode = 4429
)

// socketPeer: bagian koneksi yang berbeda antara Whiteboard dan papan task. Transport (Origin, ping, cek ulang
// sesi, antrean tulis) sama untuk keduanya, jadi ada di serveSocket.
type socketPeer struct {
	send       <-chan []byte
	readLimit  int64
	register   func() bool
	unregister func()
	setKick    func(func(reason string))
	// handle nil berarti pesan masuk dibuang (papan task hanya menerima siaran).
	handle  func(ctx context.Context, data []byte)
	tooMany string
}

// wsOriginOK: rute WebSocket berupa GET, jadi sameOrigin() melewatkannya. Tanpa cek ini situs lain bisa membuka
// WebSocket dengan cookie korban (cross-site WebSocket hijacking).
func wsOriginOK(c *gin.Context) bool {
	origin := c.GetHeader("Origin")
	u, err := url.Parse(origin)
	if origin == "" || err != nil || u.Host != c.Request.Host {
		writeError(c, http.StatusForbidden, CodeForbidden, "Permintaan ditolak karena tidak berasal dari aplikasi ini.")
		return false
	}
	return true
}

// wbSocket membuka koneksi WebSocket Whiteboard. Sesi sudah diperiksa requireSession dan requirePasswordChanged
// sebelum upgrade, lalu diperiksa ulang tiap wsRecheck selama koneksi hidup.
func (s *Server) wbSocket(c *gin.Context) {
	if !wsOriginOK(c) {
		return
	}
	user := currentUser(c)
	client := whiteboard.NewClient(user.ID, user.Name, avatarID(user.AvatarID), wbCanWrite(c))
	s.serveSocket(c, socketPeer{
		send:       client.Send,
		readLimit:  wsReadLimit,
		register:   func() bool { return s.hub.Register(client) },
		unregister: func() { s.hub.Unregister(client) },
		setKick:    func(k func(string)) { client.Kick = k },
		handle:     func(ctx context.Context, data []byte) { s.hub.Handle(ctx, client, data) },
		tooMany:    "Terlalu banyak tab Whiteboard terbuka.",
	})
}

func (s *Server) serveSocket(c *gin.Context, p socketPeer) {
	raw, _ := c.Cookie(sessionCookie)
	// Origin sudah dicek wsOriginOK dengan aturan yang sama dengan sameOrigin (host Origin sama dengan Host), jadi
	// cek bawaan pustaka dimatikan supaya hanya ada satu aturan.
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(p.readLimit)
	if !p.register() {
		_ = conn.Close(CloseTooMany, p.tooMany)
		return
	}
	defer p.unregister()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	p.setKick(func(reason string) {
		_ = conn.Close(websocket.StatusPolicyViolation, reason)
		cancel()
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case b := <-p.send:
				wctx, done := context.WithTimeout(ctx, 10*time.Second)
				err := conn.Write(wctx, websocket.MessageText, b)
				done()
				if err != nil {
					cancel()
					return
				}
			}
		}
	}()

	go func() {
		ping := time.NewTicker(wsPing)
		check := time.NewTicker(s.wsRecheck)
		defer ping.Stop()
		defer check.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ping.C:
				pctx, done := context.WithTimeout(ctx, 10*time.Second)
				err := conn.Ping(pctx)
				done()
				if err != nil {
					cancel()
					return
				}
			case <-check.C:
				// Logout, ganti password, reset, nonaktif, atau lewat batas 9 jam menutup koneksi.
				if _, ok, err := s.sessionFor(ctx, raw); err == nil && !ok {
					_ = conn.Close(CloseSession, "Sesi kamu sudah berakhir. Silakan masuk lagi.")
					cancel()
					return
				}
			}
		}
	}()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if p.handle != nil {
			p.handle(ctx, data)
		}
	}
}
