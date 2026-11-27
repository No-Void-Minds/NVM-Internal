// Logika murni Whiteboard (docs/whiteboard.md): tipe data, geometri kanvas, navigasi keyboard antar kartu, dan teks
// riwayat. Tanpa Vue dan tanpa klien API, supaya bisa diuji dengan node --test.

// Ekstensi .ts: modul ini juga diimpor test Node (tsconfig.node.json, resolusi node16).
import type { ShapeContent } from './wbShapes.ts'
import { inlineRuns, type Run } from './wbRich.ts'
import type { CardStyle } from './wbStyle.ts'
import { DEFAULT_PERSON, personLabel, personLines, type PersonContent, type PersonInfo } from './wbPerson.ts'

export type { CardStyle, PersonContent, PersonInfo }

export type CardKind =
  | 'note'
  | 'heading'
  | 'checklist'
  | 'board'
  | 'image'
  | 'file'
  | 'link'
  | 'column'
  | 'drawing'
  | 'person'
  | 'table'
  | 'shape'
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
// doc, title_doc, docs: pasangan berformat (revisi 49); teks polosnya tetap di text atau title dan dihasilkan server.
export type HeadingContent = { text: string; level: 1 | 2; doc?: PMNode | null }
export type ChecklistItem = { id: string; text: string; done: boolean; doc?: PMNode | null }
export type ChecklistContent = { title: string; title_doc?: PMNode | null; items: ChecklistItem[] }

// Papan anak yang ditunjuk kartu jenis board.
export type BoardRef = { id: string; title: string; version: number; boards: number; cards: number }
export type BoardContent = Record<string, never>
// W3: isi kartu gambar dan berkas hanya keterangan; berkasnya di field file.
// crop: potongan dan posisi zoom gambar di bingkai kartu (revisi 50), pecahan 0 sampai 1 dari sisi berkas asli.
export type Crop = { x: number; y: number; w: number; h: number }
export type CaptionContent = { caption: string; crop?: Crop | null }
export type LinkContent = { url: string; preview_id: string }
export type ColumnContent = { title: string; title_doc?: PMNode | null }
// Goresan objek gambar bebas (revisi A): piksel dunia bilangan bulat relatif ke pojok kotak batas objek, x,y berpasangan.
export type Stroke = { color: string; width: number; points: number[] }
export type DrawingContent = { strokes: Stroke[] }
export type TableContent = {
  title?: string
  title_doc?: PMNode | null
  header: boolean
  rows: string[][]
  docs?: (PMNode | null)[][]
}
export type FileInfo = {
  id: string
  name: string
  mime: string
  size: number
  thumb: boolean
  width: number
  height: number
}
export type PreviewInfo = {
  id: string
  title: string
  description: string
  site: string
  final_url: string
  status: 'ok' | 'blocked' | 'failed'
  image: string | null
}
// Model garis sudah memuat kebutuhan diagram W5: ujung bebas, port, jalur, titik belok, tujuh jenis ujung, tiga label.
export type ConnectorEndKind =
  | 'none'
  | 'arrow'
  | 'open_arrow'
  | 'triangle'
  | 'diamond'
  | 'diamond_open'
  | 'circle'
  | 'cf_one'
  | 'cf_many'
  | 'cf_zero_one'
  | 'cf_one_many'
  | 'cf_zero_many'
