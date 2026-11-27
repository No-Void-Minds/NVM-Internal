// Revisi 57c: pohon mind map dari garis, cabang tersembunyi, tata letak kiri dan kanan, dan geometri tahap corong,
// chevron, serta peristiwa garis waktu.
//   cd frontend && npm run test:unit
import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  GAP_X,
  GAP_Y,
  descendants,
  hiddenByFold,
  layoutMind,
  mindTree,
  newChildSide,
  type MindNode,
} from '../src/utils/wbMindmap.ts'
import {
  eventX,
  newEventAt,
  newShapeContent,
  shapeMarks,
  shapePath,
  stageBoxes,
} from '../src/utils/wbShapes.ts'

const node = (id: string, x: number, y: number, extra: Partial<MindNode> = {}): MindNode => ({
  id,
  x,
  y,
  w: 160,
  h: 60,
  central: false,
  ...extra,
})

test('pohon dari garis: akar topik pusat, arah garis tidak penting, anak urut dari atas', () => {
  const nodes = [
    node('c', 0, 0, { central: true, w: 200, h: 90 }),
    node('a', 300, 100),
    node('b', 300, -100),
    node('d', 500, 100),
    node('l', -300, 0),
  ]
  const tree = mindTree(nodes, [
    { from: 'c', to: 'a' },
    { from: 'b', to: 'c' },
    { from: 'a', to: 'd' },
    { from: 'c', to: 'l' },
  ])
  assert.equal(tree.parent.get('c'), null)
  assert.equal(tree.parent.get('b'), 'c')
  assert.deepEqual(tree.children.get('c'), ['b', 'l', 'a'])
  assert.deepEqual(descendants(tree, 'a'), ['d'])
  assert.equal(tree.root.get('d'), 'c')
  assert.equal(newChildSide(tree, nodes, 'c'), 'left')
  assert.equal(newChildSide(tree, nodes, 'a'), 'right')
  const folded = nodes.map((n) => (n.id === 'a' ? { ...n, folded: true } : n))
  assert.deepEqual(
    [
      ...hiddenByFold(
        folded,
        mindTree(folded, [
          { from: 'c', to: 'a' },
          { from: 'a', to: 'd' },
        ]),
      ),
    ],
    ['d'],
  )
})

test('tata letak: kanan dan kiri topik pusat, subpohon bertumpuk berpusat di induk, terlipat tidak ditata', () => {
  const root = node('c', 0, 0, { central: true, w: 200, h: 90 })
  const nodes = [root, node('r1', 400, -50), node('r2', 400, 50), node('r1a', 700, -60), node('l1', -400, 0)]
  const tree = mindTree(nodes, [
    { from: 'c', to: 'r1' },
    { from: 'c', to: 'r2' },
    { from: 'r1', to: 'r1a' },
    { from: 'c', to: 'l1' },
  ])
  const pos = layoutMind(nodes, tree, 'c')
  assert.equal(pos.get('r1')!.x, 200 + GAP_X)
  assert.equal(pos.get('r1a')!.x, 200 + GAP_X + 160 + GAP_X)
  assert.equal(pos.get('l1')!.x, -GAP_X - 160)
  // Dua cabang kanan setinggi 60 dengan jarak GAP_Y, berpusat di tengah topik pusat (y 45).
  assert.equal(pos.get('r1')!.y, Math.round(45 - (120 + GAP_Y) / 2))
  assert.equal(pos.get('r2')!.y, pos.get('r1')!.y + 60 + GAP_Y)
  assert.equal(pos.get('l1')!.y, 15)
  assert.equal(pos.get('r1a')!.y, pos.get('r1')!.y)
  const folded = nodes.map((n) => (n.id === 'r1' ? { ...n, folded: true } : n))
  assert.equal(layoutMind(folded, tree, 'c').has('r1a'), false)
  // Simpul terkunci tidak dipindah.
  assert.deepEqual(layoutMind(nodes, tree, 'c', new Set(['r2'])).get('r2'), { x: 400, y: 50 })
})

