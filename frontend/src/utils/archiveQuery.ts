import type { LocationQuery } from 'vue-router'
import type { ArchiveQuery, ArchiveSort } from '../api/archive'
import { cleanKeyword, normalizeBoardQuery, one, positiveInt } from './boardQuery'
import { urgencies, type Urgency } from './task'

export const sortOptions: { value: ArchiveSort; label: string }[] = [
  { value: 'newest', label: 'Terbaru selesai' },
  { value: 'oldest', label: 'Terlama selesai' },
  { value: 'urgency', label: 'Urgency tertinggi' },
]

const dayPattern = /^\d{4}-\d{2}-\d{2}$/

// Nilai yang tidak dikenal dibuang, supaya URL lama atau salah ketik tidak membuat Arsip galat.
export function parseArchiveQuery(q: LocationQuery, staffUsername: string | null): ArchiveQuery {
  const urgency = one(q.urgency)
  const sort = one(q.sort)
  const from = one(q.from)
  const to = one(q.to)
  const page = positiveInt(one(q.page))
  const raw: ArchiveQuery = {
    q: cleanKeyword(one(q.q)),
    scope: one(q.scope) === 'all' ? 'all' : undefined,
    pic: one(q.pic)?.trim() || undefined,
    division: positiveInt(one(q.division)),
    label: positiveInt(one(q.label)),
    urgency: urgencies.includes(urgency as Urgency) ? (urgency as Urgency) : undefined,
    from: from && dayPattern.test(from) ? from : undefined,
    to: to && dayPattern.test(to) ? to : undefined,
    sort: sort === 'oldest' || sort === 'urgency' ? sort : undefined,
    page: page && page > 1 ? page : undefined,
  }
  return normalizeBoardQuery(raw, staffUsername)
}

// Nilai bawaan (urutan terbaru, halaman 1, lingkup bawaan) tidak ditulis, supaya alamat tanpa filter tetap /arsip.
export function toArchiveRouteQuery(a: ArchiveQuery): Record<string, string> {
  const out: Record<string, string> = {}
  if (a.q) out.q = a.q
  if (a.scope === 'all') out.scope = 'all'
  if (a.pic) out.pic = a.pic
  if (a.division) out.division = String(a.division)
  if (a.label) out.label = String(a.label)
  if (a.urgency) out.urgency = a.urgency
  if (a.from) out.from = a.from
  if (a.to) out.to = a.to
  if (a.sort && a.sort !== 'newest') out.sort = a.sort
  if (a.page && a.page > 1) out.page = String(a.page)
  return out
}

export function hasArchiveFilters(a: ArchiveQuery): boolean {
  return Boolean(a.q || a.pic || a.division || a.label || a.urgency || a.from || a.to)
}