export type ConnectorStyle = {
  start: ConnectorEndKind
  end: ConnectorEndKind
  line: 'solid' | 'dashed' | 'dotted'
  color: CardColor | ''
}
export type ConnectorEnd = { port: string; x?: number; y?: number }
export type ConnectorGeometry = {
  from: ConnectorEnd
  to: ConnectorEnd
  route: 'straight' | 'orthogonal' | 'curved' | 'bebas' | 'message'
  points: [number, number][]
  // Pusat ujung awal dan akhir saat titik belok disimpan (revisi 53, garis elastis).
  anchor?: [[number, number], [number, number]]
}
export type Connector = {
  id: string
  board_id: string
  from: string | null
  to: string | null
  geometry: ConnectorGeometry
  style: ConnectorStyle
  label: string
  label_start: string
  label_end: string
  version: number
  seq: number
  // Urutan tumpuk manual (revisi 53); null berarti otomatis di atas kartu ujung yang paling depan.
  z: number | null
}

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
  content:
    | NoteContent
    | HeadingContent
    | ChecklistContent
    | BoardContent
    | CaptionContent
    | LinkContent
    | ColumnContent
    | DrawingContent
    | TableContent
    | ShapeContent
    | PersonContent
  ref?: BoardRef
  file?: FileInfo
  preview?: PreviewInfo
  // Kartu di dalam kolom: posisinya dihitung dari kolom (columnLayout), x dan y tersimpan diabaikan.
  column_id?: string | null
  col_order?: number | null
  comments?: number
  created_by_name?: string
  // Revisi 50: gaya kartu, kunci posisi, dan jalur grup dari terluar ke terdalam.
  style?: CardStyle
  locked?: boolean
  group_ids?: string[]
  // Revisi 51: data profil terkini untuk kartu orang, dari server (tidak disimpan di isi kartu).
  person?: PersonInfo
  version: number
  content_version: number
  seq: number
  updated_by: number
  updated_at: string
}

export type Rect = { x: number; y: number; w: number; h: number }
export type Viewport = { x: number; y: number; zoom: number }

export const GRID = 8
export const MIN_ZOOM = 0.1
export const MAX_ZOOM = 4
export const MIN_SIZE = 40
export const MAX_SIZE = 4000
export const COORD_LIMIT = 1_000_000

export const defaultSize: Record<CardKind, { w: number; h: number }> = {
  note: { w: 240, h: 120 },
  heading: { w: 320, h: 64 },
  checklist: { w: 260, h: 120 },
  board: { w: 200, h: 120 },
  image: { w: 240, h: 160 },
  file: { w: 240, h: 96 },
  link: { w: 280, h: 120 },
  column: { w: 280, h: 200 },
  drawing: { w: 320, h: 320 },
  table: { w: 360, h: 120 },
  shape: { w: 160, h: 100 },
  person: { w: 260, h: 82 },
}

export const kindLabel: Record<CardKind, string> = {
  note: 'Catatan',
  heading: 'Judul',
  checklist: 'Checklist',
  board: 'Papan',
  image: 'Gambar',
  file: 'Berkas',
  link: 'Tautan',
  column: 'Kolom',
  drawing: 'Gambar bebas',
  table: 'Tabel',
  shape: 'Bentuk',
  person: 'Orang',
}

// Batas gambar bebas dan tabel, sama dengan content_w3.go. Klien menyederhanakan goresan sebelum mengirim.
export const DRAWING_SPACE = 4000
export const MAX_STROKES = 300
export const MAX_STROKE_POINTS = 2000
export const MAX_DRAWING_POINTS = 6000
export const PEN_WIDTHS = [2, 4, 8] as const
export const MAX_TABLE_ROWS = 100
export const MAX_TABLE_COLS = 20

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

// Jarak gulir event wheel dalam piksel. Firefox sering mengirim baris (deltaMode 1) atau halaman (2); tanpa
// normalisasi satu tingkat roda di sana hanya bernilai sekitar 3.
export function wheelPixels(
  e: { deltaX: number; deltaY: number; deltaMode: number },
  pageHeight: number,
): { dx: number; dy: number } {
  const k = e.deltaMode === 1 ? 16 : e.deltaMode === 2 ? pageHeight : 1
  return { dx: e.deltaX * k, dy: e.deltaY * k }
}

