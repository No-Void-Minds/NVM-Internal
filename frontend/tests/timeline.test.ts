// Test unit kalender Timeline (revisi 30).
//   cd frontend && npm run test:unit
import { test } from 'node:test'
import assert from 'node:assert/strict'

import { monthGrid, shiftMonth, weekdayIndex } from '../src/utils/timeline.ts'

test('grid mulai Senin dan tanggal 1 jatuh di hari yang benar', () => {
  // 1 November 2026 hari Minggu, jadi grid mulai Senin 26 Oktober.
  const nov = monthGrid('2026-11')
  assert.equal(nov[0], '2026-10-26')
  assert.equal(nov.indexOf('2026-11-01') % 7, 6)
  assert.equal(nov.length % 7, 0)
  assert.equal(nov.at(-1), '2026-12-06')
  // Februari 2027 mulai Senin dan pas empat minggu.
  assert.deepEqual([monthGrid('2027-02')[0], monthGrid('2027-02').length], ['2027-02-01', 28])
  assert.equal(weekdayIndex('2026-11-23'), 0)
})

test('geser bulan melewati batas tahun', () => {
  assert.equal(shiftMonth('2026-12', 1), '2027-01')
  assert.equal(shiftMonth('2026-01', -1), '2025-12')
})
