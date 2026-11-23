<script setup lang="ts">
import PersonChip from './PersonChip.vue'
import { computed, ref } from 'vue'
import AttachmentPreview from './AttachmentPreview.vue'
import BaseDialog from './BaseDialog.vue'
import { ApiError } from '../api/client'
import {
  attachmentUrl,
  deleteAttachment,
  uploadAttachment,
  type AttachmentKind,
  type TaskAttachment,
} from '../api/attachments'
import FileDropzone from './FileDropzone.vue'
import UploadQueueList from './UploadQueueList.vue'
import { useUploadQueue } from '../composables/useUploadQueue'
import { formatBytes, isImage, isPdf } from '../utils/attachment'
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

// Antrean bersama dengan lampiran sub task (revisi 36). add diekspos supaya tempelan Ctrl+V di bagian induknya
// (FileDropTarget) masuk ke antrean yang sama.
const {
  queue,
  announce,
  add: addFiles,
  dismiss,
  retry,
  abort,
} = useUploadQueue(
  (file, onProgress) => uploadAttachment(props.taskId, props.kind, file, onProgress),
  (msg) => emit('changed', msg),
)
defineExpose({ add: addFiles })

// Pratinjau hanya untuk gambar dan PDF; isinya ditangani AttachmentPreview.
function canPreview(a: TaskAttachment): boolean {
  return isImage(a.mime_type) || isPdf(a.mime_type)
}
const preview = ref<TaskAttachment | null>(null)

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
  'inline-flex size-11 shrink-0 items-center justify-center rounded-card hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark active:translate-y-px'
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
            {{ formatBytes(a.size_bytes) }} ·
            <PersonChip :user-id="a.uploaded_by_id" :name="a.uploaded_by">{{ a.uploaded_by }}</PersonChip
            >, {{ formatDateTime(a.uploaded_at) }}
          </p>
        </div>
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
            <path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6-10-6-10-6" />
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
    <!-- Ctrl+V tidak ada di layar sentuh, jadi petunjuknya hanya untuk perangkat dengan mouse. -->
    <p v-if="canUpload" class="mt-1 hidden text-sm text-ink-muted pointer-fine:block">
      Atau tempel (Ctrl+V) atau seret berkas ke sini.
    </p>

    <UploadQueueList :queue="queue" :announce="announce" @abort="abort" @retry="retry" @dismiss="dismiss" />

    <AttachmentPreview :task-id="taskId" :item="preview" @close="preview = null" />

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
          class="inline-flex min-h-11 items-center rounded-card bg-urgent px-4 font-medium text-on-urgent active:translate-y-px disabled:opacity-60"
          @click="confirmDelete"
        >
          {{ deleting ? 'Menghapus...' : 'Hapus berkas' }}
        </button>
      </template>
    </BaseDialog>
  </div>
</template>