// Faktor zoom eksponensial terhadap jarak gulir: satu tingkat roda mouse (sekitar 100 px) kira-kira 22%, pinch
// touchpad (delta kecil berkali-kali) halus. Satu event dibatasi 240 px supaya roda cepat tidak melompat jauh.
export function wheelZoomFactor(dy: number): number {
  return Math.exp(-clamp(dy, -240, 240) * 0.0025)
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
// top dan bottom: jarak yang dikosongkan di atas dan bawah. Di ponsel bar pilihan menempati bagian atas kanvas dan
// kontrol zoom serta tombol Tambah bagian bawahnya, jadi kartu tidak boleh ditaruh di bawah keduanya; tanpa itu
// kartu yang baru diketuk langsung tertutup bar pilihannya sendiri.
export function fitView(rects: Rect[], width: number, height: number, top = 48, bottom = 48): Viewport {
  const b = bounds(rects)
  if (!b || width <= 0 || height <= 0) return { x: 48, y: top, zoom: 1 }
  const pad = 48
  const room = Math.max(1, height - top - bottom)
  const zoom = clamp(Math.min((width - pad * 2) / b.w, room / b.h, 1), MIN_ZOOM, MAX_ZOOM)
  return {
    zoom,
    x: (width - b.w * zoom) / 2 - b.x * zoom,
    y: top + (room - b.h * zoom) / 2 - b.y * zoom,
  }
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
  const comments = card.comments ? `, ${card.comments} komentar` : ''
  return kindText(card) + comments
}

function kindText(card: Card): string {
  switch (card.kind) {
    case 'board':
      return `Papan ${card.ref?.title ?? ''}, ${refSummary(card.ref)}`
    case 'note': {
      const t = excerpt(plainText((card.content as NoteContent).doc))
      return t ? `Catatan: ${t}` : 'Catatan kosong'
    }
    case 'heading': {
      const t = (card.content as HeadingContent).text
      return t ? `Judul: ${t}` : 'Judul kosong'
    }
    case 'image':
    case 'file': {
      const cap = (card.content as CaptionContent).caption
      return `${kindLabel[card.kind]} ${card.file?.name ?? ''}${cap ? `, ${cap}` : ''}`
    }
    case 'link': {
      const p = card.preview
      return `Tautan: ${p?.status === 'ok' && p.title ? p.title : (card.content as LinkContent).url}`
    }
    case 'column':
      return `Kolom ${(card.content as ColumnContent).title || 'tanpa judul'}`
    case 'drawing':
      return `Gambar bebas oleh ${card.created_by_name || 'anggota tim'}, ${(card.content as DrawingContent).strokes.length} goresan`
    case 'table': {
      const t = card.content as TableContent
      return `Tabel ${t.title ? `${t.title}, ` : ''}${t.rows.length} baris, ${t.rows[0]?.length ?? 0} kolom`
    }
    case 'shape': {
      const t = shapeText(card.content as ShapeContent)
      return t ? `Bentuk: ${excerpt(t)}` : 'Bentuk tanpa teks'
    }
    case 'person':
      return personLabel(card.content as PersonContent, card.person)
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
  if (kind === 'image' || kind === 'file') return { caption: '' }
  if (kind === 'link') return { url: '', preview_id: '' }
  if (kind === 'column') return { title: '' }
  if (kind === 'drawing') return { strokes: [] }
  if (kind === 'shape')
    return {
      shape: 'rect',
      rotation: 0,
      fill: 'putih',
      stroke: 'tinta',
      stroke_width: 2,
      stroke_style: 'solid',
      text: '',
      align: 'center',
      font_size: 16,
    }
  if (kind === 'person') return { user_id: 0, ...DEFAULT_PERSON }
  if (kind === 'table')
    return {
      header: true,
      rows: [
        ['', ''],
        ['', ''],
      ],
    }
  return { title: '', items: [] }
}

type Placed = Rect & {
  id: string
  kind: string
  z: number
  column_id?: string | null
  col_order?: number | null
}

// Tata letak kolom: isi kolom ditumpuk di bawah judulnya dengan jarak tetap, selebar kolom, dan kolom memanjang
// mengikuti isinya. Mengembalikan posisi tampil setiap isi kolom dan tinggi tampil setiap kolom.
export const COLUMN_HEAD = 56
export const COLUMN_GAP = 8
export function columnLayout(cards: Placed[]): {
  children: Map<string, Rect & { z: number }>
  heights: Map<string, number>
} {
  const cols = new Map(cards.filter((c) => c.kind === 'column').map((c) => [c.id, c]))
  const groups = new Map<string, Placed[]>()
  for (const c of cards) {
    if (!c.column_id || !cols.has(c.column_id)) continue
    groups.set(c.column_id, [...(groups.get(c.column_id) ?? []), c])
  }
  const children = new Map<string, Rect & { z: number }>()
  const heights = new Map<string, number>()
  for (const [id, col] of cols) {
    const list = (groups.get(id) ?? []).sort(
      (a, b) => (a.col_order ?? 0) - (b.col_order ?? 0) || (a.id < b.id ? -1 : 1),
    )
    let y = col.y + COLUMN_HEAD
    for (const c of list) {
      children.set(c.id, {
        x: col.x + COLUMN_GAP,
        y,
        w: Math.max(MIN_SIZE, col.w - COLUMN_GAP * 2),
        h: c.h,
        z: col.z,
      })
      y += c.h + COLUMN_GAP
    }
    heights.set(id, Math.max(col.h, y - col.y))
  }
  return { children, heights }
}

// Urutan baru di kolom untuk kartu yang dilepas di titik y: di antara dua tetangga, sebelum yang pertama, atau
// sesudah yang terakhir.
export function columnOrderAt(
  siblings: { y: number; h: number; col_order?: number | null }[],
  y: number,
): number {
  const list = [...siblings].sort((a, b) => a.y - b.y)
  if (list.length === 0) return 1
  const idx = list.findIndex((s) => y < s.y + s.h / 2)
  if (idx === 0) return (list[0]!.col_order ?? 0) - 1
  if (idx < 0) return (list.at(-1)!.col_order ?? 0) + 1
  return ((list[idx - 1]!.col_order ?? 0) + (list[idx]!.col_order ?? 0)) / 2
}

// Titik tempat garis dari pusat kotak ke arah titik lain keluar dari tepi kotak, supaya garis penghubung mulai
// dan berakhir di tepi kartu, bukan di tengahnya.
export function edgePoint(r: Rect, toward: { x: number; y: number }): { x: number; y: number } {
  const cx = r.x + r.w / 2
  const cy = r.y + r.h / 2
  const dx = toward.x - cx
  const dy = toward.y - cy
  if (dx === 0 && dy === 0) return { x: cx, y: cy }
  const t = Math.min(
    dx === 0 ? Infinity : r.w / 2 / Math.abs(dx),
    dy === 0 ? Infinity : r.h / 2 / Math.abs(dy),
  )
  return { x: cx + dx * t, y: cy + dy * t }
}

// Penyederhanaan Ramer-Douglas-Peucker untuk goresan (pasangan x,y datar). Ujung goresan selalu dipertahankan.
export function simplify(points: number[], tolerance: number): number[] {
  const n = Math.floor(points.length / 2)
  if (n <= 2) return points.slice(0, n * 2).map(Math.round)
  const keep = new Uint8Array(n)
  keep[0] = 1
  keep[n - 1] = 1
  const stack: [number, number][] = [[0, n - 1]]
  while (stack.length) {
    const [a, b] = stack.pop()!
    const ax = points[a * 2]!
    const ay = points[a * 2 + 1]!
    const bx = points[b * 2]!
    const by = points[b * 2 + 1]!
    const len = Math.hypot(bx - ax, by - ay)
    let worst = -1
    let dist = 0
    for (let i = a + 1; i < b; i++) {
      const px = points[i * 2]!
      const py = points[i * 2 + 1]!
      const d =
        len === 0
          ? Math.hypot(px - ax, py - ay)
          : Math.abs((bx - ax) * (ay - py) - (ax - px) * (by - ay)) / len
      if (d > dist) {
        dist = d
        worst = i
      }
    }
    if (worst >= 0 && dist > tolerance) {
      keep[worst] = 1
      stack.push([a, worst], [worst, b])
    }
  }
  const out: number[] = []
  for (let i = 0; i < n; i++)
    if (keep[i]) out.push(Math.round(points[i * 2]!), Math.round(points[i * 2 + 1]!))
  return out
}

// Goresan terdekat dari titik dunia (x, y) di antara objek gambar, dalam jarak tol. Jarak diukur ke ruas garis,
// ditambah setengah ketebalan pena, supaya klik tepat di garis tebal tetap kena.
export function hitStroke(
  objects: { id: string; x: number; y: number; content: DrawingContent }[],
  x: number,
  y: number,
  tol: number,
): { id: string; index: number } | null {
  let best: { id: string; index: number } | null = null
  let bestD = Infinity
  for (const o of objects) {
    o.content.strokes.forEach((s, index) => {
      const p = s.points
      const reach = tol + s.width / 2
      for (let i = 0; i + 1 < p.length; i += 2) {
        const ax = o.x + p[i]!
        const ay = o.y + p[i + 1]!
        const bx = i + 3 < p.length ? o.x + p[i + 2]! : ax
        const by = i + 3 < p.length ? o.y + p[i + 3]! : ay
        const len2 = (bx - ax) ** 2 + (by - ay) ** 2
        const t = len2 === 0 ? 0 : clamp(((x - ax) * (bx - ax) + (y - ay) * (by - ay)) / len2, 0, 1)
        const d = Math.hypot(x - (ax + t * (bx - ax)), y - (ay + t * (by - ay)))
        if (d <= reach && d < bestD) {
          bestD = d
          best = { id: o.id, index }
        }
      }
    })
  }
  return best
}

// Kotak batas goresan berkoordinat dunia, sama dengan perhitungan server (setengah tebal plus 2 px), untuk memutuskan
// kapan sesi pena memulai objek baru.
export function strokesBox(strokes: Stroke[]): Rect | null {
  let x1 = Infinity
  let y1 = Infinity
  let x2 = -Infinity
  let y2 = -Infinity
  for (const s of strokes) {
    const pad = s.width / 2 + 2
    for (let i = 0; i + 1 < s.points.length; i += 2) {
      x1 = Math.min(x1, s.points[i]! - pad)
      x2 = Math.max(x2, s.points[i]! + pad)
      y1 = Math.min(y1, s.points[i + 1]! - pad)
      y2 = Math.max(y2, s.points[i + 1]! + pad)
    }
  }
  return x1 === Infinity ? null : { x: x1, y: y1, w: x2 - x1, h: y2 - y1 }
}

export function drawingPoints(c: DrawingContent): number {
  return c.strokes.reduce((s, k) => s + k.points.length / 2, 0)
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
      return typeof s.template === 'string'
        ? `menambah papan dari template "${s.template}"`
        : `menambah ${label}${ex}`
    case 'board_template':
      return s.on ? 'menjadikan papan ini template' : 'melepas papan ini dari daftar template'
    case 'board_from_template':
      return `membuat papan ini dari template "${String(s.template ?? '')}"`
    case 'card_edited':
      return `mengubah isi ${label}${ex}`
    case 'cards_arranged':
      return `mengatur letak, ukuran, atau warna ${n} kartu`
    case 'cards_deleted':
      return `memindahkan ${n} kartu ke Tempat sampah${typeof s.boards === 'number' && s.boards > 0 ? `, termasuk ${s.boards} papan` : ''}`
    case 'cards_restored':
      return `memulihkan ${n} kartu dari Tempat sampah`
    case 'cards_moved_out':
      return `memindahkan ${n} kartu ke papan "${String(s.board ?? '')}"`
    case 'cards_moved_in':
      return `memindahkan ${n} kartu ke papan ini`
    case 'connector_created':
      return 'menambah garis penghubung'
    case 'connector_deleted':
      return 'memindahkan garis penghubung ke Tempat sampah'
    case 'comment_added':
      return `mengomentari kartu${ex}`
    case 'card_locked':
      return `mengunci posisi ${label}`
    case 'card_unlocked':
      return `membuka kunci posisi ${label}`
    case 'file_replaced':
      return `mengganti berkas ${label}${ex}`
    default:
      return 'mengubah papan'
  }
}

// Di bawah zoom ini isi kartu tidak terbaca, jadi kartu hanya menampilkan judul ringkas dan warnanya (rancangan
// bagian 7, level of detail).
export const LOD_ZOOM = 0.4

// Judul ringkas satu kartu untuk level of detail, Tampilan Daftar, dan ekspor.
export function cardTitle(card: Card): string {
  switch (card.kind) {
    case 'note':
      return excerpt(plainText((card.content as NoteContent).doc), 80) || 'Catatan kosong'
    case 'heading':
      return (card.content as HeadingContent).text || 'Judul kosong'
    case 'checklist':
      return (card.content as ChecklistContent).title || 'Checklist tanpa judul'
    case 'board':
      return card.ref?.title ?? 'Papan'
    case 'image':
    case 'file':
      return (card.content as CaptionContent).caption || card.file?.name || kindLabel[card.kind]
    case 'link': {
      const p = card.preview
      if (p?.status === 'ok' && p.title) return p.title
      const url = (card.content as LinkContent).url
      try {
        return new URL(url).host
      } catch {
        return url
      }
    }
    case 'column':
      return (card.content as ColumnContent).title || 'Kolom tanpa judul'
    case 'drawing':
      return 'Gambar bebas'
    case 'table':
      return (card.content as TableContent).title || 'Tabel'
    case 'shape':
      return excerpt(shapeText(card.content as ShapeContent).split('\n')[0] ?? '', 80) || 'Bentuk tanpa teks'
    case 'person':
      return card.person?.name ?? 'Pengguna tidak ditemukan'
  }
}

// Teks bentuk; kelas UML memakai nama, atribut, dan metodenya.
export function shapeText(c: ShapeContent): string {
  if (c.shape === 'uml_class') return [c.name, c.attributes, c.methods].filter(Boolean).join('\n')
  return c.text
}

// Baris teks kartu untuk ekspor dan Tampilan Daftar. text: teks polos baris itu; runs: teks yang sama dengan format
// huruf (revisi 49); rule: garis pemisah.
export type TextLine = {
  text: string
  size: number
  bold?: boolean
  muted?: boolean
  runs?: Run[]
  align?: string
  rule?: boolean
}
type LineBase = { size: number; bold?: boolean; muted?: boolean }

function inlineText(node: PMNode): string {
  if (node.type === 'text') return node.text ?? ''
  if (node.type === 'hardBreak') return '\n'
  return (node.content ?? []).map(inlineText).join('')
}

function docLines(node: PMNode, out: TextLine[], prefix: string, base: LineBase = { size: 14 }) {
  const kids = node.content ?? []
  const line = (l: LineBase) => {
    const align = typeof node.attrs?.textAlign === 'string' ? node.attrs.textAlign : undefined
    out.push({
      ...l,
      text: prefix + inlineText(node),
      runs: [...(prefix ? [{ text: prefix }] : []), ...inlineRuns(node)],
      ...(align ? { align } : {}),
    })
  }
  switch (node.type) {
    case 'paragraph':
      line(base)
      return
    case 'heading': {
      const level = Number(node.attrs?.level ?? 1)
      const k = level === 1 ? 20 / 14 : level === 2 ? 17 / 14 : 15 / 14
      line({ ...base, size: Math.round(base.size * k), bold: true })
      return
    }
    case 'horizontalRule':
      out.push({ text: '', size: base.size, rule: true })
      return
    case 'bulletList':
    case 'orderedList': {
      const start = Number(node.attrs?.start ?? 1)
      kids.forEach((item, i) =>
        (item.content ?? []).forEach((block, j) =>
          docLines(block, out, j > 0 ? '   ' : node.type === 'bulletList' ? '• ' : `${start + i}. `, base),
        ),
      )
      return
    }
    default:
      kids.forEach((k) => docLines(k, out, prefix, base))
  }
}

// Field berformat (revisi 49): dokumennya bila ada, selain itu teks polosnya.
export function fieldLines(
  text: string,
  doc: PMNode | null | undefined,
  base: LineBase,
  prefix = '',
): TextLine[] {
  if (!doc) return [{ ...base, text: prefix + text }]
  const out: TextLine[] = []
  docLines(doc, out, '', base)
  if (prefix && out[0]) {
    out[0].text = prefix + out[0].text
    out[0].runs = [{ text: prefix }, ...(out[0].runs ?? [])]
  }
  return out
}

// Baris teks kartu untuk ekspor PNG dan PDF, dengan format huruf (revisi 49). Gambar, sel tabel, bentuk, dan goresan
// digambar terpisah oleh penggambar ekspor.
export function cardLines(card: Card): TextLine[] {
  switch (card.kind) {
    case 'note': {
      const out: TextLine[] = []
      docLines((card.content as NoteContent).doc, out, '')
      return out
    }
    case 'heading': {
      const h = card.content as HeadingContent
      return fieldLines(h.text, h.doc, { size: h.level === 2 ? 18 : 22, bold: true })
    }
    case 'checklist': {
      const c = card.content as ChecklistContent
      return [
        ...(c.title ? fieldLines(c.title, c.title_doc, { size: 15, bold: true }) : []),
        ...c.items.flatMap((i) =>
          fieldLines(i.text, i.doc, { size: 14, muted: i.done }, `${i.done ? '☑' : '☐'} `),
        ),
      ]
    }
    case 'board':
      return [
        { text: card.ref?.title ?? '', size: 15, bold: true },
        { text: refSummary(card.ref), size: 13, muted: true },
      ]
    case 'image':
    case 'file': {
      const cap = (card.content as CaptionContent).caption
      const name = card.kind === 'file' ? [{ text: card.file?.name ?? '', size: 14, bold: true }] : []
      return [...name, ...(cap ? [{ text: cap, size: 13 }] : [])]
    }
    case 'link': {
      const p = card.preview?.status === 'ok' ? card.preview : undefined
      return [
        { text: cardTitle(card), size: 15, bold: true },
        ...(p?.description ? [{ text: p.description, size: 13 }] : []),
        { text: (card.content as LinkContent).url, size: 12, muted: true },
      ]
    }
    case 'column': {
      const c = card.content as ColumnContent
      return fieldLines(cardTitle(card), c.title ? c.title_doc : null, { size: 15, bold: true })
    }
    case 'table': {
      const t = card.content as TableContent
      return t.title ? fieldLines(t.title, t.title_doc, { size: 15, bold: true }) : []
    }
    case 'drawing':
      return []
    case 'shape':
      // Bentuk digambar penggambar ekspor sendiri (path dan teks di dalamnya).
      return []
    case 'person':
      // Foto digambar penggambar ekspor; baris teksnya di sini (nama tebal).
      return personLines(card.content as PersonContent, card.person).map((text, i) => ({
        text,
        size: i === 0 ? 15 : 13,
        bold: i === 0,
        muted: i > 0,
      }))
  }
}
