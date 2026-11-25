// Test unit bentuk dan diagram Whiteboard (W5): path bentuk, port setelah putar, jalur siku yang tidak menembus
// kotak asal dan tujuan, label, dan ujung garis.
//   cd frontend && npm run test:unit
import { test } from 'node:test'
import assert from 'node:assert/strict'

import {
  END_KINDS,
  SHAPES,
  autoPort,
  endMarker,
  orthogonalRoute,
  pointAlong,
  portPoint,
  routePath,
  routePoints,
  rotatedBounds,
  segmentHitsBox,
  shapePath,
  type Box,
  type End,
  type Pt,
} from '../src/utils/wbShapes.ts'

const near = (a: number, b: number) => Math.abs(a - b) < 1e-6

function end(box: Box, port: Parameters<typeof portPoint>[1], rotation = 0): End {
  const { p, dir } = portPoint(box, port, rotation)
  return { p, dir, box: rotatedBounds(box, rotation) }
}

// Bagian tengah jalur (tanpa titik port di kedua ujung) tidak boleh menembus kotak mana pun; semua segmen sejajar sumbu.
function assertClear(pts: Pt[], boxes: Box[]) {
  for (let i = 1; i < pts.length; i++) {
    const a = pts[i - 1]!
    const b = pts[i]!
    assert.ok(near(a.x, b.x) || near(a.y, b.y), `segmen miring ${JSON.stringify([a, b])}`)
    for (const r of boxes)
      assert.ok(!segmentHitsBox(a, b, r), `segmen ${JSON.stringify([a, b])} menembus ${JSON.stringify(r)}`)
  }
}

test('semua bentuk punya path SVG yang sah', () => {
  assert.equal(SHAPES.length, 22)
  for (const s of SHAPES) {
    const d = shapePath(s.id, s.w, s.h)
    assert.ok(d.startsWith('M') && !d.includes('NaN') && !d.includes('Infinity'), `${s.id}: ${d}`)
  }
})

test('port setelah putar: atas jadi kanan di 90 derajat, arah ikut berputar', () => {
  const b = { x: 0, y: 0, w: 200, h: 100 }
  const top = portPoint(b, 'top', 90)
  // Pusat (100, 50); titik atas (100, 0) diputar 90 derajat searah jarum jam menjadi (150, 50).
  assert.ok(near(top.p.x, 150) && near(top.p.y, 50), JSON.stringify(top.p))
  assert.ok(near(top.dir.x, 1) && near(top.dir.y, 0), JSON.stringify(top.dir))
  const corner = portPoint(b, 'bottom_right', 180)
  assert.ok(near(corner.p.x, 0) && near(corner.p.y, 0), JSON.stringify(corner.p))
  assert.equal(autoPort(b, { x: 500, y: 50 }), 'right')
  // Diputar 90 derajat searah jarum jam, sisi atas yang menghadap ke kanan.
  assert.equal(autoPort(b, { x: 500, y: 50 }, 90), 'top')
  const rb = rotatedBounds(b, 90)
  assert.ok(near(rb.w, 100) && near(rb.h, 200) && near(rb.x, 50) && near(rb.y, -50))
})

test('jalur siku antar dua kotak sejajar tidak menembus keduanya', () => {
  const A = { x: 0, y: 0, w: 160, h: 100 }
  const B = { x: 400, y: 200, w: 160, h: 100 }
  const pts = orthogonalRoute(end(A, 'right'), end(B, 'left'))
  assertClear(pts, [A, B])
  assert.deepEqual(pts[0], { x: 160, y: 50 })
  assert.deepEqual(pts.at(-1), { x: 400, y: 250 })
})

test('jalur siku saat port saling membelakangi memutar, tidak menembus kotak', () => {
  const A = { x: 0, y: 0, w: 160, h: 100 }
  const B = { x: 300, y: 0, w: 160, h: 100 }
  // Port kiri A dan kanan B: jalur langsung akan menembus keduanya.
  const pts = orthogonalRoute(end(A, 'left'), end(B, 'right'))
  assertClear(pts, [A, B])
  // Kotak bertumpuk vertikal, port bawah A ke atas B yang berada di atas A.
  const C = { x: 0, y: 300, w: 160, h: 100 }
  assertClear(orthogonalRoute(end(C, 'top'), end(A, 'bottom')), [A, C])
  assertClear(orthogonalRoute(end(A, 'top'), end(C, 'bottom')), [A, C])
})

test('jalur siku dari bentuk yang diputar tetap sejajar sumbu dan tidak menembus kotaknya', () => {
  const A = { x: 0, y: 0, w: 160, h: 100 }
  const B = { x: 400, y: 300, w: 120, h: 120 }
  const pts = orthogonalRoute(end(A, 'top', 30), end(B, 'left', 45))
  // Segmen pertama dan terakhir berawal dari port di dalam kotak batas putar; bagian tengahnya harus bersih.
  assertClear(pts.slice(1, -1), [rotatedBounds(A, 30), rotatedBounds(B, 45)])
})

