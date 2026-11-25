// Test unit ekspor Whiteboard (W4): batas skala, struktur berkas PDF, nama berkas, dan pemecah baris.
//   cd frontend && npm run test:unit
import { test } from 'node:test'
import assert from 'node:assert/strict'

import {
  EXPORT_SCALE,
  MAX_EXPORT_PIXELS,
  MAX_EXPORT_SIDE,
  exportName,
  exportScale,
  pdfFromJpeg,
  wrapText,
} from '../src/utils/wbExport.ts'

test('skala ekspor 2x untuk papan kecil dan diperkecil untuk papan besar', () => {
  assert.equal(exportScale(1000, 800), EXPORT_SCALE)
  const s = exportScale(20000, 12000)
  assert.ok(s < 1)
  assert.ok(20000 * s * 12000 * s <= MAX_EXPORT_PIXELS + 1)
  // Papan sangat panjang: sisi terpanjang yang membatasi.
  const long = exportScale(100000, 200)
  assert.ok(100000 * long <= MAX_EXPORT_SIDE + 1e-6)
  assert.equal(exportScale(0, 0), EXPORT_SCALE)
})

test('PDF satu halaman: header, offset xref tepat ke tiap objek, startxref, dan JPEG utuh', () => {
  const jpeg = new Uint8Array([0xff, 0xd8, 0xff, 0xe0, 1, 2, 3, 4, 0xff, 0xd9])
  const pdf = pdfFromJpeg(jpeg, 1200, 600)
  const text = new TextDecoder('latin1').decode(pdf)
  assert.ok(text.startsWith('%PDF-1.4\n'))
  assert.ok(text.trimEnd().endsWith('%%EOF'))
  const xrefAt = Number(text.match(/startxref\n(\d+)\n/)?.[1])
  assert.equal(text.slice(xrefAt, xrefAt + 4), 'xref')
  const entries = [...text.slice(xrefAt).matchAll(/^(\d{10}) 00000 n $/gm)].map((m) => Number(m[1]))
  assert.equal(entries.length, 5)
  entries.forEach((off, i) => assert.equal(text.slice(off, off + `${i + 1} 0 obj`.length), `${i + 1} 0 obj`))
  // Byte JPEG dipakai apa adanya (DCTDecode) dengan panjang yang benar.
  assert.match(text, /\/Width 1200 \/Height 600 .*\/Filter \/DCTDecode \/Length 10 >>/)
  const start = text.indexOf('stream\n') + 'stream\n'.length
  assert.deepEqual([...pdf.slice(start, start + jpeg.length)], [...jpeg])
  // 96 dpi: 1200 x 600 piksel menjadi halaman 900 x 450 poin.
  assert.match(text, /\/MediaBox \[0 0 900\.00 450\.00\]/)
  const draw = text.match(/5 0 obj\n<< \/Length (\d+) >>\nstream\n(.*)\nendstream/)
  assert.ok(draw)
  assert.equal(Number(draw[1]), draw[2]!.length)
})

test('halaman PDF dibatasi 14.400 poin per sisi', () => {
  const text = new TextDecoder('latin1').decode(pdfFromJpeg(new Uint8Array([0xff, 0xd9]), 30000, 1000))
  const [, w, h] = text.match(/\/MediaBox \[0 0 ([\d.]+) ([\d.]+)\]/)!
  assert.ok(Number(w) <= 14400 && Number(h) > 0)
})

test('nama berkas ekspor dari judul papan', () => {
  assert.equal(exportName('Kampanye Ramadan 2026!', 'png'), 'kampanye-ramadan-2026.png')
  assert.equal(exportName('Café Résumé', 'pdf'), 'cafe-resume.pdf')
  assert.equal(exportName('***', 'pdf'), 'papan.pdf')
})

test('pemecah baris: kata utuh, baris baru dipertahankan, kata panjang dipotong', () => {
  const m = (s: string) => s.length * 10
  assert.deepEqual(wrapText('satu dua tiga', 80, m), ['satu dua', 'tiga'])
  assert.deepEqual(wrapText('a\nb', 100, m), ['a', 'b'])
  assert.deepEqual(wrapText('https://contoh.test/x', 60, m), ['https:', '//cont', 'oh.tes', 't/x'])
  assert.deepEqual(wrapText('', 100, m), [''])
  for (const line of wrapText('kata kata kata panjangsekalitanpaspasi kata', 70, m)) assert.ok(m(line) <= 70)
})
