import { test } from 'node:test'
import assert from 'node:assert/strict'
import { arrange, expand, group, hasGroup, remap, tidy, ungroup, unitOf } from '../src/utils/wbGroups.ts'

const c = (id: string, x: number, y: number, g: string[] = []) => ({ id, x, y, w: 100, h: 50, group_ids: g })

test('unitOf: klik memilih grup terluar; di dalam grup memilih satu tingkat di bawahnya', () => {
  const cards = [
    c('a', 0, 0, ['A', 'B']),
    c('b', 200, 0, ['A', 'B']),
    c('s', 0, 900, ['A']),
    c('x', 500, 500),
  ]
  assert.deepEqual(unitOf(cards[0]!, cards), ['a', 'b', 's'])
  assert.deepEqual(unitOf(cards[0]!, cards, ['A']), ['a', 'b'])
  assert.deepEqual(unitOf(cards[0]!, cards, ['A', 'B']), ['a'])
  assert.deepEqual(unitOf(cards[3]!, cards, ['A']), ['x'])
  assert.deepEqual(expand(['x', 'a'], cards), ['x', 'a', 'b', 's'])
})

test('group menyisipkan grup baru di bawah jalur bersama; batas 8 tingkat; ungroup melepas satu tingkat', () => {
  const cards = [c('a', 0, 0, ['A']), c('b', 200, 0, ['A']), c('x', 500, 500)]
  assert.deepEqual(
    [...group(['a', 'b'], cards, 'N')!],
    [
      ['a', ['A', 'N']],
      ['b', ['A', 'N']],
    ],
  )
  assert.deepEqual(group(['a', 'x'], cards, 'N')!.get('a'), ['N', 'A'])
  const deep = [c('d', 0, 0, ['1', '2', '3', '4', '5', '6', '7', '8']), c('e', 0, 0)]
  assert.equal(group(['d', 'e'], deep, 'N'), null)
  assert.deepEqual(
    [...ungroup(['a', 'b'], cards)],
    [
      ['a', []],
      ['b', []],
    ],
  )
  assert.equal(hasGroup(['x'], cards), false)
  assert.equal(hasGroup(['a'], cards), true)
})

test('remap: salinan mendapat id grup baru yang konsisten', () => {
  let n = 0
  const out = remap([['A', 'B'], ['A'], []], () => `g${++n}`)
  assert.deepEqual(out, [['g1', 'g2'], ['g1'], []])
})

test('arrange: grup bergerak utuh saat rata kiri dan sebar mendatar; tidy menyusun grid', () => {
  const g1 = [c('a', 100, 0), c('b', 300, 0)]
  const g2 = [c('x', 0, 200)]
  const left = arrange([g1, g2], 'left')
  assert.deepEqual(left.get('a'), { x: 0, y: 0 })
  assert.deepEqual(left.get('b'), { x: 200, y: 0 })
  assert.deepEqual(left.get('x'), { x: 0, y: 200 })
  const spread = arrange([[c('p', 0, 0)], [c('q', 150, 0)], [c('r', 600, 0)]], 'hspread')
  assert.deepEqual(spread.get('q'), { x: 300, y: 0 })
  const grid = tidy([[c('a', 0, 0)], [c('b', 400, 0)], [c('d', 0, 400)], [c('e', 900, 900)]], 20)
  assert.deepEqual(
    [grid.get('a'), grid.get('b'), grid.get('d'), grid.get('e')],
    [
      { x: 0, y: 0 },
      { x: 120, y: 0 },
      { x: 0, y: 70 },
      { x: 120, y: 70 },
    ],
  )
})
