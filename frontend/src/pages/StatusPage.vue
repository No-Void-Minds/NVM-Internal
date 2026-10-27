<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { ApiError, apiGet } from '../api/client'

type Health = { status: string; database: string }
type CheckState =
  | { kind: 'loading' }
  | { kind: 'ok'; checkedAt: Date }
  | { kind: 'error'; message: string; code: string }

const state = ref<CheckState>({ kind: 'loading' })
const timeFormat = new Intl.DateTimeFormat('id-ID', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
let controller: AbortController | null = null

async function check() {
  // Batalkan pemeriksaan sebelumnya supaya hasil lama tidak menimpa hasil baru.
  controller?.abort()
  controller = new AbortController()
  state.value = { kind: 'loading' }

  try {
    await apiGet<Health>('/health', controller.signal)
    state.value = { kind: 'ok', checkedAt: new Date() }
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') return
    state.value =
      err instanceof ApiError
        ? { kind: 'error', message: err.message, code: err.code }
        : { kind: 'error', message: 'Terjadi kesalahan yang tidak terduga.', code: 'unknown' }
  }
}

onMounted(check)
onUnmounted(() => controller?.abort())
</script>

<template>
  <section class="max-w-xl">
    <h1 class="text-2xl font-semibold">Status sistem</h1>
    <p class="mt-1 text-ink-muted">Memeriksa apakah server aplikasi dan database bisa dihubungi.</p>

    <div aria-live="polite" class="mt-6">
      <div v-if="state.kind === 'loading'" class="rounded-card border border-line bg-surface p-4">
        <p class="text-ink-muted">Memeriksa koneksi...</p>
      </div>

      <div v-else-if="state.kind === 'ok'" class="rounded-card border border-line bg-surface p-4">
        <p class="font-medium text-progress-fg">Server dan database terhubung.</p>
        <p class="mt-1 text-sm text-ink-muted">Diperiksa pukul {{ timeFormat.format(state.checkedAt) }}</p>
      </div>

      <div v-else class="rounded-card bg-danger-bg p-4 text-danger">
        <p class="font-medium">{{ state.message }}</p>
        <p class="mt-1 text-sm">Kode: {{ state.code }}</p>
      </div>
    </div>

    <button
      type="button"
      class="mt-4 inline-flex min-h-11 items-center rounded-card border border-line bg-surface px-4 font-medium hover:bg-canvas active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60"
      :disabled="state.kind === 'loading'"
      @click="check"
    >
      Periksa lagi
    </button>
  </section>
</template>