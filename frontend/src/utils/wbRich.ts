import type { PMNode } from './whiteboard.ts'

// Format teks Whiteboard (revisi 49) tanpa Vue dan TipTap, supaya ekspor dan uji node --test bisa memakainya. Daftar
// ini satu-satunya sumber nama warna, stabilo, dan ukuran di klien; server memakai daftar yang sama
// (backend/internal/whiteboard/content.go). Token dipakai ekspor; layar memakai kelas di style.css.

export const TEXT_COLORS = [
  { value: 'abu', label: 'Abu', token: '--color-ink-muted' },
  { value: 'merah', label: 'Merah', token: '--color-danger' },
  { value: 'oranye', label: 'Oranye', token: '--color-collab-5' },
  { value: 'coklat', label: 'Cokelat', token: '--color-high' },
  { value: 'hijau', label: 'Hijau', token: '--color-collab-8' },
  { value: 'biru', label: 'Biru', token: '--color-collab-2' },
  { value: 'ungu', label: 'Ungu', token: '--color-collab-4' },
  { value: 'merah_muda', label: 'Merah muda', token: '--color-collab-7' },
] as const
export const HIGHLIGHTS = [
  { value: 'kuning', label: 'Kuning', token: '--color-ready-bg' },
  { value: 'hijau', label: 'Hijau', token: '--color-progress-bg' },
  { value: 'biru', label: 'Biru', token: '--color-completed-bg' },
  { value: 'merah', label: 'Merah', token: '--color-danger-bg' },
  { value: 'abu', label: 'Abu', token: '--color-requested-bg' },
] as const
export const FONT_SCALE: Record<string, number> = { kecil: 0.8125, besar: 1.25, sangat_besar: 1.625 }

export const colorSet = new Set<string>(TEXT_COLORS.map((c) => c.value))
export const highSet = new Set<string>(HIGHLIGHTS.map((c) => c.value))
export const sizeSet = new Set<string>(['kecil', 'besar', 'sangat_besar'])
export const alignSet = new Set<string>(['center', 'right', 'justify'])

// Nama kelas dari nilai daftar; nilai di luar daftar tidak menghasilkan kelas.
export function colorClass(v: unknown) {
  return typeof v === 'string' && colorSet.has(v) ? `wb-c-${v}` : ''
}
export function highlightClass(v: unknown) {
  return typeof v === 'string' && highSet.has(v) ? `wb-h-${v}` : ''
}
export function sizeClass(v: unknown) {
  return typeof v === 'string' && sizeSet.has(v) ? `wb-fs-${v}` : ''
}
export function alignClass(v: unknown) {
  return typeof v === 'string' && alignSet.has(v) ? `wb-al-${v}` : ''
}
export type Run = {
  text: string
  bold?: boolean
  italic?: boolean
  underline?: boolean
  strike?: boolean
  code?: boolean
  sup?: boolean
  sub?: boolean
  // Nama token warna teks dan stabilo, bukan nilai warna.
  color?: string
  highlight?: string
  scale?: number
}

const colorToken = new Map<string, string>(TEXT_COLORS.map((c) => [c.value, c.token]))
const highlightToken = new Map<string, string>(HIGHLIGHTS.map((c) => [c.value, c.token]))

// Run teks berformat dari isi paragraf atau judul; hardBreak menjadi baris baru di dalam run.
export function inlineRuns(node: PMNode): Run[] {
  const out: Run[] = []
  for (const n of node.content ?? []) {
    if (n.type === 'hardBreak') {
      out.push({ text: '\n' })
      continue
    }
    if (n.type !== 'text' || !n.text) continue
    const r: Run = { text: n.text }
    for (const m of n.marks ?? []) {
      const a = m.attrs ?? {}
      switch (m.type) {
        case 'bold':
          r.bold = true
          break
        case 'italic':
          r.italic = true
          break
        case 'underline':
        case 'link':
          r.underline = true
          break
        case 'strike':
          r.strike = true
          break
        case 'code':
          r.code = true
          break
        case 'superscript':
          r.sup = true
          break
        case 'subscript':
          r.sub = true
          break
        case 'textStyle':
          if (typeof a.color === 'string' && colorToken.has(a.color)) r.color = colorToken.get(a.color)
          if (typeof a.fontSize === 'string' && FONT_SCALE[a.fontSize]) r.scale = FONT_SCALE[a.fontSize]
          break
        case 'highlight':
          if (typeof a.color === 'string' && highlightToken.has(a.color))
            r.highlight = highlightToken.get(a.color)
          break
      }
    }
    out.push(r)
  }
  return out
}

