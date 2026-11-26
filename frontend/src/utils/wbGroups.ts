// Grup dan susunan kartu Whiteboard (revisi 50). Fungsi murni tanpa Vue supaya bisa diuji dengan node --test.
// Grup disimpan per kartu sebagai jalur group_ids dari grup terluar ke terdalam (migrasi 000026): grup A berisi grup
// B berarti setiap anggota B punya jalur yang diawali A. "Ruang" (scope) adalah jalur grup yang sedang dimasuki lewat
// dobel klik; klik di dalamnya memilih satu tingkat di bawahnya.

export type GroupCard = { id: string; x: number; y: number; w: number; h: number; group_ids?: string[] }
export const MAX_GROUP_DEPTH = 8

const path = (c: GroupCard) => c.group_ids ?? []

function startsWith(p: string[], prefix: string[]) {
  return prefix.every((g, i) => p[i] === g)
}

// unitOf: kartu yang ikut terpilih saat kartu ini diklik di ruang scope: seluruh grup satu tingkat di bawah scope,
// atau kartu itu saja bila tidak bergrup di tingkat itu. Kartu di luar scope dipilih dari tingkat terluar.
export function unitOf(card: GroupCard, cards: GroupCard[], scope: string[] = []): string[] {
  const p = path(card)
  const base = startsWith(p, scope) ? scope : []
  const g = p[base.length]
  if (!g) return [card.id]
  const prefix = [...base, g]
  return cards.filter((c) => startsWith(path(c), prefix)).map((c) => c.id)
}

// expand: semua kartu dari unit pilihan, tanpa duplikat, urutan pilihan dipertahankan.
export function expand(ids: Iterable<string>, cards: GroupCard[], scope: string[] = []): string[] {
  const byId = new Map(cards.map((c) => [c.id, c]))
  const out: string[] = []
  const seen = new Set<string>()
  for (const id of ids) {
    const c = byId.get(id)
    for (const m of c ? unitOf(c, cards, scope) : [id])
      if (!seen.has(m)) {
        seen.add(m)
        out.push(m)
      }
  }
  return out
}

// units: kelompok kartu terpilih per unit (grup atau kartu tunggal), untuk rata, sebar, dan rapikan.
export function units(ids: string[], cards: GroupCard[], scope: string[] = []): GroupCard[][] {
  const byId = new Map(cards.map((c) => [c.id, c]))
  const out: GroupCard[][] = []
  const done = new Set<string>()
  for (const id of ids) {
    if (done.has(id)) continue
    const c = byId.get(id)
    if (!c) continue
    const u = unitOf(c, cards, scope).flatMap((m) => byId.get(m) ?? [])
    u.forEach((m) => done.add(m.id))
    out.push(u)
  }
  return out
}

function commonPrefix(paths: string[][]): string[] {
  const first = paths[0] ?? []
  let n = 0
  while (n < first.length && paths.every((p) => p[n] === first[n])) n++
  return first.slice(0, n)
}

// group menyisipkan grup baru tepat di bawah jalur yang sama-sama dimiliki semua kartu terpilih, jadi grup yang
// sudah ada ikut masuk utuh (grup bersarang). Null bila hasilnya melewati batas kedalaman.
export function group(ids: string[], cards: GroupCard[], newGroup: string): Map<string, string[]> | null {
  const list = cards.filter((c) => ids.includes(c.id))
  if (list.length < 2) return null
  const prefix = commonPrefix(list.map(path))
  const out = new Map<string, string[]>()
  for (const c of list) {
    const p = path(c)
    const next = [...prefix, newGroup, ...p.slice(prefix.length)]
    if (next.length > MAX_GROUP_DEPTH) return null
    out.set(c.id, next)
  }
  return out
}

// ungroup melepas grup satu tingkat di bawah scope dari kartu terpilih (grup terluar bila tidak sedang di dalam grup).
export function ungroup(ids: string[], cards: GroupCard[], scope: string[] = []): Map<string, string[]> {
  const out = new Map<string, string[]>()
  for (const c of cards) {
    if (!ids.includes(c.id)) continue
    const p = path(c)
    const at = startsWith(p, scope) ? scope.length : 0
    if (p.length > at) out.set(c.id, [...p.slice(0, at), ...p.slice(at + 1)])
  }
  return out
}

