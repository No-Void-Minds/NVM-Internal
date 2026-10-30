<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import BaseDialog from './BaseDialog.vue'
import { ApiError } from '../api/client'
import {
  deleteTask,
  getTask,
  moveTask,
  setTaskDeliverable,
  setTaskLabels,
  type TaskDetail,
} from '../api/tasks'
import type { LabelItem } from '../api/labels'
import { useSessionStore } from '../stores/session'
import { formatDateTime } from '../utils/format'
import { describeTaskHistory, historyNote } from '../utils/taskHistory'
import {
  formatDay,
  isOverdue,
  statusClass,
  statusLabel,
  urgencyClass,
  urgencyLabel,
  type TaskStatus,
} from '../utils/task'
import AttachmentSection from './AttachmentSection.vue'

const props = defineProps<{ taskId: string | null; labels: LabelItem[] }>()
const emit = defineEmits<{ close: []; changed: [message: string]; edit: [task: TaskDetail] }>()

const session = useSessionStore()
const task = ref<TaskDetail | null>(null)
const loadState = ref<'loading' | 'ready' | 'error'>('loading')
const loadError = ref('')
const actionError = ref<string | null>(null)
const busy = ref(false)
// Dideklarasikan di atas karena watch taskId di bawah langsung berjalan (immediate) dan mengisinya.
const editingLabels = ref(false)
const chosenLabels = ref<number[]>([])

// quiet dipakai setelah unggah atau hapus berkas, supaya panel tidak berkedip ke tampilan memuat.
async function load(quiet = false) {
  if (!props.taskId) return
  if (!quiet) loadState.value = 'loading'
  try {
    task.value = await getTask(props.taskId)
    loadState.value = 'ready'
  } catch (err) {
    loadError.value = messageOf(err)
    loadState.value = 'error'
  }
}

watch(
  () => props.taskId,
  (id) => {
    actionError.value = null
    editingLabels.value = false
    if (id) void load()
  },
  { immediate: true },
)

function messageOf(err: unknown): string {
  return err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
}

// Semua aksi lewat sini: setelah berhasil detail dimuat ulang; kalau ditolak karena versi basi (409),
// detail juga dimuat ulang supaya pengguna langsung melihat keadaan terbaru.
async function run(action: () => Promise<unknown>, success: string): Promise<boolean> {
  if (busy.value) return false
  busy.value = true
  actionError.value = null
  try {
    await action()
    emit('changed', success)
    await load()
    return true
  } catch (err) {
    actionError.value = messageOf(err)
    if (err instanceof ApiError && err.status === 409) await load()
    return false
  } finally {
    busy.value = false
  }
}

const briefs = computed(() => task.value?.attachments.filter((a) => a.kind === 'brief') ?? [])
const deliverables = computed(() => task.value?.attachments.filter((a) => a.kind === 'deliverable') ?? [])

function onFilesChanged(message: string) {
  emit('changed', message)
  void load(true)
}

const readOnly = computed(() => {
  const c = task.value?.can
  return !!c && !c.edit && !c.edit_labels && !c.edit_deliverable && c.move_to.length === 0
})

const moveLabel: Record<string, string> = {
  'requested>on_progress': 'Mulai kerjakan',
  'on_progress>ready': 'Tandai Ready',
  'ready>on_progress': 'Kembalikan untuk revisi',
  'ready>completed': 'Tandai Completed',
  'completed>on_progress': 'Buka kembali',
}

// Perpindahan maju (menuju selesai) jadi tombol utama; mundur (revisi, buka kembali) jadi tombol biasa.
const forward = new Set(['requested>on_progress', 'on_progress>ready', 'ready>completed'])

const moves = computed(() =>
  (task.value?.can.move_to ?? []).map((to) => {
    const key = `${task.value?.status}>${to}`
    return { to, label: moveLabel[key] ?? statusLabel[to], primary: forward.has(key) }
  }),
)

const revisionOpen = ref(false)
const revisionNote = ref('')
const revisionError = ref<string | null>(null)

function startMove(to: TaskStatus) {
  const t = task.value
  if (!t) return
  if (t.status === 'ready' && to === 'on_progress') {
    revisionNote.value = ''
    revisionError.value = null
    revisionOpen.value = true
    return
  }
  void run(() => moveTask(t.id, t.version, to), `${t.id} dipindahkan ke ${statusLabel[to]}.`)
}

