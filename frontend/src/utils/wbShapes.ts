// Bentuk dan diagram Whiteboard (W5, revisi C). Bagian murni tanpa Vue supaya bisa diuji dengan node --test: daftar
// bentuk, path SVG dari angka, port garis setelah putar, dan jalur lurus, siku, serta melengkung.

import type { PMNode } from './whiteboard.ts'

export type Pt = { x: number; y: number }
export type Box = { x: number; y: number; w: number; h: number }

export type ShapeType =
  | 'rect'
  | 'rounded'
  | 'ellipse'
  | 'triangle'
  | 'diamond'
  | 'hexagon'
  | 'star'
  | 'process'
  | 'decision'
  | 'terminator'
  | 'data'
  | 'database'
  | 'document'
  | 'uml_class'
  | 'uml_actor'
  | 'uml_note'
  | 'uml_package'
  | 'arrow_right'
  | 'arrow_left'
  | 'arrow_up'
  | 'arrow_down'
  | 'arrow_both'

export type ShapeContent = {
  shape: ShapeType
  rotation: number
  fill: string
  stroke: string
  stroke_width: number
  stroke_style: 'solid' | 'dashed'
  text: string
  align: 'left' | 'center' | 'right'
  font_size: number
  name?: string
  attributes?: string
  methods?: string
  doc?: PMNode | null
  name_doc?: PMNode | null
  attributes_doc?: PMNode | null
  methods_doc?: PMNode | null
}

export const SHAPE_GROUPS = ['Dasar', 'Flowchart', 'UML', 'Panah blok'] as const
export const SHAPES: {
  id: ShapeType
  label: string
  group: (typeof SHAPE_GROUPS)[number]
  w: number
  h: number
}[] = [
  { id: 'rect', label: 'Persegi', group: 'Dasar', w: 160, h: 100 },
  { id: 'rounded', label: 'Persegi sudut bulat', group: 'Dasar', w: 160, h: 100 },
  { id: 'ellipse', label: 'Elips', group: 'Dasar', w: 160, h: 100 },
  { id: 'triangle', label: 'Segitiga', group: 'Dasar', w: 140, h: 120 },
  { id: 'diamond', label: 'Belah ketupat', group: 'Dasar', w: 140, h: 120 },
  { id: 'hexagon', label: 'Segi enam', group: 'Dasar', w: 160, h: 120 },
  { id: 'star', label: 'Bintang', group: 'Dasar', w: 140, h: 140 },
  { id: 'process', label: 'Proses', group: 'Flowchart', w: 180, h: 80 },
  { id: 'decision', label: 'Keputusan', group: 'Flowchart', w: 180, h: 120 },
  { id: 'terminator', label: 'Mulai atau selesai', group: 'Flowchart', w: 180, h: 64 },
  { id: 'data', label: 'Data', group: 'Flowchart', w: 180, h: 80 },
  { id: 'database', label: 'Database', group: 'Flowchart', w: 120, h: 140 },
  { id: 'document', label: 'Dokumen', group: 'Flowchart', w: 180, h: 100 },
  { id: 'uml_class', label: 'Kelas', group: 'UML', w: 220, h: 180 },
  { id: 'uml_actor', label: 'Aktor', group: 'UML', w: 80, h: 160 },
  { id: 'uml_note', label: 'Catatan UML', group: 'UML', w: 180, h: 100 },
  { id: 'uml_package', label: 'Paket', group: 'UML', w: 220, h: 160 },
  { id: 'arrow_right', label: 'Panah kanan', group: 'Panah blok', w: 180, h: 100 },
  { id: 'arrow_left', label: 'Panah kiri', group: 'Panah blok', w: 180, h: 100 },
  { id: 'arrow_up', label: 'Panah atas', group: 'Panah blok', w: 100, h: 180 },
  { id: 'arrow_down', label: 'Panah bawah', group: 'Panah blok', w: 100, h: 180 },
  { id: 'arrow_both', label: 'Panah dua arah', group: 'Panah blok', w: 200, h: 100 },
]
export const FONT_SIZES = [12, 14, 16, 20, 24, 32] as const
export const shapeLabel = (id: string) => SHAPES.find((s) => s.id === id)?.label ?? 'Bentuk'