test('tahap corong dan chevron, peristiwa garis waktu, bentuk lama tidak berubah', () => {
  const chevron = stageBoxes('chevron', 480, 70, 3)
  assert.equal(chevron.length, 3)
  assert.ok(chevron.every((b, i) => i === 0 || b.x > chevron[i - 1]!.x + chevron[i - 1]!.w))
  assert.equal(
    (shapePath('chevron', 480, 70, { stages: [{ label: 'a' }, { label: 'b' }] }).match(/M/g) ?? []).length,
    2,
  )
  assert.equal((shapePath('chevron', 180, 70).match(/M/g) ?? []).length, 1)
  const funnel = stageBoxes('funnel', 240, 180, 3)
  assert.ok(funnel[0]!.w > funnel[2]!.w && funnel[1]!.y === 60)
  assert.equal(
    shapeMarks({ shape: 'funnel', stages: [{ label: 'a' }, { label: 'b' }, { label: 'c' }] }, 240, 180)
      .length,
    2,
  )
  assert.equal(eventX(400, 25), 100)
  assert.equal(newEventAt([]), 50)
  assert.equal(
    newEventAt([
      { id: 'a', label: '', date: '', at: 10 },
      { id: 'b', label: '', date: '', at: 90 },
    ]),
    50,
  )
  const t = newShapeContent('timeline')
  assert.deepEqual(
    t.events?.map((e) => e.at),
    [8, 36, 64, 92],
  )
  assert.equal(new Set(t.events?.map((e) => e.id)).size, 4)
  assert.equal(newShapeContent('funnel').stages?.length, 3)
  assert.equal(shapeMarks({ shape: 'timeline' }, 400, 80).length, 4)
})

test('garis tidak wajar: lingkaran, dua induk, dua topik pusat; induk konsisten dan selalu berhenti', () => {
  const links = (pairs: [string, string][]) => pairs.map(([from, to]) => ({ from, to }))
  // Lingkaran C ke A ke B ke C: tiap simpul sekali, penutup lingkaran tidak dipakai.
  let nodes = [node('c', 0, 0, { central: true }), node('a', 300, -50), node('b', 300, 50)]
  let tree = mindTree(
    nodes,
    links([
      ['c', 'a'],
      ['a', 'b'],
      ['b', 'c'],
    ]),
  )
  assert.equal(tree.parent.get('a'), 'c')
  assert.equal(tree.parent.get('b'), 'c')
  assert.deepEqual(descendants(tree, 'c').sort(), ['a', 'b'])
  // Urutan garis tidak berpengaruh.
  const again = mindTree(
    nodes,
    links([
      ['b', 'c'],
      ['a', 'b'],
      ['c', 'a'],
    ]),
  )
  assert.deepEqual(Object.fromEntries(again.parent), Object.fromEntries(tree.parent))
  nodes = [node('c', 0, 0, { central: true }), node('p1', 300, -100), node('p2', 300, 100), node('x', 600, 0)]
  for (const order of [
    links([
      ['c', 'p1'],
      ['c', 'p2'],
      ['p2', 'x'],
      ['p1', 'x'],
    ]),
    links([
      ['p2', 'x'],
      ['p1', 'x'],
      ['c', 'p2'],
      ['c', 'p1'],
    ]),
  ])
    assert.equal(mindTree(nodes, order).parent.get('x'), 'p1')
  // Jarak lebih pendek menang dari letak: x tersambung langsung ke c dan lewat p1.
  assert.equal(
    mindTree(
      nodes,
      links([
        ['c', 'p1'],
        ['p1', 'x'],
        ['c', 'x'],
      ]),
    ).parent.get('x'),
    'c',
  )
  // Dua topik pusat tersambung langsung: keduanya tetap akar; cabang di antara keduanya ikut yang lebih dekat, lalu
  // yang lebih atas.
  nodes = [
    node('c1', 0, 0, { central: true }),
    node('c2', 0, 400, { central: true }),
    node('m', 300, 200),
    node('n', 300, 500),
  ]
  tree = mindTree(
    nodes,
    links([
      ['c1', 'c2'],
      ['c1', 'm'],
      ['c2', 'm'],
      ['c2', 'n'],
      ['n', 'm'],
    ]),
  )
  assert.equal(tree.parent.get('c1'), null)
  assert.equal(tree.parent.get('c2'), null)
  assert.equal(tree.parent.get('m'), 'c1')
  assert.equal(tree.parent.get('n'), 'c2')
  // Lingkaran tanpa topik pusat: akar di simpul paling atas; tata letak dan lipat berhenti.
  nodes = [node('a', 0, 100), node('b', 300, 0), node('d', 300, 200)]
  tree = mindTree(
    nodes,
    links([
      ['a', 'b'],
      ['b', 'd'],
      ['d', 'a'],
    ]),
  )
  assert.equal(tree.parent.get('b'), null)
  assert.deepEqual([...tree.parent.values()].filter((v) => v === null).length, 1)
  const folded = nodes.map((n) => (n.id === 'b' ? { ...n, folded: true } : n))
  assert.deepEqual([...hiddenByFold(folded, tree)].sort(), ['a', 'd'])
  assert.ok(layoutMind(nodes, tree, 'b').size === 2)
})
