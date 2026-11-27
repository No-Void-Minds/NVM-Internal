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

test('sequence (54d): pesan mendatar ikut lifeline asal, ke diri sendiri, tepi activation bar', async () => {
  const {
    routeConnector,
    shapeHandles,
    newShapeContent: nsc,
    LINE_PRESETS,
  } = await import('../src/utils/wbShapes.ts')
  const geom = new Map([
    ['A', { x: 0, y: 0, w: 140, h: 400 }],
    ['B', { x: 300, y: 40, w: 140, h: 400 }],
    ['bar', { x: 350, y: 100, w: 40, h: 120 }],
  ])
  const shapes = new Map([
    ['A', 'uml_lifeline'],
    ['B', 'uml_lifeline'],
    ['bar', 'uml_activation'],
  ])
  const msg = (from: string, to: string, y: number) => ({
    id: 'm',
    from,
    to,
    geometry: {
      from: { port: 'auto' },
      to: { port: 'auto' },
      route: 'message' as const,
      points: [[0, y]] as [number, number][],
    },
    style: { start: 'none' as const, end: 'arrow' as const },
  })
  const r = routeConnector(msg('A', 'B', 120), geom, new Map(), undefined, shapes)!
  assert.deepEqual(
    [r.start, r.end],
    [
      { x: 70, y: 120 },
      { x: 370, y: 120 },
    ],
  )
  // Lifeline asal digeser: pesan ikut, tetap mendatar.
  geom.set('A', { x: 0, y: 50, w: 140, h: 400 })
  const r2 = routeConnector(msg('A', 'B', 120), geom, new Map(), undefined, shapes)!
  assert.equal(r2.start.y, 170)
  assert.equal(r2.end.y, 170)
  // Draf seret: y dunia, tetap mendatar.
  const r3 = routeConnector(msg('A', 'B', 120), geom, new Map(), [[999, 260]], shapes)!
  assert.equal(r3.start.y, 260)
  assert.equal(r3.end.y, 260)
  // Ke activation bar: berhenti di tepi bar yang menghadap pengirim (7 px dari tengah).
  const r4 = routeConnector(msg('A', 'bar', 120), geom, new Map(), undefined, shapes)!
  assert.equal(r4.end.x, 363)
  // Ke diri sendiri: putaran ke kanan, pegangan di tinggi pesan.
  const self = routeConnector(msg('A', 'A', 120), geom, new Map(), undefined, shapes)!
  assert.equal(self.pts.length, 4)
  assert.deepEqual(self.start, { x: 70, y: 170 })
  assert.deepEqual(self.end, { x: 70, y: 194 })
  assert.equal(self.stops[1]!.y, 170)
  // Lifeline hanya diperpanjang ke bawah; activation bar ke atas atau bawah.
  assert.deepEqual(shapeHandles('uml_lifeline'), ['s'])
  assert.deepEqual(shapeHandles('uml_activation'), ['n', 's'])
  assert.equal(shapeHandles('rect').length, 8)
  // Fragmen: alt dua operand, opt satu; preset destroy mengakhiri lifeline.
  assert.equal(nsc('uml_fragment', 'alt').lanes?.length, 2)
  assert.equal(nsc('uml_fragment', 'opt').lanes?.length, 1)
  assert.ok(LINE_PRESETS.find((p) => p.id === 'msg_destroy')?.destroy)
  assert.equal(shapeMarks({ shape: 'uml_lifeline', variant: 'destroyed' }, 140, 400).length, 2)
  assert.ok(shapeMarks({ shape: 'uml_lifeline', variant: 'none' }, 140, 400)[0]!.dash)
})

