// Revisi 54b: anggota wadah dari geometri, wadah bersarang, lane (kotak, ubah ukuran, tambah).
//   cd frontend && npm run test:unit
import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  BAND,
  MIN_LANE,
  addLane,
  containerMembers,
  containersAt,
  laneBoxes,
  resizeLane,
} from '../src/utils/wbContainers.ts'
import { isContainer, laneOrient, newShapeContent, shapeMarks, textBox } from '../src/utils/wbShapes.ts'

const card = (
  id: string,
  x: number,
  y: number,
  w: number,
  h: number,
  z: number,
  column_id: string | null = null,
) => ({
  id,
  x,
  y,
  w,
  h,
  z,
  column_id,
})

test('anggota wadah: seluruhnya di dalam dan di atas wadah; bersarang ikut; isi kolom dan wadah diputar tidak', () => {
  const sys = card('sys', 0, 0, 1000, 600, 1)
  const pool = card('pool', 50, 50, 700, 300, 2)
  const task = card('task', 200, 100, 160, 90, 3)
  const half = card('half', 950, 100, 160, 90, 4)
  const under = card('under', 400, 400, 100, 60, 0)
  const inCol = card('inCol', 100, 450, 100, 60, 5, 'kolom')
  const all = [sys, pool, task, half, under, inCol]
  const id = (c: { id: string }) => c.id
  const m = containerMembers(
    [sys],
    all,
    (c) => c,
    () => false,
  )
  assert.deepEqual([...m].sort(), ['pool', 'task'])
  assert.deepEqual(
    [
      ...containerMembers(
        [pool],
        all,
        (c) => c,
        () => false,
      ),
    ],
    ['task'],
  )
  assert.equal(
    containerMembers(
      [sys],
      all,
      (c) => c,
      (x) => x === 'sys',
    ).size,
    0,
  )
  // Kotak tampil sesudah putar dipakai: bentuk diputar yang keluar dari wadah bukan anggota.
  const big = (c: { id: string; x: number; y: number; w: number; h: number }) =>
    c.id === 'task' ? { ...c, w: 2000 } : c
  assert.ok(!containerMembers([pool], all, big, () => false).has('task'))
  assert.deepEqual(containersAt({ x: 210, y: 110, w: 50, h: 50 }, [sys, pool], new Set()).map(id), [
    'pool',
    'sys',
  ])
  assert.deepEqual(containersAt({ x: 210, y: 110, w: 50, h: 50 }, [sys, pool], new Set(['pool'])).map(id), [
    'sys',
  ])
})

test('lane: kotak proporsional, pembatas digeser dengan batas minimal, tambah lane menumbuhkan bentuk', () => {
  const lanes = [
    { label: 'Penjual', size: 150 },
    { label: 'Gudang', size: 150 },
  ]
  const rows = laneBoxes(lanes, 600, 400, 'rows')
  assert.deepEqual(
    rows.map((b) => [b.x, b.y, b.w, b.h]),
    [
      [BAND, 0, 600 - BAND, 200],
      [BAND, 200, 600 - BAND, 200],
    ],
  )
  const cols = laneBoxes(lanes, 400, 300, 'cols')
  assert.deepEqual(
    cols.map((b) => [b.x, b.y, b.w, b.h]),
    [
      [0, BAND, 200, 300 - BAND],
      [200, BAND, 200, 300 - BAND],
    ],
  )
  // Bobot 150 + 150 tampil 400 px: geser 50 px berarti 150 + 50 * (300 / 400).
  const r = resizeLane(lanes, 0, 50, 400)
  assert.deepEqual(
    r.map((l) => l.size),
    [188, 113],
  )
  const clamped = resizeLane(lanes, 0, 10000, 400)
  assert.ok(clamped[1]!.size * (400 / 300) >= MIN_LANE - 1)
  assert.equal(resizeLane(lanes, 1, 50, 400), lanes)
  const { lanes: three, grow } = addLane(lanes, 400)
  assert.equal(three.length, 3)
  assert.equal(three[2]!.label, 'Lane 3')
  assert.equal(grow, 200)
  assert.equal(
    addLane(
      Array.from({ length: 20 }, () => ({ label: '', size: 50 })),
      400,
    ).grow,
    0,
  )
})