export function newShapeContent(shape: ShapeType): ShapeContent {
  return {
    shape,
    rotation: 0,
    fill: shape === 'uml_actor' ? '' : shape === 'uml_note' ? 'kuning' : 'putih',
    stroke: 'tinta',
    stroke_width: 2,
    stroke_style: 'solid',
    text: '',
    align: shape === 'uml_note' ? 'left' : 'center',
    font_size: 16,
    ...(shape === 'uml_class' ? { name: 'Kelas', attributes: '', methods: '' } : {}),
  }
}

const f = (n: number) => Math.round(n * 100) / 100
const poly = (pts: [number, number][]) => 'M' + pts.map(([x, y]) => `${f(x)} ${f(y)}`).join('L') + 'Z'

// Path bentuk di kotak 0..w x 0..h, sebelum diputar. Kelas UML dan paket punya garis pemisah sendiri (shapeExtras).
export function shapePath(shape: ShapeType, w: number, h: number): string {
  switch (shape) {
    case 'rect':
    case 'process':
    case 'uml_class':
      return poly([
        [0, 0],
        [w, 0],
        [w, h],
        [0, h],
      ])
    case 'rounded': {
      const r = Math.min(16, w / 4, h / 4)
      return `M${r} 0H${f(w - r)}Q${f(w)} 0 ${f(w)} ${r}V${f(h - r)}Q${f(w)} ${f(h)} ${f(w - r)} ${f(h)}H${r}Q0 ${f(h)} 0 ${f(h - r)}V${r}Q0 0 ${r} 0Z`
    }
    case 'terminator': {
      const r = Math.min(h / 2, w / 2)
      return `M${f(r)} 0H${f(w - r)}A${f(r)} ${f(h / 2)} 0 0 1 ${f(w - r)} ${f(h)}H${f(r)}A${f(r)} ${f(h / 2)} 0 0 1 ${f(r)} 0Z`
    }
    case 'ellipse':
      return `M0 ${f(h / 2)}A${f(w / 2)} ${f(h / 2)} 0 1 0 ${f(w)} ${f(h / 2)}A${f(w / 2)} ${f(h / 2)} 0 1 0 0 ${f(h / 2)}Z`
    case 'triangle':
      return poly([
        [w / 2, 0],
        [w, h],
        [0, h],
      ])
    case 'diamond':
    case 'decision':
      return poly([
        [w / 2, 0],
        [w, h / 2],
        [w / 2, h],
        [0, h / 2],
      ])
    case 'hexagon': {
      const d = Math.min(w / 4, h / 2)
      return poly([
        [d, 0],
        [w - d, 0],
        [w, h / 2],
        [w - d, h],
        [d, h],
        [0, h / 2],
      ])
    }
    case 'star': {
      const pts: [number, number][] = []
      for (let i = 0; i < 10; i++) {
        const a = -Math.PI / 2 + (i * Math.PI) / 5
        const r = i % 2 === 0 ? 0.5 : 0.2
        pts.push([w / 2 + Math.cos(a) * w * r, h / 2 + Math.sin(a) * h * r + h * 0.04])
      }
      return poly(pts)
    }
    case 'data': {
      const d = Math.min(w / 5, 24)
      return poly([
        [d, 0],
        [w, 0],
        [w - d, h],
        [0, h],
      ])
    }
    case 'database': {
      const ry = Math.min(h / 6, 16)
      return `M0 ${ry}A${f(w / 2)} ${ry} 0 0 1 ${f(w)} ${ry}V${f(h - ry)}A${f(w / 2)} ${ry} 0 0 1 0 ${f(h - ry)}Z`
    }
    case 'document': {
      const wave = Math.min(h / 6, 14)
      return `M0 0H${f(w)}V${f(h - wave)}Q${f(w * 0.75)} ${f(h - wave * 2.5)} ${f(w / 2)} ${f(h - wave)}T0 ${f(h - wave)}Z`
    }
    case 'uml_note': {
      const c = Math.min(18, w / 4, h / 4)
      return poly([
        [0, 0],
        [w - c, 0],
        [w, c],
        [w, h],
        [0, h],
      ])
    }
    case 'uml_package': {
      const tabW = Math.min(w * 0.4, 90)
      const tabH = Math.min(h * 0.18, 22)
      return `M0 0H${f(tabW)}V${f(tabH)}H${f(w)}V${f(h)}H0Z`
    }
    case 'uml_actor': {
      const cx = w / 2
      const head = Math.min(w, h) * 0.18
      const neck = head * 2
      const hip = h * 0.62
      return `M${f(cx + head)} ${f(head)}A${f(head)} ${f(head)} 0 1 0 ${f(cx - head)} ${f(head)}A${f(head)} ${f(head)} 0 1 0 ${f(cx + head)} ${f(head)}ZM${f(cx)} ${f(neck)}V${f(hip)}M${f(w * 0.1)} ${f(h * 0.35)}H${f(w * 0.9)}M${f(cx)} ${f(hip)}L${f(w * 0.12)} ${f(h)}M${f(cx)} ${f(hip)}L${f(w * 0.88)} ${f(h)}`
    }
    case 'arrow_right':
    case 'arrow_left':
    case 'arrow_both': {
      const head = Math.min(w * 0.35, h)
      const t = h * 0.25
      const right: [number, number][] = [
        [0, t],
        [w - head, t],
        [w - head, 0],
        [w, h / 2],
        [w - head, h],
        [w - head, h - t],
        [0, h - t],
      ]
      if (shape === 'arrow_right') return poly(right)
      if (shape === 'arrow_left') return poly(right.map(([x, y]) => [w - x, y]))
      const hd = Math.min(w * 0.3, h)
      return poly([
        [0, h / 2],
        [hd, 0],
        [hd, t],
        [w - hd, t],
        [w - hd, 0],
        [w, h / 2],
        [w - hd, h],
        [w - hd, h - t],
        [hd, h - t],
        [hd, h],
      ])
    }
    case 'arrow_up':
    case 'arrow_down': {
      const head = Math.min(h * 0.35, w)
      const t = w * 0.25
      const down: [number, number][] = [
        [t, 0],
        [w - t, 0],
        [w - t, h - head],
        [w, h - head],
        [w / 2, h],
        [0, h - head],
        [t, h - head],
      ]
      return poly(shape === 'arrow_down' ? down : down.map(([x, y]) => [x, h - y]))
    }
  }
}

