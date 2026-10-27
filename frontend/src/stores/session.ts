import { defineStore } from 'pinia'
import { ref } from 'vue'
import { ApiError, apiGet, apiPost } from '../api/client'
import type { Role } from '../utils/roles'

export type SessionUser = {
  id: number
  name: string
  username: string
  role: Role
  division_id: number
  division_name?: string
  must_change_password: boolean
}

export const useSessionStore = defineStore('session', () => {
  const user = ref<SessionUser | null>(null)
  const checked = ref(false)
  // Password login disimpan sebentar di memori khusus untuk layar Buat password baru, supaya password
  // sementara tidak perlu diketik dua kali. Tidak pernah ditulis ke localStorage atau sessionStorage.
  const pendingPassword = ref<string | null>(null)

  async function fetchMe() {
    try {
      const res = await apiGet<{ user: SessionUser }>('/auth/me')
      user.value = res.user
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        user.value = null
        return
      }
      throw err
    } finally {
      checked.value = true
    }
  }

  async function login(username: string, password: string) {
    const res = await apiPost<{ user: SessionUser }>('/auth/login', { username, password })
    pendingPassword.value = res.user.must_change_password ? password : null
    // /auth/me dipanggil ulang karena hanya endpoint itu yang membawa nama divisi.
    await fetchMe()
  }

  async function logout() {
    try {
      await apiPost('/auth/logout')
    } catch (err) {
      // Sesi yang sudah berakhir di server tetap dianggap berhasil keluar.
      if (!(err instanceof ApiError && err.status === 401)) throw err
    }
    clear()
  }

  async function changePassword(currentPassword: string, newPassword: string) {
    await apiPost('/auth/change-password', { current_password: currentPassword, new_password: newPassword })
    pendingPassword.value = null
    await fetchMe()
  }

  function clear() {
    user.value = null
    pendingPassword.value = null
  }

  return { user, checked, pendingPassword, fetchMe, login, logout, changePassword, clear }
})
