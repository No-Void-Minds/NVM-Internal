<script setup lang="ts">
import { ref } from 'vue'
import { ApiError } from '../api/client'
import { useSessionStore } from '../stores/session'
import { useLogout } from '../composables/useLogout'
import { roleLabel } from '../utils/roles'
import { validateNewPassword } from '../utils/password'

const session = useSessionStore()
const { logout, loggingOut, logoutError } = useLogout()

const current = ref('')
const next = ref('')
const confirm = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)
const saved = ref(false)

async function submit() {
  if (submitting.value) return
  saved.value = false
  error.value = validateNewPassword(current.value, next.value, confirm.value)
  if (error.value) return

  submitting.value = true
  try {
    await session.changePassword(current.value, next.value)
    current.value = ''
    next.value = ''
    confirm.value = ''
    saved.value = true
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <section v-if="session.user" class="max-w-xl">
    <h1 class="text-2xl font-semibold">Profil</h1>

    <dl class="mt-6 grid grid-cols-[auto_1fr] gap-x-6 gap-y-3 rounded-card border border-line bg-surface p-4">
      <dt class="text-ink-muted">Nama</dt>
      <dd>{{ session.user.name }}</dd>
      <dt class="text-ink-muted">Username</dt>
      <dd class="min-w-0 break-words">{{ session.user.username }}</dd>
      <dt class="text-ink-muted">Peran</dt>
      <dd>{{ roleLabel(session.user.role) }}</dd>
      <dt class="text-ink-muted">Divisi</dt>
      <dd>{{ session.user.division_name }}</dd>
    </dl>

    <h2 class="mt-8 text-lg font-semibold">Ganti password</h2>
    <p class="mt-1 text-sm text-ink-muted">Setelah diganti, sesi di perangkat lain otomatis berakhir.</p>

    <form class="mt-4 space-y-4" novalidate @submit.prevent="submit">
      <p v-if="error" role="alert" class="rounded-card bg-danger-bg p-3 text-sm text-danger">{{ error }}</p>
      <p
        v-if="saved"
        role="status"
        class="rounded-card border border-line bg-surface p-3 text-sm text-progress-fg"
      >
        Password berhasil diganti.
      </p>

      <div>
        <label for="profile-current" class="block text-sm font-medium">Password lama</label>
        <input
          id="profile-current"
          v-model="current"
          type="password"
          autocomplete="current-password"
          class="mt-1 block min-h-11 w-full rounded-card border border-line bg-surface px-3"
        />
      </div>
      <div>
        <label for="profile-new" class="block text-sm font-medium">Password baru</label>
        <input
          id="profile-new"
          v-model="next"
          type="password"
          autocomplete="new-password"
          aria-describedby="profile-new-hint"
          class="mt-1 block min-h-11 w-full rounded-card border border-line bg-surface px-3"
        />
        <p id="profile-new-hint" class="mt-1 text-sm text-ink-muted">Minimal 8 karakter.</p>
      </div>
      <div>
        <label for="profile-confirm" class="block text-sm font-medium">Ulangi password baru</label>
        <input
          id="profile-confirm"
          v-model="confirm"
          type="password"
          autocomplete="new-password"
          class="mt-1 block min-h-11 w-full rounded-card border border-line bg-surface px-3"
        />
      </div>

      <button
        type="submit"
        :disabled="submitting"
        class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-white active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60"
      >
        {{ submitting ? 'Menyimpan...' : 'Simpan password' }}
      </button>
    </form>

    <!-- Di layar lebar tombol Keluar ada di header; di ponsel header tidak memuatnya. -->
    <div class="mt-10 border-t border-line pt-6 md:hidden">
      <button
        type="button"
        :disabled="loggingOut"
        class="inline-flex min-h-11 items-center rounded-card border border-line bg-surface px-4 font-medium active:translate-y-px disabled:opacity-60"
        @click="logout"
      >
        Keluar
      </button>
      <p v-if="logoutError" role="alert" class="mt-2 text-sm text-danger">{{ logoutError }}</p>
    </div>
  </section>
</template>
