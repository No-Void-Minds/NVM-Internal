<script setup lang="ts">
import { computed, ref } from 'vue'
import AttachmentPreview from './AttachmentPreview.vue'
import BaseDialog from './BaseDialog.vue'
import ActivityThumb from './ActivityThumb.vue'
import { ApiError } from '../api/client'
import type { ActivityFile } from '../api/activities'
import { activityFileUrl, deleteActivityFile, uploadActivityFile } from '../api/activityFiles'
import { formatBytes, isImage, isPdf } from '../utils/attachment'
import { activityAcceptAttr, activityAllowedText, activityFilePrecheck, isVideo } from '../utils/activityFile'

// Lampiran satu jam (revisi 24). Dipakai tabel di papan task, tampilan kartu ponselnya, dan dialog sel di
// halaman Aktivitas, jadi aturan unggah, progres, dan hapus hanya ada di satu tempat.
const props = defineProps<{
  // 0 berarti jam milik sendiri, jadi papan task tidak perlu mengirim id akun.
  userId: number
  date: string
  // num: nomor slot jam. Bukan "slot", karena itu nama atribut bawaan Vue.
  num: number
  time: string
  files: ActivityFile[]
  canManage: boolean
  // compact: di dalam sel tabel yang sempit, hanya tombol dan deretan thumbnail.
  compact?: boolean
}>()
const emit = defineEmits<{ changed: [] }>()

type Pending = {
  key: number
  name: string
  percent: number
  error: string
  file: File
  abort: () => void
}

const picker = ref<HTMLInputElement | null>(null)
const pending = ref<Pending[]>([])
const preview = ref<ActivityFile | null>(null)
const confirming = ref<ActivityFile | null>(null)
const removing = ref('')
const listError = ref('')
let seq = 0

const busy = computed(() => pending.value.some((p) => !p.error))

function canPreview(f: ActivityFile): boolean {
  return isImage(f.mime_type) || isPdf(f.mime_type) || isVideo(f.mime_type)
}

function messageOf(err: unknown): string {
  if (err instanceof DOMException && err.name === 'AbortError') return ''
  return err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
}

// add dipakai tombol pilih berkas, drag and drop, dan paste gambar.
async function add(files: File[]) {
  if (!props.canManage) return
  for (const file of files) await start(file)
}

async function start(file: File) {
  const bad = activityFilePrecheck(file)
  if (bad) {
    pending.value.push({ key: ++seq, name: file.name, percent: 0, error: bad, file, abort: () => {} })
    return
  }
  const row: Pending = { key: ++seq, name: file.name, percent: 0, error: '', file, abort: () => {} }
  pending.value.push(row)
  const handle = uploadActivityFile(props.userId, props.date, props.num, file, (f) => {
    row.percent = Math.round(f * 100)
  })
  row.abort = handle.abort
  try {
    await handle.promise
    pending.value = pending.value.filter((p) => p.key !== row.key)
    emit('changed')
  } catch (err) {
    const message = messageOf(err)
    if (!message) {
      pending.value = pending.value.filter((p) => p.key !== row.key)
      return
    }
    row.error = message
  }
}

function retry(row: Pending) {
  pending.value = pending.value.filter((p) => p.key !== row.key)
  void start(row.file)
}

function dismiss(row: Pending) {
  row.abort()
  pending.value = pending.value.filter((p) => p.key !== row.key)
}

async function remove(f: ActivityFile) {
  confirming.value = null
  removing.value = f.id
  listError.value = ''
  try {
    await deleteActivityFile(f.id)
    emit('changed')
  } catch (err) {
    listError.value = messageOf(err)
  } finally {
    removing.value = ''
  }
}

function onPick(e: Event) {
  const input = e.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  // Dikosongkan supaya memilih berkas yang sama lagi tetap memicu change.
  input.value = ''
  void add(files)
}

defineExpose({ add })

const iconButton =
  'inline-flex size-11 shrink-0 items-center justify-center rounded-card text-ink-muted hover:bg-canvas hover:text-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
const smallButton =
  'inline-flex min-h-11 items-center gap-1 rounded-card border border-line px-2 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-60'
</script>

