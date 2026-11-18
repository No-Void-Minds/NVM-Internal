// Task berulang (revisi 19). Perhitungan di sini hanya untuk kalimat pratinjau di form, untuk aturan yang belum
// disimpan. Kapan kemunculan benar-benar dibuat ditentukan backend (task/recurrence.go) dengan aturan yang sama.

export type Freq = 'daily' | 'weekly' | 'monthly'

export type ScheduleForm = {
  freq: Freq | 'none'
  skip_weekend: boolean
  // Hari ISO: 1 Senin sampai 7 Minggu.
  weekdays: number[]
  month_day: number
  end_date: string
}

export const freqLabel: Record<Freq, string> = { daily: 'Harian', weekly: 'Mingguan', monthly: 'Bulanan' }

// Urutan Senin sampai Minggu, dengan angka ISO.
export const weekdayShort: { day: number; short: string; long: string }[] = [
  { day: 1, short: 'Sen', long: 'Senin' },
  { day: 2, short: 'Sel', long: 'Selasa' },
  { day: 3, short: 'Rab', long: 'Rabu' },
  { day: 4, short: 'Kam', long: 'Kamis' },
  { day: 5, short: 'Jum', long: 'Jumat' },
  { day: 6, short: 'Sab', long: 'Sabtu' },
  { day: 7, short: 'Min', long: 'Minggu' },
]

// Tanggal YYYY-MM-DD diolah sebagai UTC supaya tidak bergeser zona waktu browser.
function parse(v: string): Date {
  return new Date(`${v}T00:00:00Z`)
}

export function isoDay(v: string): string {
  return v.slice(0, 10)
}

function fmt(d: Date): string {
  return d.toISOString().slice(0, 10)
}

function addDays(v: string, n: number): string {
  const d = parse(v)
  d.setUTCDate(d.getUTCDate() + n)
  return fmt(d)
}

export function isoWeekday(v: string): number {
  const w = parse(v).getUTCDay()
  return w === 0 ? 7 : w
}

function lastDay(v: string): number {
  const d = parse(v)
  return new Date(Date.UTC(d.getUTCFullYear(), d.getUTCMonth() + 1, 0)).getUTCDate()
}

export function occursOn(s: ScheduleForm, day: string, start: string): boolean {
  if (s.freq === 'none' || day < start || (s.end_date && day > s.end_date)) return false
  if (s.freq === 'daily') {
    const w = isoWeekday(day)
    return !s.skip_weekend || (w !== 6 && w !== 7)
  }
  if (s.freq === 'weekly') return s.weekdays.includes(isoWeekday(day))
  return Number(day.slice(8, 10)) === Math.min(s.month_day, lastDay(day))
}

// Kemunculan pertama setelah tanggal after, atau null bila seri sudah habis dalam 400 hari.
export function nextOccurrence(s: ScheduleForm, after: string, start: string): string | null {
  let d = addDays(after, 1)
  if (d < start) d = start
  for (let i = 0; i < 400; i++) {
    if (s.end_date && d > s.end_date) return null
    if (occursOn(s, d, start)) return d
    d = addDays(d, 1)
  }
  return null
}

// Hari ini menurut kantor (Asia/Jakarta), sama dengan officeTZ di backend.
export function officeToday(): string {
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Jakarta' }).format(new Date())
}

const longDate = new Intl.DateTimeFormat('id-ID', {
  weekday: 'long',
  day: 'numeric',
  month: 'long',
  year: 'numeric',
  timeZone: 'UTC',
})
const plainDate = new Intl.DateTimeFormat('id-ID', {
  day: 'numeric',
  month: 'long',
  year: 'numeric',
  timeZone: 'UTC',
})

export function formatLong(v: string): string {
  return longDate.format(parse(v))
}

export function formatPlain(v: string): string {
  return plainDate.format(parse(v))
}

export function dueOffset(due: string, from: string): number | null {
  if (!due) return null
  return Math.round((parse(due).getTime() - parse(from).getTime()) / 86_400_000)
}

// Kalimat pratinjau, mis. "Muncul berikutnya Senin, 23 November 2026, deadline 23 November 2026."
export function previewSentence(next: string | null, offset: number | null): string {
  if (!next) return 'Tidak ada kemunculan berikutnya sebelum tanggal berhenti.'
  const due = offset === null ? 'tanpa deadline' : `deadline ${formatPlain(addDays(next, offset))}`
  return `Muncul berikutnya ${formatLong(next)}, ${due}.`
}

// Aturan dalam satu kalimat, untuk detail task.
export function describeRule(s: {
  freq: Freq
  skip_weekend: boolean
  weekdays: number[]
  month_day: number | null
  end_date: string | null
}): string {
  let base: string
  if (s.freq === 'daily') base = s.skip_weekend ? 'Setiap hari kerja (Senin sampai Jumat)' : 'Setiap hari'
  else if (s.freq === 'weekly') {
    const names = weekdayShort.filter((w) => s.weekdays.includes(w.day)).map((w) => w.long)
    base = `Setiap ${names.join(', ')}`
  } else base = `Setiap tanggal ${s.month_day} (hari terakhir bila bulannya lebih pendek)`
  return s.end_date ? `${base}, sampai ${formatPlain(s.end_date)}.` : `${base}.`
}
