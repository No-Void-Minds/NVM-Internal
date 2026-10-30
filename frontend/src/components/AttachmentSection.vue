<script setup lang="ts">
import { computed, ref } from 'vue'
import BaseDialog from './BaseDialog.vue'
import { ApiError } from '../api/client'
import {
  attachmentUrl,
  deleteAttachment,
  uploadAttachment,
  type AttachmentKind,
  type TaskAttachment,
  type UploadHandle,
} from '../api/attachments'
import FileDropzone from './FileDropzone.vue'
import { formatBytes, isImage, precheck } from '../utils/attachment'
import { formatDateTime } from '../utils/format'

const props = defineProps<{
  taskId: string
  kind: AttachmentKind
  items: TaskAttachment[]
  canUpload: boolean
  canDelete: boolean
}>()
const emit = defineEmits<{ changed: [message: string] }>()

const kindLabel = computed(() => (props.kind === 'brief' ? 'lampiran brief' : 'berkas deliverable'))

function messageOf(err: unknown): string {
  return err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
}

type QueueItem = {
  key: number
  file: File
  status: 'waiting' | 'uploading' | 'failed'
  progress: number
  error: string | null
}

const queue = ref<QueueItem[]>([])
const uploading = ref(false)
let nextKey = 0
let current: UploadHandle | null = null

function addFiles(files: File[]) {
  for (const file of files) {
    const error = precheck(file)
    queue.value.push({ key: nextKey++, file, status: error ? 'failed' : 'waiting', progress: 0, error })
  }
  void runQueue()
}

// Satu berkas per waktu: progresnya jelas dan Wi-Fi kantor tidak dibanjiri unggahan paralel.
async function runQueue() {
  if (uploading.value) return
  uploading.value = true
  let done = 0
  try {
    for (let item = nextWaiting(); item; item = nextWaiting()) {
      const it = item
      it.status = 'uploading'
      try {
        current = uploadAttachment(props.taskId, props.kind, it.file, (p) => (it.progress = p))
        await current.promise
        done++
        dismiss(it.key)
      } catch (err) {
        if (err instanceof DOMException && err.name === 'AbortError') {
          dismiss(it.key)
          continue
        }
        it.status = 'failed'
        it.error = `${it.file.name}: ${messageOf(err)}`
      }
    }
  } finally {
    uploading.value = false
    current = null
  }
  if (done) emit('changed', done === 1 ? '1 berkas diunggah.' : `${done} berkas diunggah.`)
}

function nextWaiting(): QueueItem | undefined {
  return queue.value.find((q) => q.status === 'waiting')
}

function dismiss(key: number) {
  queue.value = queue.value.filter((q) => q.key !== key)
}

const toDelete = ref<TaskAttachment | null>(null)
const deleting = ref(false)
const deleteError = ref<string | null>(null)

function openDelete(a: TaskAttachment) {
  deleteError.value = null
  toDelete.value = a
}

async function confirmDelete() {
  const a = toDelete.value
  if (!a || deleting.value) return
  deleting.value = true
  deleteError.value = null
  try {
    await deleteAttachment(props.taskId, a.id)
    toDelete.value = null
    emit('changed', `${a.name} dihapus.`)
  } catch (err) {
    deleteError.value = messageOf(err)
  } finally {
    deleting.value = false
  }
}

const iconButton =
  'inline-flex size-11 shrink-0 items-center justify-center rounded-card hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent active:translate-y-px'
const buttonSecondary =
  'inline-flex min-h-11 items-center rounded-card border border-line px-4 font-medium hover:bg-canvas active:translate-y-px disabled:opacity-60'
</script>

<template>
  <div>
    <ul v-if="items.length" class="mt-2 divide-y divide-line rounded-card border border-line">
      <li v-for="a in items" :key="a.id" class="flex items-center gap-3 py-1 pr-1 pl-3">
        <svg
          v-if="isImage(a.mime_type)"
          aria-hidden="true"
          viewBox="0 0 24 24"
          class="size-5 shrink-0 text-ink-muted"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <rect x="3" y="3" width="18" height="18" rx="2" />
          <circle cx="9" cy="9" r="2" />
          <path d="m21 15-5-5L5 21" />
        </svg>
        <svg
          v-else
          aria-hidden="true"
          viewBox="0 0 24 24"
          class="size-5 shrink-0 text-ink-muted"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z" />
          <path d="M14 3v5h5" />
        </svg>
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-medium" :title="a.name">{{ a.name }}</p>
          <p class="text-sm text-ink-muted">
            {{ formatBytes(a.size_bytes) }} · {{ a.uploaded_by }}, {{ formatDateTime(a.uploaded_at) }}
          </p>
        </div>
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
            <path d="M12 4v11" />
            <path d="m7 10 5 5 5-5" />
            <path d="M5 20h14" />
          </svg>
        </a>
        <button
          v-if="canDelete"
          type="button"
          :class="iconButton"
          class="text-urgent"
          :aria-label="`Hapus ${a.name}`"
          @click="openDelete(a)"
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
            <path d="M4 7h16" />
            <path d="M10 11v6M14 11v6" />
            <path d="M6 7l1 13h10l1-13" />
            <path d="M9 7V4h6v3" />
          </svg>
        </button>
      </li>
    </ul>
    <p v-else-if="!canUpload" class="mt-2 text-sm text-ink-muted">Belum ada {{ kindLabel }}.</p>

    <FileDropzone v-if="canUpload" class="mt-2" @files="addFiles" />

    <ul v-if="queue.length" class="mt-2 space-y-2">
      <li v-for="q in queue" :key="q.key" class="rounded-card border border-line p-3 text-sm">
        <div class="flex items-center gap-2">
          <p class="min-w-0 flex-1 truncate font-medium">{{ q.file.name }}</p>
          <button
            v-if="q.status === 'uploading'"
            type="button"
            :class="buttonSecondary"
            @click="current?.abort()"
          >
            Batal
          </button>
          <button
            v-else-if="q.status === 'failed'"
            type="button"
            :class="buttonSecondary"
            @click="dismiss(q.key)"
          >
            Tutup
          </button>
        </div>
        <p v-if="q.status === 'failed'" role="alert" class="mt-1 text-danger">{{ q.error }}</p>
        <template v-else>
          <progress
            class="mt-2 h-2 w-full accent-accent"
            :value="q.progress"
            max="1"
            :aria-label="`Progres unggah ${q.file.name}`"
          />
          <p class="mt-1 text-ink-muted">
            {{
              q.status === 'uploading' ? `Mengunggah ${Math.round(q.progress * 100)}%` : 'Menunggu giliran'
            }}
          </p>
        </template>
      </li>
    </ul>

    <BaseDialog :open="toDelete !== null" :title="`Hapus ${toDelete?.name ?? ''}?`" @close="toDelete = null">
      <p v-if="deleteError" role="alert" class="mb-3 rounded-card bg-danger-bg p-3 text-sm text-danger">
        {{ deleteError }}
      </p>
      <p>
        Berkas ini hilang dari task dan tidak bisa dipulihkan dari aplikasi. Jejaknya tetap ada di riwayat
        task.
      </p>
      <template #footer>
        <button type="button" :class="buttonSecondary" @click="toDelete = null">Batal</button>
        <button
          type="button"
          :disabled="deleting"
          class="inline-flex min-h-11 items-center rounded-card bg-urgent px-4 font-medium text-white active:translate-y-px disabled:opacity-60"
          @click="confirmDelete"
        >
          {{ deleting ? 'Menghapus...' : 'Hapus berkas' }}
        </button>
      </template>
    </BaseDialog>
  </div>
</template>