// Bentuk yang garisnya tidak tertutup (aktor) digambar tanpa isian.
export const openShape = (s: ShapeType) => s === 'uml_actor'

// ---------------------------------------------------------------------------------------------------------------
// Port dan putar

export type Port =
  'auto' | 'top' | 'right' | 'bottom' | 'left' | 'top_left' | 'top_right' | 'bottom_left' | 'bottom_right'
export const SIDE_PORTS: Port[] = ['top', 'right', 'bottom', 'left']
export const CORNER_PORTS: Port[] = ['top_left', 'top_right', 'bottom_left', 'bottom_right']
// Sudut hanya untuk bentuk (rancangan W5); kartu biasa hanya punya port di tengah keempat sisi.
export const portsFor = (isShape: boolean): Port[] =>
  isShape ? [...SIDE_PORTS, ...CORNER_PORTS] : SIDE_PORTS

const PORT_UNIT: Record<Exclude<Port, 'auto'>, [number, number]> = {
  top: [0.5, 0],
  right: [1, 0.5],
  bottom: [0.5, 1],
  left: [0, 0.5],
  top_left: [0, 0],
  top_right: [1, 0],
  bottom_left: [0, 1],
  bottom_right: [1, 1],
}

export function rotatePoint(p: Pt, c: Pt, deg: number): Pt {
  if (!deg) return p
  const a = (deg * Math.PI) / 180
  const cos = Math.cos(a)
  const sin = Math.sin(a)
  const dx = p.x - c.x
  const dy = p.y - c.y
  return { x: c.x + dx * cos - dy * sin, y: c.y + dx * sin + dy * cos }
}

