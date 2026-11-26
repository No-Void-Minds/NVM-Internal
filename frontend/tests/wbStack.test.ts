import { test } from 'node:test'
import assert from 'node:assert/strict'
import { lineZ, restack, stackOrder, stackRanks } from '../src/utils/wbStack.ts'
import { elastic, routeConnector, simplifyBends, snap45 } from '../src/utils/wbShapes.ts'

const near = (a: { x: number; y: number }, b: { x: number; y: number }) =>
  assert.ok(Math.hypot(a.x - b.x, a.y - b.y) < 1e-6, `${JSON.stringify(a)} != ${JSON.stringify(b)}`)

// Diagram use case: batas sistem di belakang (z -5), aktor 1, use case 2, kartu penutup 3.
const cards = [
  { id: 'sys', z: -5 },
  { id: 'aktor', z: 1 },
  { id: 'uc', z: 2 },
  { id: 'tutup', z: 3 },
]

test('lineZ: otomatis ikut ujung paling depan, manual menang, tanpa ujung paling belakang', () => {
  const zOf = new Map(cards.map((c) => [c.id, c.z]))
  assert.equal(lineZ({ id: 'l', from: 'aktor', to: 'uc' }, zOf), 2)
  assert.equal(lineZ({ id: 'l', from: 'aktor', to: null }, zOf), 1)
  assert.equal(lineZ({ id: 'l', from: 'aktor', to: 'uc', z: -9 }, zOf), -9)
  assert.equal(lineZ({ id: 'l', from: null, to: null }, zOf), -Infinity)
  assert.equal(lineZ({ id: 'l', from: null, to: null, z: 4 }, zOf), 4)
})

test('stackOrder: garis tepat di atas ujungnya, di bawah kartu yang lebih depan', () => {
  const order = stackOrder(cards, [{ id: 'L', from: 'aktor', to: 'uc' }]).map((l) => l.id)
  assert.deepEqual(order, ['sys', 'aktor', 'uc', 'L', 'tutup'])
  const ranks = stackRanks(stackOrder(cards, [{ id: 'L', from: 'aktor', to: 'uc' }]))
  assert.ok(ranks.get('L')! > ranks.get('sys')! && ranks.get('L')! < ranks.get('tutup')!)
})

test('restack garis: maju melewati satu kartu, mundur ke bawah ujungnya, depan dan belakang', () => {
  const order = stackOrder(cards, [{ id: 'L', from: 'aktor', to: 'uc' }])
  const sel = new Set(['L'])
  const z = (k: Parameters<typeof restack>[2]) => restack(order, sel, k)[0]!.z
  const pos = (k: Parameters<typeof restack>[2]) =>
    stackOrder(cards, [{ id: 'L', from: 'aktor', to: 'uc', z: z(k) }]).map((l) => l.id)
  assert.deepEqual(pos('forward'), ['sys', 'aktor', 'uc', 'tutup', 'L'])
  assert.deepEqual(pos('backward'), ['sys', 'aktor', 'L', 'uc', 'tutup'])
  assert.deepEqual(pos('front'), ['sys', 'aktor', 'uc', 'tutup', 'L'])
  assert.deepEqual(pos('back'), ['L', 'sys', 'aktor', 'uc', 'tutup'])
})

test('restack kartu: maju dan mundur satu lapis seperti sebelum revisi 53', () => {
  const order = stackOrder(cards, [])
  const r = restack(order, new Set(['aktor']), 'forward')[0]!
  assert.ok(r.z > 2 && r.z < 3)
  const b = restack(order, new Set(['uc']), 'backward')[0]!
  assert.ok(b.z > -5 && b.z < 1)
  assert.equal(restack(order, new Set(['sys']), 'back')[0]!.z, -6)
})

test('elastic: geser satu ujung, geser kedua ujung, ubah ukuran', () => {
  const a0 = { x: 0, y: 0 }
  const b0 = { x: 100, y: 0 }
  const pts = [
    { x: 30, y: 40 },
    { x: 70, y: -20 },
  ]
  // Kedua ujung bergeser sama: titik bergeser sama.
  const both = elastic(pts, a0, b0, { x: 10, y: 5 }, { x: 110, y: 5 })
  near(both[0]!, { x: 40, y: 45 })
  near(both[1]!, { x: 80, y: -15 })
  // Satu ujung menjauh dua kali: titik di skala dua kali dari ujung yang diam.
  const one = elastic(pts, a0, b0, a0, { x: 200, y: 0 })
  near(one[0]!, { x: 60, y: 80 })
  near(one[1]!, { x: 140, y: -40 })
  // Ujung diputar 90 derajat (kartu pindah ke bawah): bentuk ikut berputar, jarak relatif tetap.
  const turn = elastic(pts, a0, b0, a0, { x: 0, y: 100 })
  near(turn[0]!, { x: -40, y: 30 })
  // Ubah ukuran kartu akhir: pusatnya bergeser setengah perubahan lebar.
  const grow = elastic(pts, a0, b0, a0, { x: 120, y: 0 })
  near(grow[0]!, { x: 36, y: 48 })
  // Ujung lama berimpit: hanya digeser.
  near(elastic([{ x: 5, y: 5 }], a0, a0, { x: 10, y: 0 }, { x: 10, y: 0 })[0]!, { x: 15, y: 5 })
})

test('routeConnector memakai anchor: titik belok ikut kartu ujung yang digeser', () => {
  const c = {
    id: 'L',
    from: 'a',
    to: 'b',
    geometry: {
      from: { port: 'auto' },
      to: { port: 'auto' },
      route: 'bebas' as const,
      points: [[150, 100]] as [number, number][],
      anchor: [
        [50, 50],
        [250, 50],
      ] as [[number, number], [number, number]],
    },
    style: { start: 'none' as const, end: 'arrow' as const },
  }
  const geom = new Map([
    ['a', { x: 0, y: 0, w: 100, h: 100 }],
    ['b', { x: 200, y: 0, w: 100, h: 100 }],
  ])
  assert.deepEqual(routeConnector(c, geom, new Map())!.stops[1], { x: 150, y: 100 })
  geom.set('b', { x: 200, y: 200, w: 100, h: 100 })
  geom.set('a', { x: 0, y: 200, w: 100, h: 100 })
  near(routeConnector(c, geom, new Map())!.stops[1]!, { x: 150, y: 300 })
})

test('simplifyBends: titik segaris dibuang dengan toleransi, belokan nyata tetap', () => {
  const s = [
    { x: 0, y: 0 },
    { x: 50, y: 1 },
    { x: 100, y: 0 },
    { x: 100, y: 80 },
    { x: 100, y: 100 },
  ]
  assert.deepEqual(simplifyBends(s), [{ x: 100, y: 0 }])
  assert.deepEqual(simplifyBends([{ x: 0, y: 0 }, { x: 50, y: 30 }, { x: 100, y: 0 }]), [{ x: 50, y: 30 }])
})

test('snap45: sudut dikunci ke kelipatan 45 derajat', () => {
  near(snap45({ x: 0, y: 0 }, { x: 100, y: 10 }), { x: 100, y: 0 })
  near(snap45({ x: 0, y: 0 }, { x: 100, y: 90 }), { x: 95, y: 95 })
  near(snap45({ x: 10, y: 10 }, { x: 12, y: -90 }), { x: 10, y: -90 })
})
