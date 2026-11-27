// Revisi 57d1: letak bentuk tersambung baru tanpa menimpa kartu lain.
//   cd frontend && npm run test:unit
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { DIR_PORTS, QUICK_GAP, placeNext } from '../src/utils/wbQuickAdd.ts'

const src = { x: 0, y: 0, w: 180, h: 80 }
const size = { w: 180, h: 80 }

test('letak awal di tiap arah, sejajar tengah bentuk asal', () => {
  assert.deepEqual(placeNext(src, 'right', size, []), { x: 180 + QUICK_GAP, y: 0, w: 180, h: 80 })
  assert.deepEqual(placeNext(src, 'left', size, []), { x: -QUICK_GAP - 180, y: 0, w: 180, h: 80 })
  assert.deepEqual(placeNext(src, 'down', size, []), { x: 0, y: 80 + QUICK_GAP, w: 180, h: 80 })
  assert.deepEqual(placeNext(src, 'up', size, []), { x: 0, y: -QUICK_GAP - 80, w: 180, h: 80 })
  assert.deepEqual(DIR_PORTS.down, { from: 'bottom', to: 'top' })
})

test('menghindari kartu lain: menjauh searah arah, lalu tegak lurus', () => {
  const block = { x: 260, y: -20, w: 100, h: 120 }
  const r = placeNext(src, 'right', size, [block])
  assert.equal(r.y, 0)
  assert.ok(r.x >= block.x + block.w + 16, `x ${r.x}`)
  // Seluruh jalur ke kanan terisi: geser tegak lurus.
  const wall = { x: 200, y: -30, w: 5000, h: 140 }
  const s = placeNext(src, 'right', size, [wall])
  const clear = s.y + s.h + 16 <= wall.y || s.y >= wall.y + wall.h + 16
  assert.ok(clear, JSON.stringify(s))
})