// remap memberi id grup baru untuk salinan (tempel, duplikat), supaya salinan tidak bergabung ke grup aslinya.
export function remap(paths: string[][], newId: () => string): string[][] {
  const m = new Map<string, string>()
  return paths.map((p) =>
    p.map((g) => {
      let n = m.get(g)
      if (!n) {
        n = newId()
        m.set(g, n)
      }
      return n
    }),
  )
}

// hasGroup: pilihan memuat grup pada tingkat scope (Pisahkan grup berlaku).
export function hasGroup(ids: string[], cards: GroupCard[], scope: string[] = []): boolean {
  return cards.some(
    (c) => ids.includes(c.id) && path(c).length > (startsWith(path(c), scope) ? scope.length : 0),
  )
}

export type Box = { x: number; y: number; w: number; h: number }

export function boxOf(list: Box[]): Box | null {
  if (!list.length) return null
  const x = Math.min(...list.map((c) => c.x))
  const y = Math.min(...list.map((c) => c.y))
  return {
    x,
    y,
    w: Math.max(...list.map((c) => c.x + c.w)) - x,
    h: Math.max(...list.map((c) => c.y + c.h)) - y,
  }
}

export type AlignKind = 'left' | 'hcenter' | 'right' | 'top' | 'vcenter' | 'bottom' | 'hspread' | 'vspread'

// arrange menghitung geseran per unit untuk rata dan sebar: setiap grup bergerak utuh (semua anggotanya digeser
// sama jauh). Hasilnya posisi baru per kartu.
export function arrange(groups: GroupCard[][], kind: AlignKind): Map<string, { x: number; y: number }> {
  const boxes = groups.map((g) => ({ g, b: boxOf(g)! }))
  const all = boxOf(boxes.map((u) => u.b))
  const out = new Map<string, { x: number; y: number }>()
  if (!all || boxes.length < 2) return out
  const shift = (g: GroupCard[], dx: number, dy: number) => {
    for (const c of g) out.set(c.id, { x: Math.round(c.x + dx), y: Math.round(c.y + dy) })
  }
  if (kind === 'hspread' || kind === 'vspread') {
    const h = kind === 'hspread'
    const sorted = [...boxes].sort((a, b) => (h ? a.b.x - b.b.x : a.b.y - b.b.y))
    const total = sorted.reduce((s, u) => s + (h ? u.b.w : u.b.h), 0)
    const gap = ((h ? all.w : all.h) - total) / (sorted.length - 1)
    let pos = h ? all.x : all.y
    for (const u of sorted) {
      shift(u.g, h ? pos - u.b.x : 0, h ? 0 : pos - u.b.y)
      pos += (h ? u.b.w : u.b.h) + gap
    }
    return out
  }
  for (const u of boxes) {
    const b = u.b
    const dx =
      kind === 'left'
        ? all.x - b.x
        : kind === 'right'
          ? all.x + all.w - (b.x + b.w)
          : kind === 'hcenter'
            ? all.x + all.w / 2 - (b.x + b.w / 2)
            : 0
    const dy =
      kind === 'top'
        ? all.y - b.y
        : kind === 'bottom'
          ? all.y + all.h - (b.y + b.h)
          : kind === 'vcenter'
            ? all.y + all.h / 2 - (b.y + b.h / 2)
            : 0
    shift(u.g, dx, dy)
  }
  return out
}

// tidy menyusun unit menjadi grid rapi berurutan baca (atas ke bawah, kiri ke kanan), mulai dari pojok kiri atas
// pilihan, dengan jarak gap. Kolom grid sebanyak akar jumlah unit dibulatkan ke atas.
export function tidy(groups: GroupCard[][], gap = 24): Map<string, { x: number; y: number }> {
  const boxes = groups.map((g) => ({ g, b: boxOf(g)! }))
  const all = boxOf(boxes.map((u) => u.b))
  const out = new Map<string, { x: number; y: number }>()
  if (!all) return out
  const sorted = [...boxes].sort((a, b) => a.b.y - b.b.y || a.b.x - b.b.x)
  const cols = Math.ceil(Math.sqrt(sorted.length))
  const colW = Math.max(...sorted.map((u) => u.b.w))
  let y = all.y
  for (let r = 0; r < sorted.length; r += cols) {
    const row = sorted.slice(r, r + cols)
    row.forEach((u, i) => {
      const x = all.x + i * (colW + gap)
      for (const c of u.g) out.set(c.id, { x: Math.round(c.x + x - u.b.x), y: Math.round(c.y + y - u.b.y) })
    })
    y += Math.max(...row.map((u) => u.b.h)) + gap
  }
  return out
}
