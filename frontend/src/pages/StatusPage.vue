<script setup lang="ts">
import PageHeader from '../components/PageHeader.vue'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ApiError, apiGet } from '../api/client'
import { formatDateTime } from '../utils/format'

type Health = { status: string; database: string }
// Salinan Drive terakhir (revisi 25). never berarti job-nya belum pernah jalan.
type DriveStatus = {
  never: boolean
  finished_at: string | null
  ok: boolean
  files: number
  bytes: number
  message: string
}
type CheckState =
  { kind: 'loading' } | { kind: 'ok'; checkedAt: Date } | { kind: 'error'; message: string; code: string }

const state = ref<CheckState>({ kind: 'loading' })
const drive = ref<DriveStatus | null>(null)
const driveError = ref('')
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

// Status salinan Drive dimuat sekali; gagal memuatnya tidak menutupi hasil pemeriksaan koneksi.
async function loadDrive() {
  driveError.value = ''
  try {
    drive.value = await apiGet<DriveStatus>('/drive-export/status')
  } catch (err) {
    drive.value = null
    driveError.value = err instanceof ApiError ? err.message : 'Status salinan Drive tidak bisa dibaca.'
  }
}

const driveLine = computed(() => {
  const d = drive.value
  if (!d) return ''
  if (d.never) return 'Belum pernah jalan.'
  const waktu = d.finished_at ? formatDateTime(d.finished_at) : 'waktu tidak tercatat'
  const hasil = d.ok ? 'OK' : `galat: ${d.message || 'tanpa pesan'}`
  return `${waktu}, ${hasil}, ${d.files} berkas`
})

onMounted(check)
onMounted(loadDrive)
onUnmounted(() => controller?.abort())
</script>

<template>
  <section class="max-w-xl">
    <PageHeader
      title="Status sistem"
      description="Memeriksa apakah server aplikasi dan database bisa dihubungi."
    />

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

    <h2 class="mt-8 text-lg font-semibold">Salinan Drive terakhir</h2>
    <div class="mt-2" aria-live="polite">
      <p v-if="driveError" role="alert" class="rounded-card bg-danger-bg p-4 text-danger">{{ driveError }}</p>
      <p v-else-if="!drive" class="rounded-card border border-line bg-surface p-4 text-ink-muted">
        Memuat status salinan Drive...
      </p>
      <div v-else class="rounded-card border border-line bg-surface p-4">
        <p :class="drive.never || drive.ok ? 'font-medium' : 'font-medium text-danger'">{{ driveLine }}</p>
        <p v-if="!drive.never" class="mt-1 text-sm text-ink-muted">
          Job berjalan malam lewat cron di VM, setelah backup.
        </p>
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
