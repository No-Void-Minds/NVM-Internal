// Wadah Whiteboard (revisi 54b): pool dan lane BPMN, swimlane activity, batas sistem, dan wadah lain yang menyusul
// (state komposit, fragmen sequence). Bagian murni supaya bisa diuji dengan node --test.
//
// Keanggotaan tidak disimpan. Anggota wadah adalah kartu yang seluruhnya ada di dalam kotak wadah dan digambar di
// atasnya (z lebih tinggi), dihitung saat wadah mulai diseret. Jadi kartu yang dilepas di dalam wadah langsung jadi
// anggota, kartu yang diseret keluar langsung keluar, wadah di dalam wadah ikut bersama isinya, dan undo, realtime,
// template, serta salin tempel tidak butuh data tambahan.

export type Box = { x: number; y: number; w: number; h: number }
export type Boxy = Box & { id: string; z: number; column_id?: string | null }
export type Lane = { label: string; size: number }
export type LaneOrient = 'rows' | 'cols'

// Lebar pita judul (nama pool di kiri, judul swimlane di atas) dan pita label lane, dalam px dunia.
export const BAND = 32
export const MIN_LANE = 40
export const MAX_LANES = 20

const inside = (c: Box, box: Box, tol = 0.5) =>
  c.x >= box.x - tol &&
  c.y >= box.y - tol &&
  c.x + c.w <= box.x + box.w + tol &&
  c.y + c.h <= box.y + box.h + tol

// Anggota semua wadah yang diberikan. bounds memberi kotak tampil kartu (sesudah putar). Wadah yang diputar tidak
// punya anggota: kotak tegaknya tidak lagi sama dengan yang terlihat. Isi kolom ikut kolomnya, bukan wadah.
export function containerMembers(
  containers: Boxy[],
  cards: Boxy[],
  bounds: (c: Boxy) => Box,
  rotated: (id: string) => boolean,
): Set<string> {
  const out = new Set<string>()
  for (const w of containers) {
    if (rotated(w.id)) continue
    for (const c of cards) {
      if (c.id === w.id || c.column_id || c.z <= w.z) continue
      if (inside(bounds(c), w)) out.add(c.id)
    }
  }
  return out
}

// Wadah yang memuat kotak b, dari yang paling depan; dipakai untuk menaikkan kartu yang dilepas di dalam wadah yang
// lebih depan darinya supaya terlihat dan menjadi anggotanya.
export function containersAt(b: Box, containers: Boxy[], exclude: Set<string>): Boxy[] {
  return containers.filter((w) => !exclude.has(w.id) && inside(b, w)).sort((p, q) => q.z - p.z)
}

// Kotak tiap lane di dalam bentuk w x h. rows: pool (pita nama di kiri, lane bertumpuk ke bawah); cols: swimlane
// (pita judul di atas, lane berjajar ke kanan). Ukuran lane adalah bobot, jadi ubah ukuran kartu membagi ulang
// tinggi atau lebar secara proporsional.
// band: lebar pita sebelum lane (0 untuk operand fragmen sequence, yang tidak punya pita).
export function laneBoxes(
  lanes: Lane[],
  w: number,
  h: number,
  orient: LaneOrient,
  band = BAND,
): (Box & { label: string })[] {
  const total = lanes.reduce((n, l) => n + l.size, 0) || 1
  const span = orient === 'rows' ? h : w
  let at = 0
  return lanes.map((l) => {
    const len = (l.size / total) * span
    const b =
      orient === 'rows'
        ? { x: band, y: at, w: Math.max(0, w - band), h: len }
        : { x: at, y: band, w: len, h: Math.max(0, h - band) }
    at += len
    return { ...b, label: l.label }
  })
}

// Geser pembatas antara lane i dan i + 1 sebesar d px tampil; jumlah ukuran tetap, setiap lane minimal MIN_LANE px.
export function resizeLane(lanes: Lane[], i: number, d: number, span: number): Lane[] {
  if (i < 0 || i + 1 >= lanes.length) return lanes
  const total = lanes.reduce((n, l) => n + l.size, 0) || 1
  const k = span / total
  const a = lanes[i]!.size * k
  const b = lanes[i + 1]!.size * k
  const moved = Math.max(MIN_LANE - a, Math.min(b - MIN_LANE, d))
  return lanes.map((l, j) =>
    j === i
      ? { ...l, size: Math.round((a + moved) / k) }
      : j === i + 1
        ? { ...l, size: Math.round((b - moved) / k) }
        : l,
  )
}

// Lane baru seukuran rata-rata lane yang ada; bentuknya bertambah sepanjang lane itu supaya lane lain tidak menyempit.
export function addLane(lanes: Lane[], span: number): { lanes: Lane[]; grow: number } {
  if (lanes.length >= MAX_LANES) return { lanes, grow: 0 }
  const total = lanes.reduce((n, l) => n + l.size, 0) || 1
  const k = span / total
  const avg = lanes.length ? total / lanes.length : 150
  return {
    lanes: [...lanes, { label: `Lane ${lanes.length + 1}`, size: Math.round(avg) }],
    grow: Math.round(avg * k),
  }
}