<template>
  <div class="min-w-0">
    <ul v-if="files.length" class="flex flex-wrap items-center gap-1">
      <li v-for="f in files" :key="f.id" class="flex min-w-0 items-center gap-1">
        <button
          v-if="canPreview(f)"
          type="button"
          class="flex min-h-11 min-w-0 items-center gap-1 rounded-card border border-line px-1 hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
          :aria-label="`Pratinjau ${f.name}`"
          @click="preview = f"
        >
          <ActivityThumb v-if="isImage(f.mime_type)" :id="f.id" :name="f.name" />
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
            <path v-if="isVideo(f.mime_type)" d="M4 6h11v12H4zM15 10l5-3v10l-5-3z" />
            <path v-else d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8zM14 3v5h5" />
          </svg>
          <span class="max-w-28 truncate text-sm" :title="f.name">{{ f.name }}</span>
        </button>
        <a
          v-else
          :href="activityFileUrl(f.id)"
          class="flex min-h-11 min-w-0 items-center gap-1 rounded-card border border-line px-2 hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
          :aria-label="`Unduh ${f.name}, ${formatBytes(f.size_bytes)}`"
        >
          <svg
            aria-hidden="true"
            viewBox="0 0 24 24"
            class="size-5 shrink-0 text-ink-muted"
            fill="none"
            stroke="currentColor"
            stroke-width="1.75"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8zM14 3v5h5" />
          </svg>
          <span class="max-w-28 truncate text-sm" :title="f.name">{{ f.name }}</span>
        </a>
        <button
          v-if="canManage"
          type="button"
          :class="iconButton"
          class="hover:text-urgent"
          :disabled="removing === f.id"
          :aria-label="`Hapus ${f.name}`"
          @click="confirming = f"
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
        <span v-if="f.uploaded_by && !compact" class="text-xs text-ink-muted">
          dari {{ f.uploaded_by }}
        </span>
      </li>
    </ul>

    <!-- Satu baris per berkas yang sedang diunggah: persentase, tombol Batal, dan alasan bila gagal. -->
    <ul v-if="pending.length" class="mt-1 space-y-1">
      <li v-for="row in pending" :key="row.key" class="text-sm">
        <div class="flex items-center gap-2">
          <span class="min-w-0 flex-1 truncate" :title="row.name">{{ row.name }}</span>
          <span v-if="!row.error" class="shrink-0 font-mono">{{ row.percent }}%</span>
          <button
            v-if="row.error"
            type="button"
            :class="smallButton"
            :aria-label="`Coba lagi unggah ${row.name}`"
            @click="retry(row)"
          >
            Coba lagi
          </button>
          <button
            type="button"
            :class="smallButton"
            :aria-label="`${row.error ? 'Tutup galat' : 'Batalkan unggahan'} ${row.name}`"
            @click="dismiss(row)"
          >
            {{ row.error ? 'Tutup' : 'Batal' }}
          </button>
        </div>
        <div
          v-if="!row.error"
          class="mt-1 h-1 w-full rounded-chip bg-progress-bg"
          role="progressbar"
          :aria-valuenow="row.percent"
          aria-valuemin="0"
          aria-valuemax="100"
          :aria-label="`Progres unggah ${row.name}`"
        >
          <span class="block h-1 rounded-chip bg-progress-fg" :style="{ width: `${row.percent}%` }"></span>
        </div>
        <p v-else role="alert" class="mt-1 text-danger">{{ row.error }}</p>
      </li>
    </ul>

    <p v-if="listError" role="alert" class="mt-1 text-sm text-danger">{{ listError }}</p>

    <div v-if="canManage" class="relative mt-1">
      <input
        ref="picker"
        type="file"
        multiple
        class="sr-only"
        tabindex="-1"
        aria-hidden="true"
        :accept="activityAcceptAttr"
        @change="onPick"
      />
      <button
        type="button"
        :class="smallButton"
        :disabled="busy"
        :title="activityAllowedText"
        :aria-label="`Tambah berkas untuk jam ${time}. ${activityAllowedText}`"
        @click="picker?.click()"
      >
        <svg
          aria-hidden="true"
          viewBox="0 0 24 24"
          class="size-4 shrink-0"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path
            d="M21 12.5 12.5 21a4.95 4.95 0 0 1-7-7l8.5-8.5a3.5 3.5 0 0 1 5 5l-8.5 8.5a2 2 0 0 1-3-3l7.5-7.5"
          />
        </svg>
        Tambah berkas
      </button>
    </div>

    <AttachmentPreview
      :item="preview"
      :src="preview ? activityFileUrl(preview.id) : undefined"
      @close="preview = null"
    />

    <!-- Hapus permanen: berkasnya ikut hilang dari disk, jadi dikonfirmasi dulu. -->
    <BaseDialog :open="confirming !== null" title="Hapus berkas" @close="confirming = null">
      <p v-if="confirming">
        Hapus <span class="font-medium">{{ confirming.name }}</span> dari jam {{ time }}? Berkasnya hilang
        permanen dan tidak bisa dipulihkan.
      </p>
      <template #footer>
        <button type="button" :class="smallButton" @click="confirming = null">Batal</button>
        <button
          v-if="confirming"
          type="button"
          class="inline-flex min-h-11 items-center rounded-card bg-urgent px-4 font-medium text-on-urgent"
          @click="remove(confirming)"
        >
          Hapus
        </button>
      </template>
    </BaseDialog>
  </div>
</template>