test('pool dan swimlane: wadah, lane bawaan, pembatas digambar, judul di pita', () => {
  assert.ok(isContainer('bpmn_pool') && isContainer('uml_swimlane') && isContainer('uml_system'))
  assert.ok(!isContainer('bpmn_task'))
  const pool = newShapeContent('bpmn_pool')
  assert.equal(pool.lanes?.length, 2)
  assert.equal(newShapeContent('rect').lanes, undefined)
  assert.equal(laneOrient('bpmn_pool'), 'rows')
  assert.equal(laneOrient('uml_swimlane'), 'cols')
  const d = shapeMarks(pool, 600, 300)[0]!.d
  assert.ok(d.includes(`M${BAND} 150H600`), d)
  assert.deepEqual(textBox('bpmn_pool', 600, 300), {
    x: 0,
    y: 0,
    w: BAND,
    h: 300,
    valign: 'center',
    vertical: true,
  })
})

test('activity dan state (54c): node awal dan fork pekat, node akhir bertanda, state komposit wadah', async () => {
  const { solidShape, shapePath, isContainer: wadah } = await import('../src/utils/wbShapes.ts')
  assert.ok(solidShape('uml_initial') && solidShape('uml_fork') && !solidShape('uml_action'))
  assert.equal(shapeMarks({ shape: 'uml_final' }, 40, 40).length, 1)
  assert.equal(shapeMarks({ shape: 'uml_final' }, 40, 40)[0]!.fill, 'line')
  assert.ok(wadah('uml_state_composite') && !wadah('uml_state'))
  // Batang fork 10 px di tengah kotak 40 px.
  assert.ok(
    shapePath('uml_fork', 160, 40).startsWith('M0 15L160 15L160 25L0 25'),
    shapePath('uml_fork', 160, 40),
  )
})

test('kotak port: garis menempel di tepi batang fork dan lingkaran, juga sesudah putar dan ubah ukuran', async () => {
  const { portInset, portPoint, routeConnector } = await import('../src/utils/wbShapes.ts')
  const near = (a: { x: number; y: number }, b: { x: number; y: number }) =>
    assert.ok(Math.hypot(a.x - b.x, a.y - b.y) < 1e-6, `${JSON.stringify(a)} != ${JSON.stringify(b)}`)
  const fork = { x: 100, y: 100, w: 160, h: 40 }
  const ins = portInset('uml_fork', fork.w, fork.h)!
  near(portPoint(fork, 'top', 0, ins).p, { x: 180, y: 115 })
  near(portPoint(fork, 'bottom', 0, ins).p, { x: 180, y: 125 })
  // Putar 90: batang tegak, port atas pindah ke kanan batang (5 px dari pusat), bukan tepi kotak (20 px).
  near(portPoint(fork, 'top', 90, ins).p, { x: 185, y: 120 })
  // Ubah ukuran: batang tetap 10 px di tengah.
  const tall = { x: 0, y: 0, w: 200, h: 90 }
  near(portPoint(tall, 'top', 0, portInset('uml_fork', 200, 90)).p, { x: 100, y: 40 })
  // Lingkaran di kotak tidak persegi: port di tepi lingkaran.
  near(portPoint({ x: 0, y: 0, w: 120, h: 40 }, 'left', 0, portInset('uml_initial', 120, 40)).p, {
    x: 40,
    y: 20,
  })
  assert.equal(portInset('bpmn_task', 160, 90), null)
  // Ujung garis lurus dan siku dari aksi ke fork menyentuh batang.
  const geom = new Map([
    ['aksi', { x: 100, y: -100, w: 160, h: 60 }],
    ['fork', fork],
  ])
  const shapes = new Map([
    ['aksi', 'uml_action'],
    ['fork', 'uml_fork'],
  ])
  for (const route of ['straight', 'orthogonal'] as const) {
    const r = routeConnector(
      {
        id: 'l',
        from: 'aksi',
        to: 'fork',
        geometry: { from: { port: 'auto' }, to: { port: 'auto' }, route, points: [] },
        style: { start: 'none', end: 'arrow' },
      },
      geom,
      new Map(),
      undefined,
      shapes,
    )!
    near(r.end, { x: 180, y: 115 })
  }
})
