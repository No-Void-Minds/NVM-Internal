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

// Nilai yang tidak dikenal dibuang, supaya URL lama atau salah ketik tidak membuat papan galat.
export function parseBoardQuery(q: LocationQuery, isStaff: boolean): BoardQuery {
  const urgency = one(q.urgency)
  const due = one(q.due)
  return {
    // Hanya Staf yang punya saklar; tanpa scope di URL, Staf mulai dari "Task saya" (S4-05).
    scope: isStaff ? (one(q.scope) === 'all' ? 'all' : 'mine') : undefined,
    pic: one(q.pic)?.trim() || undefined,
    division: positiveInt(one(q.division)),
    label: positiveInt(one(q.label)),
    urgency: urgencies.includes(urgency as Urgency) ? (urgency as Urgency) : undefined,
    due: dueOptions.some((d) => d.value === due) ? (due as DueFilter) : undefined,
  }
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

export function hasFilters(b: BoardQuery): boolean {
  return Boolean(b.pic || b.division || b.label || b.urgency || b.due)
}
