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

// Garis (revisi 55): field yang diubah satu aksi garis, dalam bentuk yang dikembalikan server. Bentuknya sama dengan
// Connector di utils/whiteboard.ts (dibuat ulang di sini dengan alasan yang sama dengan Card).
type Line = {
  id: string
  from: string | null
  to: string | null
  style: unknown
  geometry: unknown
  label: string
  label_start: string
  label_end: string
}
export const LINE_FIELDS = ['style', 'geometry', 'label', 'label_start', 'label_end'] as const
export type LineField = (typeof LINE_FIELDS)[number]
export type LineSnap = Partial<Pick<Line, LineField>>

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
  // Gaya, label, jalur, dan titik belok garis (revisi 55): satu langkah per aksi (satu seret, satu isian).
  | { kind: 'line'; id: string; before: LineSnap; after: LineSnap }
  // Garis yang dihapus sendiri (revisi 55); ends dicatat untuk cek kunci karena garisnya tidak ada di papan lagi.
  | { kind: 'linesGone'; ids: string[]; batches: string[]; ends: (string | null)[] }
  // Beberapa langkah sebagai satu (revisi 54b): undo menjalankannya terbalik, redo berurutan.
  | { kind: 'batch'; steps: Step[] }

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

// lineSnap mengambil field garis yang diubah satu aksi, sebelum atau sesudahnya.
export function lineSnap(c: Line, keys: readonly string[]): LineSnap {
  const out: LineSnap = {}
  for (const k of LINE_FIELDS)
    if (keys.includes(k)) (out as Record<string, unknown>)[k] = structuredClone(c[k])
  return out
}

// Penghalang undo garis: kartu ujungnya terkunci posisinya, atau sedang diedit orang lain (kunci lunak).
export type LineBlock = 'locked' | 'busy' | null
export function lineBlocked(
  ends: (string | null)[],
  cards: Map<string, { locked?: boolean }>,
  heldByOthers: (id: string) => boolean,
): LineBlock {
  const ids = ends.filter((x): x is string => !!x)
  if (ids.some((id) => cards.get(id)?.locked)) return 'locked'
  if (ids.some(heldByOthers)) return 'busy'
  return null
}

// linePatch menyusun patch undo (kembali ke before) atau redo (kembali ke after) untuk satu langkah garis. Field yang
// nilainya sekarang bukan lagi hasil aksi itu sudah diubah orang lain, jadi dilewati dan dihitung, sama dengan
// layoutPatches untuk kartu.
export function linePatch(
  step: Extract<Step, { kind: 'line' }>,
  current: Line | undefined,
  dir: 'undo' | 'redo',
): { patch: LineSnap | null; skipped: number } {
  const expect = dir === 'undo' ? step.after : step.before
  const target = dir === 'undo' ? step.before : step.after
  const keys = Object.keys(target) as LineField[]
  if (!current) return { patch: null, skipped: keys.length }
  const patch: LineSnap = {}
  let skipped = 0
  for (const k of keys) {
    if (!same(current[k], expect[k])) {
      skipped++
      continue
    }
    ;(patch as Record<string, unknown>)[k] = target[k]
  }
  return { patch: Object.keys(patch).length ? patch : null, skipped }
}

// Sesudah undo atau redo langkah isi kartu berhasil (versi old menjadi now), langkah isi berikutnya untuk kartu yang
// sama di tumpukan yang sama ikut diperbarui versinya bila langkah itu tepat sebelumnya (versinya old - 1, jadi tidak
// ada suntingan orang lain di antaranya). Tanpa ini dua perubahan isi berturut-turut pada satu kartu hanya bisa
// dibatalkan sekali: versi kartu naik karena undo sendiri, dan langkah yang lebih lama dianggap diubah orang lain.
export function chainContentVersion(stack: Step[], id: string, old: number, now: number) {
  for (let i = stack.length - 1; i >= 0; i--) {
    const s = stack[i]!
    if (s.kind === 'content' && s.id === id) {
      if (s.version === old - 1) s.version = now
      return
    }
  }
}
