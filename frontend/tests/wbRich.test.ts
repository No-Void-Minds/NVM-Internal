import { test } from 'node:test'
import assert from 'node:assert/strict'
import { cardLines, type Card, type PMNode } from '../src/utils/whiteboard.ts'
import {
  canonDoc,
  colorClass,
  docText,
  highlightClass,
  inlineRuns,
  richValue,
  textDoc,
  wrapRuns,
} from '../src/utils/wbRich.ts'

const para = (content: PMNode[], attrs?: Record<string, unknown>): PMNode => ({
  type: 'paragraph',
  ...(attrs ? { attrs } : {}),
  content,
})

test('inlineRuns: mark format menjadi gaya run, nilai di luar daftar diabaikan', () => {
  const runs = inlineRuns(
    para([
      {
        type: 'text',
        text: 'A',
        marks: [
          { type: 'bold' },
          { type: 'textStyle', attrs: { color: 'merah', fontSize: 'besar' } },
          { type: 'highlight', attrs: { color: 'kuning' } },
        ],
      },
      { type: 'hardBreak' },
      {
        type: 'text',
        text: 'B',
        marks: [{ type: 'textStyle', attrs: { color: '#ff0000', fontSize: '72px' } }],
      },
    ]),
  )
  assert.deepEqual(runs, [
    { text: 'A', bold: true, color: '--color-danger', scale: 1.25, highlight: '--color-ready-bg' },
    { text: '\n' },
    { text: 'B' },
  ])
})

test('kelas format hanya dari daftar tetap', () => {
  assert.equal(colorClass('ungu'), 'wb-c-ungu')
  assert.equal(colorClass('collab-3'), '')
  assert.equal(highlightClass('krem'), '')
})

test('wrapRuns: pecah per kata lintas run, gaya tetap per potongan, baris baru dihormati', () => {
  const m = (_r: unknown, s: string) => s.length * 10
  const lines = wrapRuns([{ text: 'satu ' }, { text: 'dua tiga', bold: true }, { text: '\nempat' }], 90, m)
  assert.deepEqual(lines, [
    [{ text: 'satu ' }, { text: 'dua', bold: true }],
    [{ text: 'tiga', bold: true }],
    [{ text: 'empat' }],
  ])
  for (const l of wrapRuns([{ text: 'panjangsekalitanpaspasi' }], 50, m))
    assert.ok(l.reduce((n, r) => n + m(r, r.text), 0) <= 50)
})

test('richValue: dokumen tanpa format jadi teks saja, atribut null dari editor dibuang', () => {
  const plain = { type: 'doc', content: [para([{ type: 'text', text: 'Biasa' }], { textAlign: null })] }
  assert.deepEqual(richValue(plain), { text: 'Biasa', doc: null })
  assert.equal(JSON.stringify(canonDoc(plain)), JSON.stringify(textDoc('Biasa')))
  const rich = { type: 'doc', content: [para([{ type: 'text', text: 'Tengah' }], { textAlign: 'center' })] }
  assert.deepEqual(richValue(rich).doc, {
    type: 'doc',
    content: [para([{ type: 'text', text: 'Tengah' }], { textAlign: 'center' })],
  })
  assert.equal(docText(textDoc('a\nb')), 'a\nb')
})

test('cardLines: butir checklist berformat membawa run dan penanda centang', () => {
  const lines = cardLines({
    id: 'c',
    kind: 'checklist',
    content: {
      title: 'Rilis',
      items: [
        {
          id: '1',
          text: 'Foto',
          done: false,
          doc: {
            type: 'doc',
            content: [
              para([{ type: 'text', text: 'Foto', marks: [{ type: 'bold' }] }], { textAlign: 'center' }),
            ],
          },
        },
      ],
    },
  } as unknown as Card)
  assert.equal(lines[1]?.text, '☐ Foto')
  assert.deepEqual(lines[1]?.runs, [{ text: '☐ ' }, { text: 'Foto', bold: true }])
  assert.equal(lines[1]?.align, 'center')
})
