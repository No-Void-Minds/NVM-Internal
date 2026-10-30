import { apiGet, apiPost, apiPut, request } from './client'
import type { TaskStatus, Urgency } from '../utils/task'
import type { TaskAttachment } from './attachments'

export type TaskPerson = { id: number; name: string; username: string }
export type TaskLabel = { id: number; name: string; color: string; is_active: boolean }

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
}

export type TaskCan = {
  edit: boolean
  delete: boolean
  edit_labels: boolean
  edit_deliverable: boolean
  move_to: TaskStatus[]
}

export type TaskHistoryItem = {
  id: number
  occurred_at: string
  action: string
  actor_name: string
  old_value: Record<string, unknown> | null
  new_value: Record<string, unknown> | null
}

export type TaskDetail = TaskSummary & {
  description: string
  requested_by: { id: number; name: string }
  request_date: string
  deliverable_url: string
  is_migrated: boolean
  can: TaskCan
  history: TaskHistoryItem[]
  attachments: TaskAttachment[]
}

export type TaskFields = {
  title: string
  description: string
  urgency: Urgency
  pic_ids: number[]
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

export function createTask(input: TaskFields & { label_ids: number[] }) {
  return apiPost<{ id: string }>('/tasks', input)
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

export function deleteTask(id: string, version: number) {
  return request<void>('DELETE', `/tasks/${id}?version=${version}`)
}