test('titik belok dilewati berurutan; lurus dan melengkung memakai titik apa adanya', () => {
  const A = { x: 0, y: 0, w: 100, h: 100 }
  const B = { x: 600, y: 0, w: 100, h: 100 }
  const bend = { x: 300, y: 400 }
  const ortho = routePoints(end(A, 'bottom'), end(B, 'bottom'), 'orthogonal', [bend])
  assert.ok(
    ortho.some((p) => near(p.x, bend.x) && near(p.y, bend.y)),
    'titik belok tidak dilewati',
  )
  assertClear(ortho, [A, B])
  assert.equal(routePoints(end(A, 'right'), end(B, 'left'), 'straight', [bend]).length, 3)
  const curved = routePath(end(A, 'right'), end(B, 'left'), 'curved')
  assert.ok(curved.d.includes('C') && !curved.d.includes('NaN'))
  const curvedBend = routePath(end(A, 'right'), end(B, 'left'), 'curved', [bend])
  assert.equal((curvedBend.d.match(/C/g) ?? []).length, 2)
})

test('label di tengah panjang jalur dan ujung garis untuk setiap jenis', () => {
  const mid = pointAlong(
    [
      { x: 0, y: 0 },
      { x: 100, y: 0 },
      { x: 100, y: 100 },
    ],
    0.5,
  )
  assert.deepEqual(mid, { x: 100, y: 0 })
  for (const k of END_KINDS) {
    const m = endMarker(k.id, { x: 100, y: 0 }, { x: 0, y: 0 })
    if (k.id === 'none') assert.equal(m, null)
    else assert.ok(m && m.d.startsWith('M') && !m.d.includes('NaN'), k.id)
  }
})

import { angleFrom, resizeBox } from '../src/utils/wbShapes.ts'

test('ubah ukuran: pojok kanan bawah tanpa putar, sisi kiri menjaga tepi kanan, Shift menjaga proporsi', () => {
  const b = { x: 0, y: 0, w: 200, h: 100 }
  assert.deepEqual(resizeBox(b, 0, 'se', { x: 40, y: 20 }, false, 40, 4000), { x: 0, y: 0, w: 240, h: 120 })
  const w = resizeBox(b, 0, 'w', { x: 50, y: 30 }, false, 40, 4000)
  assert.ok(near(w.x, 50) && near(w.x + w.w, 200) && near(w.h, 100) && near(w.y, 0), JSON.stringify(w))
  const r = resizeBox(b, 0, 'se', { x: 100, y: 0 }, true, 40, 4000)
  assert.ok(near(r.w / r.h, 2) && near(r.w, 300), JSON.stringify(r))
  const tiny = resizeBox(b, 0, 'nw', { x: 500, y: 500 }, false, 40, 4000)
  assert.ok(near(tiny.w, 40) && near(tiny.h, 40) && near(tiny.x + tiny.w, 200) && near(tiny.y + tiny.h, 100))
})

test('ubah ukuran bentuk yang diputar: titik seberang pegangan tetap di dunia', () => {
  const b = { x: 100, y: 100, w: 200, h: 100 }
  const rot = 90
  // Pegangan timur (lokal kanan) pada putaran 90 menghadap ke bawah dunia; geser ke bawah 50 menambah lebar 50.
  const fixedBefore = portPoint(b, 'left', rot).p
  const out = resizeBox(b, rot, 'e', { x: 0, y: 50 }, false, 40, 4000)
  assert.ok(near(out.w, 250) && near(out.h, 100), JSON.stringify(out))
  const fixedAfter = portPoint(out, 'left', rot).p
  assert.ok(near(fixedBefore.x, fixedAfter.x) && near(fixedBefore.y, fixedAfter.y))
})

test('sudut putar dari penunjuk, snap 15 derajat dengan Shift', () => {
  const c = { x: 0, y: 0 }
  assert.equal(angleFrom(c, { x: 0, y: -10 }, false), 0)
  assert.equal(angleFrom(c, { x: 10, y: 0 }, false), 90)
  assert.equal(angleFrom(c, { x: -10, y: 0 }, false), 270)
  assert.equal(angleFrom(c, { x: 10, y: -9 }, true), 45)
  assert.equal(angleFrom(c, { x: 1, y: -10 }, true), 0)
})

import { routeConnector } from '../src/utils/wbShapes.ts'

test('garis lengkap: port otomatis, putar, ujung bebas, dan tiga posisi label', () => {
  const geom = new Map([
    ['a', { x: 0, y: 0, w: 160, h: 100 }],
    ['b', { x: 400, y: 0, w: 160, h: 100 }],
  ])
  const base = {
    id: 'g',
    from: 'a',
    to: 'b',
    geometry: {
      from: { port: 'auto' },
      to: { port: 'auto' },
      route: 'orthogonal' as const,
      points: [] as [number, number][],
    },
    style: { start: 'diamond' as const, end: 'arrow' as const },
  }
  const r = routeConnector(base, geom, new Map())!
  assert.deepEqual(r.start, { x: 160, y: 50 })
  assert.deepEqual(r.end, { x: 400, y: 50 })
  assert.equal(r.heads.length, 2)
  assert.ok(r.near.start.x < r.mid.x && r.mid.x < r.near.end.x)
  // Bentuk b diputar 180: port kirinya kini di kanan, jadi port otomatis memilih sisi yang menghadap a.
  const turned = routeConnector(base, geom, new Map([['b', 180]]))!
  assert.ok(near(turned.end.x, 400) && near(turned.end.y, 50), JSON.stringify(turned.end))
  // Ujung bebas tanpa kartu.
  const free = routeConnector(
    { ...base, to: null, geometry: { ...base.geometry, to: { port: '', x: 300, y: 300 } } },
    geom,
    new Map(),
  )!
  assert.deepEqual(free.end, { x: 300, y: 300 })
  // Kartu ujung tidak ada (di papan lain atau di Tempat sampah): garis tidak digambar.
  assert.equal(routeConnector({ ...base, to: 'hilang' }, geom, new Map()), null)
})
