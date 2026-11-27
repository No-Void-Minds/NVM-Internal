// Revisi 57b: layar bingkai dan letak gambar (cover, contain, posisi) yang dipakai ekspor, sama dengan object-fit
// dan object-position di layar.
//   cd frontend && npm run test:unit
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { fitImage, frameBox, frameImage, isFrame, isContainer } from '../src/utils/wbShapes.ts'

test('bingkai: jenis, wadah, dan layar', () => {
  assert.ok(isFrame('wf_phone') && isFrame('wf_browser') && isFrame('wf_image') && !isFrame('rect'))
  assert.ok(isContainer('wf_phone') && isContainer('wf_browser') && !isContainer('wf_image'))
  assert.deepEqual(frameBox('wf_image', 160, 120, 2), { x: 2, y: 2, w: 156, h: 116 })
  const phone = frameBox('wf_phone', 180, 340, 2)
  assert.ok(phone.y >= 28 && phone.y + phone.h <= 340 - 20 && phone.x > 0)
  assert.deepEqual(frameImage({}), { fit: 'cover', x: 50, y: 50 })
})

test('letak gambar cover dan contain, dengan posisi', () => {
  const box = { x: 0, y: 0, w: 100, h: 100 }
  // Gambar 200 x 100: cover menskala ke tinggi 100 (lebar 200, terpotong), contain ke lebar 100 (tinggi 50).
  assert.deepEqual(fitImage(box, 200, 100, { fit: 'cover', x: 50, y: 50 }), { x: -50, y: 0, w: 200, h: 100 })
  assert.deepEqual(fitImage(box, 200, 100, { fit: 'cover', x: 0, y: 50 }), { x: 0, y: 0, w: 200, h: 100 })
  assert.deepEqual(fitImage(box, 200, 100, { fit: 'cover', x: 100, y: 50 }), {
    x: -100,
    y: 0,
    w: 200,
    h: 100,
  })
  assert.deepEqual(fitImage(box, 200, 100, { fit: 'contain', x: 50, y: 50 }), { x: 0, y: 25, w: 100, h: 50 })
  assert.deepEqual(fitImage(box, 200, 100, { fit: 'contain', x: 50, y: 0 }), { x: 0, y: 0, w: 100, h: 50 })
})
