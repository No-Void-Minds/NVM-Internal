// Revisi 57a: tabel ERD. Tinggi mengikuti jumlah baris, port per baris di kiri dan kanan, garis tetap di barisnya saat
// baris diurutkan ulang atau tabel digeser, dan jatuh ke kotak tabel bila barisnya dihapus.
//   cd frontend && npm run test:unit
import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  ERD_HEAD,
  ERD_ROW,
  erdHeight,
  newRowId,
  newShapeContent,
  routeConnector,
  rowAt,
  rowPortPoint,
} from '../src/utils/wbShapes.ts'

test('tinggi tabel dan baris di titik', () => {
  assert.equal(erdHeight(0), ERD_HEAD + ERD_ROW)
  assert.equal(erdHeight(3), ERD_HEAD + 3 * ERD_ROW)
  assert.equal(rowAt(3, ERD_HEAD - 1), null)
  assert.equal(rowAt(3, ERD_HEAD), 0)
  assert.equal(rowAt(3, ERD_HEAD + ERD_ROW * 2.5), 2)
  assert.equal(rowAt(3, ERD_HEAD + ERD_ROW * 3), null)
  const c = newShapeContent('erd_table')
  assert.equal(c.rows?.length, 2)
  assert.deepEqual(c.rows?.[0]?.keys, ['pk'])
  assert.notEqual(c.rows?.[0]?.id, c.rows?.[1]?.id)
  const id = newRowId(['a'])
  assert.match(id, /^[a-z0-9]{1,16}$/)
})

test('port baris di tengah baris, ikut putaran', () => {
  const b = { x: 100, y: 50, w: 200, h: erdHeight(3) }
  const p = rowPortPoint(b, ['a', 'b', 'c'], 'b', 'right')!
  assert.deepEqual(p.p, { x: 300, y: 50 + ERD_HEAD + ERD_ROW * 1.5 })
  assert.deepEqual(p.dir, { x: 1, y: 0 })
  assert.equal(rowPortPoint(b, ['a'], 'x', 'left'), null)
  const r = rowPortPoint(b, ['a', 'b', 'c'], 'a', 'left', 180)!
  // Diputar 180 derajat: sisi kiri pindah ke kanan dan baris pertama ke bawah.
  assert.ok(Math.abs(r.p.x - 300) < 1e-9 && Math.abs(r.p.y - (50 + b.h - ERD_HEAD - ERD_ROW / 2)) < 1e-9)
})

const line = (fromRow?: string) => ({
  id: 'L',
  from: 'A',
  to: 'B',
  geometry: {
    from: { port: 'right', ...(fromRow ? { row: fromRow } : {}) },
    to: { port: 'left', row: 'k' },
    route: 'straight' as const,
    points: [] as [number, number][],
  },
  style: { start: 'cf_many' as const, end: 'cf_one' as const },
})

test('garis menempel ke baris, ikut urutan baru dan geseran tabel', () => {
  const geom = new Map([
    ['A', { x: 0, y: 0, w: 200, h: erdHeight(3) }],
    ['B', { x: 400, y: 0, w: 200, h: erdHeight(2) }],
  ])
  const rows = new Map([
    ['A', ['x', 'y', 'z']],
    ['B', ['j', 'k']],
  ])
  const r = routeConnector(line('z'), geom, new Map(), undefined, undefined, rows)!
  assert.deepEqual(r.start, { x: 200, y: ERD_HEAD + ERD_ROW * 2.5 })
  assert.deepEqual(r.end, { x: 400, y: ERD_HEAD + ERD_ROW * 1.5 })
  // Baris z dipindah ke atas: ujung ikut ke baris pertama.
  rows.set('A', ['z', 'x', 'y'])
  assert.equal(
    routeConnector(line('z'), geom, new Map(), undefined, undefined, rows)!.start.y,
    ERD_HEAD + ERD_ROW / 2,
  )
  // Tabel digeser: ujung ikut.
  geom.set('A', { x: 0, y: 100, w: 260, h: erdHeight(3) })
  const moved = routeConnector(line('z'), geom, new Map(), undefined, undefined, rows)!
  assert.deepEqual(moved.start, { x: 260, y: 100 + ERD_HEAD + ERD_ROW / 2 })
  // Baris dihapus: ujung jatuh ke port otomatis di kotak tabel, garisnya tetap ada.
  rows.set('A', ['x', 'y'])
  const gone = routeConnector(line('z'), geom, new Map(), undefined, undefined, rows)!
  assert.ok(gone)
  assert.deepEqual(gone.start, { x: 260, y: 100 + erdHeight(3) / 2 })
})
