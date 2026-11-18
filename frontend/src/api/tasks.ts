import type { Freq } from '../utils/recurrence'
import { apiGet, apiPost, apiPut, request } from './client'
import type { TaskStatus, Urgency } from '../utils/task'
import type { TaskAttachment } from './attachments'

export type TaskPerson = { id: number; name: string; username: string; avatar_id: string | null }
export type TaskLabel = { id: number; name: string; color: string; is_active: boolean }
export type TaskSubtask = {
  id: number
  title: string
  description: string
  // reference_url diisi yang menugaskan sebagai acuan, deliverable_url diisi yang mengerjakan sebagai hasil.
  reference_url: string
  deliverable_url: string
  created_by: string
  created_by_id: number | null
  done: boolean
  done_at: string | null
  done_by: string
  // Tenggat sub task (revisi 16), YYYY-MM-DD atau null.
  due_date: string | null
  // Catatan hasil dari yang mengerjakan (revisi 18), kosong bila belum diisi.
  result_note: string
  // Boleh mengunggah dan menghapus berkas sub task ini, dihitung server (revisi 18).
  can_attach: boolean
  // Dihitung backend dengan aturan yang sama dengan penolakan 403, jadi UI tidak menebak siapa yang boleh.
  can_check: boolean
  // Boleh diubah atau dihapus: pembuatnya, PIC-nya, dan Manajer, selama task masih Requested atau On Progress.
  can_edit: boolean
  assignees: TaskPerson[]
  attachments: TaskAttachment[]
}
// id kosong berarti sub task baru; backend memakainya untuk membedakan sub task lama yang hanya diubah.
export type SubtaskInput = { id?: number; title: string }

export type TaskSummary = {
  id: string
  title: string
  status: TaskStatus
  urgency: Urgency
  division: { id: number; name: string }
  start_date: string | null
  due_date: string | null
  solved_date: string | null
  version: number
  assignees: TaskPerson[]
  labels: TaskLabel[]
  subtask_total: number
  subtask_done: number
  // Revisi 15: PIC sub task yang cocok dengan filter PIC dan bukan PIC task. Kosong tanpa filter PIC.
  subtask_pic_matches: { id: number; name: string }[]
  // Revisi 19: null bila tidak berulang.
  series_freq: Freq | null
  occurrence_date: string | null
}

export type SeriesInfo = {
  id: number
  freq: Freq
  skip_weekend: boolean
  weekdays: number[]
  month_day: number | null
  start_date: string | null
  end_date: string | null
  active: boolean
  // Dihitung server dengan aturan pembuat kemunculan; null bila seri berhenti atau habis.
  next_date: string | null
  due_offset_days: number | null
}

// Bagian Jadwal yang dikirim ke server. Tidak dikirim berarti seri tidak disentuh.
export type ScheduleBody = {
  freq: Freq | 'none'
  skip_weekend: boolean
  weekdays: number[]
  month_day: number
  end_date: string
  apply_to_series?: boolean
}

export type TaskCan = {
  edit: boolean
  delete: boolean
  edit_labels: boolean
  edit_deliverable: boolean
  add_subtask: boolean
  move_to: TaskStatus[]
  // Pengguna ini PIC salah satu sub task (revisi 17). Hanya untuk teks; haknya sudah ada di move_to.
  subtask_pic: boolean
}

export type TaskHistoryItem = {
  id: number
  occurred_at: string
  action: string
  actor_name: string
  actor_id: number | null
  old_value: Record<string, unknown> | null
  new_value: Record<string, unknown> | null
}

export type TaskDetail = TaskSummary & {
  description: string
  requested_by: { id: number; name: string }
  request_date: string
  deliverable_url: string
  is_migrated: boolean
  series: SeriesInfo | null
  can: TaskCan
  history: TaskHistoryItem[]
  attachments: TaskAttachment[]
  subtasks: TaskSubtask[]
}

export type TaskFields = {
  schedule?: ScheduleBody
  title: string
  description: string
  urgency: Urgency
  pic_ids: number[]
  subtasks: SubtaskInput[]
  start_date: string
  due_date: string
  deliverable_url: string
}

export async function listTasks(scope: 'mine' | 'all'): Promise<TaskSummary[]> {
  const res = await apiGet<{ tasks: TaskSummary[] }>(scope === 'mine' ? '/tasks?scope=mine' : '/tasks')
  return res.tasks
}

export async function getTask(id: string): Promise<TaskDetail> {
  const res = await apiGet<{ task: TaskDetail }>(`/tasks/${id}`)
  return res.task
}

// subtask_ids urut sesuai form, dipakai untuk mengunggah lampiran acuan tiap sub task setelah task jadi.
export function createTask(
  input: Omit<TaskFields, 'subtasks'> & { label_ids: number[]; subtasks: CreateSubtaskInput[] },
) {
  return apiPost<{ id: string; subtask_ids: number[] }>('/tasks', input)
}

export function updateTask(id: string, version: number, input: TaskFields) {
  return apiPut<{ version: number }>(`/tasks/${id}`, { version, ...input })
}

export function setTaskLabels(id: string, version: number, labelIds: number[]) {
  return apiPut<{ version: number }>(`/tasks/${id}/labels`, { version, label_ids: labelIds })
}

export function setTaskDeliverable(id: string, version: number, url: string) {
  return apiPut<{ version: number }>(`/tasks/${id}/deliverable`, { version, deliverable_url: url })
}

export function moveTask(id: string, version: number, to: TaskStatus, note = '') {
  return apiPost<{ version: number; status: TaskStatus; solved_date: string | null }>(`/tasks/${id}/move`, {
    version,
    to,
    note,
  })
}

// Hentikan pengulangan (revisi 19). Kemunculan yang sudah ada tidak berubah.
export function stopSeries(id: string) {
  return apiPost<void>(`/tasks/${id}/stop-series`)
}

export function deleteTask(id: string, version: number) {
  return request<void>('DELETE', `/tasks/${id}?version=${version}`)
}

export function setSubtaskDone(taskId: string, subtaskId: number, done: boolean) {
  return apiPut<void>(`/tasks/${taskId}/subtasks/${subtaskId}/done`, { done })
}

// Sub task di luar form Manajer: PIC task menyusun pekerjaannya sendiri dan boleh menugaskannya
// ke PIC task yang lain (revisi 7). picId 0 berarti belum ditugaskan.
export type SubtaskBody = {
  title: string
  description: string
  reference_url: string
  deliverable_url: string
  pic_id: number
  due_date: string
  // Tidak dikirim berarti catatan hasil lama dipertahankan server (revisi 18).
  result_note?: string
}

// Sub task dari form Buat task (revisi 16): isinya sama dengan Tambah sub task di detail, tanpa tautan hasil.
export type CreateSubtaskInput = Omit<SubtaskBody, 'deliverable_url' | 'result_note'>

export function addSubtask(taskId: string, body: SubtaskBody) {
  return apiPost<{ id: number }>(`/tasks/${taskId}/subtasks`, body)
}

export function updateSubtask(taskId: string, subtaskId: number, body: SubtaskBody) {
  return apiPut<void>(`/tasks/${taskId}/subtasks/${subtaskId}`, body)
}

export function deleteSubtask(taskId: string, subtaskId: number) {
  return request<void>('DELETE', `/tasks/${taskId}/subtasks/${subtaskId}`)
}