export function center(b: Box): Pt {
  return { x: b.x + b.w / 2, y: b.y + b.h / 2 }
}

// Titik port di dunia setelah bentuk diputar terhadap pusatnya, beserta arah keluarnya (vektor satuan).
export function portPoint(b: Box, port: Exclude<Port, 'auto'>, rotation = 0): { p: Pt; dir: Pt } {
  const [u, v] = PORT_UNIT[port]
  const c = center(b)
  const raw = { x: b.x + u * b.w, y: b.y + v * b.h }
  const p = rotatePoint(raw, c, rotation)
  const len = Math.hypot(raw.x - c.x, raw.y - c.y) || 1
  const d0 = { x: (raw.x - c.x) / len, y: (raw.y - c.y) / len }
  // Port sisi keluar tegak lurus sisinya, bukan menuju pusat, supaya garis siku keluar lurus dari sisi itu.
  const side =
    port === 'top'
      ? { x: 0, y: -1 }
      : port === 'bottom'
        ? { x: 0, y: 1 }
        : port === 'left'
          ? { x: -1, y: 0 }
          : port === 'right'
            ? { x: 1, y: 0 }
            : d0
  const dir = rotatePoint(side, { x: 0, y: 0 }, rotation)
  return { p, dir }
}

// Port "auto": sisi yang paling menghadap titik lain.
export function autoPort(b: Box, toward: Pt, rotation = 0): Exclude<Port, 'auto'> {
  let best: Exclude<Port, 'auto'> = 'right'
  let score = -Infinity
  const c = center(b)
  const len = Math.hypot(toward.x - c.x, toward.y - c.y) || 1
  const to = { x: (toward.x - c.x) / len, y: (toward.y - c.y) / len }
  for (const port of SIDE_PORTS as Exclude<Port, 'auto'>[]) {
    const { dir } = portPoint(b, port, rotation)
    const s = dir.x * to.x + dir.y * to.y
    if (s > score) {
      score = s
      best = port
    }
  }
  return best
}

// Kotak batas setelah diputar (dipakai jalur siku untuk menghindari bentuk).
export function rotatedBounds(b: Box, rotation = 0): Box {
  if (!rotation) return b
  const c = center(b)
  const pts = [
    { x: b.x, y: b.y },
    { x: b.x + b.w, y: b.y },
    { x: b.x, y: b.y + b.h },
    { x: b.x + b.w, y: b.y + b.h },
  ].map((p) => rotatePoint(p, c, rotation))
  const xs = pts.map((p) => p.x)
  const ys = pts.map((p) => p.y)
  return {
    x: Math.min(...xs),
    y: Math.min(...ys),
    w: Math.max(...xs) - Math.min(...xs),
    h: Math.max(...ys) - Math.min(...ys),
  }
}

// ---------------------------------------------------------------------------------------------------------------
// Jalur garis

export type Route = 'straight' | 'orthogonal' | 'curved'
export type End = { p: Pt; dir: Pt | null; box: Box | null }
export const STUB = 20

function axisDir(d: Pt): Pt {
  return Math.abs(d.x) >= Math.abs(d.y) ? { x: Math.sign(d.x) || 1, y: 0 } : { x: 0, y: Math.sign(d.y) || 1 }
}

// Segmen sejajar sumbu menembus bagian dalam kotak (tepi tidak dihitung).
export function segmentHitsBox(a: Pt, b: Pt, r: Box): boolean {
  const eps = 0.5
  const left = r.x + eps
  const right = r.x + r.w - eps
  const top = r.y + eps
  const bottom = r.y + r.h - eps
  if (a.y === b.y) {
    if (a.y <= top || a.y >= bottom) return false
    return Math.max(a.x, b.x) > left && Math.min(a.x, b.x) < right
  }
  if (a.x === b.x) {
    if (a.x <= left || a.x >= right) return false
    return Math.max(a.y, b.y) > top && Math.min(a.y, b.y) < bottom
  }
  // Segmen miring (hanya di jalur lurus): uji sampel cukup untuk pemilihan rute.
  for (let t = 0.05; t < 1; t += 0.05) {
    const x = a.x + (b.x - a.x) * t
    const y = a.y + (b.y - a.y) * t
    if (x > left && x < right && y > top && y < bottom) return true
  }
  return false
}

