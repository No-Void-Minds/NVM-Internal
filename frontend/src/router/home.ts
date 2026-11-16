import type { RouteLocationRaw, Router } from 'vue-router'
import type { SessionUser } from '../stores/session'
import { returnTarget } from './redirect'

// Admin ke Pengguna; Manajer dan Staf ke Papan Task.
export function homeFor(user: SessionUser | null): RouteLocationRaw {
  if (user?.role === 'admin') return { name: 'users' }
  if (user) return { name: 'tasks' }
  return { name: 'login' }
}

// Setelah login atau ganti password paksa: ke ?kembali= bila sah, selain itu ke halaman awal.
// Tujuan di luar aplikasi (dashboard, studio) dibuka sebagai navigasi browser biasa, bukan lewat router.
export async function goAfterLogin(router: Router, kembali: unknown, user: SessionUser | null) {
  const target = returnTarget(kembali)
  if (target && 'external' in target) {
    window.location.assign(target.external)
    return
  }
  await router.replace(target?.internal ?? homeFor(user))
}

// ?kembali= ikut dibawa ke halaman ganti password paksa, supaya tujuan awal tidak hilang.
export function keepKembali(kembali: unknown): Record<string, string> {
  return typeof kembali === 'string' && kembali ? { kembali } : {}
}
