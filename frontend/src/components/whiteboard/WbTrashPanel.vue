<script setup lang="ts">
import { ref, shallowRef, watch } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import { ApiError } from '../../api/client'
import { listTrash, restoreBatch, type TrashBatch } from '../../api/whiteboard'
import { formatDateTime } from '../../utils/format'

// Tempat sampah Whiteboard sebagai panel di samping kanvas: satu baris per aksi hapus. Pulihkan mengembalikan
// seluruh batch; hapus permanen hanya oleh job harian setelah 30 hari, jadi tidak ada tombolnya.
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: []; restored: [boardId: string] }>()

const batches = shallowRef<TrashBatch[]>([])
const days = ref(30)
const canWrite = ref(false)
const state = ref<'loading' | 'error' | 'ok'>('loading')
const loadError = ref('')
const rowError = ref<Record<string, string>>({})
const busy = ref('')

async function load() {
  state.value = 'loading'
  try {
    const res = await listTrash()
    batches.value = res.batches
    days.value = res.days
    canWrite.value = res.can_write
    state.value = 'ok'
  } catch (e) {
    loadError.value = e instanceof ApiError ? e.message : 'Tempat sampah gagal dimuat. Coba lagi.'
    state.value = 'error'
  }
}
watch(
  () => props.open,
  (on) => on && load(),
  { immediate: true },
)

function heading(b: TrashBatch) {
  if (b.boards.length) {
    const rest = b.boards.length - 1
    return `Papan "${b.boards[0]}"${rest > 0 ? ` beserta ${rest} papan di dalamnya` : ''}`
  }
  // Batch tanpa kartu berisi garis penghubung saja (garis yang dihapus sendiri atau terputus saat kartu dipindah).
  return b.cards === 0 && b.connectors ? `${b.connectors} garis` : `${b.cards} kartu`
}

function detail(b: TrashBatch) {
  const parts: string[] = []
  if (b.boards.length) parts.push(`${b.cards} kartu di dalamnya`)
  else if (b.excerpts.length) parts.push(b.excerpts.join(', '))
  if (b.cards > 0 && b.connectors) parts.push(`${b.connectors} garis`)
  parts.push(`dari papan "${b.board_title}"`)
  return parts.join(', ')
}

async function restore(b: TrashBatch) {
  busy.value = b.batch
  rowError.value = { ...rowError.value, [b.batch]: '' }
  try {
    const res = (await restoreBatch(b.batch)).restored
    emit('restored', res.board_id)
    await load()
  } catch (e) {
    rowError.value = {
      ...rowError.value,
      [b.batch]: e instanceof ApiError ? e.message : 'Gagal memulihkan. Coba lagi.',
    }
  } finally {
    busy.value = ''
  }
}

const btn =
  'inline-flex min-h-11 items-center justify-center rounded-card border border-line bg-surface px-4 font-medium hover:bg-canvas active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
</script>

<template>
  <BaseDialog :open="open" title="Tempat sampah" variant="panel" @close="emit('close')">
    <div class="min-h-0 flex-1 overflow-y-auto px-5 py-4">
      <p class="mb-4 text-sm text-ink-muted">
        Isi yang dihapus bisa dipulihkan siapa saja selama {{ days }} hari, lalu dihapus permanen otomatis.
      </p>
      <p v-if="state === 'loading'" role="status" class="text-ink-muted">Memuat...</p>
      <div v-else-if="state === 'error'" role="alert">
        <p>{{ loadError }}</p>
        <button type="button" :class="btn" class="mt-3" @click="load">Coba lagi</button>
      </div>
      <p v-else-if="batches.length === 0" class="text-ink-muted">Tempat sampah kosong.</p>
      <ul v-else class="flex flex-col gap-3">
        <li
          v-for="b in batches"
          :key="b.batch"
          class="flex flex-col gap-2 border-b border-line pb-3 last:border-b-0"
        >
          <div class="min-w-0">
            <p class="font-semibold break-words">{{ heading(b) }}</p>
            <p class="text-sm break-words text-ink-muted">{{ detail(b) }}</p>
            <p class="text-sm text-ink-muted">
              Dihapus {{ b.deleted_by_name }}, {{ formatDateTime(b.deleted_at) }}. Hapus permanen
              {{ formatDateTime(b.purge_at) }}.
            </p>
            <p v-if="b.blocked_by" class="mt-1 text-sm">
              Pulihkan dulu papan "{{ b.blocked_by }}", karena isi ini berada di dalamnya.
            </p>
            <p v-if="rowError[b.batch]" role="alert" class="mt-1 text-sm text-danger">
              {{ rowError[b.batch] }}
            </p>
          </div>
          <button
            v-if="canWrite"
            type="button"
            :class="btn"
            class="self-start"
            :disabled="busy === b.batch"
            :aria-label="`Pulihkan ${heading(b)}`"
            @click="restore(b)"
          >
            Pulihkan
          </button>
        </li>
      </ul>
    </div>
  </BaseDialog>
</template>
