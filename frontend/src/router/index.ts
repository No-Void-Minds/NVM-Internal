import { createRouter, createWebHistory } from 'vue-router'
import StatusPage from '../pages/StatusPage.vue'
import NotFoundPage from '../pages/NotFoundPage.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    // Sementara jadi halaman awal; di Sprint 1 alamat "/" membuka halaman Masuk (NFR-SEC-07).
    { path: '/', name: 'status', component: StatusPage, meta: { title: 'Status sistem' } },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: NotFoundPage,
      meta: { title: 'Halaman tidak ditemukan' },
    },
  ],
})

router.afterEach((to) => {
  const title = typeof to.meta.title === 'string' ? to.meta.title : ''
  document.title = title ? `${title} | NVM Internal` : 'NVM Internal'
})