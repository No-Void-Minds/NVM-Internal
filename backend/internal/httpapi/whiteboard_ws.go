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
	// CloseTooMany: pengguna sudah membuka terlalu banyak koneksi Whiteboard.
	CloseTooMany websocket.StatusCode = 4429
)

// wbSocket membuka koneksi WebSocket Whiteboard. Rute ini GET, jadi sameOrigin() melewatkannya; Origin dicek di
// sini, karena tanpa cek ini situs lain bisa membuka WebSocket dengan cookie korban (cross-site WebSocket
// hijacking). Sesi sudah diperiksa requireSession dan requirePasswordChanged sebelum upgrade, lalu diperiksa ulang
// tiap wsRecheck selama koneksi hidup.
func (s *Server) wbSocket(c *gin.Context) {
	origin := c.GetHeader("Origin")
	u, err := url.Parse(origin)
	if origin == "" || err != nil || u.Host != c.Request.Host {
		writeError(c, http.StatusForbidden, CodeForbidden, "Permintaan ditolak karena tidak berasal dari aplikasi ini.")
		return
	}
	user := currentUser(c)
	raw, _ := c.Cookie(sessionCookie)
	client := whiteboard.NewClient(user.ID, user.Name, avatarID(user.AvatarID), wbCanWrite(c))

	// Origin sudah dicek di atas dengan aturan yang sama dengan sameOrigin (host Origin sama dengan Host), jadi cek
	// bawaan pustaka dimatikan supaya hanya ada satu aturan.
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(wsReadLimit)
	if !s.hub.Register(client) {
		_ = conn.Close(CloseTooMany, "Terlalu banyak tab Whiteboard terbuka.")
		return
	}
	defer s.hub.Unregister(client)

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	client.Kick = func(reason string) {
		_ = conn.Close(websocket.StatusPolicyViolation, reason)
		cancel()
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case b := <-client.Send:
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
		s.hub.Handle(ctx, client, data)
	}
}