function pathLength(pts: Pt[]): number {
  let n = 0
  for (let i = 1; i < pts.length; i++) n += Math.hypot(pts[i]!.x - pts[i - 1]!.x, pts[i]!.y - pts[i - 1]!.y)
  return n
}

function dedupe(pts: Pt[]): Pt[] {
  const out: Pt[] = []
  for (const p of pts) {
    const q = out.at(-1)
    if (q && Math.abs(q.x - p.x) < 0.01 && Math.abs(q.y - p.y) < 0.01) continue
    out.push(p)
  }
  // Titik di tengah garis lurus dibuang supaya jumlah belokan jujur.
  return out.filter((p, i) => {
    const a = out[i - 1]
    const b = out[i + 1]
    if (!a || !b) return true
    return !((a.x === p.x && p.x === b.x) || (a.y === p.y && p.y === b.y))
  })
}

// Jalur siku dari a ke b yang keluar lurus dari port, lalu memilih rute terpendek (belokan paling sedikit bila
// sama) yang tidak menembus kotak asal dan tujuan. Rute keliling dicoba bila rute langsung menembus.
// Potongan awal keluar dari port searah sumbu sampai lewat tepi kotak batas ditambah STUB. Pada bentuk yang diputar
// port ada di dalam kotak batas putarnya, jadi STUB saja belum keluar.
function stub(e: End): Pt {
  if (!e.dir) return e.p
  const d = axisDir(e.dir)
  const r = e.box
  let k = STUB
  if (r) {
    if (d.x > 0) k = Math.max(k, r.x + r.w - e.p.x + STUB)
    if (d.x < 0) k = Math.max(k, e.p.x - r.x + STUB)
    if (d.y > 0) k = Math.max(k, r.y + r.h - e.p.y + STUB)
    if (d.y < 0) k = Math.max(k, e.p.y - r.y + STUB)
  }
  return { x: e.p.x + d.x * k, y: e.p.y + d.y * k }
}

export function orthogonalRoute(a: End, b: End): Pt[] {
  const a1 = stub(a)
  const b1 = stub(b)
  const boxes = [a.box, b.box].filter((x): x is Box => !!x)
  const mx = (a1.x + b1.x) / 2
  const my = (a1.y + b1.y) / 2
  const candidates: Pt[][] = [
    [a1, { x: b1.x, y: a1.y }, b1],
    [a1, { x: a1.x, y: b1.y }, b1],
    [a1, { x: mx, y: a1.y }, { x: mx, y: b1.y }, b1],
    [a1, { x: a1.x, y: my }, { x: b1.x, y: my }, b1],
  ]
  if (boxes.length) {
    const minX = Math.min(...boxes.map((r) => r.x)) - STUB
    const maxX = Math.max(...boxes.map((r) => r.x + r.w)) + STUB
    const minY = Math.min(...boxes.map((r) => r.y)) - STUB
    const maxY = Math.max(...boxes.map((r) => r.y + r.h)) + STUB
    for (const X of [minX, maxX]) candidates.push([a1, { x: X, y: a1.y }, { x: X, y: b1.y }, b1])
    for (const Y of [minY, maxY]) candidates.push([a1, { x: a1.x, y: Y }, { x: b1.x, y: Y }, b1])
  }
  let best: Pt[] | null = null
  let bestScore = Infinity
  for (const c of candidates) {
    const mid = dedupe(c)
    let clear = true
    for (let i = 1; i < mid.length && clear; i++)
      for (const r of boxes) if (segmentHitsBox(mid[i - 1]!, mid[i]!, r)) clear = false
    const score = pathLength(mid) + mid.length * 4 + (clear ? 0 : 1e6)
    if (score < bestScore) {
      bestScore = score
      best = mid
    }
  }
  return dedupe([a.p, ...(best ?? [a1, b1]), b.p])
}

