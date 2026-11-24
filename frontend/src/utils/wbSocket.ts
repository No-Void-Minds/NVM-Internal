// Koneksi WebSocket Whiteboard (W2, docs/whiteboard.md bagian 3). Hanya membawa hal sementara (kehadiran, kursor,
// pilihan, gerakan, kunci, draf) dan pemberitahuan perubahan; data tetap disimpan lewat REST. Tanpa Vue supaya
// jadwal sambung ulang bisa diuji dengan node --test.

export type SocketState = 'connecting' | 'open' | 'closed'

// Jeda sambung ulang 1, 2, 4, 8, 16, lalu 30 detik, ditambah acak sampai 20%, supaya sepuluh orang tidak
// menyambung serentak setelah backend di-restart oleh rilis.sh.
export function reconnectDelay(attempt: number, random = Math.random): number {
  const base = Math.min(30, 2 ** Math.max(0, attempt)) * 1000
  return Math.round(base * (1 + random() * 0.2))
}

// Kode tutup dari server: sesi berakhir (ke halaman masuk) dan terlalu banyak tab (tidak menyambung ulang).
export const CLOSE_SESSION = 4401
export const CLOSE_TOO_MANY = 4429

export type SocketHandlers = {
  message: (m: Record<string, unknown>) => void
  state: (s: SocketState) => void
  // Dipanggil setiap kali koneksi terbuka (pertama dan setelah sambung ulang), sebelum pesan antrean dikirim.
  open: () => void
  sessionEnded: () => void
  tooMany: () => void
}

export class WbSocket {
  private ws: WebSocket | null = null
  private attempt = 0
  private timer: ReturnType<typeof setTimeout> | undefined
  private stopped = false
  state: SocketState = 'connecting'

  private readonly url: string
  private readonly h: SocketHandlers

  constructor(url: string, h: SocketHandlers) {
    this.url = url
    this.h = h
  }

  start() {
    this.stopped = false
    this.connect()
  }

  private setState(s: SocketState) {
    this.state = s
    this.h.state(s)
  }

  private connect() {
    clearTimeout(this.timer)
    this.setState('connecting')
    let ws: WebSocket
    try {
      ws = new WebSocket(this.url)
    } catch {
      this.schedule()
      return
    }
    this.ws = ws
    ws.onopen = () => {
      this.attempt = 0
      this.setState('open')
      this.h.open()
    }
    ws.onmessage = (e) => {
      try {
        this.h.message(JSON.parse(String(e.data)) as Record<string, unknown>)
      } catch {
        // Pesan rusak dibuang; server tidak pernah mengirimnya.
      }
    }
    ws.onclose = (e) => {
      if (this.ws !== ws) return
      this.ws = null
      if (e.code === CLOSE_SESSION) {
        this.stopped = true
        this.setState('closed')
        this.h.sessionEnded()
        return
      }
      if (e.code === CLOSE_TOO_MANY) {
        this.stopped = true
        this.setState('closed')
        this.h.tooMany()
        return
      }
      this.schedule()
    }
  }

  private schedule() {
    if (this.stopped) return
    this.setState('closed')
    this.timer = setTimeout(() => this.connect(), reconnectDelay(this.attempt++))
  }

  // Sambung ulang segera (mis. browser kembali online), tanpa menunggu jadwal.
  retryNow() {
    if (this.stopped || this.state === 'open') return
    this.attempt = 0
    this.connect()
  }

  // Pesan saat koneksi tidak terbuka dibuang: semuanya sementara, dan keadaan terkini dikirim ulang saat join.
  send(m: Record<string, unknown>) {
    if (this.ws?.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify(m))
  }

  stop() {
    this.stopped = true
    clearTimeout(this.timer)
    const ws = this.ws
    this.ws = null
    ws?.close(1000)
  }
}
