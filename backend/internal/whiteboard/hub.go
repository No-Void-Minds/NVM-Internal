package whiteboard

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"
)

// Hub kolaborasi langsung (docs/whiteboard.md bagian 3). Hanya hal sementara yang lewat sini: kehadiran, kursor,
// pilihan, gerakan seret, kunci lunak, dan draf teks pemegang kunci. Perubahan data tetap lewat REST; setelah
// transaksi commit, handler memanggil Changed atau Deleted supaya hub menyiarkannya. Satu hub per proses, cukup
// karena backend satu container. Hub tidak tahu soal WebSocket: transport mengirim pesan masuk ke Handle dan
// membaca pesan keluar dari Client.Send.

const (
	// MaxConnsPerUser mencegah tab yang tidak sengaja terbuka puluhan kali.
	MaxConnsPerUser = 32
	// LockTTL: kunci tanpa detak (draf atau lock ulang) selama ini dilepas, untuk koneksi setengah mati.
	LockTTL = 30 * time.Second
	// transferWait: batas menunggu tab lain milik pengguna yang sama menyimpan sebelum kunci dipindah.
	transferWait = 3 * time.Second
	// sendQueue: antrean keluar per koneksi. Klien yang terlalu lambat ditutup dan menyambung ulang dengan seq.
	sendQueue = 512
	// ratePerSecond: kursor, pilihan, dan gerakan seret per koneksi; lebihnya dibuang.
	ratePerSecond = 30
	maxIDs        = MaxBatch
)

// Client adalah satu koneksi (satu tab). Field diakses di bawah kunci hub, kecuali Send yang dibaca transport.
type Client struct {
	UserID   int64
	Name     string
	AvatarID *string
	CanWrite bool
	Send     chan []byte
	// Kick dipasang transport untuk menutup koneksi (antrean penuh).
	Kick func(reason string)

	tabID  string
	board  string
	idle   bool
	tokens float64
	last   time.Time
	gone   bool
}

func NewClient(userID int64, name string, avatarID *string, canWrite bool) *Client {
	return &Client{UserID: userID, Name: name, AvatarID: avatarID, CanWrite: canWrite, Send: make(chan []byte, sendQueue), tokens: ratePerSecond}
}

type lockState struct {
	holder  *Client
	board   string
	kind    string
	beat    time.Time
	draft   json.RawMessage
	waiting *Client
	asked   time.Time
}

type Hub struct {
	svc   *Service
	now   func() time.Time
	mu    sync.Mutex
	rooms map[string]map[*Client]bool
	conns map[int64]int
	locks map[string]*lockState
	grabs map[string]*Client
}

func NewHub(svc *Service) *Hub {
	return &Hub{
		svc: svc, now: time.Now,
		rooms: map[string]map[*Client]bool{}, conns: map[int64]int{},
		locks: map[string]*lockState{}, grabs: map[string]*Client{},
	}
}

// Register mendaftarkan koneksi baru; false bila pengguna sudah memakai MaxConnsPerUser koneksi.
func (h *Hub) Register(c *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[c.UserID] >= MaxConnsPerUser {
		return false
	}
	h.conns[c.UserID]++
	c.last = h.now()
	return true
}

// Unregister dipanggil saat koneksi berakhir: kunci dan pegangan seretnya dilepas dan kehadiran diperbarui.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c.gone {
		return
	}
	c.gone = true
	h.conns[c.UserID]--
	if h.conns[c.UserID] <= 0 {
		delete(h.conns, c.UserID)
	}
	h.leave(c)
}

// leave mengeluarkan klien dari papannya. Dipanggil dengan kunci hub terpegang.
func (h *Hub) leave(c *Client) {
	if c.board == "" {
		return
	}
	board := c.board
	for card, l := range h.locks {
		if l.waiting == c {
			l.waiting = nil
		}
		if l.holder == c {
			h.releaseLock(card, l)
		}
	}
	released := []string{}
	for card, g := range h.grabs {
		if g == c {
			delete(h.grabs, card)
			released = append(released, card)
		}
	}
	delete(h.rooms[board], c)
	if len(h.rooms[board]) == 0 {
		delete(h.rooms, board)
	}
	c.board = ""
	if len(released) > 0 {
		h.broadcast(board, nil, msg{"t": "released", "ids": released})
	}
	h.broadcastPresence(board)
}

