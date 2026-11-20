import { apiGet, apiPut } from './client'

// Aktivitas Harian (revisi 21). Daftar slot jam datang dari backend, frontend tidak menulisnya sendiri.
// Lampiran satu jam (revisi 24).
export type ActivityFile = {
  id: string
  name: string
  mime_type: string
  size_bytes: number
  // Nama pengunggah, hanya terisi bila bukan pemilik jamnya.
  uploaded_by: string
}

export type ActivityCell = {
  slot: number
  time: string
  activity: string
  output: string
  // Nama penyimpan terakhir, hanya terisi bila bukan pemilik isian (Manajer yang mengubah isian orang lain).
  updated_by: string
  files: ActivityFile[]
  // Revisi 33, rantai sambung jam. linked: jam ini menyambung ke jam sebelumnya; isinya salinan jam kepala
  // (head) dari server dan hanya dibaca. span: rentang seluruh rantai, mis. "09.00-11.00".
  linked: boolean
  head: number
  span: string
  // joinable: ada tombol rantai ke jam sebelumnya. link_hint: alasan tombol itu tidak bisa dipakai.
  joinable: boolean
  link_hint: string
}

// Satu kolom tabel Aktivitas: seluruh sel orang itu di rentang yang diminta, dikelompokkan per tanggal.
export type ActivityPerson = {
  id: number
  name: string
  division_name: string
  avatar_id: string | null
  is_active: boolean
  // filled dan total mengisi ringkasan "n dari N jam terisi" untuk rentang yang tampil.
  filled: number
  total: number
  days: Record<string, ActivityCell[]>
}

export type MyActivities = { date: string; slots: string[]; cells: ActivityCell[] }

export type ActivityRange = {
  from: string
  to: string
  // Tanggal yang benar-benar ditampilkan: hari kerja selalu ada, Sabtu dan Minggu hanya bila ada isinya.
  dates: string[]
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

// Satu panggilan memuat seluruh rentang: week (Senin sampai Jumat minggu tanggal itu), from dan to, atau
// date untuk satu hari. Halaman mingguan tidak pernah memanggil API per hari.
export function getActivityRange(
  q: { week?: string; from?: string; to?: string; date?: string },
  signal?: AbortSignal,
): Promise<ActivityRange> {
  const params = new URLSearchParams()
  for (const [k, v] of Object.entries(q)) if (v) params.set(k, v)
  return apiGet<ActivityRange>(`/activities?${params}`, signal)
}

export function saveActivity(input: SaveActivityInput): Promise<MyActivities> {
  return apiPut<MyActivities>('/activities', input)
}

// Memasang atau melepas sambungan ke jam sebelumnya. Jawabannya isian sehari penuh, karena melepas satu jam
// bisa ikut melepas jam setelahnya.
export function linkActivity(input: { user_id?: number; date: string; slot: number; linked: boolean }) {
  return apiPut<MyActivities>('/activities/link', input)
}

// Unduhan memakai tautan biasa, bukan fetch, supaya nama berkas dari Content-Disposition yang dipakai.
export function activityExportUrl(from: string, to: string): string {
  return `/api/activities/export?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`
}
