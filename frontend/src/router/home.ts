import type { RouteLocationRaw } from 'vue-router'
import type { SessionUser } from '../stores/session'

// Sementara Admin dan Manajer ke Pengguna, Staf ke Status. Sprint 4: Manajer dan Staf ke Papan Task.
export function homeFor(user: SessionUser | null): RouteLocationRaw {
  if (user?.role === 'admin' || user?.role === 'manager') return { name: 'users' }
  return { name: 'status' }
}
