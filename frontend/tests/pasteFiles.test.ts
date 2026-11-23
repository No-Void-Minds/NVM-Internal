import { test } from 'node:test'
import assert from 'node:assert/strict'
import { pastedFiles, pasteStamp } from '../src/utils/pasteFiles.ts'

// Revisi 36: tempelan Ctrl+V di bagian lampiran task dan sub task.
const at = new Date(2026, 10, 23, 9, 5, 7)
const clip = (...files: File[]) => ({ files })

test('cap waktu YYYYMMDD-HHMMSS', () => {
  assert.equal(pasteStamp(at), '20261123-090507')
})

test('tempelan teks: tidak ada berkas, jadi browser menempel teks seperti biasa', () => {
  assert.deepEqual(pastedFiles(clip(), at), [])
  assert.deepEqual(pastedFiles(null, at), [])
})

test('tangkapan layar diberi nama tempel-<waktu> dengan ekstensi sesuai jenisnya', () => {
  const [png] = pastedFiles(clip(new File(['x'], 'image.png', { type: 'image/png' })), at)
  assert.equal(png.name, 'tempel-20261123-090507.png')
  assert.equal(png.type, 'image/png')
  const [jpg] = pastedFiles(clip(new File(['x'], '', { type: 'image/jpeg' })), at)
  assert.equal(jpg.name, 'tempel-20261123-090507.jpg')
})

test('beberapa gambar sekaligus diberi nomor, berkas bernama tetap memakai namanya', () => {
  const out = pastedFiles(
    clip(
      new File(['a'], 'image.png', { type: 'image/png' }),
      new File(['b'], 'laporan.pdf', { type: 'application/pdf' }),
      new File(['c'], 'image.png', { type: 'image/png' }),
      new File(['d'], 'foto-produk.png', { type: 'image/png' }),
    ),
    at,
  )
  assert.deepEqual(
    out.map((f) => f.name),
    ['tempel-20261123-090507.png', 'laporan.pdf', 'tempel-20261123-090507-2.png', 'foto-produk.png'],
  )
})
