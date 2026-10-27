import { createRouter, createWebHistory } from 'vue-router'
import { setUnauthorizedHandler } from '../api/client'
import { useSessionStore } from '../stores/session'
import LoginPage from '../pages/LoginPage.vue'
import ForcePasswordPage from '../pages/ForcePasswordPage.vue'
import StatusPage from '../pages/StatusPage.vue'
import ProfilePage from '../pages/ProfilePage.vue'
import NotFoundPage from '../pages/NotFoundPage.vue'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    public?: boolean
    layout?: 'auth' | 'app'
  }
}

// Sementara semua peran ke Status. Sprint 2: Admin ke Pengguna; Sprint 4: Manajer dan Staf ke Papan Task.
const home = { name: 'status' }

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
    { path: '/profil', name: 'profile', component: ProfilePage, meta: { title: 'Profil' } },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: NotFoundPage,
      meta: { title: 'Halaman tidak ditemukan', public: true },
    },
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
  if (to.name === 'login') return user ? home : true
  if (to.meta.public) return true
  if (!user) return { name: 'login', query: { kembali: to.fullPath } }
  // Password sementara wajib diganti sebelum halaman lain bisa dibuka (FR-AUTH-07).
  if (user.must_change_password && to.name !== 'force-password') return { name: 'force-password' }
  if (!user.must_change_password && to.name === 'force-password') return home
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
