import type { TaskStatus } from './task'

type Progress = { status: TaskStatus; subtask_total: number; subtask_done: number }

// Aturan yang sama dengan backend (CountOpenSubtasks di Move). Backend tetap penentu akhir;
// ini hanya supaya pengguna tahu alasannya sebelum menekan tombol.
export function readyBlock(t: Progress): string | null {
  if (t.status !== 'on_progress') return null
  const open = t.subtask_total - t.subtask_done
  if (open <= 0) return null
  return `Masih ${open} sub task yang belum dicentang. Centang semuanya dulu sebelum menandai Ready.`
}
