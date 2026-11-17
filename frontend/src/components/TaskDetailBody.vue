<script setup lang="ts">
import LoadingSkeleton from './LoadingSkeleton.vue'
import SubtaskFiles from './SubtaskFiles.vue'
import { computed, ref, watch } from 'vue'
import BaseDialog from './BaseDialog.vue'
import { ApiError } from '../api/client'
import {
  deleteTask,
  getTask,
  moveTask,
  addSubtask,
  deleteSubtask,
  updateSubtask,
  setSubtaskDone,
  setTaskDeliverable,
  setTaskLabels,
  type TaskDetail,
  type TaskSubtask,
} from '../api/tasks'
import type { LabelItem } from '../api/labels'
import { useSessionStore } from '../stores/session'
import { formatDateTime } from '../utils/format'
import { describeTaskHistory, historyNote } from '../utils/taskHistory'
import { formatDay, isOverdue, statusClass, statusLabel, type TaskStatus } from '../utils/task'
import AttachmentSection from './AttachmentSection.vue'
import UrgencyLabel from './UrgencyLabel.vue'
import UserAvatar from './UserAvatar.vue'
import { listPeople, type Person } from '../api/board'
import { readyBlock } from '../utils/subtaskGate'

// wide dipakai halaman penuh /task/:id; tanpa wide isinya satu kolom untuk panel samping.
const props = withDefaults(defineProps<{ taskId: string; labels: LabelItem[]; wide?: boolean }>(), {
  wide: false,
})
const emit = defineEmits<{ close: []; changed: [message: string]; edit: [task: TaskDetail] }>()

// Di panel judul dialog adalah h2, di halaman penuh h1 adalah id task. Tingkat heading mengikuti
// supaya urutannya tidak melompat.
const titleTag = computed(() => (props.wide ? 'h2' : 'h3'))
const sectionTag = computed(() => (props.wide ? 'h3' : 'h4'))

const session = useSessionStore()
const task = ref<TaskDetail | null>(null)
const loadState = ref<'loading' | 'ready' | 'error'>('loading')
const loadError = ref('')
const actionError = ref<string | null>(null)
const busy = ref(false)
// Riwayat task bisa puluhan baris dan mendorong isi lain jauh ke bawah; yang terbaru saja yang
// ditampilkan sampai pengguna meminta sisanya.
const historyLimit = 3
const historyAll = ref(false)
const historyShown = computed(() =>
  historyAll.value ? (task.value?.history ?? []) : (task.value?.history ?? []).slice(0, historyLimit),
)

// Menyusun sub task di luar form Manajer (revisi 7): PIC task membagi pekerjaan, termasuk ke PIC lain
// di task yang sama. Hak per barisnya datang dari server lewat can_edit.
const newTitle = ref('')
const addingSub = ref(false)
const editingSub = ref<number | null>(null)
const editTitle = ref('')
// 0 berarti belum ditugaskan. -1 hanya dipakai sebelum task termuat, supaya pilihan awal bisa diisi
// sekali dengan nama sendiri tanpa menimpa pilihan yang sudah diubah pengguna.
const newPic = ref(-1)
const editPic = ref(0)
const newDesc = ref('')
const editDesc = ref('')
const newLink = ref('')
const editRef = ref('')
// Tautan hasil diisi langsung di Detail, tanpa masuk mode Ubah, karena itu yang paling sering
// diperbarui: satu sub task bisa berganti tautan hasil beberapa kali sampai disetujui.
const resultLink = ref('')
const savingLink = ref(false)
// Calon PIC sub task adalah semua akun aktif, bukan hanya PIC task (revisi 11), jadi daftarnya diambil
// dari /people. Sekali per panel: isinya jarang berubah dan dipakai semua baris sub task.
const people = ref<Person[]>([])
type PicOption = { id: number; name: string }
// Manajer tidak ikut: dia membagi pekerjaan di sub task, bukan menerimanya (revisi 12).
// PIC task ditaruh di atas karena merekalah yang paling sering dipilih.
const picOptions = computed<PicOption[]>(() => {
  const staf = people.value.filter((p) => p.role === 'staff')
  const picked = new Set((task.value?.assignees ?? []).map((p) => p.id))
  return [...staf.filter((p) => picked.has(p.id)), ...staf.filter((p) => !picked.has(p.id))]
})

