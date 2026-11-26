// Urutan tumpuk kartu dan garis (revisi 53). Bagian murni supaya bisa diuji dengan node --test dan dipakai kanvas
// serta ekspor dengan urutan yang sama.

export type StackLine = { id: string; from: string | null; to: string | null; z?: number | null }
export type Layer = { id: string; z: number; line: boolean }
export type StackKind = 'front' | 'forward' | 'backward' | 'back'

// z garis: manual bila diatur; otomatis sama dengan kartu ujung yang paling depan (lalu digambar tepat di atasnya
// karena garis menang seri melawan kartu). Garis tanpa z dan tanpa kartu ujung ada di paling belakang, seperti
// sebelum revisi 53.
export function lineZ(l: StackLine, zOf: Map<string, number>): number {
  if (l.z !== null && l.z !== undefined) return l.z
  let z = -Infinity
  for (const id of [l.from, l.to]) {
    const v = id ? zOf.get(id) : undefined
    if (v !== undefined && v > z) z = v
  }
  return z
}

// Urutan gambar dari belakang ke depan: z, lalu kartu sebelum garis pada z yang sama, lalu id supaya stabil.
export function stackOrder(cards: { id: string; z: number }[], lines: StackLine[]): Layer[] {
  const zOf = new Map(cards.map((c) => [c.id, c.z]))
  const out: Layer[] = [
    ...cards.map((c) => ({ id: c.id, z: c.z, line: false })),
    ...lines.map((l) => ({ id: l.id, z: lineZ(l, zOf), line: true })),
  ]
  return out.sort((a, b) =>
    a.z !== b.z ? a.z - b.z : a.line !== b.line ? (a.line ? 1 : -1) : a.id < b.id ? -1 : a.id > b.id ? 1 : 0,
  )
}

// Nilai z-index CSS: peringkat 1..n, jadi z pecahan (hasil maju dan mundur) tidak lagi bertabrakan karena dibulatkan.
export function stackRanks(order: Layer[]): Map<string, number> {
  return new Map(order.map((l, i) => [l.id, i + 1]))
}

// z baru untuk yang terpilih. Paling depan dan paling belakang melompati semuanya; maju dan mundur melewati satu
// lapis lain (posisi di urutan gambar, bukan angka z saja, supaya garis otomatis yang seri dengan kartunya terhitung).
export function restack(order: Layer[], sel: Set<string>, kind: StackKind): { id: string; z: number }[] {
  const pos = new Map(order.map((l, i) => [l.id, i]))
  const finite = order.filter((l) => Number.isFinite(l.z))
  const others = order.filter((l) => !sel.has(l.id))
  const list = order.filter((l) => sel.has(l.id))
  const top = finite.at(-1)?.z ?? 0
  const bottom = finite[0]?.z ?? 0
  return list.map((c, i) => {
    let z = Number.isFinite(c.z) ? c.z : bottom - 1
    const at = pos.get(c.id)!
    if (kind === 'front') z = top + 1 + i
    else if (kind === 'back') z = bottom - list.length + i
    else if (kind === 'forward') {
      const k = others.findIndex((o) => pos.get(o.id)! > at && Number.isFinite(o.z))
      if (k >= 0) {
        const up = others.slice(k + 1).find((o) => o.z > others[k]!.z)
        z = up ? (others[k]!.z + up.z) / 2 + i * 1e-3 : others[k]!.z + 1 + i
      }
    } else {
      let k = -1
      for (let j = others.length - 1; j >= 0; j--)
        if (pos.get(others[j]!.id)! < at && Number.isFinite(others[j]!.z)) {
          k = j
          break
        }
      if (k >= 0) {
        const down = others
          .slice(0, k)
          .reverse()
          .find((o) => Number.isFinite(o.z) && o.z < others[k]!.z)
        z = down ? (down.z + others[k]!.z) / 2 + i * 1e-3 : others[k]!.z - list.length + i
      }
    }
    return { id: c.id, z }
  })
}