// Titik jalur untuk garis lurus dan siku. Titik belok pengguna (points) dilewati berurutan; di jalur siku setiap
// bagian antar titik dibuat siku.
export function routePoints(a: End, b: End, route: Route, points: Pt[] = []): Pt[] {
  if (route === 'straight' || route === 'curved') return [a.p, ...points, b.p]
  if (!points.length) return orthogonalRoute(a, b)
  const out: Pt[] = []
  const stops: End[] = [a, ...points.map((p) => ({ p, dir: null, box: null })), b]
  for (let i = 1; i < stops.length; i++) {
    const seg = orthogonalRoute(stops[i - 1]!, stops[i]!)
    out.push(...(i === 1 ? seg : seg.slice(1)))
  }
  return dedupe(out)
}

// Path SVG. Jalur melengkung memakai Bezier kubik dari arah port; dengan titik belok, kurva halus melewati titiknya.
export function routePath(a: End, b: End, route: Route, points: Pt[] = []): { d: string; pts: Pt[] } {
  const pts = routePoints(a, b, route, points)
  if (route !== 'curved') return { d: 'M' + pts.map((p) => `${f(p.x)} ${f(p.y)}`).join('L'), pts }
  if (!points.length) {
    const k = Math.max(40, Math.hypot(b.p.x - a.p.x, b.p.y - a.p.y) / 3)
    const c1 = a.dir ? { x: a.p.x + a.dir.x * k, y: a.p.y + a.dir.y * k } : a.p
    const c2 = b.dir ? { x: b.p.x + b.dir.x * k, y: b.p.y + b.dir.y * k } : b.p
    const d = `M${f(a.p.x)} ${f(a.p.y)}C${f(c1.x)} ${f(c1.y)} ${f(c2.x)} ${f(c2.y)} ${f(b.p.x)} ${f(b.p.y)}`
    return { d, pts: [a.p, c1, c2, b.p] }
  }
  // Catmull-Rom ke Bezier: kurva melewati setiap titik.
  let d = `M${f(pts[0]!.x)} ${f(pts[0]!.y)}`
  for (let i = 0; i < pts.length - 1; i++) {
    const p0 = pts[i - 1] ?? pts[i]!
    const p1 = pts[i]!
    const p2 = pts[i + 1]!
    const p3 = pts[i + 2] ?? p2
    const c1 = { x: p1.x + (p2.x - p0.x) / 6, y: p1.y + (p2.y - p0.y) / 6 }
    const c2 = { x: p2.x - (p3.x - p1.x) / 6, y: p2.y - (p3.y - p1.y) / 6 }
    d += `C${f(c1.x)} ${f(c1.y)} ${f(c2.x)} ${f(c2.y)} ${f(p2.x)} ${f(p2.y)}`
  }
  return { d, pts }
}

// Titik di sepanjang polyline pada pecahan t (0..1) dari panjangnya, untuk label.
export function pointAlong(pts: Pt[], t: number): Pt {
  const total = pathLength(pts)
  let left = total * t
  for (let i = 1; i < pts.length; i++) {
    const a = pts[i - 1]!
    const b = pts[i]!
    const seg = Math.hypot(b.x - a.x, b.y - a.y)
    if (left <= seg && seg > 0)
      return { x: a.x + ((b.x - a.x) * left) / seg, y: a.y + ((b.y - a.y) * left) / seg }
    left -= seg
  }
  return pts.at(-1) ?? { x: 0, y: 0 }
}

