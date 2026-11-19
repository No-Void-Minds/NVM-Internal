import { apiGet, apiPost, apiPut, request } from './client'
import type { AbsenceType, TimelineCategory, TimelineEvent } from '../utils/timeline'

// Timeline (revisi 30). can_write dari server: hanya Manajer. Tipe entri ada di utils/timeline.ts supaya test
// unit kalender tidak ikut memuat klien API.
export type { AbsenceType, TimelineCategory, TimelineEvent }
export type TimelineEventInput = {
  category: TimelineCategory
  title: string
  description: string
  start_date: string
  end_date: string
  user_id: number
  absence_type: AbsenceType | ''
}

export type TimelineNote = { id: number; title: string; body: string }

export function listTimelineEvents(month: string, signal?: AbortSignal) {
  return apiGet<{ events: TimelineEvent[]; can_write: boolean }>(`/timeline/events?month=${month}`, signal)
}

export function createTimelineEvent(input: TimelineEventInput) {
  return apiPost<{ id: number }>('/timeline/events', input)
}

export function updateTimelineEvent(id: number, input: TimelineEventInput) {
  return apiPut<void>(`/timeline/events/${id}`, input)
}

export function deleteTimelineEvent(id: number) {
  return request<void>('DELETE', `/timeline/events/${id}`)
}

export function listTimelineNotes(signal?: AbortSignal) {
  return apiGet<{ notes: TimelineNote[]; can_write: boolean }>('/timeline/notes', signal)
}

export function createTimelineNote(title: string, body: string) {
  return apiPost<{ note: TimelineNote }>('/timeline/notes', { title, body })
}

export function updateTimelineNote(id: number, title: string, body: string) {
  return apiPut<void>(`/timeline/notes/${id}`, { title, body })
}

export function deleteTimelineNote(id: number) {
  return request<void>('DELETE', `/timeline/notes/${id}`)
}

export function reorderTimelineNotes(ids: number[]) {
  return apiPut<void>('/timeline/notes/order', { ids })
}
