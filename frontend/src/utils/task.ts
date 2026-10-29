export type TaskStatus = 'requested' | 'on_progress' | 'ready' | 'completed'
export type Urgency = 'low' | 'medium' | 'high' | 'urgent'

// Nama status mengikuti dropdown Excel lama (FR-TASK-04), jadi tetap berbahasa Inggris.
export const statusLabel: Record<TaskStatus, string> = {
  requested: 'Requested',
  on_progress: 'On Progress',
  ready: 'Ready',
  completed: 'Completed',
}

export const statusClass: Record<TaskStatus, string> = {
  requested: 'bg-requested-bg text-requested-fg',
  on_progress: 'bg-progress-bg text-progress-fg',
  ready: 'bg-ready-bg text-ready-fg',
  completed: 'bg-completed-bg text-completed-fg',
}

export const urgencies: Urgency[] = ['low', 'medium', 'high', 'urgent']

export const urgencyLabel: Record<Urgency, string> = {
  low: 'Low',
  medium: 'Medium',
  high: 'High',
  urgent: 'Urgent',
}

// Hanya High dan Urgent yang diberi warna, supaya mata langsung tertuju ke yang mendesak.
export const urgencyClass: Record<Urgency, string> = {
  low: 'text-ink-muted',
  medium: 'text-ink-muted',
  high: 'font-medium text-high',
  urgent: 'font-semibold text-urgent',
}

const dateFormat = new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'short', year: 'numeric' })

// Tanggal dari backend berbentuk "2026-11-05" tanpa jam; dibaca sebagai tanggal lokal supaya tidak bergeser sehari.
export function parseDay(value: string): Date {
  const [y, m, d] = value.split('-').map(Number)
  return new Date(y, m - 1, d)
}

export function formatDay(value: string): string {
  return dateFormat.format(parseDay(value))
}

export function isOverdue(due: string | null, status: TaskStatus): boolean {
  if (!due || status === 'completed') return false
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return parseDay(due) < today
}

// Untuk cap waktu lengkap seperti solved_date: tanggalnya dibaca dalam zona waktu browser.
export function formatMoment(value: string): string {
  return dateFormat.format(new Date(value))
}
