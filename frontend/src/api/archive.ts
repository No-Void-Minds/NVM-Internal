import { apiGet } from './client'
import type { BoardTask } from './board'
import type { Urgency } from '../utils/task'

export type ArchiveSort = 'newest' | 'oldest' | 'urgency'

export type ArchiveQuery = {
  q?: string
  scope?: 'mine' | 'all'
  pic?: string
  division?: number
  label?: number
  urgency?: Urgency
  // Tanggal selesai (YYYY-MM-DD), keduanya inklusif.
  from?: string
  to?: string
  sort?: ArchiveSort
  page?: number
}

export type ArchiveResult = { tasks: BoardTask[]; total: number; page: number; page_size: number }

export function getArchive(q: ArchiveQuery, signal?: AbortSignal): Promise<ArchiveResult> {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(q)) {
    if (value !== undefined && value !== '') params.set(key, String(value))
  }
  const qs = params.toString()
  return apiGet<ArchiveResult>(qs ? `/archive?${qs}` : '/archive', signal)
}
