import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '../stores/session'

// Dipakai tombol Keluar di header, di Profil, dan di layar Buat password baru.
export function useLogout() {
  const session = useSessionStore()
  const router = useRouter()
  const loggingOut = ref(false)
  const logoutError = ref<string | null>(null)

  async function logout() {
    loggingOut.value = true
    logoutError.value = null
    try {
      await session.logout()
      await router.replace({ name: 'login' })
    } catch {
      logoutError.value = 'Gagal keluar. Periksa koneksi lalu coba lagi.'
    } finally {
      loggingOut.value = false
    }
  }

  return { logout, loggingOut, logoutError }
}
