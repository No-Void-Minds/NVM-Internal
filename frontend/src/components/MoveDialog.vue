<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import BaseDialog from './BaseDialog.vue'
import { ApiError } from '../api/client'
import { moveTask } from '../api/tasks'
import type { BoardTask } from '../api/board'
import { statusLabel, type TaskStatus } from '../utils/task'

const props = defineProps<{ task: BoardTask | null }>()
const emit = defineEmits<{ close: []; moved: [message: string]; stale: [message: string] }>()

// Setiap tujuan diberi penjelasan, supaya pilihannya tidak hanya berupa nama status.
function describe(from: TaskStatus, to: TaskStatus): string {
  if (from === 'requested') return 'Mulai dikerjakan.'
  if (from === 'ready' && to === 'on_progress') return 'Kembalikan untuk revisi. Catatan revisi wajib diisi.'
  if (to === 'ready') return 'Kirim ke Manajer untuk ditinjau.'
  if (to === 'completed') return 'Tutup task. Solved Date terisi otomatis.'
  if (from === 'completed') return 'Buka kembali. Solved Date dikosongkan.'
  return ''
}

const target = ref<TaskStatus | null>(null)
const note = ref('')
const error = ref<string | null>(null)
const busy = ref(false)

const isRevision = computed(() => props.task?.status === 'ready' && target.value === 'on_progress')

watch(
  () => props.task,
  (t) => {
    // Kalau tujuannya hanya satu, langsung terpilih supaya cukup satu ketukan lagi.
    target.value = t && t.move_to.length === 1 ? t.move_to[0] : null
    note.value = ''
    error.value = null
  },
)

async function submit() {
  const t = props.task
  if (!t || busy.value) return
  if (!target.value) {
    error.value = 'Pilih status tujuan.'
    return
  }
  if (isRevision.value && !note.value.trim()) {
    error.value = 'Tulis catatan revisi supaya PIC tahu apa yang perlu diperbaiki.'
    return
  }
  busy.value = true
  try {
    await moveTask(t.id, t.version, target.value, isRevision.value ? note.value.trim() : '')
    emit('moved', `${t.id} dipindahkan ke ${statusLabel[target.value]}.`)
  } catch (err) {
    // Kartu di layar sudah basi: dialog ditutup dan papan dimuat ulang, karena mencoba lagi dengan
    // versi lama pasti ditolak lagi (AC-10).
    if (err instanceof ApiError && err.status === 409) {
      emit(
        'stale',
        `${t.id} sudah diubah orang lain. Papan dimuat ulang, cek statusnya sebelum memindahkan lagi.`,
      )
      return
    }
    if (err instanceof ApiError && err.status === 403) {
      emit('stale', `Kamu tidak lagi punya hak memindahkan ${t.id}. Papan dimuat ulang.`)
      return
    }
    if (err instanceof ApiError && err.status === 404) {
      emit('stale', `${t.id} sudah dihapus. Papan dimuat ulang.`)
      return
    }
    error.value = err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
  } finally {
    busy.value = false
  }
}

const buttonSecondary =
  'inline-flex min-h-11 items-center rounded-card border border-line px-4 font-medium hover:bg-canvas active:translate-y-px'
</script>

<template>
  <BaseDialog :open="task !== null" :title="task ? `Pindahkan ${task.id}` : ''" @close="emit('close')">
    <form v-if="task" id="move-form" class="space-y-4" novalidate @submit.prevent="submit">
      <p class="text-sm text-ink-muted">Status saat ini: {{ statusLabel[task.status] }}</p>
      <p v-if="error" role="alert" class="rounded-card bg-danger-bg p-3 text-sm text-danger">{{ error }}</p>

      <fieldset>
        <legend class="text-sm font-medium">Pilih status tujuan</legend>
        <div class="mt-1 space-y-2">
          <label
            v-for="to in task.move_to"
            :key="to"
            class="flex min-h-11 cursor-pointer items-start gap-3 rounded-card border border-line p-3 has-[:checked]:border-accent has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-accent"
          >
            <input
              v-model="target"
              type="radio"
              name="move-target"
              :value="to"
              class="mt-1 size-4 accent-accent"
            />
            <span>
              <span class="block font-medium">{{ statusLabel[to] }}</span>
              <span class="block text-sm text-ink-muted">{{ describe(task.status, to) }}</span>
            </span>
          </label>
        </div>
      </fieldset>

      <div v-if="isRevision">
        <label for="move-note" class="block text-sm font-medium">Catatan revisi</label>
        <p id="move-note-hint" class="text-sm text-ink-muted">
          {{ task.id }} akan kembali ke On Progress. Catatan revisi tersimpan di riwayat task.
        </p>
        <textarea
          id="move-note"
          v-model="note"
          rows="3"
          maxlength="500"
          required
          aria-describedby="move-note-hint"
          class="mt-1 block w-full rounded-card border border-line bg-surface p-3"
        />
      </div>
    </form>
    <template #footer>
      <button type="button" :class="buttonSecondary" @click="emit('close')">Batal</button>
      <button
        type="submit"
        form="move-form"
        :disabled="busy"
        class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-white active:translate-y-px disabled:opacity-60"
      >
        {{ busy ? 'Memindahkan...' : isRevision ? 'Kembalikan ke On Progress' : 'Pindahkan' }}
      </button>
    </template>
  </BaseDialog>
</template>
