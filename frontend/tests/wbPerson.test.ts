import { test } from 'node:test'
import assert from 'node:assert/strict'
import { gridAt, personLabel, personLines, personSize } from '../src/utils/wbPerson.ts'

const p = {
  id: 4,
  name: 'Staf Uji',
  job_title: 'Desainer',
  division: 'Marketing',
  active: false,
  avatar_url: null,
}

test('personLines: sesuai tampilan, nonaktif berupa teks', () => {
  assert.deepEqual(personLines({ user_id: 4, view: 'foto_nama', size: 'sedang', layout: 'samping' }, p), [
    'Staf Uji',
    'Nonaktif',
  ])
  assert.deepEqual(
    personLines({ user_id: 4, view: 'lengkap', size: 'sedang', layout: 'samping' }, { ...p, active: true }),
    ['Staf Uji', 'Desainer', 'Marketing'],
  )
  assert.equal(
    personLabel({ user_id: 4, view: 'lengkap', size: 'kecil', layout: 'atas' }, p),
    'Orang: Staf Uji, Desainer, Marketing, nonaktif',
  )
})

test('gridAt: lima kartu tiga kolom, berpusat di titik tujuan', () => {
  const g = gridAt(5, { w: 100, h: 50 }, { x: 0, y: 0 }, 10)
  assert.deepEqual(g[0], { x: -160, y: -55 })
  assert.deepEqual(g[3], { x: -160, y: 5 })
  assert.ok(
    personSize({ view: 'lengkap', size: 'besar', layout: 'atas' }).h >
      personSize({ view: 'foto_nama', size: 'besar', layout: 'atas' }).h,
  )
})
