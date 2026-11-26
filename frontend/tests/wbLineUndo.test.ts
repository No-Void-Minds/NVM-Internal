// Revisi 55: undo dan redo garis (snapshot, patch, field yang diubah orang lain dilewati, kunci) dan jalur siku dengan
// titik belok yang tidak memotong dirinya sendiri.
//   cd frontend && npm run test:unit
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { chainContentVersion, lineBlocked, linePatch, lineSnap, push, type Step } from '../src/utils/wbHistory.ts'
import { routeConnector } from '../src/utils/wbShapes.ts'

const line = (over: Record<string, unknown> = {}) => ({
  id: 'L',
  from: 'a',
  to: 'b',
  style: { start: 'none', end: 'arrow', line: 'solid', color: '' },
  geometry: { from: { port: 'auto' }, to: { port: 'auto' }, route: 'bebas', points: [[10, 20]] },
  label: '',
  label_start: '',
  label_end: '',
  ...over,
})

test('lineSnap hanya mengambil field yang diubah, sebagai salinan', () => {
  const c = line()
  const s = lineSnap(c, ['geometry', 'z'])
  assert.deepEqual(Object.keys(s), ['geometry'])
  ;(c.geometry as { points: number[][] }).points[0]![0] = 99
  assert.deepEqual((s.geometry as { points: number[][] }).points[0], [10, 20])
})

test('linePatch: undo ke before, redo ke after, perubahan orang lain dilewati', () => {
  const before = line()
  const after = line({ label: 'lalu', geometry: { ...before.geometry, points: [[30, 40]] } })
  const step: Extract<Step, { kind: 'line' }> = {
    kind: 'line',
    id: 'L',
    before: lineSnap(before, ['label', 'geometry']),
    after: lineSnap(after, ['label', 'geometry']),
  }
  const u = linePatch(step, after, 'undo')
  assert.equal(u.skipped, 0)
  assert.deepEqual(u.patch, { label: '', geometry: before.geometry })
  const r = linePatch(step, before, 'redo')
  assert.deepEqual(r.patch, { label: 'lalu', geometry: after.geometry })
  // Orang lain mengubah label sesudahnya: label tidak dibatalkan, titik belok tetap.
  const theirs = linePatch(step, { ...after, label: 'milik Nala' }, 'undo')
  assert.equal(theirs.skipped, 1)
  assert.deepEqual(theirs.patch, { geometry: before.geometry })
  // Garis sudah dihapus: semua dilewati.
  assert.deepEqual(linePatch(step, undefined, 'undo'), { patch: null, skipped: 2 })
})

test('lineBlocked: kartu ujung terkunci atau sedang diedit orang lain', () => {
  const cards = new Map([
    ['a', { locked: false }],
    ['b', { locked: true }],
    ['c', {}],
  ])
  assert.equal(lineBlocked(['a', 'b'], cards, () => false), 'locked')
  assert.equal(lineBlocked(['a', 'c'], cards, (id) => id === 'c'), 'busy')
  assert.equal(lineBlocked(['a', null], cards, () => false), null)
})

test('riwayat garis memakai tumpukan yang sama dengan kartu', () => {
  const h = { undo: [] as Step[], redo: [] as Step[] }
  push(h, { kind: 'layout', cards: [] })
  push(h, { kind: 'line', id: 'L', before: {}, after: {} })
  assert.equal(h.undo.at(-1)!.kind, 'line')
})

const o = (p: { x: number; y: number }, q: { x: number; y: number }, r: { x: number; y: number }) =>
  Math.sign((q.x - p.x) * (r.y - p.y) - (q.y - p.y) * (r.x - p.x))
function selfCross(p: { x: number; y: number }[]) {
  for (let i = 0; i + 1 < p.length; i++)
    for (let j = i + 2; j + 1 < p.length; j++) {
      const [a, b, c, d] = [p[i]!, p[i + 1]!, p[j]!, p[j + 1]!]
      if (o(a, b, c) * o(a, b, d) < 0 && o(c, d, a) * o(c, d, b) < 0) return true
    }
  return false
}
const spike = (p: { x: number; y: number }[]) =>
  p.some(
    (q, i) =>
      i > 1 && (p[i - 1]!.x - p[i - 2]!.x) * (q.x - p[i - 1]!.x) + (p[i - 1]!.y - p[i - 2]!.y) * (q.y - p[i - 1]!.y) < -0.01,
  )
