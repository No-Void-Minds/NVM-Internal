// Test unit hitungan tanggal halaman Aktivitas (revisi 21), tanpa dependensi baru: pemeriksa test bawaan
// Node (node:test) dan Node yang menjalankan TypeScript langsung.
//   cd frontend && npm run test:unit
import { test } from 'node:test'
import assert from 'node:assert/strict'

import { monthRange, rangeLabel, shiftDay, weekDays, weekStart } from '../src/utils/activityWeek.ts'

test('weekStart selalu Senin minggu tanggal itu', () => {
  for (const d of ['2026-11-16', '2026-11-18', '2026-11-20', '2026-11-21', '2026-11-22']) {
    assert.equal(weekStart(d), '2026-11-16', d)
  }
})

test('weekDays Senin sampai Jumat, termasuk pergantian bulan dan tahun', () => {
  assert.deepEqual(weekDays('2026-11-18'), [
    '2026-11-16',
    '2026-11-17',
    '2026-11-18',
    '2026-11-19',
    '2026-11-20',
  ])
  // Senin 30 November 2026, Jumatnya sudah Desember.
  assert.deepEqual(weekDays('2026-12-02'), [
    '2026-11-30',
    '2026-12-01',
    '2026-12-02',
    '2026-12-03',
    '2026-12-04',
  ])
  // Senin 28 Desember 2026, Jumatnya 1 Januari 2027.
  assert.deepEqual(weekDays('2026-12-31'), [
    '2026-12-28',
    '2026-12-29',
    '2026-12-30',
    '2026-12-31',
    '2027-01-01',
  ])
  // Tahun kabisat: 29 Februari 2028 hari Selasa.
  assert.deepEqual(weekDays('2028-03-01'), [
    '2028-02-28',
    '2028-02-29',
    '2028-03-01',
    '2028-03-02',
    '2028-03-03',
  ])
})

test('shiftDay melewati batas bulan dan tahun', () => {
  assert.equal(shiftDay('2026-11-30', 1), '2026-12-01')
  assert.equal(shiftDay('2027-01-01', -1), '2026-12-31')
  assert.equal(shiftDay('2026-11-16', -7), '2026-11-09')
  assert.equal(shiftDay('2028-02-28', 1), '2028-02-29')
})

test('rangeLabel menyebut bulan dan tahun hanya bila perlu', () => {
  assert.equal(rangeLabel('2026-11-16', '2026-11-20'), '16 sampai 20 November 2026')
  assert.equal(rangeLabel('2026-11-30', '2026-12-04'), '30 November sampai 4 Desember 2026')
  assert.equal(rangeLabel('2026-12-28', '2027-01-01'), '28 Desember 2026 sampai 1 Januari 2027')
  assert.equal(rangeLabel('2026-11-18', '2026-11-18'), 'Rabu, 18 November 2026')
})

test('monthRange sampai hari terakhir bulan', () => {
  assert.deepEqual(monthRange('2026-11-18'), { from: '2026-11-01', to: '2026-11-30' })
  assert.deepEqual(monthRange('2028-02-10'), { from: '2028-02-01', to: '2028-02-29' })
  assert.deepEqual(monthRange('2026-12-31'), { from: '2026-12-01', to: '2026-12-31' })
})
