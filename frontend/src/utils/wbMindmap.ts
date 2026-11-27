// Mind map (revisi 57c): pohon dari garis antar bentuk mind map, cabang yang tersembunyi karena induknya dilipat, dan
// tata letak otomatis kiri dan kanan topik pusat. Bagian murni tanpa Vue supaya bisa diuji dengan node --test.
//
// Struktur tidak disimpan terpisah: induk dan anak diturunkan dari garis yang kedua ujungnya bentuk mind map, mulai
// dari topik pusat (BFS, jadi arah garis tidak penting). Komponen tanpa topik pusat berakar di simpul yang tidak
// pernah menjadi ujung akhir garis. Jadi salin tempel, undo, realtime, dan template tidak butuh data tambahan.

export type MindNode = {
  id: string
  x: number
  y: number
  w: number
  h: number
  central: boolean
  folded?: boolean
}
export type MindLink = { from: string; to: string }
export type MindTree = {
  parent: Map<string, string | null>
  children: Map<string, string[]>
  root: Map<string, string>
}

export const GAP_X = 56
export const GAP_Y = 16

export function mindTree(nodes: MindNode[], links: MindLink[]): MindTree {
  const byId = new Map(nodes.map((n) => [n.id, n]))
  const adj = new Map<string, string[]>(nodes.map((n) => [n.id, []]))
  const isTo = new Set<string>()
  for (const l of links) {
    if (!byId.has(l.from) || !byId.has(l.to) || l.from === l.to) continue
    adj.get(l.from)!.push(l.to)
    adj.get(l.to)!.push(l.from)
    isTo.add(l.to)
  }
  const parent = new Map<string, string | null>()
  const children = new Map<string, string[]>(nodes.map((n) => [n.id, []]))
  const root = new Map<string, string>()
  const walk = (start: string) => {
    if (parent.has(start)) return
    parent.set(start, null)
    root.set(start, start)
    const queue = [start]
    while (queue.length) {
      const id = queue.shift()!
      for (const next of adj.get(id)!) {
        if (parent.has(next)) continue
        parent.set(next, id)
        root.set(next, start)
        children.get(id)!.push(next)
        queue.push(next)
      }
    }
  }
  for (const n of nodes) if (n.central) walk(n.id)
  for (const n of nodes) if (!isTo.has(n.id)) walk(n.id)
  for (const n of nodes) walk(n.id)
  // Anak diurutkan dari atas ke bawah, jadi tata letak mempertahankan urutan yang terlihat.
  for (const list of children.values())
    list.sort((a, b) => byId.get(a)!.y - byId.get(b)!.y || byId.get(a)!.x - byId.get(b)!.x)
  return { parent, children, root }
}

// Semua keturunan (tanpa simpul itu sendiri), termasuk yang tersembunyi.
export function descendants(tree: MindTree, id: string): string[] {
  const out: string[] = []
  const stack = [...(tree.children.get(id) ?? [])]
  while (stack.length) {
    const c = stack.pop()!
    out.push(c)
    stack.push(...(tree.children.get(c) ?? []))
  }
  return out
}

// Simpul yang tersembunyi karena salah satu leluhurnya dilipat.
export function hiddenByFold(nodes: MindNode[], tree: MindTree): Set<string> {
  const out = new Set<string>()
  for (const n of nodes) if (n.folded) for (const d of descendants(tree, n.id)) out.add(d)
  return out
}

// Sisi cabang terhadap topik pusat: anak langsung akar menurut letaknya sekarang, keturunannya ikut induknya.
export function sideOf(tree: MindTree, byId: Map<string, MindNode>, id: string): 'left' | 'right' {
  let cur = id
  let up = tree.parent.get(cur)
  while (up && tree.parent.get(up)) {
    cur = up
    up = tree.parent.get(cur)
  }
  if (!up) return 'right'
  const r = byId.get(up)!
  const n = byId.get(cur)!
  return n.x + n.w / 2 >= r.x + r.w / 2 ? 'right' : 'left'
}

// Posisi baru setiap keturunan yang terlihat dari satu akar: kolom per tingkat, GAP_X dari induknya, subpohon
// bertumpuk dengan GAP_Y dan berpusat di tengah induknya. Akar tidak berpindah. Simpul terkunci tidak dipindah
// (keturunannya tetap ditata relatif ke posisinya).
export function layoutMind(
  nodes: MindNode[],
  tree: MindTree,
  rootId: string,
  locked: Set<string> = new Set(),
): Map<string, { x: number; y: number }> {
  const byId = new Map(nodes.map((n) => [n.id, n]))
  const out = new Map<string, { x: number; y: number }>()
  const kids = (id: string) => (byId.get(id)?.folded ? [] : (tree.children.get(id) ?? []))
  const heightOf = new Map<string, number>()
  const span = (id: string): number => {
    const ks = kids(id)
    const own = byId.get(id)!.h
    const sub = ks.reduce((t, k) => t + span(k), 0) + Math.max(0, ks.length - 1) * GAP_Y
    const v = Math.max(own, sub)
    heightOf.set(id, v)
    return v
  }
  const place = (id: string, side: 'left' | 'right', at: { x: number; y: number }) => {
    const ks = kids(id)
    if (!ks.length) return
    const p = byId.get(id)!
    const total = ks.reduce((t, k) => t + heightOf.get(k)!, 0) + (ks.length - 1) * GAP_Y
    let y = at.y + p.h / 2 - total / 2
    for (const k of ks) {
      const n = byId.get(k)!
      const hk = heightOf.get(k)!
      const x = side === 'right' ? at.x + p.w + GAP_X : at.x - GAP_X - n.w
      const pos = locked.has(k)
        ? { x: n.x, y: n.y }
        : { x: Math.round(x), y: Math.round(y + hk / 2 - n.h / 2) }
      out.set(k, pos)
      place(k, side, pos)
      y += hk + GAP_Y
    }
  }
  const root = byId.get(rootId)
  if (!root) return out
  const all = tree.children.get(rootId) ?? []
  const groups = { right: [] as string[], left: [] as string[] }
  for (const k of root.folded ? [] : all) groups[sideOf(tree, byId, k)].push(k)
  for (const side of ['right', 'left'] as const) {
    const ks = groups[side]
    ks.forEach((k) => span(k))
    const total = ks.reduce((t, k) => t + heightOf.get(k)!, 0) + Math.max(0, ks.length - 1) * GAP_Y
    let y = root.y + root.h / 2 - total / 2
    for (const k of ks) {
      const n = byId.get(k)!
      const hk = heightOf.get(k)!
      const x = side === 'right' ? root.x + root.w + GAP_X : root.x - GAP_X - n.w
      const pos = locked.has(k)
        ? { x: n.x, y: n.y }
        : { x: Math.round(x), y: Math.round(y + hk / 2 - n.h / 2) }
      out.set(k, pos)
      place(k, side, pos)
      y += hk + GAP_Y
    }
  }
  return out
}

// Sisi untuk anak baru: anak topik pusat ke sisi yang cabangnya lebih sedikit (kanan bila sama), lainnya sisi induk.
export function newChildSide(tree: MindTree, nodes: MindNode[], parentId: string): 'left' | 'right' {
  const byId = new Map(nodes.map((n) => [n.id, n]))
  if (tree.parent.get(parentId)) return sideOf(tree, byId, parentId)
  let left = 0
  let right = 0
  for (const k of tree.children.get(parentId) ?? []) {
    if (sideOf(tree, byId, k) === 'left') left++
    else right++
  }
  return left < right ? 'left' : 'right'
}
