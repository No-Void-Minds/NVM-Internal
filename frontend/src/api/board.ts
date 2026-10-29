import { apiGet } from './client'
import type { TaskSummary } from './tasks'
import type { TaskStatus, Urgency } from '../utils/task'

export type DueFilter = 'overdue' | 'week' | 'none'

export type BoardQuery = {
  scope?: 'mine' | 'all'
  pic?: string
  division?: number
  label?: number
  urgency?: Urgency
  due?: DueFilter
}

export type BoardTask = TaskSummary & { move_to: TaskStatus[] }
export type BoardColumn = { status: TaskStatus; count: number; tasks: BoardTask[] }
export type Board = { columns: BoardColumn[]; completed_days: number }

export type Person = {
  id: number
  name: string
  username: string
  role: 'staff' | 'manager'
  division_name: string
}

export function getBoard(q: BoardQuery, signal?: AbortSignal): Promise<Board> {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(q)) {
    if (value !== undefined && value !== '') params.set(key, String(value))
  }
  const qs = params.toString()
  return apiGet<Board>(qs ? `/board?${qs}` : '/board', signal)
}

export async function listPeople(): Promise<Person[]> {
  return (await apiGet<{ people: Person[] }>('/people')).people
}
