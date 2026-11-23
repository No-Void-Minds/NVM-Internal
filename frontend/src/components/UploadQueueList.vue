<script setup lang="ts">
import type { QueueItem } from '../composables/useUploadQueue'

// Daftar antrean unggah (progres, galat, coba lagi), dipakai lampiran task dan lampiran sub task (revisi 36).
// announce dibacakan pembaca layar saat unggah selesai atau gagal.
defineProps<{ queue: QueueItem[]; announce: string }>()
const emit = defineEmits<{ abort: []; retry: [key: number]; dismiss: [key: number] }>()

const button =
  'inline-flex min-h-11 items-center rounded-card border border-line px-3 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
</script>

<template>
  <p class="sr-only" aria-live="polite">{{ announce }}</p>
  <ul v-if="queue.length" class="mt-2 space-y-2">
    <li v-for="q in queue" :key="q.key" class="rounded-card border border-line p-3 text-sm">
      <div class="flex flex-wrap items-center gap-2">
        <p class="min-w-0 flex-1 truncate font-medium" :title="q.file.name">{{ q.file.name }}</p>
        <button v-if="q.status === 'uploading'" type="button" :class="button" @click="emit('abort')">
          Batal
        </button>
        <template v-else-if="q.status === 'failed'">
          <button v-if="q.retryable" type="button" :class="button" @click="emit('retry', q.key)">
            Coba lagi
          </button>
          <button type="button" :class="button" @click="emit('dismiss', q.key)">Tutup</button>
        </template>
      </div>
      <p v-if="q.status === 'failed'" class="mt-1 text-danger">{{ q.error }}</p>
      <template v-else>
        <progress
          class="mt-2 h-2 w-full accent-mark"
          :value="q.progress"
          max="1"
          :aria-label="`Progres unggah ${q.file.name}`"
        />
        <p class="mt-1 text-ink-muted">
          {{ q.status === 'uploading' ? `Mengunggah ${Math.round(q.progress * 100)}%` : 'Menunggu giliran' }}
        </p>
      </template>
    </li>
  </ul>
</template>
