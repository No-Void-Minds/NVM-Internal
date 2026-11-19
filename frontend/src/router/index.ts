import { createRouter, createWebHistory } from 'vue-router'
import { setUnauthorizedHandler } from '../api/client'
import { useSessionStore } from '../stores/session'
import type { Role } from '../utils/roles'
import { homeFor, keepKembali } from './home'
import { returnTarget } from './redirect'
import LoginPage from '../pages/LoginPage.vue'
import ForcePasswordPage from '../pages/ForcePasswordPage.vue'
import StatusPage from '../pages/StatusPage.vue'
import ProfilePage from '../pages/ProfilePage.vue'
import UsersPage from '../pages/UsersPage.vue'
import NotFoundPage from '../pages/NotFoundPage.vue'
import SettingsPage from '../pages/SettingsPage.vue'
import AuditLogPage from '../pages/AuditLogPage.vue'
import BoardPage from '../pages/BoardPage.vue'
import TaskPage from '../pages/TaskPage.vue'
import TaskNewPage from '../pages/TaskNewPage.vue'
import ArchivePage from '../pages/ArchivePage.vue'
import ActivityPage from '../pages/ActivityPage.vue'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    public?: boolean
    layout?: 'auth' | 'app'
    // Kosong berarti semua peran boleh membuka. Backend tetap menolak dengan 403 bila URL dipanggil langsung.
    roles?: Role[]
    // needsCreateTask: hanya akun yang boleh membuat task (can_create_task dari server).
    needsCreateTask?: boolean
    // Peran di luar roles melihat halaman 404, bukan dialihkan ke beranda, supaya keberadaan halaman ini tidak
    // terungkap (revisi 23).
    hideFromOthers?: boolean
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
    {
      path: '/status',
      name: 'status',
      component: StatusPage,
      meta: { title: 'Status sistem', roles: ['admin'], hideFromOthers: true },
    },
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
    { path: '/task', name: 'tasks', component: BoardPage, meta: { title: 'Papan Task' } },
    // Rute statis selalu menang atas /task/:id di vue-router, jadi "baru" tidak pernah terbaca sebagai ID task.
    // Yang boleh membuka ditentukan can_create_task dari server (revisi 27), bukan daftar peran di sini.
    // Yang tidak boleh melihat 404 dengan URL tetap sama (pola revisi 23).
    {
      path: '/task/baru',
      name: 'task-new',
      component: TaskNewPage,
      meta: { title: 'Buat task', needsCreateTask: true, hideFromOthers: true },
    },
    { path: '/task/:id', name: 'task-detail', component: TaskPage, meta: { title: 'Detail task' } },
    { path: '/arsip', name: 'archive', component: ArchivePage, meta: { title: 'Arsip' } },
    { path: '/aktivitas', name: 'activity', component: ActivityPage, meta: { title: 'Aktivitas Harian' } },
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
  if (to.name === 'login') {
    if (!user) return true
    if (user.must_change_password) return { name: 'force-password', query: keepKembali(to.query.kembali) }
    // Sudah login tetapi dikirim ke sini oleh dashboard (mis. cookie lama): langsung ke tujuan bila sah.
    const target = returnTarget(to.query.kembali)
    if (target && 'external' in target) {
      window.location.assign(target.external)
      return false
    }
    return target?.internal ?? homeFor(user)
  }
  if (to.meta.public) return true
  if (!user) return { name: 'login', query: { kembali: to.fullPath } }
  // Password sementara wajib diganti sebelum halaman lain bisa dibuka (FR-AUTH-07).
  if (user.must_change_password && to.name !== 'force-password')
    return { name: 'force-password', query: keepKembali(to.fullPath) }
  if (!user.must_change_password && to.name === 'force-password') return homeFor(user)
  if (to.meta.needsCreateTask && !user.can_create_task) {
    return {
      name: 'not-found',
      params: { pathMatch: to.path.slice(1).split('/') },
      query: to.query,
    }
  }
  if (to.meta.roles && !to.meta.roles.includes(user.role)) {
    if (!to.meta.hideFromOthers) return homeFor(user)
    // Tampilkan 404 yang sudah ada dengan URL tetap sama, seperti alamat yang memang tidak ada.
    return {
      name: 'not-found',
      params: { pathMatch: to.path.slice(1).split('/') },
      query: to.query,
      hash: to.hash,
      replace: true,
    }
  }
  return true
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} | NVM Internal` : 'NVM Internal'
})

// Sesi habis di tengah pemakaian (umur sesi 9 jam, logout di perangkat lain, akun dinonaktifkan).
setUnauthorizedHandler(() => {
  const session = useSessionStore()
  if (!session.user) return
  const current = router.currentRoute.value.fullPath
  session.clear()
  void router.replace({ name: 'login', query: { alasan: 'sesi-berakhir', kembali: current } })
})