const route = (points: [number, number][], aktor = { x: 0, y: 0, w: 80, h: 160 }) =>
  routeConnector(
    {
      id: 'L',
      from: 'aktor',
      to: 'uc',
      geometry: { from: { port: 'auto' }, to: { port: 'auto' }, route: 'orthogonal', points, anchor: [[40, 80], [490, 85]] },
      style: { start: 'none', end: 'arrow' },
    },
    new Map([
      ['aktor', aktor],
      ['uc', { x: 400, y: 40, w: 180, h: 90 }],
    ]),
    new Map(),
  )!

test('jalur siku dengan titik belok dekat aktor tidak memotong dirinya dan tidak berbalik arah', () => {
  // Kasus nyata sebelum revisi 55: titik belok di atas port kanan aktor membuat jalur maju lalu berbalik.
  const r = route([
    [31, -17],
    [167, 4],
    [312, 131],
  ])
  assert.ok(!selfCross(r.pts), JSON.stringify(r.pts))
  assert.ok(!spike(r.pts), JSON.stringify(r.pts))
  // Acak berbiji: titik belok di luar kotak ujung tidak pernah menghasilkan silang atau tekukan bolak-balik.
  let seed = 7
  const rnd = () => (seed = (seed * 1103515245 + 12345) % 2147483648) / 2147483648
  const inside = (q: { x: number; y: number }, b: { x: number; y: number; w: number; h: number }) =>
    q.x > b.x && q.x < b.x + b.w && q.y > b.y && q.y < b.y + b.h
  let checked = 0
  for (let t = 0; t < 1500; t++) {
    const k = 2 + Math.floor(rnd() * 2)
    const pts: [number, number][] = []
    for (let i = 0; i < k; i++)
      pts.push([Math.round(-60 + (i + 1) * (460 / (k + 1)) + rnd() * 60 - 30), Math.round(-40 + rnd() * 260)])
    const r = route(pts)
    if (r.stops.slice(1, -1).some((q) => inside(q, { x: 0, y: 0, w: 80, h: 160 }) || inside(q, { x: 400, y: 40, w: 180, h: 90 })))
      continue
    checked++
    assert.ok(!selfCross(r.pts), `silang ${JSON.stringify(pts)}`)
    assert.ok(!spike(r.pts), `bolak-balik ${JSON.stringify(pts)}`)
    assert.ok(r.pts.every((q, i) => !i || Math.abs(q.x - r.pts[i - 1]!.x) < 0.01 || Math.abs(q.y - r.pts[i - 1]!.y) < 0.01))
  }
  assert.ok(checked > 500)
})

test('dua perubahan isi berturut-turut pada satu kartu bisa dibatalkan dua kali (revisi 54b)', () => {
  // Langkah 1 menghasilkan versi 5, langkah 2 versi 6. Undo langkah 2 membuat kartu versi 7.
  const s1: Step = { kind: 'content', id: 'k', before: 'a', after: 'b', version: 5 }
  const s2: Step = { kind: 'content', id: 'k', before: 'b', after: 'c', version: 6 }
  const other: Step = { kind: 'content', id: 'lain', before: 'x', after: 'y', version: 6 }
  const undo: Step[] = [s1, other]
  chainContentVersion(undo, 'k', 6, 7)
  assert.equal((s1 as { version: number }).version, 7)
  assert.equal((other as { version: number }).version, 6)
  // Ada suntingan orang lain di antaranya (versi tidak bersambung): langkah lama tetap dilewati.
  const s0: Step = { kind: 'content', id: 'k', before: 'z', after: 'a', version: 3 }
  chainContentVersion([s0], 'k', 6, 7)
  assert.equal((s0 as { version: number }).version, 3)
  void s2
})