async function submitRevision() {
  const t = task.value
  if (!t) return
  if (!revisionNote.value.trim()) {
    revisionError.value = 'Tulis apa yang perlu direvisi.'
    return
  }
  const ok = await run(
    () => moveTask(t.id, t.version, 'on_progress', revisionNote.value),
    `${t.id} dikembalikan untuk revisi.`,
  )
  if (ok) revisionOpen.value = false
  else revisionError.value = actionError.value
}

// Staf hanya boleh memasang label divisinya atau label semua divisi (FR-TASK-21). Label divisi lain
// yang sudah menempel tetap tampil tercentang tetapi tidak bisa dilepas olehnya.
function inScope(l: LabelItem) {
  return (
    session.user?.role === 'manager' || l.division_id === null || l.division_id === session.user?.division_id
  )
}
const labelChoices = computed(() => {
  const attached = new Set(task.value?.labels.map((l) => l.id))
  return props.labels
    .filter((l) => attached.has(l.id) || (l.is_active && inScope(l)))
    .map((l) => ({ ...l, locked: !inScope(l) }))
})

function startLabels() {
  chosenLabels.value = task.value?.labels.map((l) => l.id) ?? []
  editingLabels.value = true
}

function toggleLabel(id: number) {
  chosenLabels.value = chosenLabels.value.includes(id)
    ? chosenLabels.value.filter((l) => l !== id)
    : [...chosenLabels.value, id]
}

async function saveLabels() {
  const t = task.value
  if (!t) return
  const ok = await run(() => setTaskLabels(t.id, t.version, chosenLabels.value), `Label ${t.id} disimpan.`)
  if (ok) editingLabels.value = false
}

const deliverable = ref('')
watch(task, (t) => (deliverable.value = t?.deliverable_url ?? ''))

function saveDeliverable() {
  const t = task.value
  if (!t) return
  const url = deliverable.value.trim()
  if (url && !/^https?:\/\//i.test(url)) {
    actionError.value = 'Tautan deliverable harus diawali http:// atau https://.'
    return
  }
  void run(() => setTaskDeliverable(t.id, t.version, url), `Tautan deliverable ${t.id} disimpan.`)
}

const deleteOpen = ref(false)
async function confirmDelete() {
  const t = task.value
  if (!t) return
  const ok = await run(() => deleteTask(t.id, t.version), `${t.id} dihapus.`)
  deleteOpen.value = false
  if (ok) emit('close')
}

const buttonPrimary =
  'inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-white active:translate-y-px disabled:opacity-60'
const buttonSecondary =
  'inline-flex min-h-11 items-center rounded-card border border-line px-4 font-medium hover:bg-canvas active:translate-y-px disabled:opacity-60'
</script>

