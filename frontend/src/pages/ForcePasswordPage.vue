<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import AuthCard from '../components/AuthCard.vue'
import { ApiError } from '../api/client'
import { useSessionStore } from '../stores/session'
import { useLogout } from '../composables/useLogout'
import { validateNewPassword } from '../utils/password'

const session = useSessionStore()
const router = useRouter()
const { logout, loggingOut, logoutError } = useLogout()

// Setelah halaman dimuat ulang, password login sudah hilang dari memori, jadi perlu diketik lagi.
const needsCurrent = session.pendingPassword === null
const current = ref(session.pendingPassword ?? '')
const next = ref('')
const confirm = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

async function submit() {
  if (submitting.value) return
  error.value = validateNewPassword(current.value, next.value, confirm.value)
  if (error.value) return

  submitting.value = true
  try {
    await session.changePassword(current.value, next.value)
    await router.replace({ name: 'status' })
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <AuthCard title="Buat password baru" subtitle="Password akun sementara harus diganti sebelum melanjutkan.">
    <form class="mt-4 space-y-4" novalidate @submit.prevent="submit">
      <p
        v-if="error"
        id="password-error"
        role="alert"
        class="rounded-card bg-danger-bg p-3 text-sm text-danger"
      >
        {{ error }}
      </p>

      <div v-if="needsCurrent">
        <label for="current-password" class="block text-sm font-medium">Password sementara</label>
        <input
          id="current-password"
          v-model="current"
          type="password"
          autocomplete="current-password"
          class="mt-1 block min-h-11 w-full rounded-card border border-line bg-surface px-3"
        />
      </div>

      <div>
        <label for="new-password" class="block text-sm font-medium">Password baru</label>
        <input
          id="new-password"
          v-model="next"
          type="password"
          autocomplete="new-password"
          aria-describedby="new-password-hint"
          class="mt-1 block min-h-11 w-full rounded-card border border-line bg-surface px-3"
        />
        <p id="new-password-hint" class="mt-1 text-sm text-ink-muted">Minimal 8 karakter.</p>
      </div>

      <div>
        <label for="confirm-password" class="block text-sm font-medium">Ulangi password baru</label>
        <input
          id="confirm-password"
          v-model="confirm"
          type="password"
          autocomplete="new-password"
          class="mt-1 block min-h-11 w-full rounded-card border border-line bg-surface px-3"
        />
      </div>

      <button
        type="submit"
        :disabled="submitting"
        class="flex min-h-11 w-full items-center justify-center rounded-card bg-accent px-4 font-medium text-white active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60"
      >
        {{ submitting ? 'Menyimpan...' : 'Simpan password' }}
      </button>
    </form>

    <button
      type="button"
      :disabled="loggingOut"
      class="mt-3 flex min-h-11 w-full items-center justify-center rounded-card px-4 text-sm font-medium text-ink-muted hover:bg-canvas active:translate-y-px disabled:opacity-60"
      @click="logout"
    >
      Keluar
    </button>
    <p v-if="logoutError" role="alert" class="mt-2 text-sm text-danger">{{ logoutError }}</p>
  </AuthCard>
</template>
