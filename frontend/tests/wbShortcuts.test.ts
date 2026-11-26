import { test } from 'node:test'
import assert from 'node:assert/strict'
import { SHORTCUTS, find, label, matches } from '../src/utils/wbShortcuts.ts'

const ev = (code: string, m: { ctrl?: boolean; alt?: boolean; shift?: boolean } = {}) => ({
  code,
  ctrlKey: !!m.ctrl,
  metaKey: false,
  altKey: !!m.alt,
  shiftKey: !!m.shift,
})

test('label dan matches: kombinasi persis, tombol fisik', () => {
  assert.equal(label('copyStyle'), 'Ctrl+Alt+C')
  assert.equal(label('front'), 'Ctrl+Shift+]')
  assert.equal(label('zoomSel'), 'Shift+2')
  assert.ok(matches(ev('KeyG', { ctrl: true }), 'group'))
  assert.ok(!matches(ev('KeyG', { ctrl: true, shift: true }), 'group'))
  assert.equal(find(ev('KeyG', { ctrl: true, shift: true }), ['group', 'ungroup']), 'ungroup')
})

test('setiap pintasan unik', () => {
  const seen = new Map<string, string>()
  for (const [id, c] of Object.entries(SHORTCUTS)) {
    const k = JSON.stringify([
      !!('ctrl' in c && c.ctrl),
      !!('alt' in c && c.alt),
      !!('shift' in c && c.shift),
      c.code,
    ])
    assert.ok(!seen.has(k), `${id} bentrok dengan ${seen.get(k)}`)
    seen.set(k, id)
  }
})
