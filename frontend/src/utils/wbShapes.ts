// Bentuk dan diagram Whiteboard (W5, revisi C). Bagian murni tanpa Vue supaya bisa diuji dengan node --test: daftar
// bentuk, path SVG dari angka, port garis setelah putar, dan jalur lurus, siku, serta melengkung.

import type { PMNode } from './whiteboard.ts'
import catalog from './wbShapeCatalog.json' with { type: 'json' }

export type Pt = { x: number; y: number }
export type Box = { x: number; y: number; w: number; h: number }

// Jenis bentuk diturunkan dari tabel geometri GEO di bawah; daftar, label, ukuran, dan varian dari katalog JSON yang
// juga dibaca test Go (daftar putih server), jadi keduanya tidak bisa menyimpang diam-diam.
export type ShapeType = keyof typeof GEO

export type ShapeContent = {
  shape: ShapeType
  // Varian bentuk (revisi 54), mis. jenis event atau task BPMN; kosong berarti varian pertama di katalog.
  variant?: string
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
  // Balik mendatar dan tegak (revisi 50): hanya gambar bentuknya.
  flip_x?: boolean
  flip_y?: boolean
  doc?: PMNode | null
  name_doc?: PMNode | null
  attributes_doc?: PMNode | null
  methods_doc?: PMNode | null
}

export type ShapeVariant = { id: string; label: string }
export type ShapeInfo = {
  id: ShapeType
  label: string
  group: string
  w: number
  h: number
  variants?: ShapeVariant[]
}
export const SHAPE_GROUPS: string[] = catalog.groups
export const SHAPES = catalog.shapes as ShapeInfo[]
const INFO = new Map(SHAPES.map((s) => [s.id, s]))
export const shapeInfo = (id: string) => INFO.get(id as ShapeType)
export const shapeLabel = (id: string) => INFO.get(id as ShapeType)?.label ?? 'Bentuk'
export const FONT_SIZES = [12, 14, 16, 20, 24, 32] as const

// Entri palet: satu per bentuk, atau satu per varian untuk bentuk bervarian (mis. "Event mulai pesan").
export type PaletteEntry = {
  key: string
  shape: ShapeType
  variant?: string
  label: string
  group: string
  w: number
  h: number
}
export const PALETTE: PaletteEntry[] = SHAPES.flatMap((s): PaletteEntry[] =>
  s.variants?.length
    ? s.variants.map((v) => ({
        key: `${s.id}:${v.id}`,
        shape: s.id,
        variant: v.id,
        label: v.label,
        group: s.group,
        w: s.w,
        h: s.h,
      }))
    : [{ key: s.id, shape: s.id, label: s.label, group: s.group, w: s.w, h: s.h }],
)

export function newShapeContent(shape: ShapeType, variant?: string): ShapeContent {
  const g = GEO[shape] as Geo
  return {
    shape,
    ...(INFO.get(shape)?.variants?.length ? { variant: variant ?? INFO.get(shape)!.variants![0]!.id } : {}),
    rotation: 0,
    fill: g.fill ?? 'putih',
    stroke: 'tinta',
    stroke_width: 2,
    stroke_style: 'solid',
    text: g.text ?? '',
    align: g.align ?? 'center',
    font_size: 16,
    ...(shape === 'uml_class' ? { name: 'Kelas', attributes: '', methods: '' } : {}),
  }
}

const f = (n: number) => Math.round(n * 100) / 100
const poly = (pts: [number, number][]) => 'M' + pts.map(([x, y]) => `${f(x)} ${f(y)}`).join('L') + 'Z'

