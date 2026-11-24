// Test unit logika kanvas Whiteboard (W1).
//   cd frontend && npm run test:unit
import { test } from 'node:test'
import assert from 'node:assert/strict'

import {
  activityText,
  cardLabel,
  fitView,
  intersects,
  nearestInDirection,
  readingOrder,
  snap,
  toWorld,
  visibleWorld,
  zoomAt,
  type Card,
} from '../src/utils/whiteboard.ts'

const r = (id: string, x: number, y: number, w = 100, h = 50) => ({ id, x, y, w, h })

test('snap ke grid 8 px hanya bila grid menyala', () => {
  assert.equal(snap(13, true), 16)
  assert.equal(snap(11, true), 8)
  assert.equal(snap(13.4, false), 13)
})

test('zoom di sekitar penunjuk menjaga titik dunia di bawahnya', () => {
  const v = { x: 40, y: 20, zoom: 1 }
  const before = toWorld(v, 300, 200)
  const z = zoomAt(v, 300, 200, 1.5)
  const after = toWorld(z, 300, 200)
  assert.ok(Math.abs(before.x - after.x) < 1e-9 && Math.abs(before.y - after.y) < 1e-9)
  assert.equal(zoomAt(v, 0, 0, 10).zoom, 2)
  assert.equal(zoomAt(v, 0, 0, 0.01).zoom, 0.25)
})

test('pas ke layar memuat semua kartu dan tidak memperbesar di atas 100%', () => {
  const cards = [r('a', 0, 0), r('b', 2000, 1000)]
  const v = fitView(cards, 1000, 600)
  const world = visibleWorld(v, 1000, 600, 0)
  for (const c of cards) assert.ok(intersects(world, c))
  assert.ok(v.zoom < 1)
  assert.equal(fitView([r('a', 0, 0)], 1000, 600).zoom, 1)
})

test('urutan baca: baris dulu, lalu kiri ke kanan, toleransi 24 px', () => {
  const order = readingOrder([r('c', 0, 200), r('b', 300, 10), r('a', 0, 0)]).map((c) => c.id)
  assert.deepEqual(order, ['a', 'b', 'c'])
})

test('panah memilih kartu terdekat di arah itu, bukan yang di belakang', () => {
  const from = r('from', 500, 500)
  const cards = [
    from,
    r('kanan', 700, 510),
    r('kanan-jauh', 1200, 500),
    r('kiri', 200, 500),
    r('bawah', 500, 800),
  ]
  assert.equal(nearestInDirection(from, cards, 'right')?.id, 'kanan')
  assert.equal(nearestInDirection(from, cards, 'left')?.id, 'kiri')
  assert.equal(nearestInDirection(from, cards, 'down')?.id, 'bawah')
  assert.equal(nearestInDirection(from, cards, 'up'), null)
})

test('label kartu untuk pembaca layar', () => {
  const base = {
    id: 'x',
    board_id: 'b',
    x: 0,
    y: 0,
    w: 1,
    h: 1,
    z: 0,
    color: null,
    version: 1,
    content_version: 1,
    seq: 1,
    updated_by: 1,
    updated_at: '',
  }
  const note: Card = {
    ...base,
    kind: 'note',
    content: {
      doc: {
        type: 'doc',
        content: [{ type: 'paragraph', content: [{ type: 'text', text: 'Brief klien' }] }],
      },
    },
  }
  const list: Card = {
    ...base,
    kind: 'checklist',
    content: {
      title: 'Rilis',
      items: [
        { id: '1', text: 'a', done: true },
        { id: '2', text: 'b', done: false },
      ],
    },
  }
  assert.equal(cardLabel(note), 'Catatan: Brief klien')
  assert.equal(cardLabel(list), 'Checklist Rilis, 1 dari 2 selesai')
  const board: Card = {
    ...base,
    kind: 'board',
    content: {},
    ref: { id: 'r', title: '2026', version: 1, boards: 22, cards: 4 },
  }
  assert.equal(cardLabel(board), 'Papan 2026, 22 papan, 4 kartu')
})

test('kalimat riwayat', () => {
  const a = (action: string, summary: Record<string, unknown>) =>
    activityText({ id: 1, action, summary, created_at: '', updated_at: '', user_name: 'Rina' })
  assert.equal(a('cards_arranged', { count: 3 }), 'mengatur letak, ukuran, atau warna 3 kartu')
  assert.equal(a('card_edited', { kind: 'note', excerpt: 'Halo' }), 'mengubah isi catatan: "Halo"')
  assert.equal(
    a('cards_deleted', { count: 2, boards: 1 }),
    'memindahkan 2 kartu ke Tempat sampah, termasuk 1 papan',
  )
  assert.equal(a('board_renamed', { from: 'A', to: 'B' }), 'mengganti judul papan dari "A" menjadi "B"')
})

import { reconnectDelay } from '../src/utils/wbSocket.ts'
import { layoutPatches, push, MAX_STEPS, type Step } from '../src/utils/wbHistory.ts'

test('jeda sambung ulang 1, 2, 4, 8, 16, lalu 30 detik ditambah acak sampai 20%', () => {
  const none = () => 0
  assert.deepEqual(
    [0, 1, 2, 3, 4, 5, 9].map((a) => reconnectDelay(a, none)),
    [1000, 2000, 4000, 8000, 16000, 30000, 30000],
  )
  assert.equal(
    reconnectDelay(0, () => 0.999),
    1200,
  )
  assert.ok(reconnectDelay(5, () => 0.5) > 30000 && reconnectDelay(5, () => 0.5) <= 36000)
})

test('undo tata letak melewati field yang sudah diubah orang lain', () => {
  const base = {
    board_id: 'b',
    kind: 'note' as const,
    z: 0,
    content: {},
    version: 1,
    content_version: 1,
    seq: 1,
    updated_by: 1,
    updated_at: '',
  }
  const step: Extract<Step, { kind: 'layout' }> = {
    kind: 'layout',
    cards: [
      { id: 'a', before: { x: 0, y: 0, color: null }, after: { x: 100, y: 0, color: 'kuning' } },
      { id: 'b', before: { x: 10 }, after: { x: 20 } },
    ],
  }
  // Kartu a: x masih hasil aksi sendiri, warna sudah diganti orang lain. Kartu b sudah dihapus.
  const current = new Map([['a', { ...base, id: 'a', x: 100, y: 0, w: 1, h: 1, color: 'biru' as const }]])
  const undo = layoutPatches(step, current, 'undo')
  assert.deepEqual(undo.patches, [{ id: 'a', x: 0, y: 0 }])
  assert.equal(undo.skipped, 2)
  const redoState = new Map([['a', { ...base, id: 'a', x: 0, y: 0, w: 1, h: 1, color: null }]])
  assert.deepEqual(layoutPatches(step, redoState, 'redo').patches, [
    { id: 'a', x: 100, y: 0, color: 'kuning' },
  ])
})

test('riwayat dibatasi dan redo dibuang setelah aksi baru', () => {
  const stack = { undo: [] as Step[], redo: [{ kind: 'create', ids: ['x'] }] as Step[] }
  for (let i = 0; i < MAX_STEPS + 5; i++) push(stack, { kind: 'create', ids: [String(i)] })
  assert.equal(stack.undo.length, MAX_STEPS)
  assert.equal(stack.redo.length, 0)
})
