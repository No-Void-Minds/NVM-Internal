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
  assert.equal(zoomAt(v, 0, 0, 10).zoom, 4)
  assert.equal(zoomAt(v, 0, 0, 0.01).zoom, 0.1)
})

test('pas ke layar memuat semua kartu dan tidak memperbesar di atas 100%', () => {
  const cards = [r('a', 0, 0), r('b', 2000, 1000)]
  const v = fitView(cards, 1000, 600)
  const world = visibleWorld(v, 1000, 600, 0)
  for (const c of cards) assert.ok(intersects(world, c))
  assert.ok(v.zoom < 1)
  assert.equal(fitView([r('a', 0, 0)], 1000, 600).zoom, 1)
})

test('pas ke layar dengan jarak atas dan bawah: kartu tidak masuk area bar pilihan dan kontrol ponsel', () => {
  const cards = [r('a', 0, 0, 240, 120), r('b', 0, 400, 360, 120)]
  const v = fitView(cards, 360, 580, 160, 120)
  // Tepi atas kartu teratas di layar = v.y + 0 * zoom.
  assert.ok(v.y >= 160 - 1e-9, `atas ${v.y}`)
  const bottom = v.y + 520 * v.zoom
  assert.ok(bottom <= 580 - 120 + 1e-9, `bawah ${bottom}`)
  // Bawaan 48 sama dengan perilaku lama: isi di tengah tinggi kanvas.
  const d = fitView(cards, 1000, 600)
  assert.ok(Math.abs(d.y + (520 * d.zoom) / 2 - 300) < 1e-9)
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

// W3: kolom, garis penghubung, penyederhanaan goresan.
import { columnLayout, columnOrderAt, edgePoint, simplify } from '../src/utils/whiteboard.ts'

test('isi kolom ditumpuk sesuai urutan dan kolom memanjang mengikuti isinya', () => {
  const col = { id: 'k', kind: 'column', x: 100, y: 50, w: 280, h: 100, z: 3 }
  const a = { id: 'a', kind: 'note', x: 999, y: 999, w: 240, h: 80, z: 9, column_id: 'k', col_order: 2 }
  const b = { id: 'b', kind: 'note', x: 0, y: 0, w: 240, h: 40, z: 1, column_id: 'k', col_order: 1 }
  const lost = { id: 'c', kind: 'note', x: 5, y: 5, w: 10, h: 10, z: 1, column_id: 'hilang' }
  const { children, heights } = columnLayout([col, a, b, lost])
  assert.deepEqual(children.get('b'), { x: 108, y: 106, w: 264, h: 40, z: 3 })
  assert.deepEqual(children.get('a'), { x: 108, y: 154, w: 264, h: 80, z: 3 })
  assert.equal(children.has('c'), false)
  assert.equal(heights.get('k'), 192)
})

test('urutan kolom saat dilepas di antara, sebelum, dan sesudah tetangga', () => {
  const sib = [
    { y: 0, h: 40, col_order: 1 },
    { y: 50, h: 40, col_order: 2 },
  ]
  assert.equal(columnOrderAt(sib, 5), 0)
  assert.equal(columnOrderAt(sib, 40), 1.5)
  assert.equal(columnOrderAt(sib, 200), 3)
  assert.equal(columnOrderAt([], 10), 1)
})

test('garis penghubung mulai di tepi kartu', () => {
  const r = { x: 0, y: 0, w: 100, h: 50 }
  assert.deepEqual(edgePoint(r, { x: 500, y: 25 }), { x: 100, y: 25 })
  assert.deepEqual(edgePoint(r, { x: 50, y: -300 }), { x: 50, y: 0 })
  assert.deepEqual(edgePoint(r, { x: 50, y: 25 }), { x: 50, y: 25 })
})

test('penyederhanaan goresan membuang titik segaris dan menjaga ujung serta sudut', () => {
  assert.deepEqual(simplify([0, 0, 10, 0, 20, 0, 30, 0], 1), [0, 0, 30, 0])
  assert.deepEqual(simplify([0, 0, 50, 50, 100, 0], 1), [0, 0, 50, 50, 100, 0])
  assert.deepEqual(simplify([1.4, 2.6], 1), [1, 3])
})

// Revisi A: gambar bebas tanpa kartu.
import { hitStroke, strokesBox } from '../src/utils/whiteboard.ts'

test('klik tepat di garis memilih goresan, klik di sebelahnya tembus', () => {
  const obj = {
    id: 'g',
    x: 100,
    y: 100,
    content: {
      strokes: [
        { color: 'tinta', width: 4, points: [0, 0, 100, 0] },
        { color: 'biru', width: 8, points: [0, 50] },
      ],
    },
  }
  assert.deepEqual(hitStroke([obj], 150, 103, 4), { id: 'g', index: 0 })
  assert.equal(hitStroke([obj], 150, 120, 4), null)
  assert.deepEqual(hitStroke([obj], 106, 150, 4), { id: 'g', index: 1 })
})

test('kotak batas goresan sama dengan perhitungan server', () => {
  assert.deepEqual(strokesBox([{ color: 'tinta', width: 8, points: [20, 50, 400, 600] }]), {
    x: 14,
    y: 44,
    w: 392,
    h: 562,
  })
  assert.equal(strokesBox([]), null)
})

// Zoom roda: normalisasi deltaMode dan faktor eksponensial.
import { wheelPixels, wheelZoomFactor } from '../src/utils/whiteboard.ts'

test('wheel baris dan halaman dinormalisasi ke piksel', () => {
  assert.deepEqual(wheelPixels({ deltaX: 0, deltaY: 3, deltaMode: 1 }, 800), { dx: 0, dy: 48 })
  assert.deepEqual(wheelPixels({ deltaX: 1, deltaY: 1, deltaMode: 2 }, 800), { dx: 800, dy: 800 })
  assert.deepEqual(wheelPixels({ deltaX: 2, deltaY: -5, deltaMode: 0 }, 800), { dx: 2, dy: -5 })
})

test('zoom roda sebanding, simetris, dan dibatasi per event', () => {
  const notch = wheelZoomFactor(100)
  assert.ok(notch > 0.75 && notch < 0.8)
  assert.ok(Math.abs(wheelZoomFactor(100) * wheelZoomFactor(-100) - 1) < 1e-12)
  assert.equal(wheelZoomFactor(10000), wheelZoomFactor(240))
  assert.ok(wheelZoomFactor(-2) > 1 && wheelZoomFactor(-2) < 1.01)
})

// W4: judul ringkas (level of detail, Tampilan Daftar) dan baris teks ekspor.
import { cardLines, cardTitle } from '../src/utils/whiteboard.ts'

const base = {
  board_id: 'b',
  x: 0,
  y: 0,
  w: 240,
  h: 120,
  z: 1,
  color: null,
  version: 1,
  content_version: 1,
  seq: 1,
  updated_by: 1,
  updated_at: '',
}
const doc = (...content: unknown[]) => ({ doc: { type: 'doc', content } })
const para = (text: string) => ({ type: 'paragraph', content: [{ type: 'text', text }] })

test('cardTitle memberi judul ringkas untuk setiap jenis kartu', () => {
  const t = (c: Partial<Card>) => cardTitle({ ...base, id: 'x', ...c } as Card)
  assert.equal(t({ kind: 'note', content: doc(para('Rencana konten')) as Card['content'] }), 'Rencana konten')
  assert.equal(t({ kind: 'note', content: doc() as Card['content'] }), 'Catatan kosong')
  assert.equal(t({ kind: 'checklist', content: { title: '', items: [] } }), 'Checklist tanpa judul')
  assert.equal(t({ kind: 'link', content: { url: 'https://contoh.test/a', preview_id: '' } }), 'contoh.test')
  assert.equal(t({ kind: 'drawing', content: { strokes: [] } }), 'Gambar bebas')
  assert.equal(t({ kind: 'table', content: { header: true, rows: [] } }), 'Tabel')
})

test('cardLines: paragraf, judul, daftar bernomor, dan checklist dengan penanda selesai', () => {
  const lines = cardLines({
    ...base,
    id: 'n',
    kind: 'note',
    content: doc(
      { type: 'heading', attrs: { level: 1 }, content: [{ type: 'text', text: 'Judul' }] },
      para('Isi'),
      { type: 'orderedList', attrs: { start: 3 }, content: [{ type: 'listItem', content: [para('tiga')] }] },
      { type: 'bulletList', content: [{ type: 'listItem', content: [para('poin')] }] },
    ) as Card['content'],
  } as Card)
  assert.deepEqual(
    lines.map((l) => [l.text, !!l.bold]),
    [
      ['Judul', true],
      ['Isi', false],
      ['3. tiga', false],
      ['• poin', false],
    ],
  )
  const check = cardLines({
    ...base,
    id: 'c',
    kind: 'checklist',
    content: {
      title: 'Rilis',
      items: [
        { id: '1', text: 'Foto', done: true },
        { id: '2', text: 'Copy', done: false },
      ],
    },
  } as Card)
  assert.deepEqual(
    check.map((l) => [l.text, !!l.muted]),
    [
      ['Rilis', false],
      ['☑ Foto', true],
      ['☐ Copy', false],
    ],
  )
})