test('aktor dan lifeline aktor: port kiri dan kanan di ujung tangan, juga sesudah putar dan ubah ukuran', async () => {
  const { portInset, portPoint } = await import('../src/utils/wbShapes.ts')
  const near = (a: { x: number; y: number }, b: { x: number; y: number }) =>
    assert.ok(Math.hypot(a.x - b.x, a.y - b.y) < 1e-6, `${JSON.stringify(a)} != ${JSON.stringify(b)}`)
  const b = { x: 0, y: 0, w: 80, h: 160 }
  const ins = portInset('uml_actor', 80, 160)
  near(portPoint(b, 'left', 0, ins).p, { x: 8, y: 56 })
  near(portPoint(b, 'right', 0, ins).p, { x: 72, y: 56 })
  near(portPoint(b, 'top', 0, ins).p, { x: 40, y: 0 })
  near(portPoint(b, 'bottom', 0, ins).p, { x: 40, y: 160 })
  // Diputar 90 derajat terhadap pusat (40, 80): tangan kanan (72, 56) pindah ke (64, 112).
  near(portPoint(b, 'right', 90, ins).p, { x: 64, y: 112 })
  // Diubah ukurannya: tetap di ujung tangan.
  near(portPoint({ x: 0, y: 0, w: 120, h: 200 }, 'left', 0, portInset('uml_actor', 120, 200)).p, {
    x: 12,
    y: 70,
  })
  // Lifeline aktor: tangan di kepala setinggi 70 px.
  near(portPoint({ x: 0, y: 0, w: 100, h: 400 }, 'right', 0, portInset('uml_lifeline_actor', 100, 400)).p, {
    x: 90,
    y: 24.5,
  })
})

test("ERD dan class (54e): crow's foot, preset, varian kelas, atribut kunci, entitas lemah", async () => {
  const w = await import('../src/utils/wbShapes.ts')
  const tip = { x: 100, y: 0 }
  const from = { x: 0, y: 0 }
  for (const k of ['cf_one', 'cf_many', 'cf_zero_one', 'cf_one_many', 'cf_zero_many'] as const) {
    const m = w.endMarker(k, tip, from)!
    assert.ok(m.d.startsWith('M') && !m.d.includes('NaN'), k)
    // Hanya varian "nol" yang punya lingkaran berisi warna permukaan; garis cabang tidak terisi.
    assert.equal(m.fill, k.startsWith('cf_zero') ? 'surface' : 'none', k)
    assert.ok(w.END_KINDS.some((e) => e.id === k))
  }
  // Banyak: tiga cabang menyentuh tepi entitas (titik ujung).
  assert.equal((w.endMarker('cf_many', tip, from)!.d.match(/L100 /g) ?? []).length, 3)
  const p = w.LINE_PRESETS.find((x) => x.id === 'erd_one_many')!
  assert.deepEqual([p.start, p.end], ['cf_one', 'cf_many'])
  assert.equal(w.classStereotype('interface'), '«interface»')
  assert.equal(w.classStereotype('enum'), '«enumeration»')
  assert.equal(w.classStereotype('abstract'), '')
  assert.equal(w.newShapeContent('uml_class').variant, 'class')
  assert.ok(w.underlineText('erd_key_attribute') && !w.underlineText('erd_attribute'))
  assert.equal(shapeMarks({ shape: 'erd_weak_entity' }, 160, 70).length, 1)
})

test('lainnya (54f): kelompok baru, mind map, kotak centang, garis waktu, corong', async () => {
  const w = await import('../src/utils/wbShapes.ts')
  const count = (g: string) => w.SHAPES.filter((s) => s.group === g).length
  assert.equal(count('Arsitektur'), 9)
  assert.equal(count('Wireframe'), 7)
  assert.equal(count('Mind map'), 2)
  assert.equal(count('Lainnya'), 4)
  assert.ok(w.SHAPES.filter((s) => s.id.startsWith('fc_')).length === 7)
  assert.ok(w.isMindmap('mm_central') && !w.isMindmap('rect') && !w.isMindmap(undefined))
  assert.equal(w.newShapeContent('mm_central').font_size, 20)
  assert.equal(w.newShapeContent('mm_central').fill, 'kuning')
  assert.equal(shapeMarks({ shape: 'wf_checkbox', variant: 'unchecked' }, 180, 40).length, 0)
  assert.equal(shapeMarks({ shape: 'wf_checkbox', variant: 'checked' }, 180, 40).length, 1)
  assert.ok(w.openShape('timeline'))
  assert.equal(shapeMarks({ shape: 'timeline' }, 420, 80).length, 4)
  // Garis bantu corong berada di dalam tepinya.
  const d = shapeMarks({ shape: 'funnel' }, 160, 140)[0]!.d
  const [x1, , x2] = (d.match(/-?\d+(\.\d+)?/g) ?? []).map(Number)
  assert.ok(x1! > 0 && x2! < 160 && x1! < x2!, d)
})
