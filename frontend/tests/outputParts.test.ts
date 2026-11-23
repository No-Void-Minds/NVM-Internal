import { test } from 'node:test'
import assert from 'node:assert/strict'
import { outputParts, type OutputPart } from '../src/utils/outputLink.ts'

// Revisi 35: satu parser untuk semua tampilan Output Aktivitas.
const links = (parts: OutputPart[]) =>
  parts.flatMap((p) => (p.type === 'link' ? [p.label + ' ' + p.href] : []))
const texts = (parts: OutputPart[]) => parts.flatMap((p) => (p.type === 'text' ? [p.text] : []))

test('tanpa URL: satu potongan teks utuh', () => {
  assert.deepEqual(outputParts('Naskah final di folder tim'), [
    { type: 'text', text: 'Naskah final di folder tim' },
  ])
  assert.deepEqual(outputParts(''), [])
  assert.deepEqual(outputParts('javascript:alert(1)'), [{ type: 'text', text: 'javascript:alert(1)' }])
})

test('satu URL Drive jadi File Drive', () => {
  const p = outputParts('https://drive.google.com/file/d/1AbC/view')
  assert.equal(p.length, 1)
  assert.equal(p[0].type, 'link')
  assert.equal(p[0].type === 'link' && p[0].label, 'File Drive')
  assert.equal(p[0].type === 'link' && p[0].driveFileId, '1AbC')
})

test('dua dan tiga URL: label sama diberi nomor, dipisah baris baru, spasi, atau koma', () => {
  assert.deepEqual(
    links(outputParts('https://drive.google.com/file/d/a/view\nhttps://drive.google.com/file/d/b/view')),
    [
      'File Drive https://drive.google.com/file/d/a/view',
      'File Drive 2 https://drive.google.com/file/d/b/view',
    ],
  )
  const tiga = outputParts(
    'https://drive.google.com/file/d/a/view, https://www.canva.com/design/x https://drive.google.com/file/d/c/view',
  )
  assert.deepEqual(links(tiga), [
    'File Drive https://drive.google.com/file/d/a/view',
    'www.canva.com https://www.canva.com/design/x',
    'File Drive 2 https://drive.google.com/file/d/c/view',
  ])
  assert.deepEqual(texts(tiga), [], 'koma dan spasi pemisah tidak tampil sebagai teks')
})

test('tanda baca di ujung URL dibuang: titik, koma, kurung tutup', () => {
  assert.deepEqual(links(outputParts('Lihat https://a.test/x.')), ['a.test https://a.test/x'])
  assert.deepEqual(links(outputParts('https://a.test/x, lalu')), ['a.test https://a.test/x'])
  const kurung = outputParts('(hasil: https://a.test/y)')
  assert.deepEqual(links(kurung), ['a.test https://a.test/y'])
  assert.deepEqual(texts(kurung), ['(hasil: ', ')'])
  assert.deepEqual(links(outputParts('https://id.wikipedia.org/wiki/Foo_(bar)')), [
    'id.wikipedia.org https://id.wikipedia.org/wiki/Foo_(bar)',
  ])
})

test('teks campur URL tetap di urutan aslinya', () => {
  const p = outputParts(
    'Draf: https://docs.google.com/document/d/1/edit\nFinal di https://drive.google.com/file/d/2/view ya',
  )
  assert.deepEqual(
    p.map((x) => (x.type === 'text' ? `T:${x.text}` : `L:${x.label}`)),
    ['T:Draf: ', 'L:Google Docs', 'T:\nFinal di ', 'L:File Drive', 'T: ya'],
  )
})

test('URL di baris berbeda tanpa spasi pemisah', () => {
  assert.deepEqual(links(outputParts('https://a.test/1\nhttps://a.test/2\r\nhttps://b.test/3')), [
    'a.test https://a.test/1',
    'a.test 2 https://a.test/2',
    'b.test https://b.test/3',
  ])
})
