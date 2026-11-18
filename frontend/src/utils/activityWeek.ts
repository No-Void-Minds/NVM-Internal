// Hitungan tanggal halaman Aktivitas (revisi 21). Semua tanggal berbentuk "2026-11-18" dan dihitung dalam UTC,
// supaya tidak bergeser satu hari di zona waktu browser mana pun. Aturannya sama dengan activity.WeekRange
// di backend: minggu kerja Senin sampai Jumat.
export type ActivityMode = 'minggu' | 'hari'

const long = new Intl.DateTimeFormat('id-ID', {
  weekday: 'long',
  day: 'numeric',
  month: 'long',
  year: 'numeric',
  timeZone: 'UTC',
})
const monthYear = new Intl.DateTimeFormat('id-ID', { month: 'long', year: 'numeric', timeZone: 'UTC' })
const dayMonth = new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'long', timeZone: 'UTC' })
const shortDay = new Intl.DateTimeFormat('id-ID', { weekday: 'short', timeZone: 'UTC' })

export function parseDay(date: string): Date {
  return new Date(`${date}T00:00:00Z`)
}

export function dayKey(d: Date): string {
  return d.toISOString().slice(0, 10)
}

export function isDay(value: unknown): value is string {
  return typeof value === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(value) && !isNaN(parseDay(value).getTime())
}

export function shiftDay(date: string, days: number): string {
  const d = parseDay(date)
  d.setUTCDate(d.getUTCDate() + days)
  return dayKey(d)
}

// Senin minggu tanggal itu. Hari Sabtu dan Minggu masih masuk minggu Senin yang sudah lewat.
export function weekStart(date: string): string {
  const d = parseDay(date)
  return shiftDay(date, -((d.getUTCDay() + 6) % 7))
}

// Senin sampai Jumat. Sabtu dan Minggu tidak ikut di sini: backend yang menambahkannya bila ada isinya.
export function weekDays(date: string): string[] {
  const monday = weekStart(date)
  return Array.from({ length: 5 }, (_, i) => shiftDay(monday, i))
}

export function dayLabel(date: string): string {
  return long.format(parseDay(date))
}

export function shortDayLabel(date: string): string {
  return shortDay.format(parseDay(date))
}

// Label rentang: "16 sampai 20 November 2026", atau menyebut bulan dan tahun kedua sisi bila berbeda.
export function rangeLabel(from: string, to: string): string {
  if (from === to) return dayLabel(from)
  const a = parseDay(from)
  const b = parseDay(to)
  if (a.getUTCFullYear() !== b.getUTCFullYear()) {
    return `${dayMonth.format(a)} ${a.getUTCFullYear()} sampai ${dayMonth.format(b)} ${b.getUTCFullYear()}`
  }
  if (a.getUTCMonth() !== b.getUTCMonth()) {
    return `${dayMonth.format(a)} sampai ${dayMonth.format(b)} ${b.getUTCFullYear()}`
  }
  return `${a.getUTCDate()} sampai ${b.getUTCDate()} ${monthYear.format(b)}`
}

// Hari terakhir bulan tanggal itu, untuk pilihan ekspor "Bulan ini".
export function monthRange(date: string): { from: string; to: string } {
  const d = parseDay(date)
  const from = `${date.slice(0, 7)}-01`
  const last = new Date(Date.UTC(d.getUTCFullYear(), d.getUTCMonth() + 1, 0))
  return { from, to: dayKey(last) }
}