// PIC lama yang sekarang tidak memenuhi syarat, misalnya Manajer yang ditugaskan sebelum revisi 12,
// tetap ditampilkan di barisnya sendiri supaya menyimpan perubahan lain tidak memaksa menggantinya.
function picOptionsFor(s: TaskSubtask): PicOption[] {
  const cur = s.assignees[0]
  if (!cur || picOptions.value.some((p) => p.id === cur.id)) return picOptions.value
  return [{ id: cur.id, name: cur.name }, ...picOptions.value]
}
// Hanya satu sub task terbuka sekaligus: isinya (deskripsi dan berkas) panjang, dan dua kartu terbuka
// berdampingan membuat daftar sulit dibaca.
const openSub = ref<number | null>(null)

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
    // Sub task paling sering dibuat untuk diri sendiri, jadi itu pilihan awalnya kalau pelaku PIC task.
    if (newPic.value === -1) {
      const me = session.user?.id ?? 0
      newPic.value = task.value.assignees.some((p) => p.id === me) ? me : 0
    }
  } catch (err) {
    // Saat memuat ulang diam-diam, isi yang sudah tampil dipertahankan; galatnya cukup diberitahukan.
    if (quiet && task.value) {
      actionError.value = messageOf(err)
      return
    }
    loadError.value = messageOf(err)
    loadState.value = 'error'
  }
}

