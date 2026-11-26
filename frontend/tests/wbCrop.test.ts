import { test } from 'node:test'
import assert from 'node:assert/strict'
import { clampCrop, frameCrop, frameOf } from '../src/utils/wbCrop.ts'

test('frameCrop: zoom 1 menutupi bingkai, posisi menggeser, frameOf membalikkannya', () => {
  // Gambar 400x200 di bingkai persegi: lebar terpotong separuh, di tengah.
  assert.deepEqual(frameCrop(400, 200, 1, 1, 0.5, 0.5), { x: 0.25, y: 0, w: 0.5, h: 1 })
  const c = frameCrop(400, 200, 1, 2, 1, 0)
  assert.deepEqual(c, { x: 0.75, y: 0, w: 0.25, h: 0.5 })
  const f = frameOf(400, 200, 1, c)
  assert.equal(f.zoom, 2)
  assert.equal(f.px, 1)
  assert.equal(f.py, 0)
})

test('clampCrop menjaga potongan di dalam gambar dan minimal 2%', () => {
  assert.deepEqual(clampCrop({ x: 0.9, y: -1, w: 0.5, h: 0.001 }), { x: 0.5, y: 0, w: 0.5, h: 0.02 })
})
