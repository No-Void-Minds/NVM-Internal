import { apiGet, apiPut } from './client'

// Aktivitas Harian (revisi 21). Daftar slot jam datang dari backend, frontend tidak menulisnya sendiri.
export type ActivityCell = {
  slot: number
  time: string
  activity: string
  output: string
  // Nama penyimpan terakhir, hanya terisi bila bukan pemilik isian (Manajer yang mengubah isian orang lain).
  updated_by: string
}

export type ActivityPerson = {
  id: number
  name: string
  division_name: string
  avatar_id: string | null
  is_active: boolean
  filled: boolean
  cells: ActivityCell[]
}

export type MyActivities = { date: string; slots: string[]; cells: ActivityCell[] }

export type ActivityDay = {
  date: string
  today: string
  slots: string[]
  people: ActivityPerson[]
  can_edit: boolean
}

export type SaveActivityInput = {
  user_id?: number
  date: string
  slot: number
  activity: string
  output: string
}

export function getMyActivities(signal?: AbortSignal): Promise<MyActivities> {
  return apiGet<MyActivities>('/activities/me', signal)
}

export function getActivityDay(date: string, signal?: AbortSignal): Promise<ActivityDay> {
  return apiGet<ActivityDay>(`/activities?date=${encodeURIComponent(date)}`, signal)
}

export function saveActivity(input: SaveActivityInput): Promise<MyActivities> {
  return apiPut<MyActivities>('/activities', input)
}

// Unduhan memakai tautan biasa, bukan fetch, supaya nama berkas dari Content-Disposition yang dipakai.
export function activityExportUrl(from: string, to: string): string {
  return `/api/activities/export?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`
}