// Ujung garis: path di titik tip dengan arah datang dari `from`. filled false berarti hanya garis tepi (isi surface).
export type EndKind = 'none' | 'arrow' | 'open_arrow' | 'triangle' | 'diamond' | 'diamond_open' | 'circle'
export const END_KINDS: { id: EndKind; label: string }[] = [
  { id: 'none', label: 'Tanpa ujung' },
  { id: 'arrow', label: 'Panah' },
  { id: 'open_arrow', label: 'Panah terbuka' },
  { id: 'triangle', label: 'Segitiga kosong' },
  { id: 'diamond', label: 'Belah ketupat' },
  { id: 'diamond_open', label: 'Belah ketupat kosong' },
  { id: 'circle', label: 'Lingkaran' },
]
export function endMarker(
  kind: EndKind,
  tip: Pt,
  from: Pt,
): { d: string; fill: 'line' | 'surface' | 'none' } | null {
  if (kind === 'none') return null
  const len = Math.hypot(tip.x - from.x, tip.y - from.y) || 1
  const ux = (tip.x - from.x) / len
  const uy = (tip.y - from.y) / len
  const at = (back: number, side: number) => ({
    x: tip.x - ux * back - uy * side,
    y: tip.y - uy * back + ux * side,
  })
  const P = (p: Pt) => `${f(p.x)} ${f(p.y)}`
  switch (kind) {
    case 'arrow':
      return { d: `M${P(tip)}L${P(at(12, 6))}L${P(at(12, -6))}Z`, fill: 'line' }
    case 'triangle':
      return { d: `M${P(tip)}L${P(at(14, 7))}L${P(at(14, -7))}Z`, fill: 'surface' }
    case 'open_arrow':
      return { d: `M${P(at(12, 6))}L${P(tip)}L${P(at(12, -6))}`, fill: 'none' }
    case 'diamond':
    case 'diamond_open':
      return {
        d: `M${P(tip)}L${P(at(9, 5))}L${P(at(18, 0))}L${P(at(9, -5))}Z`,
        fill: kind === 'diamond' ? 'line' : 'surface',
      }
    case 'circle': {
      const c = at(5, 0)
      return {
        d: `M${P({ x: c.x + 5, y: c.y })}A5 5 0 1 0 ${P({ x: c.x - 5, y: c.y })}A5 5 0 1 0 ${P({ x: c.x + 5, y: c.y })}Z`,
        fill: 'surface',
      }
    }
  }
}

// ---------------------------------------------------------------------------------------------------------------
// Ubah ukuran dan putar

export type Handle = 'n' | 's' | 'e' | 'w' | 'ne' | 'nw' | 'se' | 'sw'
export const HANDLES: Handle[] = ['nw', 'n', 'ne', 'e', 'se', 's', 'sw', 'w']
const HANDLE_SIGN: Record<Handle, [number, number]> = {
  n: [0, -1],
  s: [0, 1],
  e: [1, 0],
  w: [-1, 0],
  ne: [1, -1],
  nw: [-1, -1],
  se: [1, 1],
  sw: [-1, 1],
}
export const handleSign = (h: Handle) => HANDLE_SIGN[h]

// Ubah ukuran dari satu pegangan dengan geser dunia d. Pada bentuk yang diputar, geser diproyeksikan ke sumbu lokal
// bentuk, dan titik di seberang pegangan tetap di tempatnya. keepRatio (Shift) menjaga perbandingan sisi.
export function resizeBox(
  b: Box,
  rotation: number,
  handle: Handle,
  d: Pt,
  keepRatio: boolean,
  min: number,
  max: number,
): Box {
  const [sx, sy] = HANDLE_SIGN[handle]
  const local = rotatePoint(d, { x: 0, y: 0 }, -rotation)
  let w = sx ? b.w + sx * local.x : b.w
  let h = sy ? b.h + sy * local.y : b.h
  if (keepRatio && b.w > 0 && b.h > 0) {
    const k = sx && sy ? Math.max(w / b.w, h / b.h) : sx ? w / b.w : h / b.h
    w = b.w * k
    h = b.h * k
  }
  w = Math.min(max, Math.max(min, w))
  h = Math.min(max, Math.max(min, h))
  const c = center(b)
  const fixed = rotatePoint({ x: c.x - (sx * b.w) / 2, y: c.y - (sy * b.h) / 2 }, c, rotation)
  const nc = { x: fixed.x, y: fixed.y }
  const off = rotatePoint({ x: (sx * w) / 2, y: (sy * h) / 2 }, { x: 0, y: 0 }, rotation)
  nc.x += off.x
  nc.y += off.y
  return { x: nc.x - w / 2, y: nc.y - h / 2, w, h }
}