type msg map[string]any

func encode(m msg) []byte {
	b, _ := json.Marshal(m)
	return b
}

// send tidak pernah menunggu: antrean penuh berarti klien terlalu lambat, jadi koneksinya ditutup supaya tidak
// menahan klien lain. Klien menyambung ulang dan menyusul lewat seq.
func (h *Hub) send(c *Client, m msg) {
	if c.gone {
		return
	}
	select {
	case c.Send <- encode(m):
	default:
		if c.Kick != nil {
			go c.Kick("Koneksi terlalu lambat, menyambung ulang.")
		}
	}
}

func (h *Hub) broadcast(board string, except *Client, m msg) {
	b := encode(m)
	for c := range h.rooms[board] {
		if c == except || c.gone {
			continue
		}
		select {
		case c.Send <- b:
		default:
			if c.Kick != nil {
				go c.Kick("Koneksi terlalu lambat, menyambung ulang.")
			}
		}
	}
}

// Person adalah satu orang di daftar kehadiran: beberapa tab milik orang yang sama digabung.
type Person struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	AvatarID *string `json:"avatar_id"`
	Color    int     `json:"color"`
	Idle     bool    `json:"idle"`
	Tabs     int     `json:"tabs"`
	Editing  string  `json:"editing,omitempty"`
}

// ColorIndex memberi warna kolaborasi tetap per orang (1 sampai 8), sama di semua papan dan sesi.
func ColorIndex(userID int64) int { return int((userID%8+8)%8) + 1 }

