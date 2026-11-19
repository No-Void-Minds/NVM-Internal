// Hitungan kalender Timeline (revisi 30). Tanggal berbentuk "2026-11-18" dan dihitung dalam UTC seperti
// utils/activityWeek.ts, supaya tidak bergeser di zona waktu browser mana pun; "hari ini" dari officeToday.
import { parseDay, shiftDay, weekStart } from './activityWeek.ts'

// Alasan kehadiran yang pribadi sudah disaring server: absence_type null berarti pembaca ini tidak berhak melihatnya.
export type TimelineCategory = 'launching' | 'produksi' | 'kehadiran'
export type AbsenceType =
  'sakit' | 'izin_pribadi' | 'cuti_pribadi' | 'cuti_khusus' | 'tukar_libur' | 'tukar_shift'

export type TimelineEvent = {
  id: number
  category: TimelineCategory
  title: string
  description: string
  start_date: string
  // null berarti satu hari.
  end_date: string | null
  user: { id: number; name: string } | null
  absence_type: AbsenceType | null
  absence_label?: string
}

export const categories: TimelineCategory[] = ['launching', 'produksi', 'kehadiran']
export const categoryLabel: Record<TimelineCategory, string> = {
  launching: 'Launching',
  produksi: 'Produksi',
  kehadiran: 'Kehadiran',
}

// Urutan sama dengan pilihan di form; labelnya sama dengan backend (timeline.absenceLabels).
export const absenceTypes: { value: AbsenceType; label: string }[] = [
  { value: 'sakit', label: 'Sakit' },
  { value: 'izin_pribadi', label: 'Izin pribadi' },
  { value: 'cuti_pribadi', label: 'Cuti pribadi' },
  { value: 'cuti_khusus', label: 'Cuti khusus' },
  { value: 'tukar_libur', label: 'Tukar libur' },
  { value: 'tukar_shift', label: 'Tukar shift' },
]

const monthYear = new Intl.DateTimeFormat('id-ID', { month: 'long', year: 'numeric', timeZone: 'UTC' })

export function monthLabel(month: string): string {
  return monthYear.format(parseDay(`${month}-01`))
}

export function isMonth(v: unknown): v is string {
  return typeof v === 'string' && /^\d{4}-(0[1-9]|1[0-2])$/.test(v)
}

export function shiftMonth(month: string, n: number): string {
  const d = parseDay(`${month}-01`)
  d.setUTCMonth(d.getUTCMonth() + n)
  return d.toISOString().slice(0, 7)
}

// Minggu mulai Senin. Grid selalu kelipatan 7 hari, dari Senin sebelum tanggal 1 sampai Minggu sesudah hari
// terakhir, jadi tanggal jatuh di kolom hari yang benar.
export function monthGrid(month: string): string[] {
  const first = `${month}-01`
  const last = shiftDay(`${shiftMonth(month, 1)}-01`, -1)
  const start = weekStart(first)
  const end = shiftDay(weekStart(last), 6)
  const days: string[] = []
  for (let d = start; d <= end; d = shiftDay(d, 1)) days.push(d)
  return days
}

// 0 Senin sampai 6 Minggu.
export function weekdayIndex(date: string): number {
  return (parseDay(date).getUTCDay() + 6) % 7
}

export function eventCovers(e: TimelineEvent, date: string): boolean {
  return e.start_date <= date && (e.end_date ?? e.start_date) >= date
}

// Teks ringkas satu entri, dipakai di sel, agenda, dan nama aksesibel. Kehadiran tanpa alasan memakai label
// dari server ("Tidak hadir"), bukan tebakan frontend.
export function eventSummary(e: TimelineEvent): string {
  if (e.category === 'kehadiran')
    return `${e.user?.name ?? 'Tanpa nama'}: ${e.absence_label ?? 'Tidak hadir'}`
  return e.title
}
