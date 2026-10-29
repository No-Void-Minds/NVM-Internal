import type { LocationQuery } from 'vue-router'
import type { BoardQuery, DueFilter } from '../api/board'
import { urgencies, type Urgency } from './task'

export const dueOptions: { value: DueFilter; label: string }[] = [
  { value: 'overdue', label: 'Terlambat' },
  { value: 'week', label: '7 hari ke depan' },
  { value: 'none', label: 'Tanpa tenggat' },
]

function one(value: LocationQuery[string]): string | undefined {
  const v = Array.isArray(value) ? value[0] : value
  return v ?? undefined
}

function positiveInt(value: string | undefined): number | undefined {
  const n = Number(value)
  return Number.isInteger(n) && n > 0 ? n : undefined
}

// staffUsername diisi hanya untuk akun Staf; Manajer dan Admin tidak punya saklar lingkup.
// Aturan S4-06 dikumpulkan di sini supaya tombol, select, dan URL yang diketik tangan berujung pada keadaan yang sama.
export function normalizeBoardQuery(b: BoardQuery, staffUsername: string | null): BoardQuery {
  if (!staffUsername) return { ...b, scope: undefined }
  const pic = b.pic?.toLowerCase()
  if (pic === staffUsername.toLowerCase()) return { ...b, scope: 'mine', pic: undefined }
  if (pic) return { ...b, scope: 'all' }
  return { ...b, scope: b.scope === 'all' ? 'all' : 'mine' }
}

// Nilai yang tidak dikenal dibuang, supaya URL lama atau salah ketik tidak membuat papan galat.
export function parseBoardQuery(q: LocationQuery, staffUsername: string | null): BoardQuery {
  const urgency = one(q.urgency)
  const due = one(q.due)
  const raw: BoardQuery = {
    scope: one(q.scope) === 'all' ? 'all' : undefined,
    pic: one(q.pic)?.trim() || undefined,
    division: positiveInt(one(q.division)),
    label: positiveInt(one(q.label)),
    urgency: urgencies.includes(urgency as Urgency) ? (urgency as Urgency) : undefined,
    due: dueOptions.some((d) => d.value === due) ? (due as DueFilter) : undefined,
  }
  return normalizeBoardQuery(raw, staffUsername)
}

// Nilai bawaan tidak ditulis ke URL, supaya alamat papan tanpa filter tetap /task.
export function toRouteQuery(b: BoardQuery): Record<string, string> {
  const out: Record<string, string> = {}
  if (b.scope === 'all') out.scope = 'all'
  if (b.pic) out.pic = b.pic
  if (b.division) out.division = String(b.division)
  if (b.label) out.label = String(b.label)
  if (b.urgency) out.urgency = b.urgency
  if (b.due) out.due = b.due
  return out
}

export function sameRouteQuery(a: Record<string, string>, b: LocationQuery): boolean {
  const keys = new Set([...Object.keys(a), ...Object.keys(b)])
  return [...keys].every((k) => a[k] === b[k])
}

export function hasFilters(b: BoardQuery): boolean {
  return Boolean(b.pic || b.division || b.label || b.urgency || b.due)
}
