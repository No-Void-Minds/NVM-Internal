import { apiGet, apiPost, apiPut } from './client'
import type { Role } from '../utils/roles'

export type UserStatus = 'aktif' | 'terkunci' | 'nonaktif'

export type UserItem = {
  id: number
  name: string
  username: string
  role: Role
  division_id: number
  division_name: string
  status: UserStatus
  locked_until: string | null
  last_login_at: string | null
}

export type UserProfileInput = { name: string; role: Role; division_id: number }
export type NewUserInput = UserProfileInput & { username: string; password: string }

export async function listUsers(): Promise<UserItem[]> {
  const res = await apiGet<{ users: UserItem[] }>('/users')
  return res.users
}

export function createUser(input: NewUserInput) {
  return apiPost<{ id: number }>('/users', input)
}

export function updateUser(id: number, input: UserProfileInput) {
  return apiPut<void>(`/users/${id}`, input)
}

export function setUserActive(id: number, active: boolean) {
  return apiPost<void>(`/users/${id}/${active ? 'activate' : 'deactivate'}`)
}

export function resetUserPassword(id: number, password: string) {
  return apiPost<void>(`/users/${id}/reset-password`, { password })
}

export function unlockUser(id: number) {
  return apiPost<void>(`/users/${id}/unlock`)
}
