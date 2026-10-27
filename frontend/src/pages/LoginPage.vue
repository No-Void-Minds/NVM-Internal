<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AuthCard from '../components/AuthCard.vue'
import { ApiError } from '../api/client'
import { useSessionStore } from '../stores/session'
import { safeRedirect } from '../router/redirect'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()

const username = ref('')
const password = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

const sessionEnded = computed(() => route.query.alasan === 'sesi-berakhir' && !error.value)
const timeFormat = new Intl.DateTimeFormat('id-ID', { hour: '2-digit', minute: '2-digit' })

function messageFor(err: unknown): string {
  if (!(err instanceof ApiError)) return 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
  // Stitch menampilkan jam buka kunci, bukan sisa menit; jamnya dihitung dari header Retry-After.
  if (err.code === 'account_locked' && err.retryAfterSeconds) {
    const until = new Date(Date.now() + err.retryAfterSeconds * 1000)
    return `Akun dikunci sementara. Coba lagi pukul ${timeFormat.format(until)}.`
  }
  return err.message
}

async function submit() {
  if (submitting.value) return
  error.value = null
  if (!username.value.trim() || !password.value) {
    error.value = 'Username dan password wajib diisi.'
    return
  }

  submitting.value = true
  try {
    await session.login(username.value.trim(), password.value)
    if (session.user?.must_change_password) {
      await router.replace({ name: 'force-password' })
    } else {
      await router.replace(safeRedirect(route.query.kembali) ?? { name: 'status' })
    }
  } catch (err) {
    error.value = messageFor(err)
  } finally {
    password.value = ''
    submitting.value = false
  }
}
</script>

<template>
  <AuthCard title="Masuk" subtitle="Gunakan akun yang dibuat Admin atau Manajer.">
    <p v-if="sessionEnded" role="status" class="mt-4 rounded-card bg-canvas p-3 text-sm">
      Sesi kamu sudah berakhir. Silakan masuk lagi.
    </p>

    <form class="mt-4 space-y-4" novalidate @submit.prevent="submit">
      <p v-if="error" id="login-error" role="alert" class="rounded-card bg-danger-bg p-3 text-sm text-danger">
        {{ error }}
      </p>

      <div>
        <label for="username" class="block text-sm font-medium">Username</label>
        <input
          id="username"
          v-model="username"
          name="username"
          type="text"
          autocomplete="username"
          autocapitalize="none"
          spellcheck="false"
          autofocus
          :aria-describedby="error ? 'login-error' : undefined"
          class="mt-1 block min-h-11 w-full rounded-card border border-line bg-surface px-3"
        />
      </div>

      <div>
        <label for="password" class="block text-sm font-medium">Password</label>
        <input
          id="password"
          v-model="password"
          name="password"
          type="password"
          autocomplete="current-password"
          :aria-describedby="error ? 'login-error' : undefined"
          class="mt-1 block min-h-11 w-full rounded-card border border-line bg-surface px-3"
        />
      </div>

      <button
        type="submit"
        :disabled="submitting"
        class="flex min-h-11 w-full items-center justify-center rounded-card bg-accent px-4 font-medium text-white active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60"
      >
        {{ submitting ? 'Memproses...' : 'Masuk' }}
      </button>
    </form>
  </AuthCard>
</template>
