import type { CreateSubtaskInput, SubtaskInput, TaskDetail } from '../api/tasks'

// Isi satu sub task di form (revisi 16): dipakai Tambah sub task di detail, Ubah sub task, dan form Buat task.
// files hanya dipakai saat menambah: lampiran acuan diunggah setelah sub task punya id.
export type SubtaskForm = {
  title: string
  description: string
  reference_url: string
  pic_id: number
  due_date: string
  files: File[]
}

export type SubtaskDraft = SubtaskForm & { key: string; id: number }

// Sama dengan batas di backend (task/subtasks.go).
export const maxSubtasks = 50

let counter = 0
// key hanya untuk v-for. id database tidak bisa dipakai karena sub task baru belum punya id.
export function newKey(): string {
  counter += 1
  return `sub-${counter}`
}

export function emptyForm(): SubtaskForm {
  return { title: '', description: '', reference_url: '', pic_id: 0, due_date: '', files: [] }
}

export function emptyDraft(): SubtaskDraft {
  return { ...emptyForm(), key: newKey(), id: 0 }
}

export function draftsFromTask(t: TaskDetail | null): SubtaskDraft[] {
  return (t?.subtasks ?? []).map((s) => ({ ...emptyForm(), key: newKey(), id: s.id, title: s.title }))
}

// Form Ubah task hanya mengirim nama dan urutan; isi lain diubah dari detail supaya tidak tertimpa.
export function toPayload(drafts: SubtaskDraft[]): SubtaskInput[] {
  return drafts.map((d) => (d.id ? { id: d.id, title: d.title.trim() } : { title: d.title.trim() }))
}

export function toCreatePayload(drafts: SubtaskDraft[]): CreateSubtaskInput[] {
  return drafts.map((d) => ({
    title: d.title.trim(),
    description: d.description.trim(),
    reference_url: d.reference_url.trim(),
    pic_id: d.pic_id,
    due_date: d.due_date,
  }))
}

// Sama dengan pesan backend (checkSubtaskDue), supaya galat muncul sebelum dikirim.
export function subtaskDueProblem(due: string, taskDue: string | null): string | null {
  if (!due || !taskDue || due <= taskDue) return null
  return `Deadline sub task tidak boleh melewati deadline task (${formatLongDay(taskDue)}).`
}

const longDay = new Intl.DateTimeFormat('id-ID', {
  day: 'numeric',
  month: 'long',
  year: 'numeric',
  timeZone: 'UTC',
})
function formatLongDay(v: string): string {
  return longDay.format(new Date(`${v}T00:00:00Z`))
}