// Path bentuk dasar di kotak 0..w x 0..h, sebelum diputar; dipakai tabel GEO. Jenis tak dikenal digambar persegi.
function basePath(shape: string, w: number, h: number): string {
  switch (shape) {
    default:
    case 'rect':
    case 'process':
    case 'uml_class':
    case 'uml_system':
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
    case 'uml_usecase':
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

// ---------------------------------------------------------------------------------------------------------------
// Definisi bentuk sebagai data (revisi 54): geometri, tanda di dalam bentuk (ikon BPMN dan sejenisnya), area teks, dan
// perilaku. Satu renderer (WbShape.vue) dan ekspor (exportBoard.ts) membaca tabel yang sama; tidak ada komponen per
// bentuk. Ikon digambar sendiri dari garis dan bangun sederhana.

// Area teks di dalam kotak bentuk, sebagai nilai CSS (persen dari tinggi atau lebar, atau px). auto di bawah berarti
// tinggi mengikuti isi (judul di atas).
export type Inset = { top: string; right: string; bottom: string; left: string }
// Tanda: path di kotak bentuk yang sama. fill dan stroke: line memakai warna garis bentuk, surface warna permukaan.
export type Mark = {
  d: string
  fill?: 'line' | 'surface' | 'none'
  stroke?: 'line' | 'surface' | 'none'
  width?: number
}
type Geo = {
  path: (w: number, h: number) => string
  marks?: (w: number, h: number, variant: string) => Mark[]
  inset?: Inset
  valign?: 'top' | 'center'
  // Garis terbuka (tanpa isian), mis. aktor dan anotasi.
  open?: boolean
  // Pengali tebal garis tepi, mis. event selesai BPMN yang bergaris tebal.
  strokeScale?: number
  fill?: string
  text?: string
  align?: 'left' | 'center' | 'right'
}

const DEFAULT_INSET: Inset = { top: '8px', bottom: '8px', left: '10px', right: '10px' }
const pct = (top: number, bottom: number, left: number, right: number): Inset => ({
  top: `${top}%`,
  bottom: `${bottom}%`,
  left: `${left}%`,
  right: `${right}%`,
})
// Label di bawah bentuk kecil (event, gateway, data), seperti nama di bawah aktor.
const BELOW: Inset = { top: '104%', bottom: '-48px', left: '-80%', right: '-80%' }

const circ = (cx: number, cy: number, r: number) =>
  `M${f(cx + r)} ${f(cy)}A${f(r)} ${f(r)} 0 1 0 ${f(cx - r)} ${f(cy)}A${f(r)} ${f(r)} 0 1 0 ${f(cx + r)} ${f(cy)}Z`
const roundRect = (x: number, y: number, w: number, h: number, r: number) => {
  r = Math.min(r, w / 2, h / 2)
  return `M${f(x + r)} ${f(y)}H${f(x + w - r)}Q${f(x + w)} ${f(y)} ${f(x + w)} ${f(y + r)}V${f(y + h - r)}Q${f(x + w)} ${f(y + h)} ${f(x + w - r)} ${f(y + h)}H${f(x + r)}Q${f(x)} ${f(y + h)} ${f(x)} ${f(y + h - r)}V${f(y + r)}Q${f(x)} ${f(y)} ${f(x + r)} ${f(y)}Z`
}
// Titik relatif (satuan s) di sekitar pusat, sebagai poligon tertutup.
const rel = (cx: number, cy: number, s: number, pts: [number, number][]) =>
  poly(pts.map(([x, y]) => [cx + x * s, cy + y * s]))

// Amplop: kotak dan tutup. filled: amplop penuh warna garis dengan tutup warna permukaan (BPMN "melempar").
function envelope(x: number, y: number, w: number, h: number, filled: boolean): Mark[] {
  const box = poly([
    [x, y],
    [x + w, y],
    [x + w, y + h],
    [x, y + h],
  ])
  const flap = `M${f(x)} ${f(y)}L${f(x + w / 2)} ${f(y + h * 0.55)}L${f(x + w)} ${f(y)}`
  return filled
    ? [
        { d: box, fill: 'line' },
        { d: flap, stroke: 'surface' },
      ]
    : [{ d: box }, { d: flap }]
}

function eventMarks(w: number, h: number, variant: string, filled: boolean): Mark[] {
  const cx = w / 2
  const cy = h / 2
  const r = Math.min(w, h) / 2
  const s = r * 0.5
  const fill = filled ? 'line' : 'none'
  switch (variant) {
    case 'message':
      return envelope(cx - s, cy - s * 0.68, s * 2, s * 1.36, filled)
    case 'timer':
      return [
        { d: circ(cx, cy, r * 0.6) },
        { d: `M${f(cx)} ${f(cy - r * 0.42)}L${f(cx)} ${f(cy)}L${f(cx + r * 0.3)} ${f(cy + r * 0.12)}` },
      ]
    case 'error':
      return [
        {
          d: rel(cx, cy, s * 1.1, [
            [-0.55, 0.6],
            [-0.2, -0.5],
            [0.12, 0.12],
            [0.55, -0.6],
            [0.2, 0.5],
            [-0.12, -0.12],
          ]),
          fill,
        },
      ]
    case 'signal':
      return [
        {
          d: rel(cx, cy, s * 1.15, [
            [0, -0.65],
            [0.6, 0.45],
            [-0.6, 0.45],
          ]),
          fill,
        },
      ]
    case 'cancel':
      return [
        {
          d: `M${f(cx - s * 0.7)} ${f(cy - s * 0.7)}L${f(cx + s * 0.7)} ${f(cy + s * 0.7)}M${f(cx + s * 0.7)} ${f(cy - s * 0.7)}L${f(cx - s * 0.7)} ${f(cy + s * 0.7)}`,
          width: filled ? 2 : 1.5,
        },
      ]
    default:
      return []
  }
}

// Ikon jenis task di pojok kiri atas, kotak 18 px.
function taskIcon(variant: string): Mark[] {
  const x = 10
  const y = 10
  switch (variant) {
    case 'user':
      return [
        { d: circ(x + 9, y + 5, 4) },
        {
          d: `M${x + 2} ${y + 18}Q${x + 2} ${y + 10} ${x + 9} ${y + 10}Q${x + 16} ${y + 10} ${x + 16} ${y + 18}Z`,
        },
      ]
    case 'service': {
      const pts: [number, number][] = []
      for (let i = 0; i < 16; i++) {
        const a = (i * Math.PI) / 8
        const rr = i % 2 ? 6 : 8.5
        pts.push([x + 9 + Math.cos(a) * rr, y + 9 + Math.sin(a) * rr])
      }
      return [{ d: poly(pts) }, { d: circ(x + 9, y + 9, 2.5) }]
    }
    case 'script':
      return [
        {
          d: `M${x + 3} ${y + 1}H${x + 15}V${y + 17}H${x + 3}ZM${x + 6} ${y + 6}H${x + 12}M${x + 6} ${y + 9.5}H${x + 12}M${x + 6} ${y + 13}H${x + 12}`,
        },
      ]
    case 'manual':
      return [
        {
          d: `M${x + 2} ${y + 9}V${y + 17}H${x + 13}Q${x + 17} ${y + 17} ${x + 17} ${y + 13}V${y + 7}M${x + 5} ${y + 10}V${y + 3}M${x + 8.5} ${y + 10}V${y + 1.5}M${x + 12} ${y + 10}V${y + 3}M${x + 15} ${y + 10}V${y + 5}`,
        },
      ]
    case 'send':
      return envelope(x, y + 3, 18, 12, true)
    case 'receive':
      return envelope(x, y + 3, 18, 12, false)
    default:
      return []
  }
}

function gatewayMarks(w: number, h: number, variant: string): Mark[] {
  const cx = w / 2
  const cy = h / 2
  const s = Math.min(w, h) * 0.2
  switch (variant) {
    case 'exclusive':
      return [
        {
          d: `M${f(cx - s)} ${f(cy - s)}L${f(cx + s)} ${f(cy + s)}M${f(cx + s)} ${f(cy - s)}L${f(cx - s)} ${f(cy + s)}`,
          width: 2,
        },
      ]
    case 'parallel':
      return [
        {
          d: `M${f(cx)} ${f(cy - s * 1.2)}V${f(cy + s * 1.2)}M${f(cx - s * 1.2)} ${f(cy)}H${f(cx + s * 1.2)}`,
          width: 2,
        },
      ]
    case 'inclusive':
      return [{ d: circ(cx, cy, s * 1.15), width: 1.75 }]
    case 'event': {
      const pent: [number, number][] = []
      for (let i = 0; i < 5; i++) {
        const a = -Math.PI / 2 + (i * 2 * Math.PI) / 5
        pent.push([cx + Math.cos(a) * s * 0.75, cy + Math.sin(a) * s * 0.75])
      }
      return [{ d: circ(cx, cy, s * 1.35) }, { d: circ(cx, cy, s * 1.1) }, { d: poly(pent) }]
    }
    default:
      return []
  }
}

const base = (id: string, rest: Omit<Geo, 'path'> = {}): Geo => ({
  path: (w, h) => basePath(id, w, h),
  ...rest,
})

const GEO = {
  rect: base('rect'),
  rounded: base('rounded'),
  ellipse: base('ellipse', { inset: pct(16, 16, 14, 14) }),
  triangle: base('triangle', { inset: pct(45, 8, 24, 24) }),
  diamond: base('diamond', { inset: pct(28, 28, 24, 24) }),
  hexagon: base('hexagon'),
  star: base('star', { inset: pct(28, 28, 24, 24) }),
  process: base('process'),
  decision: base('decision', { inset: pct(28, 28, 24, 24) }),
  terminator: base('terminator'),
  data: base('data'),
  database: base('database', { inset: { top: '24%', bottom: '10%', left: '8px', right: '8px' } }),
  document: base('document'),
  uml_class: base('uml_class'),
  uml_actor: base('uml_actor', {
    open: true,
    fill: '',
    inset: { top: '102%', bottom: '-30%', left: '-40%', right: '-40%' },
  }),
  uml_note: base('uml_note', { fill: 'kuning', align: 'left' }),
  uml_package: base('uml_package', { inset: { top: '22%', bottom: '8px', left: '8px', right: '8px' } }),
  uml_system: base('uml_system', {
    fill: '',
    text: 'Sistem',
    valign: 'top',
    inset: { top: '8px', bottom: 'auto', left: '10px', right: '10px' },
  }),
  uml_usecase: base('uml_usecase', { text: 'Use case', inset: pct(16, 16, 14, 14) }),
  bpmn_start: {
    path: (w, h) => circ(w / 2, h / 2, Math.min(w, h) / 2),
    marks: (w, h, v) => eventMarks(w, h, v, false),
    inset: BELOW,
  },
  bpmn_intermediate: {
    path: (w, h) => circ(w / 2, h / 2, Math.min(w, h) / 2),
    marks: (w, h, v) => [
      { d: circ(w / 2, h / 2, Math.min(w, h) / 2 - 4) },
      ...eventMarks(w, h, v, v === 'error' || v === 'cancel'),
    ],
    inset: BELOW,
  },
  bpmn_end: {
    path: (w, h) => circ(w / 2, h / 2, Math.min(w, h) / 2),
    marks: (w, h, v) => eventMarks(w, h, v, true),
    strokeScale: 2,
    inset: BELOW,
  },
  bpmn_task: {
    path: (w, h) => roundRect(0, 0, w, h, 10),
    marks: (_w, _h, v) => taskIcon(v),
    inset: { top: '10px', bottom: '8px', left: '12px', right: '12px' },
  },
  bpmn_subprocess: {
    path: (w, h) => roundRect(0, 0, w, h, 10),
    marks: (w, h) => [
      { d: `M${f(w / 2 - 8)} ${f(h - 22)}H${f(w / 2 + 8)}V${f(h - 6)}H${f(w / 2 - 8)}Z` },
      { d: `M${f(w / 2)} ${f(h - 19)}V${f(h - 9)}M${f(w / 2 - 5)} ${f(h - 14)}H${f(w / 2 + 5)}` },
    ],
    inset: { top: '8px', bottom: '26px', left: '10px', right: '10px' },
  },
  bpmn_gateway: { path: (w, h) => basePath('diamond', w, h), marks: gatewayMarks, inset: BELOW },
  bpmn_data_object: {
    path: (w, h) => {
      const c = Math.min(14, w / 3, h / 3)
      return poly([
        [0, 0],
        [w - c, 0],
        [w, c],
        [w, h],
        [0, h],
      ])
    },
    marks: (w, h) => {
      const c = Math.min(14, w / 3, h / 3)
      return [{ d: `M${f(w - c)} 0V${f(c)}H${f(w)}` }]
    },
    inset: BELOW,
  },
  bpmn_data_store: base('database', { inset: BELOW }),
  bpmn_annotation: {
    path: (w, h) => {
      const b = Math.min(16, w * 0.15)
      return `M${f(b)} 0H0V${f(h)}H${f(b)}`
    },
    open: true,
    fill: '',
    align: 'left',
    inset: { top: '8px', bottom: '8px', left: '14px', right: '6px' },
  },
  arrow_right: base('arrow_right', { inset: pct(26, 26, 14, 14) }),
  arrow_left: base('arrow_left', { inset: pct(26, 26, 14, 14) }),
  arrow_up: base('arrow_up'),
  arrow_down: base('arrow_down'),
  arrow_both: base('arrow_both', { inset: pct(26, 26, 14, 14) }),
} satisfies Record<string, Geo>

const geoOf = (s: string): Geo => (GEO as Record<string, Geo>)[s] ?? GEO.rect
export const shapePath = (shape: string, w: number, h: number) => geoOf(shape).path(w, h)
export const shapeMarks = (c: Pick<ShapeContent, 'shape' | 'variant'>, w: number, h: number): Mark[] =>
  geoOf(c.shape).marks?.(w, h, c.variant || shapeInfo(c.shape)?.variants?.[0]?.id || '') ?? []
// Bentuk yang garisnya tidak tertutup (aktor, anotasi) digambar tanpa isian.
export const openShape = (s: string) => !!geoOf(s).open
export const strokeScale = (s: string) => geoOf(s).strokeScale ?? 1
export const shapeInset = (s: string): Inset => geoOf(s).inset ?? DEFAULT_INSET
export const shapeValign = (s: string) => geoOf(s).valign ?? 'center'
export const hasGeometry = (s: string) => s in GEO

// Kotak teks dalam angka (untuk ekspor), dari nilai CSS yang sama dengan layar.
export function textBox(s: string, w: number, h: number): Box & { valign: 'top' | 'center' } {
  const i = shapeInset(s)
  const v = (x: string, size: number) =>
    x === 'auto' ? null : x.endsWith('%') ? (parseFloat(x) / 100) * size : parseFloat(x)
  const top = v(i.top, h) ?? 0
  const left = v(i.left, w) ?? 0
  const right = v(i.right, w) ?? 0
  const bottom = v(i.bottom, h)
  return {
    x: left,
    y: top,
    w: w - left - right,
    h: bottom === null ? h - top : h - top - bottom,
    valign: shapeValign(s),
  }
}

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

// bebas (revisi 53): ruas lurus antar titik belok dengan sudut apa pun.
export type Route = 'straight' | 'orthogonal' | 'curved' | 'bebas'
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
  if (route !== 'orthogonal') return [a.p, ...points, b.p]
  if (!points.length) return orthogonalRoute(a, b)
  return orthogonalThrough(a, b, points)
}

// Segmen ab dan cd saling memotong di bagian dalam keduanya (bersentuhan di ujung tidak dihitung).
function crosses(a: Pt, b: Pt, c: Pt, d: Pt): boolean {
  const o = (p: Pt, q: Pt, r: Pt) => Math.sign((q.x - p.x) * (r.y - p.y) - (q.y - p.y) * (r.x - p.x))
  return o(a, b, c) * o(a, b, d) < 0 && o(c, d, a) * o(c, d, b) < 0
}

// Jalur siku melewati titik belok berurutan (revisi 55). Sebelumnya setiap ruas dirutekan sendiri dan ruas pertama
// selalu keluar lewat potongan port 20 px: bentuk L tiap ruas dipilih tanpa melihat ruas lain, dan titik belok yang
// dekat atau di belakang potongan itu memaksa jalur maju lalu berbalik. Hasilnya tekukan tajam yang memotong jalurnya
// sendiri, makin sering sesudah titik dipetakan elastis. Sekarang setiap ruas punya beberapa bentuk L (langsung, atau
// lewat potongan port untuk ruas pertama dan terakhir) dan pencarian beam membawa BEAM jalur terbaik ke ruas
// berikutnya. Skor: tidak berbalik arah, keluar searah port, tiba di port tujuan dari luar, tidak memotong jalur yang
// sudah ada, tidak menembus kotak ujung, lalu yang terpendek.
const BEAM = 8
function orthogonalThrough(a: End, b: End, points: Pt[]): Pt[] {
  const boxes = [a.box, b.box].filter((x): x is Box => !!x)
  const out0 = a.dir ? axisDir(a.dir) : null
  const into = b.dir ? { x: -axisDir(b.dir).x, y: -axisDir(b.dir).y } : null
  const dot = (u: Pt, v: Pt) => u.x * v.x + u.y * v.y
  const legDir = (p: Pt, q: Pt) => ({ x: q.x - p.x, y: q.y - p.y })
  const ell = (cur: Pt, q: Pt): Pt[][] =>
    [
      { x: q.x, y: cur.y },
      { x: cur.x, y: q.y },
    ].map((mid) =>
      [cur, mid, q].filter(
        (p, k, arr) => k === 0 || Math.hypot(p.x - arr[k - 1]!.x, p.y - arr[k - 1]!.y) > 0.01,
      ),
    )
  // Biaya menambah legs (diawali titik terakhir path) ke path.
  const cost = (path: Pt[], legs: Pt[], first: boolean, last: boolean) => {
    let sc = 0
    if (legs.length < 2) return sc
    const d0 = legDir(legs[0]!, legs[1]!)
    if (first && out0) sc += dot(d0, out0) < -0.01 ? 1000 : dot(d0, out0) < 0.01 ? 5 : 0
    else if (path.length > 1 && dot(d0, legDir(path[path.length - 2]!, path[path.length - 1]!)) < -0.01)
      sc += 1000
    if (last && into) {
      const dl = legDir(legs[legs.length - 2]!, legs[legs.length - 1]!)
      sc += dot(dl, into) < -0.01 ? 1000 : dot(dl, into) < 0.01 ? 200 : 0
    }
    for (let k = 1; k < legs.length; k++) {
      if (k > 1 && dot(legDir(legs[k - 2]!, legs[k - 1]!), legDir(legs[k - 1]!, legs[k]!)) < -0.01) sc += 1000
      for (let m = 1; m < path.length - 1; m++)
        if (crosses(legs[k - 1]!, legs[k]!, path[m - 1]!, path[m]!)) sc += 100
      for (let m = 1; m < k - 1; m++) if (crosses(legs[k - 1]!, legs[k]!, legs[m - 1]!, legs[m]!)) sc += 100
      for (const r of boxes) if (segmentHitsBox(legs[k - 1]!, legs[k]!, r)) sc += 50
    }
    return sc + pathLength(legs) * 0.001
  }
  let beam: { path: Pt[]; score: number }[] = [{ path: [a.p], score: 0 }]
  const stops = [...points, b.p]
  stops.forEach((q, ti) => {
    const first = ti === 0
    const last = ti === stops.length - 1
    const next: { path: Pt[]; score: number }[] = []
    for (const st of beam) {
      const cur = st.path.at(-1)!
      let cands = ell(cur, q)
      if (first && a.dir) cands = cands.concat(ell(stub(a), q).map((l) => [cur, ...l]))
      if (last && b.dir) cands = cands.concat(ell(cur, stub(b)).map((l) => [...l, b.p]))
      if (first && last && a.dir && b.dir)
        cands = cands.concat(ell(stub(a), stub(b)).map((l) => [cur, ...l, b.p]))
      for (const c of cands)
        next.push({ path: [...st.path, ...c.slice(1)], score: st.score + cost(st.path, c, first, last) })
    }
    next.sort((x, y) => x.score - y.score)
    beam = next.slice(0, BEAM)
  })
  return dedupe(beam[0]!.path)
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
  geometry: {
    from: ConnEnd
    to: ConnEnd
    route: Route
    points: [number, number][]
    anchor?: [[number, number], [number, number]]
  }
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
  // Titik belok tersimpan relatif terhadap pusat ujung saat disimpan (anchor); titik draf sudah di posisi tampil.
  const an = c.geometry.anchor
  const raw = (points ?? c.geometry.points ?? []).map(([x, y]) => ({ x, y }))
  const pts =
    !points && an ? elastic(raw, { x: an[0][0], y: an[0][1] }, { x: an[1][0], y: an[1][1] }, ca, cb) : raw
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

// ---------------------------------------------------------------------------------------------------------------
// Garis elastis dan titik belok (revisi 53)

// Memetakan titik belok dari pasangan ujung lama ke pasangan ujung baru dengan transformasi kemiripan (geser, putar,
// skala seragam), satu-satunya pemetaan affine yang ditentukan dua pasang titik. Bentuk garis terjaga: titik dekat
// ujung yang diam hampir tidak bergerak. Ujung lama yang berimpit hanya digeser rata-rata perpindahan kedua ujung.
export function elastic(points: Pt[], a0: Pt, b0: Pt, a1: Pt, b1: Pt): Pt[] {
  const vx = b0.x - a0.x
  const vy = b0.y - a0.y
  const len = vx * vx + vy * vy
  if (len < 1) {
    const dx = (a1.x - a0.x + b1.x - b0.x) / 2
    const dy = (a1.y - a0.y + b1.y - b0.y) / 2
    return points.map((p) => ({ x: p.x + dx, y: p.y + dy }))
  }
  const ux = b1.x - a1.x
  const uy = b1.y - a1.y
  // s = u / v sebagai bilangan kompleks.
  const sr = (ux * vx + uy * vy) / len
  const si = (uy * vx - ux * vy) / len
  return points.map((p) => {
    const qx = p.x - a0.x
    const qy = p.y - a0.y
    return { x: a1.x + sr * qx - si * qy, y: a1.y + si * qx + sr * qy }
  })
}

// Titik belok yang segaris dengan tetangganya (jarak tegak lurus ke ruas tetangga di bawah tol) dibuang. stops:
// ujung awal, titik belok, ujung akhir; yang dikembalikan hanya titik beloknya.
export function simplifyBends(stops: Pt[], tol = 2): Pt[] {
  const out = [...stops]
  for (let i = 1; i < out.length - 1;) {
    const a = out[i - 1]!
    const p = out[i]!
    const b = out[i + 1]!
    const dx = b.x - a.x
    const dy = b.y - a.y
    const l = Math.hypot(dx, dy)
    const dist =
      l < 0.01 ? Math.hypot(p.x - a.x, p.y - a.y) : Math.abs(dx * (a.y - p.y) - dy * (a.x - p.x)) / l
    const t = l < 0.01 ? 0 : ((p.x - a.x) * dx + (p.y - a.y) * dy) / (l * l)
    if (dist < tol && t > 0 && t < 1) out.splice(i, 1)
    else if (dist < tol && l < 0.01) out.splice(i, 1)
    else i++
  }
  return out.slice(1, -1)
}

// Shift saat menyeret titik belok: sudut dari titik sebelumnya dikunci ke kelipatan 45 derajat, panjangnya tetap
// proyeksi ke arah itu.
export function snap45(from: Pt, p: Pt): Pt {
  const dx = p.x - from.x
  const dy = p.y - from.y
  const step = Math.PI / 4
  const ang = Math.round(Math.atan2(dy, dx) / step) * step
  const ux = Math.round(Math.cos(ang) * 1e9) / 1e9
  const uy = Math.round(Math.sin(ang) * 1e9) / 1e9
  const k = (dx * ux + dy * uy) / (ux * ux + uy * uy)
  return { x: from.x + ux * k, y: from.y + uy * k }
}

// Terakhir dipakai di palet: yang baru di depan, tanpa duplikat, paling banyak max.
export function pushRecent(list: string[], key: string, max = 12): string[] {
  return [key, ...list.filter((k) => k !== key)].slice(0, max)
}

// Preset garis per jenis diagram (revisi 53 dan 54). text: label tengah yang diisi preset (tetap bisa diubah).
export type LinePreset = {
  id: string
  label: string
  group: string
  line: 'solid' | 'dashed' | 'dotted'
  start: EndKind
  end: EndKind
  text?: string
}
export const LINE_PRESETS: LinePreset[] = [
  {
    id: 'include',
    label: 'Include',
    group: 'Use case',
    line: 'dashed',
    start: 'none',
    end: 'open_arrow',
    text: '«include»',
  },
  {
    id: 'extend',
    label: 'Extend',
    group: 'Use case',
    line: 'dashed',
    start: 'none',
    end: 'open_arrow',
    text: '«extend»',
  },
  { id: 'sequence_flow', label: 'Sequence flow', group: 'BPMN', line: 'solid', start: 'none', end: 'arrow' },
  {
    id: 'message_flow',
    label: 'Message flow',
    group: 'BPMN',
    line: 'dashed',
    start: 'circle',
    end: 'open_arrow',
  },
  { id: 'association', label: 'Association', group: 'BPMN', line: 'dotted', start: 'none', end: 'none' },
]
