// Logika murni Whiteboard (docs/whiteboard.md): tipe data, geometri kanvas, navigasi keyboard antar kartu, dan teks
// riwayat. Tanpa Vue dan tanpa klien API, supaya bisa diuji dengan node --test.

export type CardKind = 'note' | 'heading' | 'checklist' | 'board'
export type CardColor = 'abu' | 'hijau' | 'kuning' | 'biru' | 'krem' | 'merah'

export type PMMark = { type: string; attrs?: Record<string, unknown> }
export type PMNode = {
  type: string
  attrs?: Record<string, unknown>
  content?: PMNode[]
  text?: string
  marks?: PMMark[]
}
export type NoteContent = { doc: PMNode }
export type HeadingContent = { text: string; level: 1 | 2 }
export type ChecklistItem = { id: string; text: string; done: boolean }
export type ChecklistContent = { title: string; items: ChecklistItem[] }

// Papan anak yang ditunjuk kartu jenis board.
export type BoardRef = { id: string; title: string; version: number; boards: number; cards: number }
export type BoardContent = Record<string, never>

export type Card = {
  id: string
  board_id: string
  kind: CardKind
  x: number
  y: number
  w: number
  h: number
  z: number
  color: CardColor | null
  content: NoteContent | HeadingContent | ChecklistContent | BoardContent
  ref?: BoardRef
  version: number
  content_version: number
  seq: number
  updated_by: number
  updated_at: string
}

export type Rect = { x: number; y: number; w: number; h: number }
export type Viewport = { x: number; y: number; zoom: number }

export const GRID = 8
export const MIN_ZOOM = 0.25
export const MAX_ZOOM = 2
export const MIN_SIZE = 40
export const MAX_SIZE = 4000
export const COORD_LIMIT = 1_000_000

export const defaultSize: Record<CardKind, { w: number; h: number }> = {
  note: { w: 240, h: 120 },
  heading: { w: 320, h: 64 },
  checklist: { w: 260, h: 120 },
  board: { w: 200, h: 120 },
}

export const kindLabel: Record<CardKind, string> = {
  note: 'Catatan',
  heading: 'Judul',
  checklist: 'Checklist',
  board: 'Papan',
}

export const colors: { value: CardColor | null; label: string }[] = [
  { value: null, label: 'Putih' },
  { value: 'abu', label: 'Abu' },
  { value: 'hijau', label: 'Hijau' },
  { value: 'kuning', label: 'Kuning' },
  { value: 'biru', label: 'Biru' },
  { value: 'krem', label: 'Krem' },
  { value: 'merah', label: 'Merah' },
]

export function clamp(v: number, min: number, max: number) {
  return Math.min(max, Math.max(min, v))
}

export function snap(v: number, on: boolean) {
  return on ? Math.round(v / GRID) * GRID : Math.round(v)
}

// Titik layar (relatif ke elemen kanvas) menjadi koordinat dunia, dan sebaliknya.
export function toWorld(v: Viewport, sx: number, sy: number) {
  return { x: (sx - v.x) / v.zoom, y: (sy - v.y) / v.zoom }
}

// Zoom di sekitar satu titik layar: titik dunia di bawah penunjuk tetap di tempatnya.
export function zoomAt(v: Viewport, sx: number, sy: number, nextZoom: number): Viewport {
  const zoom = clamp(nextZoom, MIN_ZOOM, MAX_ZOOM)
  const w = toWorld(v, sx, sy)
  return { zoom, x: sx - w.x * zoom, y: sy - w.y * zoom }
}

export function intersects(a: Rect, b: Rect) {
  return a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h
}

export function bounds(rects: Rect[]): Rect | null {
  if (rects.length === 0) return null
  let x1 = Infinity
  let y1 = Infinity
  let x2 = -Infinity
  let y2 = -Infinity
  for (const r of rects) {
    x1 = Math.min(x1, r.x)
    y1 = Math.min(y1, r.y)
    x2 = Math.max(x2, r.x + r.w)
    y2 = Math.max(y2, r.y + r.h)
  }
  return { x: x1, y: y1, w: x2 - x1, h: y2 - y1 }
}

// Pas ke layar: semua kartu terlihat dengan tepi 48 px, zoom tidak lebih dari 100% supaya papan kecil tidak
// diperbesar berlebihan.
export function fitView(rects: Rect[], width: number, height: number): Viewport {
  const b = bounds(rects)
  if (!b || width <= 0 || height <= 0) return { x: 48, y: 48, zoom: 1 }
  const pad = 48
  const zoom = clamp(Math.min((width - pad * 2) / b.w, (height - pad * 2) / b.h, 1), MIN_ZOOM, MAX_ZOOM)
  return { zoom, x: (width - b.w * zoom) / 2 - b.x * zoom, y: (height - b.h * zoom) / 2 - b.y * zoom }
}

// Area dunia yang terlihat plus satu layar margin di setiap sisi: kartu di luar ini tidak di-mount (culling).
export function visibleWorld(v: Viewport, width: number, height: number, margin = 1): Rect {
  const w = width / v.zoom
  const h = height / v.zoom
  const tl = toWorld(v, 0, 0)
  return { x: tl.x - w * margin, y: tl.y - h * margin, w: w * (1 + 2 * margin), h: h * (1 + 2 * margin) }
}