// Sudut putar dari posisi penunjuk terhadap pusat bentuk: 0 berarti pegangan tepat di atas pusat. snap (Shift)
// membulatkan ke kelipatan 15 derajat. Hasil selalu 0 sampai 359.
export function angleFrom(c: Pt, p: Pt, snap: boolean): number {
  let deg = (Math.atan2(p.y - c.y, p.x - c.x) * 180) / Math.PI + 90
  if (snap) deg = Math.round(deg / 15) * 15
  deg = Math.round(deg)
  return ((deg % 360) + 360) % 360
}

// ---------------------------------------------------------------------------------------------------------------
// Satu garis penghubung lengkap, dipakai lapisan garis, pegangan diagram, dan ekspor supaya jalurnya selalu sama.

export type ConnEnd = { port: string; x?: number; y?: number }
export type ConnLike = {
  id: string
  from: string | null
  to: string | null
  geometry: { from: ConnEnd; to: ConnEnd; route: Route; points: [number, number][] }
  style: { start: EndKind; end: EndKind }
}
export type Routed = {
  id: string
  d: string
  pts: Pt[]
  start: Pt
  end: Pt
  heads: { d: string; fill: 'line' | 'surface' | 'none' }[]
  mid: Pt
  near: { start: Pt; end: Pt }
  // Titik henti untuk pegangan titik belok: ujung awal, titik belok, ujung akhir.
  stops: Pt[]
}

function resolveEnd(
  card: string | null,
  e: ConnEnd,
  geom: Map<string, Box>,
  rot: Map<string, number>,
  toward: Pt,
): End | null {
  if (!card)
    return e.x !== undefined && e.y !== undefined ? { p: { x: e.x, y: e.y }, dir: null, box: null } : null
  const b = geom.get(card)
  if (!b) return null
  const r = rot.get(card) ?? 0
  const port = !e.port || e.port === 'auto' ? autoPort(b, toward, r) : (e.port as Exclude<Port, 'auto'>)
  const { p, dir } = portPoint(b, port, r)
  return { p, dir, box: rotatedBounds(b, r) }
}

function centerOf(card: string | null, e: ConnEnd, geom: Map<string, Box>): Pt | null {
  if (!card) return e.x !== undefined && e.y !== undefined ? { x: e.x, y: e.y } : null
  const b = geom.get(card)
  return b ? center(b) : null
}

export function routeConnector(
  c: ConnLike,
  geom: Map<string, Box>,
  rot: Map<string, number>,
  points?: [number, number][],
): Routed | null {
  const ca = centerOf(c.from, c.geometry.from, geom)
  const cb = centerOf(c.to, c.geometry.to, geom)
  if (!ca || !cb) return null
  const pts = (points ?? c.geometry.points ?? []).map(([x, y]) => ({ x, y }))
  const a = resolveEnd(c.from, c.geometry.from, geom, rot, pts[0] ?? cb)
  const b = resolveEnd(c.to, c.geometry.to, geom, rot, pts.at(-1) ?? ca)
  if (!a || !b) return null
  const route = c.geometry.route || 'straight'
  const { d, pts: line } = routePath(a, b, route, pts)
  // Arah ujung: dari titik kendali atau titik jalur terdekat ke ujungnya.
  const beforeEnd = line[line.length - 2] ?? a.p
  const afterStart = line[1] ?? b.p
  const heads = [endMarker(c.style.end, b.p, beforeEnd), endMarker(c.style.start, a.p, afterStart)].filter(
    (h): h is NonNullable<typeof h> => !!h,
  )
  const poly = route === 'curved' ? [a.p, ...pts, b.p] : line
  const total = pathLength(poly) || 1
  const at = (dist: number) => pointAlong(poly, Math.min(0.45, dist / total))
  const back = (dist: number) => pointAlong(poly, Math.max(0.55, 1 - dist / total))
  return {
    id: c.id,
    d,
    pts: line,
    start: a.p,
    end: b.p,
    heads,
    mid: pointAlong(poly, 0.5),
    near: { start: at(28), end: back(28) },
    stops: [a.p, ...pts, b.p],
  }
}
