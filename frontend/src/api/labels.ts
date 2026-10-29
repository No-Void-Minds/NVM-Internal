import { apiGet, apiPost, apiPut } from './client'

export type LabelItem = {
  id: number
  name: string
  color: string
  division_id: number | null
  division_name?: string
  is_active: boolean
  task_count: number
}

export type LabelInput = { name: string; color: string; division_id: number | null }

export async function listLabels(): Promise<{ labels: LabelItem[]; palette: string[] }> {
  return apiGet<{ labels: LabelItem[]; palette: string[] }>('/labels')
}

export function createLabel(input: LabelInput) {
  return apiPost<{ id: number }>('/labels', input)
}

export function updateLabel(id: number, input: LabelInput) {
  return apiPut<void>(`/labels/${id}`, input)
}

export function setLabelActive(id: number, active: boolean) {
  return apiPost<void>(`/labels/${id}/${active ? 'activate' : 'deactivate'}`)
}
