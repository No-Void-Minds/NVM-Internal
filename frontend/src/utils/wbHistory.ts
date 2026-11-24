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
}

export type Geo = { x?: number; y?: number; w?: number; h?: number; z?: number; color?: CardColor | null }
export type GeoPatch = {
  id: string
  x?: number
  y?: number
  w?: number
  h?: number
  z?: number
  color?: CardColor | ''
}

export type ChecklistInverse = { undo: Record<string, unknown>; redo: Record<string, unknown> }

export type Step =
  | { kind: 'layout'; cards: { id: string; before: Geo; after: Geo }[] }
  | { kind: 'content'; id: string; before: Card['content']; after: Card['content']; version: number }
  | { kind: 'create'; ids: string[]; batch?: string }
  | { kind: 'delete'; ids: string[]; batch: string }
  | { kind: 'checklist'; id: string; ops: ChecklistInverse }

export const MAX_STEPS = 100

const fields = ['x', 'y', 'w', 'h', 'z', 'color'] as const

// geo mengambil field tata letak sebuah kartu, untuk dicatat sebelum dan sesudah aksi.
export function geo(card: Card, only: readonly (typeof fields)[number][] = fields): Geo {
  const out: Geo = {}
  for (const f of only) (out as Record<string, unknown>)[f] = card[f]
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
      const now = card[f] ?? null
      const was = (expect as Record<string, unknown>)[f] ?? null
      if (now !== was) {
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