const same = (a: Run, b: Run) =>
  a.bold === b.bold &&
  a.italic === b.italic &&
  a.underline === b.underline &&
  a.strike === b.strike &&
  a.code === b.code &&
  a.sup === b.sup &&
  a.sub === b.sub &&
  a.color === b.color &&
  a.highlight === b.highlight &&
  a.scale === b.scale

// Memecah run ke baris selebar maxW: per kata, kata yang lebih lebar dari satu baris dipecah per huruf, dan baris
// baru di teks selalu memulai baris. measure mengukur teks dengan gaya run itu.
export function wrapRuns(runs: Run[], maxW: number, measure: (r: Run, s: string) => number): Run[][] {
  const lines: Run[][] = [[]]
  let w = 0
  const put = (r: Run, t: string) => {
    const line = lines[lines.length - 1]!
    const last = line[line.length - 1]
    if (last && same(last, r)) last.text += t
    else line.push({ ...r, text: t })
    w += measure(r, t)
  }
  const breakLine = () => {
    lines.push([])
    w = 0
  }
  for (const r of runs) {
    for (const part of r.text.split(/(\n)/)) {
      if (part === '\n') {
        breakLine()
        continue
      }
      for (const word of part.split(/(?<=\s)/)) {
        if (!word) continue
        const ww = measure(r, word.trimEnd())
        if (w + ww <= maxW) put(r, word)
        else if (ww <= maxW) {
          breakLine()
          put(r, word)
        } else
          for (const ch of word) {
            if (w > 0 && w + measure(r, ch) > maxW) breakLine()
            put(r, ch)
          }
      }
    }
  }
  for (const line of lines) {
    const last = line[line.length - 1]
    if (last) last.text = last.text.trimEnd()
  }
  return lines.map((l) => l.filter((r) => r.text))
}

// Dokumen pembanding: atribut null dan objek atribut kosong dibuang, supaya dokumen dari editor (yang menulis setiap
// atribut, termasuk textAlign null) bisa dibandingkan dengan dokumen dari server atau dari teks polos.
export function canonDoc(n: PMNode): PMNode {
  const out: PMNode = { type: n.type }
  if (n.attrs) {
    const attrs = Object.fromEntries(Object.entries(n.attrs).filter(([, v]) => v !== null && v !== undefined))
    if (Object.keys(attrs).length) out.attrs = attrs
  }
  if (n.text !== undefined) out.text = n.text
  if (n.marks?.length)
    out.marks = n.marks
      .map((m) => {
        const attrs = Object.fromEntries(
          Object.entries(m.attrs ?? {}).filter(([, v]) => v !== null && v !== undefined),
        )
        return Object.keys(attrs).length ? { type: m.type, attrs } : { type: m.type }
      })
      .filter((m) => m.type !== 'textStyle' || m.attrs)
  if (out.marks && !out.marks.length) delete out.marks
  if (n.content?.length) out.content = n.content.map(canonDoc)
  return out
}

// Teks polos jadi dokumen: satu paragraf per baris (teks bentuk dan bagian kelas UML boleh berbaris).
export function textDoc(text: string): PMNode {
  return {
    type: 'doc',
    content: (text ? text.split('\n') : ['']).map((line) =>
      line ? { type: 'paragraph', content: [{ type: 'text', text: line }] } : { type: 'paragraph' },
    ),
  }
}

// Teks polos dokumen, sama dengan yang dihasilkan server: blok dipisah baris baru.
export function docText(n: PMNode): string {
  const blocks: string[] = []
  const walk = (x: PMNode) => {
    if (x.type === 'paragraph' || x.type === 'heading') {
      const t: string[] = []
      const inline = (y: PMNode) => {
        if (y.type === 'text') t.push(y.text ?? '')
        else if (y.type === 'hardBreak') t.push('\n')
        else (y.content ?? []).forEach(inline)
      }
      inline(x)
      blocks.push(t.join(''))
    } else (x.content ?? []).forEach(walk)
  }
  walk(n)
  return blocks.join('\n').trim()
}

// Dokumen tanpa format apa pun (paragraf teks polos saja) disimpan sebagai teks saja, sama dengan aturan server.
export function plainDoc(n: PMNode): boolean {
  const c = canonDoc(n)
  return (c.content ?? []).every(
    (p) =>
      p.type === 'paragraph' && !p.attrs && (p.content ?? []).every((t) => t.type === 'text' && !t.marks),
  )
}

// Nilai yang dikirim ke server untuk field berformat: teks polos plus dokumen bila ada formatnya.
export function richValue(doc: PMNode): { text: string; doc: PMNode | null } {
  return { text: docText(doc), doc: plainDoc(doc) ? null : canonDoc(doc) }
}