<template>
  <BaseDialog
    :open="taskId !== null"
    :title="task ? task.id : 'Detail task'"
    variant="panel"
    @close="emit('close')"
  >
    <p v-if="loadState === 'loading'" class="text-ink-muted">Memuat task...</p>

    <div v-else-if="loadState === 'error'" role="alert" class="rounded-card bg-danger-bg p-4 text-danger">
      <p class="font-medium">{{ loadError }}</p>
      <button
        type="button"
        class="mt-3 inline-flex min-h-11 items-center rounded-card border border-danger px-4 font-medium"
        @click="load()"
      >
        Muat ulang
      </button>
    </div>

    <div v-else-if="task" class="space-y-6">
      <div>
        <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
          <span class="rounded-chip px-2 py-0.5 font-medium" :class="statusClass[task.status]">{{
            statusLabel[task.status]
          }}</span>
          <span :class="urgencyClass[task.urgency]">{{ urgencyLabel[task.urgency] }}</span>
        </div>
        <h3 class="mt-2 text-xl font-semibold">{{ task.title }}</h3>
        <p v-if="readOnly" class="mt-2 text-sm text-ink-muted">Kamu hanya bisa melihat task ini.</p>
      </div>

      <div aria-live="polite">
        <p v-if="actionError" role="alert" class="rounded-card bg-danger-bg p-3 text-sm text-danger">
          {{ actionError }}
        </p>
      </div>

      <section>
        <h4 class="font-semibold">Detail</h4>
        <dl class="mt-2 grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2 text-sm">
          <dt class="text-ink-muted">Diminta oleh</dt>
          <dd>{{ task.requested_by.name }}</dd>
          <dt class="text-ink-muted">PIC</dt>
          <dd>{{ task.assignees.map((a) => a.name).join(', ') }}</dd>
          <dt class="text-ink-muted">Divisi</dt>
          <dd>{{ task.division.name }}</dd>
          <dt class="text-ink-muted">Tanggal mulai</dt>
          <dd>{{ task.start_date ? formatDay(task.start_date) : 'Belum diisi' }}</dd>
          <dt class="text-ink-muted">Tenggat</dt>
          <dd :class="isOverdue(task.due_date, task.status) ? 'font-medium text-urgent' : ''">
            <template v-if="task.due_date">
              {{ isOverdue(task.due_date, task.status) ? 'Terlambat, ' : '' }}{{ formatDay(task.due_date) }}
            </template>
            <template v-else>Belum diisi</template>
          </dd>
          <template v-if="task.solved_date">
            <dt class="text-ink-muted">Selesai</dt>
            <dd>{{ formatDateTime(task.solved_date) }}</dd>
          </template>
        </dl>
      </section>

      <section>
        <div class="flex items-center justify-between gap-3">
          <h4 class="font-semibold">Label</h4>
          <button
            v-if="task.can.edit_labels && !editingLabels"
            type="button"
            :class="buttonSecondary"
            @click="startLabels"
          >
            Ubah label
          </button>
        </div>
        <div v-if="!editingLabels" class="mt-2 flex flex-wrap gap-1">
          <span
            v-for="l in task.labels"
            :key="l.id"
            class="rounded-chip px-2 py-0.5 text-sm"
            :style="{ backgroundColor: l.color }"
          >
            {{ l.name }}{{ l.is_active ? '' : ' (nonaktif)' }}
          </span>
          <span v-if="task.labels.length === 0" class="text-sm text-ink-muted">Belum ada label.</span>
        </div>
        <fieldset v-else class="mt-2">
          <legend class="sr-only">Pilih label</legend>
          <div class="flex flex-wrap gap-2">
            <label
              v-for="l in labelChoices"
              :key="l.id"
              class="inline-flex min-h-11 items-center gap-2 rounded-chip border border-line px-3 text-sm has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-accent"
              :class="l.locked ? 'cursor-not-allowed opacity-70' : 'cursor-pointer'"
              :style="{ backgroundColor: l.color }"
            >
              <input
                type="checkbox"
                class="size-4 accent-accent"
                :checked="chosenLabels.includes(l.id)"
                :disabled="l.locked"
                @change="toggleLabel(l.id)"
              />
              {{ l.name }}
            </label>
          </div>
          <p v-if="labelChoices.some((l) => l.locked)" class="mt-1 text-sm text-ink-muted">
            Label divisi lain hanya bisa diubah Manajer.
          </p>
          <div class="mt-3 flex gap-2">
            <button type="button" :class="buttonPrimary" :disabled="busy" @click="saveLabels">
              Simpan label
            </button>
            <button type="button" :class="buttonSecondary" @click="editingLabels = false">Batal</button>
          </div>
        </fieldset>
      </section>

      <section v-if="task.description">
        <h4 class="font-semibold">Deskripsi</h4>
        <p class="mt-2 text-sm whitespace-pre-line break-words">{{ task.description }}</p>
      </section>
      <section>
        <h4 class="font-semibold">Lampiran brief</h4>
        <AttachmentSection
          :task-id="task.id"
          kind="brief"
          :items="briefs"
          :can-upload="task.can.edit"
          :can-delete="task.can.edit"
          @changed="onFilesChanged"
        />
      </section>
      <section>
        <h4 class="font-semibold">Deliverable</h4>
        <form
          v-if="task.can.edit_deliverable"
          class="mt-2 flex flex-col gap-2 sm:flex-row"
          novalidate
          @submit.prevent="saveDeliverable"
        >
          <label for="detail-deliverable" class="sr-only">Tautan deliverable</label>
          <input
            id="detail-deliverable"
            v-model="deliverable"
            type="url"
            inputmode="url"
            placeholder="https://"
            class="min-h-11 min-w-0 flex-1 rounded-card border border-line bg-surface px-3"
          />
          <button type="submit" :class="buttonSecondary" :disabled="busy">Simpan tautan</button>
        </form>
        <p v-if="task.deliverable_url" class="mt-2 text-sm break-all">
          <a
            :href="task.deliverable_url"
            target="_blank"
            rel="noopener noreferrer"
            class="text-accent underline"
          >
            {{ task.deliverable_url }}
          </a>
        </p>
        <p v-else-if="!task.can.edit_deliverable" class="mt-2 text-sm text-ink-muted">Belum ada tautan.</p>
        <AttachmentSection
          :task-id="task.id"
          kind="deliverable"
          :items="deliverables"
          :can-upload="task.can.edit_deliverable"
          :can-delete="task.can.edit_deliverable"
          @changed="onFilesChanged"
        />
      </section>

      <section>
        <h4 class="font-semibold">Riwayat</h4>
        <ol class="mt-2 space-y-3 text-sm">
          <li v-for="h in task.history" :key="h.id">
            <p>
              <span class="font-medium">{{ h.actor_name || 'Sistem' }}</span> {{ describeTaskHistory(h) }}
            </p>
            <p v-if="historyNote(h)" class="mt-1 border-l-2 border-line pl-2">
              Catatan: {{ historyNote(h) }}
            </p>
            <p class="text-ink-muted">{{ formatDateTime(h.occurred_at) }}</p>
          </li>
        </ol>
      </section>
    </div>

    <template
      v-if="task && loadState === 'ready' && (moves.length || task.can.edit || task.can.delete)"
      #footer
    >
      <div class="flex w-full flex-wrap justify-end gap-2">
        <button
          v-if="task.can.delete"
          type="button"
          class="mr-auto inline-flex min-h-11 items-center rounded-card px-3 font-medium text-urgent hover:bg-canvas"
          @click="deleteOpen = true"
        >
          Hapus
        </button>
        <button v-if="task.can.edit" type="button" :class="buttonSecondary" @click="emit('edit', task)">
          Ubah
        </button>
        <button
          v-for="m in moves"
          :key="m.to"
          type="button"
          :class="m.primary ? buttonPrimary : buttonSecondary"
          :disabled="busy"
          @click="startMove(m.to)"
        >
          {{ m.label }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <BaseDialog :open="revisionOpen" title="Kembalikan untuk revisi" @close="revisionOpen = false">
    <form id="revision-form" novalidate @submit.prevent="submitRevision">
      <p v-if="revisionError" role="alert" class="mb-3 rounded-card bg-danger-bg p-3 text-sm text-danger">
        {{ revisionError }}
      </p>
      <label for="revision-note" class="block text-sm font-medium">Apa yang perlu direvisi?</label>
      <textarea
        id="revision-note"
        v-model="revisionNote"
        rows="4"
        maxlength="500"
        autofocus
        class="mt-1 block w-full rounded-card border border-line bg-surface px-3 py-2"
      />
      <p class="mt-1 text-sm text-ink-muted">Catatan ini tampil di riwayat task. Maksimal 500 karakter.</p>
    </form>
    <template #footer>
      <button type="button" :class="buttonSecondary" @click="revisionOpen = false">Batal</button>
      <button type="submit" form="revision-form" :class="buttonPrimary" :disabled="busy">Kembalikan</button>
    </template>
  </BaseDialog>

  <BaseDialog :open="deleteOpen" :title="`Hapus ${task?.id ?? ''}?`" @close="deleteOpen = false">
    <p>
      Task ini hilang dari semua daftar dan tidak bisa dipulihkan dari aplikasi. Jejaknya tetap ada di audit
      log.
    </p>
    <template #footer>
      <button type="button" :class="buttonSecondary" @click="deleteOpen = false">Batal</button>
      <button
        type="button"
        :disabled="busy"
        class="inline-flex min-h-11 items-center rounded-card bg-urgent px-4 font-medium text-white active:translate-y-px disabled:opacity-60"
        @click="confirmDelete"
      >
        Hapus task
      </button>
    </template>
  </BaseDialog>
</template>
