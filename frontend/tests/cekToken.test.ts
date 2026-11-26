// Test unit cek token CSS (scripts/cek-token.mjs), termasuk kasus bug warna kolaborasi: token terang hilang dari
// build sementara token gelapnya ada.
//   cd frontend && npm run test:unit
import { test } from 'node:test'
import assert from 'node:assert/strict'

// @ts-expect-error modul .mjs tanpa deklarasi tipe
import { builtTokens, compare, references, sourceTokens } from '../scripts/cek-token.mjs'

const source = `
@theme {
  /* --color-komentar: bukan definisi */
  --color-ink: #1a1c20;
  --color-collab-1: #b42318;
}
:root {
  --color-on-collab-1: #ffffff;
}
:root[data-theme='gelap'] {
  --color-ink: #e8ebef;
  --color-collab-1: #ff8a7a;
}
.wb-card-krem > * {
  --color-lokal: #fff;
}`

test('definisi sumber per tema; timpaan lokal bukan sumber token', () => {
  const s = sourceTokens(source)
  assert.deepEqual([...s.get('--color-ink')].sort(), ['gelap', 'terang'])
  assert.deepEqual([...s.get('--color-on-collab-1')], ['terang'])
  assert.equal(s.has('--color-lokal'), false)
  assert.equal(s.has('--color-komentar'), false)
})

test('kasus bug kolaborasi: nilai terang hilang dari build dilaporkan, gelap yang ada tidak', () => {
  // Build ala Tailwind v4: --color-collab-1 terang tidak dikeluarkan karena hanya dirujuk dinamis.
  const built = builtTokens(
    '@layer theme{:root,:host{--color-ink:#1a1c20;--color-on-collab-1:#fff}}:root[data-theme=gelap]{--color-ink:#e8ebef;--color-collab-1:#ff8a7a}',
  )
  const missing = compare(sourceTokens(source), built, new Set(['--color-ink', '--color-collab-1', '--color-lain']))
  assert.deepEqual(missing, ['--color-collab-1 (terang)'])
})

test('rujukan: var(), string, getPropertyValue; definisi tidak dihitung; nama dinamis dilaporkan', () => {
  const { refs, dynamic } = references(`
const a = 'var(--color-surface)'
ctx.fillStyle = tok('--color-ink')
const s = { '--wb-ring': '2px' }
.x { --color-def: red; color: var(--color-tag-ink); }
const bad = \`var(--color-collab-\${n})\`
`)
  for (const r of ['--color-surface', '--color-ink', '--wb-ring', '--color-tag-ink']) assert.ok(refs.has(r), r)
  assert.equal(refs.has('--color-def'), false)
  assert.equal(dynamic.length, 1)
  assert.equal(dynamic[0].line, 6)
})
