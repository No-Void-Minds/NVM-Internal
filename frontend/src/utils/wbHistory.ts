// Undo dan redo Whiteboard (W2): hanya aksi sendiri, per tab. Langkah tata letak hanya mengubah field yang nilainya
// masih hasil aksi sendiri; field yang sudah diubah orang lain sesudahnya dilewati, jadi undo tidak pernah
// membatalkan perubahan orang lain. Tanpa Vue dan klien API supaya bisa diuji dengan node --test.
// Tipe dibuat ulang di sini (bukan diimpor dari whiteboard.ts) supaya test Node tanpa resolusi modul bundler bisa
// memuat berkas ini. Bentuknya sama dengan Card di utils/whiteboard.ts.
type CardColor = 'abu' | 'hijau' | 'kuning' | 'biru' | 'krem' | 'merah'
type Card = {
  id: string
  x: number
  y: number
  w: number
  h: number
  z: number
  color: CardColor | null
  content: unknown
  content_version: number
  style?: Record<string, unknown>
  locked?: boolean
  group_ids?: string[]
  file?: { id: string }
}

// Revisi 50: gaya, kunci, grup, dan berkas ikut langkah tata letak (salin gaya, kunci, grup, ganti gambar).
export type Geo = {
  x?: number
  y?: number
  w?: number
  h?: number
  z?: number
  color?: CardColor | null
  style?: Record<string, unknown>
  locked?: boolean
  group_ids?: string[]
  file_id?: string
}
export type GeoPatch = {
  id: string
  x?: number
  y?: number
  w?: number
  h?: number
  z?: number
  color?: CardColor | ''
  style?: Record<string, unknown>
  locked?: boolean
  group_ids?: string[]
  file_id?: string
}

export type ChecklistInverse = { undo: Record<string, unknown>; redo: Record<string, unknown> }

export type Step =
  | { kind: 'layout'; cards: { id: string; before: Geo; after: Geo }[] }
  | { kind: 'content'; id: string; before: Card['content']; after: Card['content']; version: number }
  | { kind: 'create'; ids: string[]; batch?: string }
  | { kind: 'delete'; ids: string[]; batch: string }
  | { kind: 'checklist'; id: string; ops: ChecklistInverse }
  // Garis baru dari Hubungkan dengan garis (revisi 50); batches diisi saat undo untuk redo lewat pemulihan.
  | { kind: 'connectors'; ids: string[]; batches: string[] }
  // Urutan tumpuk garis (revisi 53); null berarti otomatis.
  | { kind: 'lineZ'; id: string; before: number | null; after: number | null }

export const MAX_STEPS = 100

const fields = ['x', 'y', 'w', 'h', 'z', 'color', 'style', 'locked', 'group_ids', 'file_id'] as const
type Field = (typeof fields)[number]

function valueOf(card: Card, f: Field): unknown {
  if (f === 'file_id') return card.file?.id
  if (f === 'style') return card.style ?? {}
  if (f === 'locked') return card.locked ?? false
  if (f === 'group_ids') return card.group_ids ?? []
  return card[f]
}

// Nilai objek dan larik (gaya, jalur grup) dibandingkan isinya.
const same = (a: unknown, b: unknown) => JSON.stringify(a ?? null) === JSON.stringify(b ?? null)

// geo mengambil field tata letak sebuah kartu, untuk dicatat sebelum dan sesudah aksi.
export function geo(card: Card, only: readonly Field[] = ['x', 'y', 'w', 'h', 'z', 'color']): Geo {
  const out: Geo = {}
  for (const f of only) (out as Record<string, unknown>)[f] = valueOf(card, f)
  return out
}

// layoutPatches menyusun patch untuk undo (kembali ke before) atau redo (kembali ke after). Field yang nilainya di
// kartu sekarang tidak sama dengan hasil aksi itu sudah diubah orang lain, jadi dilewati dan dihitung.
export function layoutPatches(
  step: Extract<Step, { kind: 'layout' }>,
  current: Map<string, Card>,
  dir: 'undo' | 'redo',
): { patches: GeoPatch[]; skipped: number } {
  const patches: GeoPatch[] = []
  let skipped = 0
  for (const entry of step.cards) {
    const card = current.get(entry.id)
    const expect = dir === 'undo' ? entry.after : entry.before
    const target = dir === 'undo' ? entry.before : entry.after
    if (!card) {
      skipped += Object.keys(target).length
      continue
    }
    const p: GeoPatch = { id: entry.id }
    let any = false
    for (const f of fields) {
      if (!(f in target)) continue
      const now = valueOf(card, f) ?? null
      const was = (expect as Record<string, unknown>)[f] ?? null
      if (!same(now, was)) {
        skipped++
        continue
      }
      const v = (target as Record<string, unknown>)[f]
      if (f === 'color') p.color = (v as CardColor | null) ?? ''
      else (p as Record<string, unknown>)[f] = v
      any = true
    }
    if (any) patches.push(p)
  }
  return { patches, skipped }
}

// push menambah langkah baru: redo dibuang dan riwayat dibatasi MAX_STEPS langkah.
export function push(stack: { undo: Step[]; redo: Step[] }, step: Step) {
  stack.undo.push(step)
  if (stack.undo.length > MAX_STEPS) stack.undo.shift()
  stack.redo.length = 0
}
