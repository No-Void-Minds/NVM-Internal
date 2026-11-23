<script setup lang="ts">
import { ref } from 'vue'
import { ApiError } from '../api/client'
import { attachmentUrl, deleteAttachment, uploadAttachment, type TaskAttachment } from '../api/attachments'
import AttachmentPreview from './AttachmentPreview.vue'
import UploadQueueList from './UploadQueueList.vue'
import { useUploadQueue } from '../composables/useUploadQueue'
import { acceptAttr, formatBytes, isImage, isPdf } from '../utils/attachment'

// Lampiran sub task sengaja lebih ringkas daripada AttachmentSection milik task induk: di dalam kartu
// sub task tidak ada ruang untuk dropzone besar, dan isinya biasanya satu atau dua berkas. Tempel dan seret
// ditangani FileDropTarget di bagian induknya (revisi 36), yang menyerahkan berkasnya lewat add.
const props = defineProps<{
  taskId: string
  subtaskId: number
  items: TaskAttachment[]
  canManage: boolean
  // brief: berkas acuan, deliverable: berkas hasil (revisi 18).
  kind: 'brief' | 'deliverable'
  // Nama kelompok untuk label tombol, supaya dua tombol Tambah berkas di satu sub task bisa dibedakan.
  group: string
}>()
const emit = defineEmits<{ changed: [message: string] }>()

const preview = ref<TaskAttachment | null>(null)

function canPreview(a: TaskAttachment): boolean {
  return isImage(a.mime_type) || isPdf(a.mime_type)
}

const picker = ref<HTMLInputElement | null>(null)
const busy = ref(false)
const error = ref('')

function messageOf(err: unknown): string {
  return err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
}

const { queue, uploading, announce, add, dismiss, retry, abort } = useUploadQueue(
  (file, onProgress) => uploadAttachment(props.taskId, props.kind, file, onProgress, props.subtaskId),
  (msg) => emit('changed', msg),
)
defineExpose({ add })

function onPick(e: Event) {
  const input = e.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  // Dikosongkan supaya memilih berkas yang sama lagi (misalnya setelah gagal) tetap memicu change.
  input.value = ''
  if (files.length) add(files)
}

async function remove(a: TaskAttachment) {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    await deleteAttachment(props.taskId, a.id)
    emit('changed', `${a.name} dihapus.`)
  } catch (err) {
    error.value = messageOf(err)
  } finally {
    busy.value = false
  }
}

const iconButton =
  'inline-flex size-11 shrink-0 items-center justify-center rounded-card text-ink-muted hover:bg-canvas hover:text-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
</script>

<template>
  <div>
    <ul v-if="items.length" class="space-y-1">
      <li
        v-for="a in items"
        :key="a.id"
        class="flex items-center gap-2 rounded-card border border-line bg-surface py-1 pr-1 pl-3"
      >
        <span class="min-w-0 flex-1 truncate text-sm" :title="a.name">{{ a.name }}</span>
        <span class="shrink-0 text-sm text-ink-muted">{{ formatBytes(a.size_bytes) }}</span>
        <button
          v-if="canPreview(a)"
          type="button"
          :class="iconButton"
          :aria-label="`Pratinjau ${a.name}`"
          @click="preview = a"
        >
          <svg
            aria-hidden="true"
            viewBox="0 0 24 24"
            class="size-5"
            fill="none"
            stroke="currentColor"
            stroke-width="1.75"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6-10-6-10-6Z" />
            <circle cx="12" cy="12" r="3" />
          </svg>
        </button>
        <a :href="attachmentUrl(taskId, a.id)" :class="iconButton" :aria-label="`Unduh ${a.name}`">
          <svg
            aria-hidden="true"
            viewBox="0 0 24 24"
            class="size-5"
            fill="none"
            stroke="currentColor"
            stroke-width="1.75"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M12 4v12M7 11l5 5 5-5M5 20h14" />
          </svg>
        </a>
        <button
          v-if="canManage"
          type="button"
          :class="iconButton"
          class="hover:text-urgent"
          :disabled="busy"
          :aria-label="`Hapus ${a.name}`"
          @click="remove(a)"
        >
          <svg
            aria-hidden="true"
            viewBox="0 0 24 24"
            class="size-5"
            fill="none"
            stroke="currentColor"
            stroke-width="1.75"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M5 7h14M10 7V5h4v2M7 7l1 12h8l1-12" />
          </svg>
        </button>
      </li>
    </ul>

    <div v-if="canManage" class="relative mt-2">
      <input
        ref="picker"
        type="file"
        class="sr-only"
        tabindex="-1"
        aria-hidden="true"
        multiple
        :accept="acceptAttr"
        @change="onPick"
      />
      <button
        type="button"
        class="inline-flex min-h-11 items-center rounded-card border border-line px-3 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
        :aria-label="`Tambah berkas ${group}`"
        @click="picker?.click()"
      >
        {{ uploading ? 'Tambah berkas lagi' : 'Tambah berkas' }}
      </button>
      <!-- Ctrl+V tidak ada di layar sentuh, jadi petunjuknya hanya untuk perangkat dengan mouse. -->
      <p class="mt-1 hidden text-sm text-ink-muted pointer-fine:block">
        Atau tempel (Ctrl+V) atau seret berkas ke sini.
      </p>
    </div>
    <UploadQueueList :queue="queue" :announce="announce" @abort="abort" @retry="retry" @dismiss="dismiss" />

    <p v-if="error" role="alert" class="mt-2 rounded-card bg-danger-bg p-2 text-sm text-danger">
      {{ error }}
    </p>

    <AttachmentPreview :task-id="taskId" :item="preview" @close="preview = null" />
  </div>
</template>