// Urutan baca: atas ke bawah, lalu kiri ke kanan. Kartu yang tingginya bertumpuk dianggap satu baris bila
// selisih y-nya kurang dari 24 px, supaya kartu sejajar tidak melompat karena beda beberapa piksel.
export function readingOrder<T extends Rect & { id: string }>(cards: T[]): T[] {
  return [...cards].sort((a, b) => {
    if (Math.abs(a.y - b.y) >= 24) return a.y - b.y
    if (a.x !== b.x) return a.x - b.x
    return a.id < b.id ? -1 : 1
  })
}

export type Direction = 'up' | 'down' | 'left' | 'right'

// Kartu terdekat di arah panah, diukur dari pusat ke pusat. Kartu di luar kerucut arah itu tidak dipilih, dan
// penyimpangan dari sumbu arah diberi bobot dua kali supaya yang lurus didahulukan.
export function nearestInDirection<T extends Rect & { id: string }>(
  from: T,
  cards: T[],
  dir: Direction,
): T | null {
  const cx = from.x + from.w / 2
  const cy = from.y + from.h / 2
  let best: T | null = null
  let bestScore = Infinity
  for (const c of cards) {
    if (c.id === from.id) continue
    const dx = c.x + c.w / 2 - cx
    const dy = c.y + c.h / 2 - cy
    const along = dir === 'left' ? -dx : dir === 'right' ? dx : dir === 'up' ? -dy : dy
    const across = dir === 'left' || dir === 'right' ? Math.abs(dy) : Math.abs(dx)
    if (along <= 0 || across > along * 2) continue
    const score = along + across * 2
    if (score < bestScore) {
      bestScore = score
      best = c
    }
  }
  return best
}

export function plainText(node: PMNode | undefined): string {
  if (!node) return ''
  if (node.type === 'text') return node.text ?? ''
  if (node.type === 'hardBreak') return ' '
  return (node.content ?? []).map(plainText).join(node.type === 'doc' ? ' ' : '')
}

export function excerpt(text: string, max = 60) {
  const t = text.replace(/\s+/g, ' ').trim()
  return t.length > max ? `${t.slice(0, max)}...` : t
}

// Ringkasan isi papan anak, mis. "2 papan, 4 kartu".
export function refSummary(ref: BoardRef | undefined): string {
  if (!ref) return ''
  if (ref.boards === 0 && ref.cards === 0) return 'Kosong'
  const parts: string[] = []
  if (ref.boards) parts.push(`${ref.boards} papan`)
  if (ref.cards) parts.push(`${ref.cards} kartu`)
  return parts.join(', ')
}

// Label kartu untuk pembaca layar dan daftar: jenis plus awal isinya.
export function cardLabel(card: Card): string {
  if (card.kind === 'board') return `Papan ${card.ref?.title ?? ''}, ${refSummary(card.ref)}`
  if (card.kind === 'note') {
    const t = excerpt(plainText((card.content as NoteContent).doc))
    return t ? `Catatan: ${t}` : 'Catatan kosong'
  }
  if (card.kind === 'heading') {
    const t = (card.content as HeadingContent).text
    return t ? `Judul: ${t}` : 'Judul kosong'
  }
  const c = card.content as ChecklistContent
  const done = c.items.filter((i) => i.done).length
  return `Checklist ${c.title || 'tanpa judul'}, ${done} dari ${c.items.length} selesai`
}

export function newId(): string {
  return crypto.randomUUID()
}

// Isi awal kartu baru, sama dengan DefaultContent di backend.
export function emptyContent(kind: CardKind): Card['content'] {
  if (kind === 'note') return { doc: { type: 'doc', content: [{ type: 'paragraph' }] } }
  if (kind === 'heading') return { text: '', level: 1 }
  if (kind === 'board') return {}
  return { title: '', items: [] }
}

export type Activity = {
  id: number
  action: string
  summary: Record<string, unknown>
  created_at: string
  updated_at: string
  user_name: string
}

// Kalimat riwayat per papan. Nama pelaku ditulis terpisah oleh tampilan.
export function activityText(a: Activity): string {
  const s = a.summary
  const kind = typeof s.kind === 'string' ? s.kind : ''
  const label = (kindLabel[kind as CardKind] ?? 'kartu').toLowerCase()
  const ex = typeof s.excerpt === 'string' && s.excerpt ? `: "${s.excerpt}"` : ''
  const n = typeof s.count === 'number' ? s.count : 0
  switch (a.action) {
    case 'board_renamed':
      return `mengganti judul papan dari "${String(s.from ?? '')}" menjadi "${String(s.to ?? '')}"`
    case 'card_created':
      return `menambah ${label}${ex}`
    case 'card_edited':
      return `mengubah isi ${label}${ex}`
    case 'cards_arranged':
      return `mengatur letak, ukuran, atau warna ${n} kartu`
    case 'cards_deleted':
      return `memindahkan ${n} kartu ke Tempat sampah${typeof s.boards === 'number' && s.boards > 0 ? `, termasuk ${s.boards} papan` : ''}`
    case 'cards_restored':
      return `memulihkan ${n} kartu dari Tempat sampah`
    default:
      return 'mengubah papan'
  }
}
