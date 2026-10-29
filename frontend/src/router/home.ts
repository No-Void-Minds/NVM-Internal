import type { RouteLocationRaw } from 'vue-router'
import type { SessionUser } from '../stores/session'

// Sementara Admin dan Manajer ke Pengguna, Staf ke Status. Sprint 4: Manajer dan Staf ke Papan Task.
// Admin ke Pengguna; Manajer dan Staf ke Task. Sprint 4: Task diganti Papan Task.
export function homeFor(user: SessionUser | null): RouteLocationRaw {
  if (user?.role === 'admin') return { name: 'users' }
  if (user) return { name: 'tasks' }
  return { name: 'login' }
}
