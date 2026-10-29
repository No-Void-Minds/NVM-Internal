import { createRouter, createWebHistory } from 'vue-router'
import { setUnauthorizedHandler } from '../api/client'
import { useSessionStore } from '../stores/session'
import type { Role } from '../utils/roles'
import { homeFor } from './home'
import LoginPage from '../pages/LoginPage.vue'
import ForcePasswordPage from '../pages/ForcePasswordPage.vue'
import StatusPage from '../pages/StatusPage.vue'
import ProfilePage from '../pages/ProfilePage.vue'
import UsersPage from '../pages/UsersPage.vue'
import NotFoundPage from '../pages/NotFoundPage.vue'
import SettingsPage from '../pages/SettingsPage.vue'
import AuditLogPage from '../pages/AuditLogPage.vue'
import TasksPage from '../pages/TasksPage.vue'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    public?: boolean
    layout?: 'auth' | 'app'
    // Kosong berarti semua peran boleh membuka. Backend tetap menolak dengan 403 bila URL dipanggil langsung.
    roles?: Role[]
  }
}

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'login',
      component: LoginPage,
      meta: { title: 'Masuk', public: true, layout: 'auth' },
    },
    {
      path: '/ganti-password',
      name: 'force-password',
      component: ForcePasswordPage,
      meta: { title: 'Buat password baru', layout: 'auth' },
    },
    { path: '/status', name: 'status', component: StatusPage, meta: { title: 'Status sistem' } },
    {
      path: '/pengguna',
      name: 'users',
      component: UsersPage,
      meta: { title: 'Pengguna', roles: ['admin', 'manager'] },
    },
    { path: '/profil', name: 'profile', component: ProfilePage, meta: { title: 'Profil' } },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: NotFoundPage,
      meta: { title: 'Halaman tidak ditemukan', public: true },
    },
    {
      path: '/pengaturan',
      name: 'settings',
      component: SettingsPage,
      meta: { title: 'Pengaturan', roles: ['admin', 'manager'] },
    },
    {
      path: '/audit-log',
      name: 'audit-log',
      component: AuditLogPage,
      meta: { title: 'Audit Log', roles: ['admin', 'manager'] },
    },
    { path: '/task', name: 'tasks', component: TasksPage, meta: { title: 'Task' } },
  ],
})

router.beforeEach(async (to) => {
  const session = useSessionStore()
  if (!session.checked) {
    try {
      await session.fetchMe()
    } catch {
      // Backend tidak bisa dihubungi: anggap belum masuk, pesan galatnya muncul saat mencoba login.
      session.clear()
    }
  }

  const user = session.user
  if (to.name === 'login') return user ? homeFor(user) : true
  if (to.meta.public) return true
  if (!user) return { name: 'login', query: { kembali: to.fullPath } }
  // Password sementara wajib diganti sebelum halaman lain bisa dibuka (FR-AUTH-07).
  if (user.must_change_password && to.name !== 'force-password') return { name: 'force-password' }
  if (!user.must_change_password && to.name === 'force-password') return homeFor(user)
  if (to.meta.roles && !to.meta.roles.includes(user.role)) return homeFor(user)
  return true
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} | NVM Internal` : 'NVM Internal'
})

// Sesi habis di tengah pemakaian (idle 30 menit, logout di perangkat lain, akun dinonaktifkan).
setUnauthorizedHandler(() => {
  const session = useSessionStore()
  if (!session.user) return
  const current = router.currentRoute.value.fullPath
  session.clear()
  void router.replace({ name: 'login', query: { alasan: 'sesi-berakhir', kembali: current } })
})
