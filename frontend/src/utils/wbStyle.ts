// Gaya kartu Whiteboard (revisi 50). Nilai dari daftar yang sama dengan server (backend/internal/whiteboard/style.go)
// dan dirender sebagai kelas CSS statis di style.css (wb-st-*), bukan atribut style, supaya nama token CSS tidak
// pernah dirakit dinamis. Opacity satu-satunya nilai angka, dipasang lewat CSSOM (:style), yang tidak diatur CSP.

export type CardStyle = {
  bg?: 'transparan'
  border?: 'tanpa' | 'tipis' | 'sedang' | 'tebal'
  border_color?: string
  border_style?: 'putus' | 'titik'
  radius?: 'siku' | 'kecil' | 'sedang' | 'bulat'
  shadow?: 'tanpa' | 'halus' | 'tegas'
  opacity?: number
  padding?: 'rapat' | 'normal' | 'lega'
}

export type StyleKey = keyof CardStyle

// Bagian gaya yang berlaku per jenis kartu. Bentuk punya isian dan garis sendiri (panel Gaya bentuk); gambar tanpa
// latar dan jarak dalam; gambar bebas hanya transparansi.
const ALL: StyleKey[] = [
  'bg',
  'border',
  'border_color',
  'border_style',
  'radius',
  'shadow',
  'opacity',
  'padding',
]
export function styleKeys(kind: string): StyleKey[] {
  if (kind === 'shape') return ['shadow', 'opacity']
  if (kind === 'image') return ['border', 'border_color', 'border_style', 'radius', 'shadow', 'opacity']
  if (kind === 'drawing') return ['opacity']
  return ALL
}

export const STYLE_OPTIONS = {
  bg: [
    { value: '', label: 'Warna kartu' },
    { value: 'transparan', label: 'Tanpa latar' },
  ],
  border: [
    { value: 'tanpa', label: 'Tanpa' },
    { value: '', label: 'Tipis' },
    { value: 'sedang', label: 'Sedang' },
    { value: 'tebal', label: 'Tebal' },
  ],
  border_style: [
    { value: '', label: 'Penuh' },
    { value: 'putus', label: 'Putus-putus' },
    { value: 'titik', label: 'Titik-titik' },
  ],
  radius: [
    { value: 'siku', label: 'Siku' },
    { value: '', label: 'Kecil' },
    { value: 'sedang', label: 'Sedang' },
    { value: 'bulat', label: 'Bulat' },
  ],
  shadow: [
    { value: 'tanpa', label: 'Tanpa' },
    { value: '', label: 'Halus' },
    { value: 'tegas', label: 'Tegas' },
  ],
  padding: [
    { value: 'rapat', label: 'Rapat' },
    { value: '', label: 'Normal' },
    { value: 'lega', label: 'Lega' },
  ],
} as const

// Kelas statis per nilai; nilai di luar daftar tidak menghasilkan kelas.
const classMap: Record<string, Record<string, string>> = {
  bg: { transparan: 'wb-st-clear' },
  border: { tanpa: 'wb-st-b0', sedang: 'wb-st-b2', tebal: 'wb-st-b4' },
  border_color: {
    tinta: 'wb-st-bc-tinta',
    abu: 'wb-st-bc-abu',
    hijau: 'wb-st-bc-hijau',
    kuning: 'wb-st-bc-kuning',
    biru: 'wb-st-bc-biru',
    krem: 'wb-st-bc-krem',
    merah: 'wb-st-bc-merah',
  },
  border_style: { putus: 'wb-st-dashed', titik: 'wb-st-dotted' },
  radius: { siku: 'wb-st-r0', sedang: 'wb-st-r2', bulat: 'wb-st-r3' },
  shadow: { tanpa: 'wb-st-sh0', tegas: 'wb-st-sh2' },
  padding: { rapat: 'wb-st-p1', lega: 'wb-st-p3' },
}

export function styleClasses(style: CardStyle | undefined, kind: string): string[] {
  if (!style) return []
  const out: string[] = []
  for (const k of styleKeys(kind)) {
    const v = style[k]
    if (k === 'opacity' || v === undefined) continue
    const c = classMap[k]?.[String(v)]
    if (c) out.push(c)
  }
  return out
}

export function styleOpacity(style: CardStyle | undefined, kind: string): number | undefined {
  const o = style?.opacity
  return o && o >= 25 && o < 100 && styleKeys(kind).includes('opacity') ? o / 100 : undefined
}

// normalize membuang nilai bawaan, sama dengan server, supaya gaya yang sama dibandingkan dengan isi yang sama.
export function normalize(style: CardStyle): CardStyle {
  const out: CardStyle = {}
  for (const [k, v] of Object.entries(style) as [StyleKey, unknown][]) {
    if (v === '' || v === undefined || v === null) continue
    if ((k === 'opacity' && v === 100) || (k === 'radius' && v === 'kecil')) continue
    if (
      (k === 'padding' && v === 'normal') ||
      (k === 'shadow' && v === 'halus') ||
      (k === 'border' && v === 'tipis')
    )
      continue
    ;(out as Record<string, unknown>)[k] = v
  }
  if (out.border === 'tanpa') {
    delete out.border_color
    delete out.border_style
  }
  return out
}

// pasteStyle: gaya salinan diterapkan hanya pada bagian yang berlaku untuk jenis kartu tujuan; bagian lain tetap.
export function pasteStyle(target: CardStyle | undefined, copied: CardStyle, kind: string): CardStyle {
  const keys = styleKeys(kind)
  const out: CardStyle = { ...(target ?? {}) }
  for (const k of keys) {
    const v = copied[k]
    if (v === undefined) delete out[k]
    else (out as Record<string, unknown>)[k] = v
  }
  return normalize(out)
}
