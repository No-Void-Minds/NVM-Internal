import type { LocationQuery } from 'vue-router'
import type { BoardQuery, DueFilter } from '../api/board'
import { urgencies, type Urgency } from './task'

export const dueOptions: { value: DueFilter; label: string }[] = [
  { value: 'overdue', label: 'Terlambat' },
  { value: 'today', label: 'Hari ini' },
  { value: 'week', label: '7 hari ke depan' },
  { value: 'none', label: 'Tanpa deadline' },
]

export function one(value: LocationQuery[string]): string | undefined {
  const v = Array.isArray(value) ? value[0] : value
  return v ?? undefined
}

export function positiveInt(value: string | undefined): number | undefined {
  const n = Number(value)
  return Number.isInteger(n) && n > 0 ? n : undefined
}

// Sama dengan backend (MinKeywordLen). Kata kunci yang lebih pendek dibuang dari URL, bukan dikirim lalu ditolak.
export const minKeyword = 3

export function cleanKeyword(value: string | undefined): string | undefined {
  const text = value?.trim()
  return text && [...text].length >= minKeyword ? text : undefined
}

// staffUsername diisi hanya untuk akun Staf. managerScope: Manajer juga punya saklar Task saya (revisi 47), tetapi
// bawaannya Semua task, jadi scope=mine miliknya ditulis ke URL. Admin tidak punya saklar lingkup.
// Aturan S4-06 dikumpulkan di sini supaya tombol, select, dan URL yang diketik tangan berujung pada keadaan yang sama.
export function normalizeBoardQuery<T extends Pick<BoardQuery, 'scope' | 'pic'>>(
  b: T,
  staffUsername: string | null,
  managerScope = false,
): T {
  // Filter Orang tidak diubah (revisi 45): pic nama sendiri di akun Manajer tetap filter Orang, bukan Task saya.
  if (!staffUsername)
    return managerScope && b.scope === 'mine' && !b.pic ? { ...b, scope: 'mine' } : { ...b, scope: undefined }
  const pic = b.pic?.toLowerCase()
  if (pic === staffUsername.toLowerCase()) return { ...b, scope: 'mine', pic: undefined }
  if (pic) return { ...b, scope: 'all' }
  return { ...b, scope: b.scope === 'all' ? 'all' : 'mine' }
}

// Nilai yang tidak dikenal dibuang, supaya URL lama atau salah ketik tidak membuat papan galat.
export function parseBoardQuery(
  q: LocationQuery,
  staffUsername: string | null,
  managerScope = false,
): BoardQuery {
  const urgency = one(q.urgency)
  const due = one(q.due)
  const raw: BoardQuery = {
    q: cleanKeyword(one(q.q)),
    scope: parseScope(one(q.scope)),
    pic: one(q.pic)?.trim() || undefined,
    division: positiveInt(one(q.division)),
    label: positiveInt(one(q.label)),
    urgency: urgencies.includes(urgency as Urgency) ? (urgency as Urgency) : undefined,
    due: dueOptions.some((d) => d.value === due) ? (due as DueFilter) : undefined,
  }
  return normalizeBoardQuery(raw, staffUsername, managerScope)
}

export function parseScope(v: string | undefined): 'mine' | 'all' | undefined {
  return v === 'all' || v === 'mine' ? v : undefined
}

// Nilai bawaan tidak ditulis ke URL, supaya alamat papan tanpa filter tetap /task. scope=mine hanya ditulis untuk
// Manajer (managerScope), karena bagi Staf itu bawaan.
export function toRouteQuery(b: BoardQuery, managerScope = false): Record<string, string> {
  const out: Record<string, string> = {}
  if (b.q) out.q = b.q
  if (b.scope === 'all' || (managerScope && b.scope === 'mine')) out.scope = b.scope
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
  return Boolean(b.q || b.pic || b.division || b.label || b.urgency || b.due)
}