func (h *Hub) presence(board string) []Person {
	by := map[int64]*Person{}
	for c := range h.rooms[board] {
		p := by[c.UserID]
		if p == nil {
			p = &Person{ID: c.UserID, Name: c.Name, AvatarID: c.AvatarID, Color: ColorIndex(c.UserID), Idle: true}
			by[c.UserID] = p
		}
		p.Tabs++
		p.Idle = p.Idle && c.idle
	}
	for card, l := range h.locks {
		if l.board == board && by[l.holder.UserID] != nil {
			by[l.holder.UserID].Editing = card
		}
	}
	out := make([]Person, 0, len(by))
	for _, p := range by {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (h *Hub) broadcastPresence(board string) {
	if board == "" {
		return
	}
	h.broadcast(board, nil, msg{"t": "presence", "people": h.presence(board)})
}

// allow menerapkan batas laju token bucket per koneksi untuk pesan yang sering (kursor, pilihan, gerakan).
func (h *Hub) allow(c *Client) bool {
	now := h.now()
	c.tokens += now.Sub(c.last).Seconds() * ratePerSecond
	if c.tokens > ratePerSecond {
		c.tokens = ratePerSecond
	}
	c.last = now
	if c.tokens < 1 {
		return false
	}
	c.tokens--
	return true
}

func person(c *Client) msg {
	return msg{"id": c.UserID, "name": c.Name, "color": ColorIndex(c.UserID), "tab": c.tabID}
}

// In adalah satu pesan dari klien.
type In struct {
	T       string          `json:"t"`
	Board   string          `json:"board"`
	LastSeq int64           `json:"last_seq"`
	TabID   string          `json:"tab_id"`
	X       float64         `json:"x"`
	Y       float64         `json:"y"`
	IDs     []string        `json:"ids"`
	Moves   []Move          `json:"moves"`
	Card    string          `json:"card"`
	Content json.RawMessage `json:"content"`
	Force   bool            `json:"force"`
	// Goresan yang sedang digambar (revisi A): titik koordinat dunia sejauh ini.
	Points []int  `json:"points"`
	Color  string `json:"color"`
	Width  int    `json:"width"`
	// All: grab semua atau tidak sama sekali (seret grup, revisi 50).
	All bool `json:"all"`
}

type Move struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	W  float64 `json:"w"`
	H  float64 `json:"h"`
}

// Handle memproses satu pesan klien. Pesan tidak dikenal dan pesan tulis dari Admin dibuang dengan jawaban
// "denied", tidak pernah diteruskan.
func (h *Hub) Handle(ctx context.Context, c *Client, raw []byte) {
	var in In
	if err := json.Unmarshal(raw, &in); err != nil {
		return
	}
	switch in.T {
	case "join":
		h.join(ctx, c, in)
		return
	case "lock":
		h.lock(ctx, c, in)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if c.board == "" {
		return
	}
	board := c.board
	writeOnly := in.T == "select" || in.T == "grab" || in.T == "move" || in.T == "release" || in.T == "draft" || in.T == "unlock" || in.T == "ink" || in.T == "ink_end"
	if writeOnly && !c.CanWrite {
		h.send(c, msg{"t": "denied", "for": in.T, "message": "Admin hanya bisa melihat Whiteboard, tidak bisa mengubahnya."})
		return
	}
	switch in.T {
	case "cursor":
		if h.allow(c) && finite(in.X) && finite(in.Y) {
			h.broadcast(board, c, msg{"t": "cursor", "user": person(c), "x": in.X, "y": in.Y})
		}
	case "select":
		if len(in.IDs) <= maxIDs && h.allow(c) {
			h.broadcast(board, c, msg{"t": "select", "user": person(c), "ids": in.IDs})
		}
	case "grab":
		h.grab(c, board, in.IDs, in.All)
	case "move":
		if !h.allow(c) {
			return
		}
		moves := []Move{}
		for _, m := range in.Moves {
			if h.grabs[m.ID] == c && finite(m.X) && finite(m.Y) && finite(m.W) && finite(m.H) {
				moves = append(moves, m)
			}
		}
		if len(moves) > 0 {
			h.broadcast(board, c, msg{"t": "moving", "user": person(c), "moves": moves})
		}
	case "ink":
		// Goresan langsung hanya tampilan; yang tersimpan tetap lewat REST. Titik divalidasi seperti di server supaya
		// klien lain tidak menerima goresan yang tidak akan diterima.
		if !h.allow(c) || len(in.Points) < 2 || len(in.Points)%2 != 0 || len(in.Points) > MaxStrokePoints*2 ||
			(in.Color != drawingInk && !Colors[in.Color]) || !DrawingWidths[in.Width] {
			return
		}
		for _, p := range in.Points {
			if p < -maxCoord || p > maxCoord {
				return
			}
		}
		h.broadcast(board, c, msg{"t": "ink", "user": person(c), "points": in.Points, "color": in.Color, "width": in.Width})
	case "ink_end":
		h.broadcast(board, c, msg{"t": "ink_end", "user": person(c)})
	case "release":
		ids := []string{}
		for _, id := range in.IDs {
			if h.grabs[id] == c {
				delete(h.grabs, id)
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			h.broadcast(board, c, msg{"t": "released", "ids": ids})
		}
	case "draft":
		l := h.locks[in.Card]
		if l == nil || l.holder != c {
			return
		}
		// Draf divalidasi dengan pembersih yang sama dengan REST sebelum diteruskan, jadi klien lain tidak pernah
		// menerima isi yang tidak akan diterima server.
		clean, _, err := CleanContent(l.kind, in.Content)
		if err != nil {
			return
		}
		l.draft = clean
		l.beat = h.now()
		h.broadcast(board, c, msg{"t": "draft", "card": in.Card, "content": json.RawMessage(clean), "user": person(c)})
	case "unlock":
		if l := h.locks[in.Card]; l != nil && l.holder == c {
			h.releaseLock(in.Card, l)
			h.broadcastPresence(board)
		}
	case "idle", "active":
		if idle := in.T == "idle"; c.idle != idle {
			c.idle = idle
			h.broadcastPresence(board)
		}
	}
}

func finite(v float64) bool { return v == v && v > -2*maxCoord && v < 2*maxCoord }

// join memindahkan klien ke papan lain, mengirim daftar kehadiran dan kunci yang sedang berjalan, lalu menyusulkan
// perubahan sejak last_seq (sinkron ulang setelah putus).
func (h *Hub) join(ctx context.Context, c *Client, in In) {
	board, err := h.svc.ResolveBoard(ctx, Actor{ID: c.UserID}, in.Board)
	if err != nil {
		h.send(c, msg{"t": "board_missing", "board": in.Board})
		return
	}
	var sync Sync
	resync := false
	if in.LastSeq > 0 {
		sync, err = h.svc.Since(ctx, board, in.LastSeq)
		resync = err != nil || sync.TooMany
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if c.gone {
		return
	}
	if c.board != board {
		h.leave(c)
		c.board = board
		if h.rooms[board] == nil {
			h.rooms[board] = map[*Client]bool{}
		}
		h.rooms[board][c] = true
	}
	if len(in.TabID) <= 64 {
		c.tabID = in.TabID
	}
	c.idle = false
	locks := []msg{}
	for card, l := range h.locks {
		if l.board == board && l.holder != c {
			locks = append(locks, msg{"card": card, "user": person(l.holder)})
		}
	}
	grabbed := map[string]msg{}
	for card, g := range h.grabs {
		if g.board == board && g != c {
			grabbed[card] = person(g)
		}
	}
	reply := msg{"t": "joined", "board": board, "locks": locks, "grabs": grabbed}
	switch {
	case resync:
		reply["resync"] = true
	case in.LastSeq > 0:
		reply["cards"] = sync.Cards
		reply["deleted"] = sync.Deleted
		reply["connectors"] = sync.Connectors
		reply["connectors_gone"] = sync.ConnectorsGone
	}
	h.send(c, reply)
	h.broadcastPresence(board)
}

func (h *Hub) grab(c *Client, board string, ids []string, all bool) {
	if len(ids) > maxIDs {
		return
	}
	granted, denied := []string{}, []msg{}
	for _, id := range ids {
		if g := h.grabs[id]; g != nil && g != c {
			denied = append(denied, msg{"id": id, "user": person(g)})
			continue
		}
		granted = append(granted, id)
	}
	// Grup bergerak utuh: satu anggota yang dipegang orang lain menolak seluruh grab.
	if all && len(denied) > 0 {
		granted = []string{}
	}
	for _, id := range granted {
		h.grabs[id] = c
	}
	h.send(c, msg{"t": "grab_result", "granted": granted, "denied": denied})
	if len(granted) > 0 {
		h.broadcast(board, c, msg{"t": "grabbed", "ids": granted, "user": person(c)})
	}
}

// lock memberi kunci lunak satu kartu. Pemegang lain dari akun yang sama (tab lain) bisa diminta menyerahkan kunci
// dengan force: tab lama diminta menyimpan dan melepas, dan kunci pindah paling lambat transferWait kemudian.
func (h *Hub) lock(ctx context.Context, c *Client, in In) {
	if !c.CanWrite {
		h.mu.Lock()
		h.send(c, msg{"t": "denied", "for": "lock", "message": "Admin hanya bisa melihat Whiteboard, tidak bisa mengubahnya."})
		h.mu.Unlock()
		return
	}
	h.mu.Lock()
	board := c.board
	h.mu.Unlock()
	if board == "" {
		return
	}
	kind, err := h.svc.CardKind(ctx, board, in.Card)
	h.mu.Lock()
	defer h.mu.Unlock()
	if err != nil || c.board != board {
		h.send(c, msg{"t": "lock_denied", "card": in.Card, "missing": true})
		return
	}
	now := h.now()
	l := h.locks[in.Card]
	switch {
	case l == nil:
		h.locks[in.Card] = &lockState{holder: c, board: board, kind: kind, beat: now}
		h.send(c, msg{"t": "lock_granted", "card": in.Card})
		h.broadcast(board, c, msg{"t": "locked", "card": in.Card, "user": person(c)})
		h.broadcastPresence(board)
	case l.holder == c:
		l.beat = now
	case in.Force && l.holder.UserID == c.UserID:
		l.waiting = c
		l.asked = now
		h.send(l.holder, msg{"t": "lock_release", "card": in.Card})
	default:
		h.send(c, msg{"t": "lock_denied", "card": in.Card, "user": person(l.holder), "same_user": l.holder.UserID == c.UserID})
	}
}

// releaseLock melepas kunci; bila ada tab yang menunggu (pindah kunci), kunci langsung diberikan kepadanya.
func (h *Hub) releaseLock(card string, l *lockState) {
	delete(h.locks, card)
	h.broadcast(l.board, nil, msg{"t": "unlocked", "card": card})
	if w := l.waiting; w != nil && !w.gone && w.board == l.board {
		h.locks[card] = &lockState{holder: w, board: l.board, kind: l.kind, beat: h.now()}
		h.send(w, msg{"t": "lock_granted", "card": card})
		h.broadcast(l.board, w, msg{"t": "locked", "card": card, "user": person(w)})
	}
}

// Sweep melepas kunci yang kehilangan detak dan pindah kunci yang tidak dijawab tab lama. Dipanggil berkala.
func (h *Hub) Sweep() {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.now()
	for card, l := range h.locks {
		expired := now.Sub(l.beat) > LockTTL
		transfer := l.waiting != nil && now.Sub(l.asked) > transferWait
		if expired || transfer {
			if transfer {
				h.send(l.holder, msg{"t": "lock_lost", "card": card})
			}
			h.releaseLock(card, l)
			h.broadcastPresence(l.board)
		}
	}
}

// Run menjalankan Sweep tiap 5 detik sampai ctx selesai.
func (h *Hub) Run(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.Sweep()
		}
	}
}

// Drafts mengembalikan draf terakhir kartu-kartu yang sedang dikunci, untuk disimpan sebelum kartu dihapus.
func (h *Hub) Drafts(ids []string) map[string]json.RawMessage {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := map[string]json.RawMessage{}
	for _, id := range ids {
		if l := h.locks[id]; l != nil && l.draft != nil {
			out[id] = l.draft
		}
	}
	return out
}

// Changed menyiarkan kartu hasil transaksi yang sudah commit ke semua koneksi di papan itu, termasuk tab lain
// milik pengirim. Klien mengabaikan versi yang tidak lebih baru dari yang dipegang.
func (h *Hub) Changed(board string, cards []Card, by int64) {
	if len(cards) == 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.broadcast(board, nil, msg{"t": "changed", "cards": cards, "by": by})
}

// Deleted menyiarkan kartu yang masuk Tempat sampah. Pemegang kunci kartu itu diberi tahu siapa penghapusnya
// lewat pesan yang sama; kuncinya dilepas.
func (h *Hub) Deleted(board string, ids []string, batch, byName string, boards []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, id := range ids {
		delete(h.locks, id)
		delete(h.grabs, id)
	}
	h.broadcast(board, nil, msg{"t": "deleted", "ids": ids, "batch": batch, "by": byName})
	// Papan anak yang ikut terhapus: siapa pun yang sedang membukanya diberi tahu.
	for _, b := range boards {
		h.broadcast(b, nil, msg{"t": "board_deleted", "batch": batch, "by": byName})
	}
	h.broadcastPresence(board)
}

// Resync meminta semua koneksi di papan itu memuat ulang papan utuh (mis. setelah pulihkan atau ganti judul papan
// anak, yang mengubah beberapa hal sekaligus).
func (h *Hub) Resync(boards ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, b := range boards {
		h.broadcast(b, nil, msg{"t": "resync"})
	}
}

// Renamed menyiarkan judul papan baru ke papan itu sendiri.
func (h *Hub) Renamed(board, title string, version int32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.broadcast(board, nil, msg{"t": "renamed", "board": board, "title": title, "version": version})
}

// Connectors menyiarkan garis yang dibuat atau diubah; ConnectorsGone garis yang masuk Tempat sampah.
func (h *Hub) Connectors(board string, items []Connector) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.broadcast(board, nil, msg{"t": "connectors", "items": items})
}

func (h *Hub) ConnectorsGone(board string, ids []string, batch string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.broadcast(board, nil, msg{"t": "connectors_gone", "ids": ids, "batch": batch})
}

// Moved: kartu keluar dari papan asal (kunci dan genggamannya dilepas), papan tujuan dimuat ulang.
func (h *Hub) Moved(res MoveResult, byName string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, id := range res.Moved {
		delete(h.locks, id)
		delete(h.grabs, id)
	}
	h.broadcast(res.From, nil, msg{"t": "moved", "ids": res.Moved, "connectors_gone": res.ConnectorsGone, "to": res.To, "by": byName})
	h.broadcast(res.To, nil, msg{"t": "resync"})
	h.broadcastPresence(res.From)
}

// LockedByOther mengembalikan nama pemegang kunci kartu bila kuncinya dipegang akun lain. Tabel dan gambar bebas
// hanya bisa diubah pemegang kunci (atau siapa pun bila tidak ada yang memegang), sama dengan editor catatan.
func (h *Hub) LockedByOther(card string, userID int64) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if l := h.locks[card]; l != nil && l.holder.UserID != userID {
		return l.holder.Name
	}
	return ""
}
