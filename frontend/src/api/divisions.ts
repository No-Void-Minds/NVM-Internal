import { apiGet, apiPost, apiPut } from './client'

export type DivisionItem = { id: number; name: string; is_active: boolean; active_users: number }

export async function listDivisions(): Promise<DivisionItem[]> {
  const res = await apiGet<{ divisions: DivisionItem[] }>('/divisions')
  return res.divisions
}

export function createDivision(name: string) {
  return apiPost<{ id: number }>('/divisions', { name })
}

export function renameDivision(id: number, name: string) {
  return apiPut<void>(`/divisions/${id}`, { name })
}

export function setDivisionActive(id: number, active: boolean) {
  return apiPost<void>(`/divisions/${id}/${active ? 'activate' : 'deactivate'}`)
}