watch(
  () => props.taskId,
  (id) => {
    actionError.value = null
    editingLabels.value = false
    historyAll.value = false
    newTitle.value = ''
    editingSub.value = null
    newPic.value = -1
    newDesc.value = ''
    newLink.value = ''
    openSub.value = null
    if (id) {
      void load()
      if (people.value.length === 0) {
        void listPeople()
          .then((rows) => {
            people.value = rows
          })
          // Gagal memuat daftar orang tidak boleh menghalangi detail: pilihan PIC jatuh ke PIC task saja.
          .catch(() => {})
      }
    }
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

const readyReason = computed(() => (task.value ? readyBlock(task.value) : null))

const moves = computed(() =>
  (task.value?.can.move_to ?? []).map((to) => {
    const key = `${task.value?.status}>${to}`
    return {
      to,
      label: moveLabel[key] ?? statusLabel[to],
      primary: forward.has(key),
      blocked: to === 'ready' && readyReason.value !== null,
    }
  }),
)

const checking = ref<number | null>(null)
const rowErrors = ref<Record<number, string>>({})
const announce = ref('')

async function submitNewSubtask() {
  const t = task.value
  const title = newTitle.value.trim()
  if (!t || !title || addingSub.value) return
  addingSub.value = true
  actionError.value = null
  try {
    await addSubtask(t.id, {
      title,
      description: newDesc.value.trim(),
      reference_url: newLink.value.trim(),
      deliverable_url: '',
      pic_id: newPic.value === -1 ? 0 : newPic.value,
    })
    newTitle.value = ''
    newDesc.value = ''
    newLink.value = ''
    announce.value = `Sub task "${title}" ditambahkan.`
    await load(true)
  } catch (err) {
    actionError.value = messageOf(err)
  } finally {
    addingSub.value = false
  }
}

function startEdit(s: TaskSubtask) {
  editingSub.value = s.id
  editTitle.value = s.title
  editDesc.value = s.description
  editRef.value = s.reference_url
  editPic.value = s.assignees[0]?.id ?? 0
  rowErrors.value = { ...rowErrors.value, [s.id]: '' }
}

// Berkas sub task memakai endpoint lampiran, jadi detailnya dimuat ulang diam-diam seperti aksi lain.
async function onSubtaskFiles(message: string) {
  announce.value = message
  await load(true)
}

// Sub task yang jadi urusan pengguna ini: dikerjakan dia, atau dia yang menugaskannya ke orang lain.
function isMine(s: TaskSubtask): boolean {
  const me = session.user?.id ?? 0
  return s.assignees.some((p) => p.id === me)
}

async function submitEdit(s: TaskSubtask) {
  const t = task.value
  const title = editTitle.value.trim()
  if (!t || !title) return
  const desc = editDesc.value.trim()
  const acuan = editRef.value.trim()
  if (
    title === s.title &&
    desc === s.description &&
    acuan === s.reference_url &&
    editPic.value === (s.assignees[0]?.id ?? 0)
  ) {
    editingSub.value = null
    return
  }
  try {
    await updateSubtask(t.id, s.id, {
      title,
      description: desc,
      reference_url: acuan,
      // Tautan hasil punya kotak isian sendiri di Detail; dikirim apa adanya supaya tidak terhapus.
      deliverable_url: s.deliverable_url,
      pic_id: editPic.value,
    })
    editingSub.value = null
    announce.value = `Sub task "${title}" disimpan.`
    await load(true)
  } catch (err) {
    rowErrors.value = { ...rowErrors.value, [s.id]: messageOf(err) }
  }
}

async function removeSubtask(s: TaskSubtask) {
  const t = task.value
  if (!t) return
  try {
    await deleteSubtask(t.id, s.id)
    announce.value = `Sub task "${s.title}" dihapus.`
    await load(true)
  } catch (err) {
    rowErrors.value = { ...rowErrors.value, [s.id]: messageOf(err) }
  }
}

// Hasil dari server yang dipercaya: kotak centang dikembalikan ke keadaan semula kalau gagal,
// karena DOM tidak ikut berubah sendiri bila nilai model tidak berubah.
async function toggleSubtask(s: TaskSubtask, e: Event) {
  const t = task.value
  const input = e.target as HTMLInputElement
  if (!t || checking.value !== null || busy.value) {
    input.checked = s.done
    return
  }
  const next = input.checked
  checking.value = s.id
  rowErrors.value = { ...rowErrors.value, [s.id]: '' }
  try {
    await setSubtaskDone(t.id, s.id, next)
    const verb = next ? 'dicentang' : 'dilepas centangnya'
    announce.value = `Sub task "${s.title}" ${verb}.`
    emit('changed', `${t.id}: sub task "${s.title}" ${verb}.`)
    await load(true)
  } catch (err) {
    input.checked = s.done
    rowErrors.value = { ...rowErrors.value, [s.id]: messageOf(err) }
    // Galat 4xx berarti keadaan di server berbeda (status berubah, hak dicabut), jadi muat ulang.
    // Galat jaringan tidak, karena memuat ulang juga akan gagal.
    if (err instanceof ApiError && err.status >= 400 && err.status < 500) await load(true)
  } finally {
    checking.value = null
  }
}

const revisionOpen = ref(false)
const revisionNote = ref('')
const revisionError = ref<string | null>(null)

function startMove(to: TaskStatus) {
  const t = task.value
  if (!t) return
  if (to === 'ready' && readyReason.value) {
    actionError.value = readyReason.value
    return
  }
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

const picSelect =
  'min-h-11 min-w-0 flex-1 rounded-card border border-line bg-surface px-3 text-sm sm:flex-none'
const subtaskPercent = computed(() => {
  const t = task.value
  return t && t.subtask_total ? Math.round((t.subtask_done / t.subtask_total) * 100) : 0
})

function toggleDetail(s: TaskSubtask) {
  openSub.value = openSub.value === s.id ? null : s.id
  resultLink.value = s.deliverable_url
  rowErrors.value = { ...rowErrors.value, [s.id]: '' }
}

// Menyimpan tautan hasil saja; isi lain dikirim ulang apa adanya supaya tidak tertimpa kosong.
async function saveResultLink(s: TaskSubtask) {
  const t = task.value
  const link = resultLink.value.trim()
  if (!t || savingLink.value || link === s.deliverable_url) return
  savingLink.value = true
  rowErrors.value = { ...rowErrors.value, [s.id]: '' }
  try {
    await updateSubtask(t.id, s.id, {
      title: s.title,
      description: s.description,
      reference_url: s.reference_url,
      deliverable_url: link,
      pic_id: s.assignees[0]?.id ?? 0,
    })
    announce.value = link ? 'Tautan hasil disimpan.' : 'Tautan hasil dihapus.'
    await load(true)
  } catch (err) {
    rowErrors.value = { ...rowErrors.value, [s.id]: messageOf(err) }
  } finally {
    savingLink.value = false
  }
}

// Dua tautan sub task: acuan dari yang menugaskan, hasil dari yang mengerjakan (revisi 10).
function subtaskLinks(s: TaskSubtask): { label: string; url: string }[] {
  const out: { label: string; url: string }[] = []
  if (s.reference_url) out.push({ label: 'Acuan', url: s.reference_url })
  if (s.deliverable_url) out.push({ label: 'Hasil', url: s.deliverable_url })
  return out
}

function linkCount(s: TaskSubtask): number {
  return subtaskLinks(s).length
}

// Nama di pilihan PIC: akun sendiri ditandai supaya tidak perlu mengingat nama sendiri di daftar.
function picName(p: { id: number; name: string }): string {
  return p.id === session.user?.id ? `${p.name} (kamu)` : p.name
}

const linkButton =
  'inline-flex min-h-11 items-center rounded-card px-2 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
const buttonPrimary =
  'inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-on-accent hover:bg-accent-hover active:translate-y-px disabled:opacity-60'
const buttonSecondary =
  'inline-flex min-h-11 items-center rounded-card border border-line px-4 font-medium hover:bg-canvas active:translate-y-px disabled:opacity-60'
</script>

<template>
  <LoadingSkeleton v-if="loadState === 'loading'" label="Memuat task..." shape="detail" />

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

  <!-- flex + min-h-full + mt-auto di bar aksi: itu yang mendorong bar ke dasar panel saat isinya pendek.
       sticky saja tidak cukup, karena sticky hanya menahan saat digulir, tidak memindahkan elemen.
       gap dipakai, bukan space-y, karena space-y menulis margin-top ke tiap anak dan menimpa mt-auto. -->
  <div v-else-if="task" class="flex min-h-full flex-col gap-6">
    <div>
      <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
        <span class="rounded-chip px-2 py-0.5 font-medium" :class="statusClass[task.status]">{{
          statusLabel[task.status]
        }}</span>
        <UrgencyLabel :urgency="task.urgency" />
      </div>
      <component
        :is="titleTag"
        class="mt-2 font-semibold break-words"
        :class="wide ? 'text-2xl' : 'text-xl'"
        >{{ task.title }}</component
      >
      <p v-if="readOnly" class="mt-2 text-sm text-ink-muted">Kamu hanya bisa melihat task ini.</p>
    </div>

    <div aria-live="polite">
      <p v-if="actionError" role="alert" class="rounded-card bg-danger-bg p-3 text-sm text-danger">
        {{ actionError }}
      </p>
    </div>

    <section>
      <component :is="sectionTag" class="font-semibold">Detail</component>
      <!-- Di halaman lebar pasangan label dan nilai disusun empat kolom supaya tidak memanjang ke bawah. -->
      <dl
        class="mt-2 grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2 text-sm"
        :class="wide ? 'lg:grid-cols-[auto_minmax(0,1fr)_auto_minmax(0,1fr)] lg:gap-x-6' : ''"
      >
        <dt class="text-ink-muted">Diminta oleh</dt>
        <dd>{{ task.requested_by.name }}</dd>
        <dt class="text-ink-muted">PIC</dt>
        <dd>
          {{ task.assignees.length ? task.assignees.map((a) => a.name).join(', ') : 'Belum ada PIC' }}
        </dd>
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

    <div
      :class="
        wide ? 'grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] lg:items-start lg:gap-8' : 'space-y-6'
      "
    >
      <div class="min-w-0 space-y-6">
        <section>
          <component :is="sectionTag" class="font-semibold">
            Sub task
            <span v-if="task.subtasks.length" class="font-normal text-ink-muted">
              ({{ task.subtask_done }} dari {{ task.subtask_total }} tercentang)
            </span>
          </component>
          <p v-if="task.subtasks.length && task.status !== 'on_progress'" class="mt-1 text-sm text-ink-muted">
            Sub task hanya bisa dicentang saat task berstatus On Progress.
          </p>
          <p v-else-if="readyReason" class="mt-1 text-sm text-ink-muted">
            Dicentang oleh PIC sub task atau yang menugaskannya. Semua harus tercentang sebelum task bisa
            Ready.
          </p>
          <p class="sr-only" role="status">{{ announce }}</p>
          <div
            v-if="task.subtask_total > 0"
            class="mt-2 h-1.5 overflow-hidden rounded-[2px] border border-line bg-canvas"
            aria-hidden="true"
          >
            <div
              class="h-full bg-mark transition-[width] duration-200"
              :style="{ width: `${subtaskPercent}%` }"
            />
          </div>

          <ol v-if="task.subtasks.length" class="mt-3 space-y-2">
            <!-- Garis tebal di tepi kiri menandai sub task yang dikerjakan pengguna ini, supaya pekerjaannya
                 sendiri langsung ketemu di task yang sub task-nya banyak. -->
            <li
              v-for="(s, i) in task.subtasks"
              :key="s.id"
              class="rounded-card border border-line p-3"
              :class="isMine(s) ? 'border-l-[3px] border-l-mark bg-canvas' : 'bg-surface'"
            >
              <form v-if="editingSub === s.id" class="space-y-2" @submit.prevent="submitEdit(s)">
                <label :for="`sub-edit-${s.id}`" class="sr-only">Nama sub task {{ i + 1 }}</label>
                <input
                  :id="`sub-edit-${s.id}`"
                  v-model="editTitle"
                  type="text"
                  maxlength="200"
                  autocomplete="off"
                  class="min-h-11 w-full rounded-card border border-line bg-surface px-3 text-sm"
                />
                <label :for="`sub-edit-desc-${s.id}`" class="sr-only">Deskripsi sub task {{ i + 1 }}</label>
                <textarea
                  :id="`sub-edit-desc-${s.id}`"
                  v-model="editDesc"
                  rows="3"
                  maxlength="2000"
                  placeholder="Deskripsi (opsional)"
                  class="w-full rounded-card border border-line bg-surface p-3 text-sm"
                ></textarea>
                <label :for="`sub-edit-ref-${s.id}`" class="sr-only">Tautan acuan sub task {{ i + 1 }}</label>
                <input
                  :id="`sub-edit-ref-${s.id}`"
                  v-model="editRef"
                  type="url"
                  maxlength="2000"
                  inputmode="url"
                  placeholder="Tautan acuan dari yang menugaskan (opsional)"
                  class="min-h-11 w-full rounded-card border border-line bg-surface px-3 text-sm"
                />
                <div class="flex flex-wrap items-center gap-2">
                  <label :for="`sub-edit-pic-${s.id}`" class="text-sm text-ink-muted">Dikerjakan</label>
                  <select :id="`sub-edit-pic-${s.id}`" v-model.number="editPic" :class="picSelect">
                    <option :value="0">Belum ditugaskan</option>
                    <option v-for="p in picOptionsFor(s)" :key="p.id" :value="p.id">
                      {{ picName(p) }}
                    </option>
                  </select>
                  <button type="submit" :class="buttonSecondary" :disabled="!editTitle.trim()">Simpan</button>
                  <button type="button" :class="buttonSecondary" @click="editingSub = null">Batal</button>
                </div>
              </form>

              <template v-else>
                <div class="flex items-start gap-2">
                  <label
                    class="flex min-h-11 min-w-0 flex-1 items-start gap-3 rounded-card has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-mark"
                    :class="s.can_check ? 'cursor-pointer' : 'cursor-default'"
                  >
                    <input
                      type="checkbox"
                      class="mt-0.5 size-5 shrink-0 accent-mark"
                      :checked="s.done"
                      :disabled="!s.can_check"
                      :aria-busy="checking === s.id"
                      @change="toggleSubtask(s, $event)"
                    />
                    <span
                      class="min-w-0 text-sm font-medium break-words"
                      :class="s.done ? 'text-ink-muted' : ''"
                    >
                      <span class="text-ink-muted">{{ i + 1 }}.</span> {{ s.title }}
                    </span>
                  </label>
                  <button
                    type="button"
                    :class="linkButton"
                    class="shrink-0"
                    :aria-expanded="openSub === s.id"
                    :aria-controls="`sub-detail-${s.id}`"
                    @click="toggleDetail(s)"
                  >
                    {{ openSub === s.id ? 'Tutup' : 'Detail' }}
                  </button>
                </div>

                <div class="mt-1 ml-8 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
                  <span
                    v-if="s.assignees.length"
                    class="inline-flex items-center gap-1.5 rounded-chip border border-line bg-surface py-0.5 pr-2 pl-0.5"
                  >
                    <UserAvatar
                      :user-id="s.assignees[0].id"
                      :name="s.assignees[0].name"
                      :avatar-id="s.assignees[0].avatar_id"
                      size="sm"
                      decorative
                    />
                    {{ picName(s.assignees[0]) }}
                  </span>
                  <span
                    v-else
                    class="rounded-chip border border-dashed border-line px-2 py-0.5 text-ink-muted"
                  >
                    Belum ditugaskan
                  </span>
                  <span v-if="s.created_by" class="text-ink-muted">dari {{ s.created_by }}</span>
                  <span v-if="s.attachments.length" class="text-ink-muted">
                    {{ s.attachments.length }} berkas
                  </span>
                  <span v-if="linkCount(s)" class="text-ink-muted">{{ linkCount(s) }} tautan</span>
                  <span v-if="s.done" class="text-ink-muted">
                    Dicentang {{ s.done_by }}{{ s.done_at ? `, ${formatDateTime(s.done_at)}` : '' }}
                  </span>
                </div>

                <div v-if="openSub === s.id" :id="`sub-detail-${s.id}`" class="mt-3 ml-8 space-y-3">
                  <p v-if="s.description" class="text-sm whitespace-pre-line break-words">
                    {{ s.description }}
                  </p>
                  <p v-else class="text-sm text-ink-muted">Belum ada deskripsi.</p>

                  <a
                    v-for="l in subtaskLinks(s)"
                    :key="l.url"
                    :href="l.url"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="inline-flex min-h-11 max-w-full items-center gap-2 rounded-card border border-line bg-surface px-3 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
                  >
                    <svg
                      aria-hidden="true"
                      viewBox="0 0 24 24"
                      class="size-5 shrink-0"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="1.75"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                    >
                      <path d="M14 4h6v6M20 4l-9 9M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5" />
                    </svg>
                    <span class="shrink-0 text-ink-muted">{{ l.label }}</span>
                    <span class="truncate">{{ l.url }}</span>
                  </a>

                  <form
                    v-if="s.can_edit"
                    class="flex flex-wrap items-center gap-2"
                    @submit.prevent="saveResultLink(s)"
                  >
                    <label :for="`sub-result-${s.id}`" class="sr-only">Tautan hasil sub task</label>
                    <input
                      :id="`sub-result-${s.id}`"
                      v-model="resultLink"
                      type="url"
                      maxlength="2000"
                      inputmode="url"
                      placeholder="Tautan hasil, misalnya Figma atau Drive"
                      class="min-h-11 min-w-0 flex-1 rounded-card border border-line bg-surface px-3 text-sm"
                    />
                    <button type="submit" :class="buttonSecondary" :disabled="savingLink">
                      {{ savingLink ? 'Menyimpan...' : 'Simpan tautan' }}
                    </button>
                  </form>

                  <SubtaskFiles
                    :task-id="task.id"
                    :subtask-id="s.id"
                    :items="s.attachments"
                    :can-manage="task.can.edit_deliverable"
                    @changed="onSubtaskFiles"
                  />

                  <div v-if="s.can_edit" class="flex gap-1">
                    <button type="button" :class="linkButton" @click="startEdit(s)">Ubah</button>
                    <button type="button" :class="linkButton" class="text-urgent" @click="removeSubtask(s)">
                      Hapus
                    </button>
                  </div>
                </div>
              </template>

              <p
                v-if="rowErrors[s.id]"
                role="alert"
                class="mt-2 rounded-card bg-danger-bg p-2 text-sm text-danger"
              >
                {{ rowErrors[s.id] }}
              </p>
            </li>
          </ol>
          <p v-else class="mt-2 text-sm text-ink-muted">Belum ada sub task.</p>

          <form
            v-if="task.can.add_subtask"
            class="mt-3 space-y-2 rounded-card border border-dashed border-line p-3"
            @submit.prevent="submitNewSubtask"
          >
            <label for="sub-new" class="sr-only">Nama sub task baru</label>
            <input
              id="sub-new"
              v-model="newTitle"
              type="text"
              maxlength="200"
              autocomplete="off"
              placeholder="Pekerjaan yang perlu dilakukan"
              class="min-h-11 w-full rounded-card border border-line bg-surface px-3 text-sm"
            />
            <label for="sub-new-desc" class="sr-only">Deskripsi sub task baru</label>
            <textarea
              id="sub-new-desc"
              v-model="newDesc"
              rows="2"
              maxlength="2000"
              placeholder="Deskripsi (opsional)"
              class="w-full rounded-card border border-line bg-surface p-3 text-sm"
            ></textarea>
            <label for="sub-new-link" class="sr-only">Tautan acuan sub task baru</label>
            <input
              id="sub-new-link"
              v-model="newLink"
              type="url"
              maxlength="2000"
              inputmode="url"
              placeholder="Tautan acuan, misalnya brief atau contoh (opsional)"
              class="min-h-11 w-full rounded-card border border-line bg-surface px-3 text-sm"
            />
            <div class="flex flex-wrap items-center gap-2">
              <label for="sub-new-pic" class="text-sm text-ink-muted">Dikerjakan</label>
              <select id="sub-new-pic" v-model.number="newPic" :class="picSelect">
                <option :value="0">Belum ditugaskan</option>
                <option v-for="p in picOptions" :key="p.id" :value="p.id">{{ picName(p) }}</option>
              </select>
              <button type="submit" :class="buttonSecondary" :disabled="addingSub || !newTitle.trim()">
                {{ addingSub ? 'Menambah...' : 'Tambah' }}
              </button>
            </div>
          </form>
        </section>

        <section v-if="task.description">
          <component :is="sectionTag" class="font-semibold">Deskripsi</component>
          <p class="mt-2 text-sm whitespace-pre-line break-words">{{ task.description }}</p>
        </section>

        <section>
          <component :is="sectionTag" class="font-semibold">Lampiran brief</component>
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
          <component :is="sectionTag" class="font-semibold">Deliverable</component>
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
              class="text-mark underline"
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
      </div>

      <div class="min-w-0 space-y-6">
        <section>
          <div class="flex items-center justify-between gap-3">
            <component :is="sectionTag" class="font-semibold">Label</component>
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
              :style="{ backgroundColor: l.color, color: 'var(--color-tag-ink)' }"
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
                class="inline-flex min-h-11 items-center gap-2 rounded-chip border border-line px-3 text-sm has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-mark"
                :class="l.locked ? 'cursor-not-allowed opacity-70' : 'cursor-pointer'"
                :style="{ backgroundColor: l.color, color: 'var(--color-tag-ink)' }"
              >
                <input
                  type="checkbox"
                  class="size-4 accent-mark"
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

        <section>
          <component :is="sectionTag" class="font-semibold">Riwayat</component>
          <ol class="mt-2 space-y-3 border-l border-line pl-3 text-sm">
            <li v-for="h in historyShown" :key="h.id">
              <p>
                <span class="font-medium">{{ h.actor_name || 'Sistem' }}</span> {{ describeTaskHistory(h) }}
              </p>
              <p v-if="historyNote(h)" class="mt-1 border-l-2 border-line pl-2">
                Catatan: {{ historyNote(h) }}
              </p>
              <p class="text-ink-muted">{{ formatDateTime(h.occurred_at) }}</p>
            </li>
          </ol>
          <button
            v-if="task.history.length > historyLimit"
            type="button"
            class="mt-2 inline-flex min-h-11 items-center rounded-card px-3 text-sm font-medium text-ink-muted hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
            :aria-expanded="historyAll"
            @click="historyAll = !historyAll"
          >
            {{
              historyAll
                ? 'Tampilkan lebih sedikit'
                : `Tampilkan ${task.history.length - historyLimit} riwayat lainnya`
            }}
          </button>
        </section>
      </div>
    </div>

    <!-- Bar aksi menempel di bawah area gulir. Di halaman penuh ia naik di atas navigasi bawah ponsel,
         dan margin bawahnya menutup padding bawah <main> (pb-20 ponsel, pb-10 desktop) supaya saat
         gulir mentok bar tidak berhenti menggantung di atas tepi layar. -->
    <div
      v-if="moves.length || task.can.edit || task.can.delete"
      class="sticky z-10 mt-auto border-t border-line bg-surface py-3"
      :class="
        wide
          ? '-mb-6 bottom-[calc(3.5rem+env(safe-area-inset-bottom))] md:-mb-10 md:bottom-0'
          : '-mx-5 -mb-4 bottom-[-1rem] px-5'
      "
    >
      <div class="flex w-full flex-wrap justify-end gap-2">
        <p
          v-if="readyReason && moves.some((m) => m.blocked)"
          id="ready-hint"
          class="w-full text-right text-sm text-ink-muted"
        >
          {{ readyReason }}
        </p>
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
          :class="[m.primary ? buttonPrimary : buttonSecondary, m.blocked ? 'opacity-60' : '']"
          :disabled="busy"
          :aria-disabled="m.blocked"
          :aria-describedby="m.blocked ? 'ready-hint' : undefined"
          @click="startMove(m.to)"
        >
          {{ m.label }}
        </button>
      </div>
    </div>
  </div>

  <!-- Dialog turunan dipindah ke body supaya tidak bersarang di dalam dialog panel. -->
  <Teleport to="body">
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
          class="inline-flex min-h-11 items-center rounded-card bg-urgent px-4 font-medium text-on-urgent active:translate-y-px disabled:opacity-60"
          @click="confirmDelete"
        >
          Hapus task
        </button>
      </template>
    </BaseDialog>
  </Teleport>
</template>
