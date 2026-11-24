<script setup lang="ts">
import LoadingSkeleton from '../components/LoadingSkeleton.vue'
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { toArchiveRouteQuery } from '../utils/archiveQuery'
import KeywordSearch from '../components/KeywordSearch.vue'
import TaskCard from '../components/TaskCard.vue'
import TaskDetailPanel from '../components/TaskDetailPanel.vue'
import TaskFormPanel from '../components/TaskFormPanel.vue'
import { ApiError } from '../api/client'
import { getBoard, listPeople, type Board, type BoardQuery, type BoardTask, type Person } from '../api/board'
import { moveTask, type TaskDetail } from '../api/tasks'
import { listUsers, type UserItem } from '../api/users'
import { listLabels, type LabelItem } from '../api/labels'
import { listDivisions, type DivisionItem } from '../api/divisions'
import { useSessionStore } from '../stores/session'
import {
  dueOptions,
  hasFilters,
  normalizeBoardQuery,
  parseBoardQuery,
  sameRouteQuery,
  toRouteQuery,
} from '../utils/boardQuery'
import {
  staleMoveMessage,
  statusClass,
  statusLabel,
  urgencies,
  urgencyLabel,
  type TaskStatus,
  type Urgency,
} from '../utils/task'
import type { DueFilter } from '../api/board'
import MoveDialog from '../components/MoveDialog.vue'
import BaseDialog from '../components/BaseDialog.vue'
import ProfileCard from '../components/ProfileCard.vue'
import TodoCard from '../components/TodoCard.vue'
import TodoPanel from '../components/TodoPanel.vue'
import ProfileForm from '../components/ProfileForm.vue'
import { getTaskCounts, type TaskCounts } from '../api/profile'
import { readyBlock } from '../utils/subtaskGate'
import PageHeader from '../components/PageHeader.vue'
import DailyActivityCard from '../components/DailyActivityCard.vue'
import FilterToolbar from '../components/FilterToolbar.vue'
import { filterControl } from '../utils/ui'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const isManager = computed(() => session.user?.role === 'manager')
// Revisi 27: Staf juga boleh membuat task. Nilainya dari server, bukan dihitung dari peran di sini.
const canCreateTask = computed(() => session.user?.can_create_task === true)
const isStaff = computed(() => session.user?.role === 'staff')
const staffUsername = computed(() => (isStaff.value ? (session.user?.username ?? null) : null))

// URL adalah sumber filter: bisa di-refresh, dibagikan, dan tombol Back browser tetap bekerja.
const query = computed(() => parseBoardQuery(route.query, staffUsername.value))
// Revisi 40: kolom Completed paling banyak 10 task dari server. Tautan ke Arsip (yang isinya memang hanya task
// Completed) membawa filter papan yang juga dikenal Arsip; filter deadline tidak ada di Arsip.
const archiveLink = computed(() => {
  const b = query.value
  return {
    name: 'archive',
    query: toArchiveRouteQuery({
      q: b.q,
      scope: b.scope,
      pic: b.pic,
      division: b.division,
      label: b.label,
      urgency: b.urgency,
    }),
  }
})
const filtered = computed(() => hasFilters(query.value))
// Kata kunci tidak dihitung: kolom cari selalu terlihat, jumlah ini untuk tombol Filter yang dilipat di ponsel.
const activeFilterCount = computed(
  () =>
    [query.value.pic, query.value.division, query.value.label, query.value.urgency, query.value.due].filter(
      Boolean,
    ).length,
)
// Di "Task saya" isinya sudah pasti milik sendiri, jadi filter PIC baru berguna di "Semua task".
const showPicFilter = computed(() => !isStaff.value || query.value.scope === 'all')
// Kunci berupa teks supaya papan hanya dimuat ulang saat filternya benar-benar berubah,
// bukan setiap kali objek query dibuat ulang.
const queryKey = computed(() => new URLSearchParams(toRouteQuery(query.value)).toString())

function setQuery(patch: Partial<BoardQuery>) {
  void router.push({
    query: toRouteQuery(normalizeBoardQuery({ ...query.value, ...patch }, staffUsername.value)),
  })
}

// URL yang tidak sesuai aturan dirapikan dengan replace, supaya tidak menambah langkah di tombol Back.
watch(
  () => route.query,
  (q) => {
    const canonical = toRouteQuery(query.value)
    if (!sameRouteQuery(canonical, q)) void router.replace({ query: canonical })
  },
  { immediate: true },
)

function selected(e: Event): string {
  return (e.target as HTMLSelectElement).value
}

function clearFilters() {
  setQuery({
    q: undefined,
    pic: undefined,
    division: undefined,
    label: undefined,
    urgency: undefined,
    due: undefined,
  })
}

const board = ref<Board | null>(null)
const loadState = ref<'loading' | 'ready' | 'error'>('loading')
const loadError = ref('')
const notice = ref<{ kind: 'success' | 'error'; text: string } | null>(null)
const total = computed(() => board.value?.columns.reduce((n, c) => n + c.count, 0) ?? 0)

function messageOf(err: unknown): string {
  return err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
}

let controller: AbortController | null = null

// quiet dipakai setelah aksi, supaya papan tidak berkedip ke tampilan memuat.
async function load(quiet = false) {
  // Permintaan lama dibatalkan saat filter berganti, supaya jawaban yang datang terlambat tidak menimpa yang baru.
  controller?.abort()
  const c = new AbortController()
  controller = c
  if (!quiet) loadState.value = 'loading'
  // Angka kartu profil tidak bergantung pada filter, jadi hanya dimuat ulang setelah aksi, bukan tiap filter berubah.
  if (quiet) void loadCounts()
  try {
    board.value = await getBoard(query.value, c.signal)
    loadState.value = 'ready'
  } catch (err) {
    if (c.signal.aborted) return
    if (quiet && board.value) {
      notice.value = { kind: 'error', text: messageOf(err) }
      return
    }
    loadError.value = messageOf(err)
    loadState.value = 'error'
  }
}
watch(queryKey, () => load())

// Admin tidak pernah menjadi PIC (hanya Staf dan Manajer), jadi tidak punya "task saya".
const showCounts = computed(() => session.user?.role !== 'admin')
const counts = ref<TaskCounts | null>(null)
const countsState = ref<'loading' | 'ready' | 'error'>('loading')

let countsController: AbortController | null = null

async function loadCounts() {
  if (!showCounts.value) return
  // Dua aksi beruntun memicu dua permintaan; yang lama dibatalkan supaya jawabannya tidak menimpa angka terbaru.
  countsController?.abort()
  const c = new AbortController()
  countsController = c
  countsState.value = 'loading'
  try {
    counts.value = await getTaskCounts(c.signal)
    countsState.value = 'ready'
  } catch {
    if (c.signal.aborted) return
    // Angka lama tetap tampil bila sudah ada; galat hanya terlihat saat belum pernah berhasil dimuat.
    countsState.value = 'error'
  }
}

// Klik angka menampilkan task saya di papan: filter lain dikosongkan supaya jumlahnya sama dengan angka di kartu.
function pickCount(status: TaskStatus) {
  const self = session.user?.username
  const mine: Partial<BoardQuery> = isStaff.value ? { scope: 'mine' } : { pic: self }
  void router.push({
    query: toRouteQuery(normalizeBoardQuery({ ...mine }, staffUsername.value)),
  })
  chooseColumn(status)
}

const profileOpen = ref(false)

// Papan dimuat ulang karena kartu task membawa nama dan foto PIC dari data papan, bukan dari data sesi.
async function onProfileSaved(message: string) {
  notice.value = { kind: 'success', text: message }
  await load(true)
}

const labels = ref<LabelItem[]>([])
const divisions = ref<DivisionItem[]>([])
const people = ref<Person[]>([])
const users = ref<UserItem[]>([])

onMounted(async () => {
  void load()
  void loadCounts()
  try {
    const [l, d, p] = await Promise.all([listLabels(), listDivisions(), listPeople()])
    labels.value = l.labels
    divisions.value = d
    people.value = p
    // Daftar akun hanya bisa dibaca pengelola, dan hanya dibutuhkan Manajer untuk memilih PIC di form.
    if (isManager.value) users.value = await listUsers()
  } catch {
    notice.value = {
      kind: 'error',
      text: 'Pilihan filter gagal dimuat. Muat ulang halaman untuk memakai filter.',
    }
  }
})

const detailId = ref<string | null>(null)
const formOpen = ref(false)
const editing = ref<TaskDetail | null>(null)
const moving = ref<BoardTask | null>(null)
const movingTarget = ref<TaskStatus | null>(null)

function openMove(t: BoardTask) {
  movingTarget.value = null
  moving.value = t
}

function closeMove() {
  moving.value = null
  movingTarget.value = null
}

async function onMoved(message: string) {
  closeMove()
  notice.value = { kind: 'success', text: message }
  await load(true)
}

async function onStale(message: string) {
  closeMove()
  notice.value = { kind: 'error', text: message }
  await load(true)
}

// Drag-and-drop hanya untuk mouse di desktop. Layar sentuh dan keyboard memakai tombol Pindahkan (S4-03).
const dragging = ref<BoardTask | null>(null)
const dropTarget = ref<TaskStatus | null>(null)

function canDrop(status: TaskStatus): boolean {
  return dragging.value?.move_to.includes(status) ?? false
}

// Kartu tampil terangkat (bayangan dan miring 1 derajat) hanya saat browser mengambil gambar drag-nya,
// lalu kartu asli di kolom menjadi bayangan pudar sebagai penanda asal.
const lifting = ref<string | null>(null)

function onDragStart(e: DragEvent, t: BoardTask) {
  dragging.value = t
  lifting.value = t.id
  setTimeout(() => (lifting.value = null))
  // Firefox tidak memulai drag kalau dataTransfer kosong.
  e.dataTransfer?.setData('text/plain', t.id)
  if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
}

function onDragEnd() {
  dragging.value = null
  dropTarget.value = null
}

// Selama drag: kolom tujuan yang sah bergaris putus-putus olive, kolom yang tidak boleh meredup, dan kolom yang
// sedang disentuh menunjukkan sebelum dilepas apakah perpindahan akan diterima.
function columnDragClass(status: TaskStatus): string {
  const t = dragging.value
  if (!t || t.status === status) return 'border-line'
  if (!canDrop(status))
    return dropTarget.value === status ? 'border-danger bg-danger-bg' : 'border-line opacity-60'
  return dropTarget.value === status
    ? 'border-mark bg-surface outline-2 outline-mark'
    : 'border-dashed border-mark'
}

function onDragOver(e: DragEvent, status: TaskStatus) {
  if (!dragging.value || dragging.value.status === status) return
  // Semua kolom lain menerima drop, termasuk yang tidak boleh, supaya bisa dijawab dengan pesan
  // (alur alternatif 3a), bukan diam-diam tidak terjadi apa-apa.
  e.preventDefault()
  dropTarget.value = status
}

function onDragLeave(e: DragEvent, status: TaskStatus) {
  // dragleave juga terpicu saat kursor masuk ke kartu di dalam kolom; hanya keluar kolom yang dihitung.
  const into = e.relatedTarget as Node | null
  if (dropTarget.value === status && !(e.currentTarget as HTMLElement).contains(into)) dropTarget.value = null
}

async function onDrop(e: DragEvent, status: TaskStatus) {
  e.preventDefault()
  const t = dragging.value
  onDragEnd()
  if (!t || t.status === status) return
  if (!t.move_to.includes(status)) {
    const allowed = t.move_to.map((s) => statusLabel[s]).join(' atau ')
    notice.value = {
      kind: 'error',
      text: `${t.id} tidak bisa dipindahkan ke ${statusLabel[status]}. Tujuan yang boleh: ${allowed}.`,
    }
    return
  }
  const block = status === 'ready' ? readyBlock(t) : null
  if (block) {
    notice.value = { kind: 'error', text: `${t.id}: ${block}` }
    return
  }
  // Revisi butuh catatan (PIC task, Manajer) atau pilihan sub task (PIC sub task, revisi 22), jadi tetap lewat
  // dialog dengan tujuan sudah terpilih.
  if (t.status === 'ready' && status === 'on_progress') {
    movingTarget.value = status
    moving.value = t
    return
  }
  try {
    await moveTask(t.id, t.version, status)
    notice.value = { kind: 'success', text: `${t.id} dipindahkan ke ${statusLabel[status]}.` }
  } catch (err) {
    notice.value = { kind: 'error', text: staleMoveMessage(err, t.id) ?? messageOf(err) }
  }
  await load(true)
}

// Di layar sempit hanya satu kolom yang tampil (S4-04).
const activeColumn = ref<TaskStatus>('requested')
let columnChosen = false

// Saat pertama dimuat, kolom yang dibuka adalah kolom pertama yang berisi task, bukan kolom kosong.
watch(board, (b) => {
  if (!b || columnChosen) return
  activeColumn.value = b.columns.find((c) => c.count > 0)?.status ?? 'requested'
  columnChosen = true
})

function chooseColumn(status: TaskStatus) {
  activeColumn.value = status
  columnChosen = true
}

function openCreate() {
  editing.value = null
  notice.value = null
  formOpen.value = true
}

// Panel ubah menggantikan panel detail, lalu detail dibuka lagi setelah tersimpan.
function onEdit(t: TaskDetail) {
  editing.value = t
  detailId.value = null
  formOpen.value = true
}

async function onSaved(id: string, message: string, partial: boolean) {
  formOpen.value = false
  notice.value = { kind: partial ? 'error' : 'success', text: message }
  await load(true)
  detailId.value = id
}

async function onChanged(message: string) {
  notice.value = { kind: 'success', text: message }
  await load(true)
}

const scopeButton =
  'inline-flex min-h-11 items-center px-4 font-medium first:rounded-l-card last:rounded-r-card aria-pressed:bg-accent aria-pressed:text-on-accent aria-pressed:outline aria-pressed:outline-1 aria-pressed:-outline-offset-1 aria-pressed:outline-mark'
const buttonSecondary =
  'inline-flex min-h-11 items-center rounded-card border border-line bg-surface px-4 font-medium hover:bg-canvas active:translate-y-px'
</script>

<template>
  <!-- Container query: kartu profil jadi kolom kiri hanya bila kontainer papan selebar 80rem atau lebih. -->
  <section class="@container/board">
    <div class="grid items-start gap-6 @min-[90rem]/board:grid-cols-[17rem_minmax(0,1fr)]">
      <!-- Kolom kiri: kartu profil, lalu kartu To Do. Lebarnya tetap 17rem dan kolom papan memakai
           minmax(0,1fr), jadi daftar yang panjang tidak pernah menyempitkan papan (revisi 28). -->
      <aside class="sticky top-4 hidden space-y-4 @min-[90rem]/board:block">
        <ProfileCard
          variant="card"
          :counts="counts"
          :counts-state="countsState"
          :show-counts="showCounts"
          @edit="profileOpen = true"
          @pick="pickCount"
          @retry="loadCounts"
        />
        <TodoCard />
      </aside>
      <div class="min-w-0">
        <PageHeader title="Papan Task">
          <template v-if="canCreateTask" #actions>
            <button
              type="button"
              class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-on-accent hover:bg-accent-hover active:translate-y-px"
              @click="openCreate"
            >
              Buat task
            </button>
          </template>
        </PageHeader>

        <div class="mt-4 flex flex-wrap items-start gap-2 @min-[90rem]/board:hidden">
          <TodoPanel />
          <ProfileCard
            class="min-w-0 flex-1"
            variant="bar"
            :counts="counts"
            :counts-state="countsState"
            :show-counts="showCounts"
            @edit="profileOpen = true"
            @pick="pickCount"
            @retry="loadCounts"
          />
        </div>

        <div
          v-if="isStaff"
          class="mt-4 inline-flex rounded-card border border-line bg-surface"
          role="group"
          aria-label="Lingkup task"
        >
          <button
            type="button"
            :class="scopeButton"
            :aria-pressed="query.scope === 'mine'"
            @click="setQuery({ scope: 'mine', pic: undefined })"
          >
            Task saya
          </button>
          <button
            type="button"
            :class="scopeButton"
            :aria-pressed="query.scope === 'all'"
            @click="setQuery({ scope: 'all' })"
          >
            Semua task
          </button>
        </div>

        <!-- Hanya Staf yang punya tabel aktivitas harian (revisi 21). -->
        <DailyActivityCard v-if="isStaff" />

        <FilterToolbar
          label="Filter papan"
          :active-count="activeFilterCount"
          :can-clear="filtered"
          @clear="clearFilters"
        >
          <template #search>
            <KeywordSearch id="board-search" :value="query.q" @search="setQuery({ q: $event })" />
            <!-- Papan hanya memuat Completed 7 hari terakhir, jadi task lama yang cocok baru terlihat di Arsip. -->
            <RouterLink
              v-if="query.q"
              :to="{ name: 'archive', query: { q: query.q } }"
              class="inline-flex min-h-11 items-center rounded-card text-sm font-medium text-mark underline underline-offset-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
            >
              Cari "{{ query.q }}" juga di Arsip
            </RouterLink>
          </template>
          <div v-if="showPicFilter" class="min-w-0">
            <label for="filter-pic" class="sr-only">PIC</label>
            <select
              id="filter-pic"
              aria-describedby="filter-pic-hint"
              :class="filterControl"
              :value="query.pic ?? ''"
              @change="setQuery({ pic: selected($event) || undefined })"
            >
              <option value="">Semua orang</option>
              <option v-for="p in people" :key="p.id" :value="p.username">
                {{ p.name }} ({{ p.username === staffUsername ? 'kamu' : p.division_name }})
              </option>
            </select>
            <!-- Revisi 15: selalu dibaca pembaca layar; terlihat saat filter PIC dipakai. -->
            <p id="filter-pic-hint" :class="query.pic ? 'mt-1 text-sm text-ink-muted' : 'sr-only'">
              Termasuk task yang sub task-nya dipegang orang ini.
            </p>
          </div>
          <div class="min-w-0">
            <label for="filter-division" class="sr-only">Divisi</label>
            <select
              id="filter-division"
              :class="filterControl"
              :value="query.division ?? ''"
              @change="setQuery({ division: Number(selected($event)) || undefined })"
            >
              <option value="">Semua divisi</option>
              <option v-for="d in divisions" :key="d.id" :value="d.id">
                {{ d.name }}{{ d.is_active ? '' : ' (nonaktif)' }}
              </option>
            </select>
          </div>
          <div class="min-w-0">
            <label for="filter-label" class="sr-only">Label</label>
            <select
              id="filter-label"
              :class="filterControl"
              :value="query.label ?? ''"
              @change="setQuery({ label: Number(selected($event)) || undefined })"
            >
              <option value="">Semua label</option>
              <option v-for="l in labels" :key="l.id" :value="l.id">
                {{ l.name }}{{ l.is_active ? '' : ' (nonaktif)' }}
              </option>
            </select>
          </div>
          <div class="min-w-0">
            <label for="filter-urgency" class="sr-only">Urgency</label>
            <select
              id="filter-urgency"
              :class="filterControl"
              :value="query.urgency ?? ''"
              @change="setQuery({ urgency: (selected($event) || undefined) as Urgency | undefined })"
            >
              <option value="">Semua urgency</option>
              <option v-for="u in [...urgencies].reverse()" :key="u" :value="u">{{ urgencyLabel[u] }}</option>
            </select>
          </div>
          <div class="min-w-0">
            <label for="filter-due" class="sr-only">Deadline</label>
            <select
              id="filter-due"
              :class="filterControl"
              :value="query.due ?? ''"
              @change="setQuery({ due: (selected($event) || undefined) as DueFilter | undefined })"
            >
              <option value="">Semua deadline</option>
              <option v-for="d in dueOptions" :key="d.value" :value="d.value">{{ d.label }}</option>
            </select>
          </div>
        </FilterToolbar>

        <div aria-live="polite">
          <Transition name="notice">
            <p
              v-if="notice"
              :key="notice.text"
              :role="notice.kind === 'error' ? 'alert' : 'status'"
              class="mt-4 rounded-card p-3 text-sm"
              :class="
                notice.kind === 'error'
                  ? 'bg-danger-bg text-danger'
                  : 'border border-line bg-surface text-progress-fg'
              "
            >
              {{ notice.text }}
            </p>
          </Transition>
        </div>

        <LoadingSkeleton v-if="loadState === 'loading'" class="mt-6" label="Memuat papan..." shape="board" />

        <div
          v-else-if="loadState === 'error'"
          role="alert"
          class="mt-6 rounded-card bg-danger-bg p-4 text-danger"
        >
          <p class="font-medium">{{ loadError }}</p>
          <button
            type="button"
            class="mt-3 inline-flex min-h-11 items-center rounded-card border border-danger px-4 font-medium"
            @click="load()"
          >
            Muat ulang
          </button>
        </div>

        <template v-else-if="board">
          <!-- Kolom kosong sudah memberi tahu sendiri; baris ini hanya saat filter atau pencarian membuat papan kosong. -->
          <p
            v-if="total === 0 && filtered"
            role="status"
            class="mt-6 flex flex-wrap items-center gap-3 text-ink-muted"
          >
            Tidak ada task yang cocok dengan pencarian atau filter ini.
            <button type="button" :class="buttonSecondary" @click="clearFilters">Hapus filter</button>
          </p>

          <!-- Container query, bukan breakpoint layar: lebar papan bergantung pada sidebar AppShell, bukan lebar jendela. -->
          <div class="@container mt-6">
            <div class="grid grid-cols-4 gap-1 @2xl:hidden" role="group" aria-label="Pilih kolom">
              <button
                v-for="col in board.columns"
                :key="col.status"
                type="button"
                :aria-pressed="activeColumn === col.status"
                class="flex min-h-11 min-w-0 flex-col items-center justify-center rounded-card border border-line bg-surface px-1 py-1 text-xs font-medium aria-pressed:border-mark aria-pressed:bg-accent aria-pressed:text-on-accent aria-pressed:outline aria-pressed:outline-1 aria-pressed:-outline-offset-1 aria-pressed:outline-mark"
                @click="chooseColumn(col.status)"
              >
                <span>{{ statusLabel[col.status] }}</span>
                <span class="font-mono">{{ col.count }}<span class="sr-only"> task</span></span>
              </button>
            </div>

            <div class="mt-3 grid items-start gap-4 @2xl:mt-0 @2xl:grid-cols-2 @4xl:grid-cols-4">
              <section
                v-for="col in board.columns"
                :key="col.status"
                class="min-w-0 rounded-card border bg-canvas p-2 transition-colors duration-150"
                :class="[col.status === activeColumn ? '' : 'hidden @2xl:block', columnDragClass(col.status)]"
                :aria-labelledby="`kolom-${col.status}`"
                @dragover="onDragOver($event, col.status)"
                @dragleave="onDragLeave($event, col.status)"
                @drop="onDrop($event, col.status)"
              >
                <h2 :id="`kolom-${col.status}`" class="flex min-h-11 items-center gap-2 px-1">
                  <span class="rounded-chip px-2 py-0.5 text-sm font-medium" :class="statusClass[col.status]">
                    {{ statusLabel[col.status] }}
                  </span>
                  <span class="rounded-chip border border-line bg-surface px-2 text-sm font-medium font-mono">
                    {{ col.count }}<span class="sr-only"> task</span>
                  </span>
                </h2>
                <p
                  v-if="dragging && dragging.status !== col.status"
                  class="px-1 pb-2 text-sm font-medium"
                  :class="canDrop(col.status) ? 'text-mark' : 'text-ink-muted'"
                >
                  {{
                    canDrop(col.status) ? 'Lepas di sini untuk memindahkan' : 'Tidak bisa dipindah ke sini'
                  }}
                </p>
                <p v-if="col.status === 'completed'" class="px-1 pb-2 text-sm text-ink-muted">
                  Hanya yang selesai {{ board.completed_days }} hari terakhir.
                </p>
                <!-- Kartu yang masuk kolom (mis. setelah dipindah) muncul dengan transisi singkat sebagai umpan balik. -->
                <TransitionGroup v-if="col.tasks.length" tag="ul" name="card" class="space-y-2">
                  <li
                    v-for="t in col.tasks"
                    :key="t.id"
                    :draggable="t.move_to.length > 0"
                    class="rounded-card"
                    :class="[
                      t.move_to.length ? 'cursor-grab' : '',
                      lifting === t.id ? 'rotate-1 shadow-drag' : dragging?.id === t.id ? 'opacity-40' : '',
                    ]"
                    @dragstart="onDragStart($event, t)"
                    @dragend="onDragEnd"
                  >
                    <TaskCard :task="t" @open="detailId = $event" @move="openMove" />
                  </li>
                </TransitionGroup>
                <p v-else class="px-1 py-4 text-center text-sm text-ink-muted">Tidak ada task.</p>
                <p
                  v-if="col.status === 'completed' && col.count > col.tasks.length"
                  class="px-1 pt-3 text-sm text-ink-muted"
                >
                  Menampilkan {{ col.tasks.length }} dari {{ col.count }} task.
                  <RouterLink
                    :to="archiveLink"
                    class="inline-flex min-h-11 items-center font-medium text-ink underline underline-offset-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
                  >
                    Lihat semua di Arsip
                  </RouterLink>
                </p>
              </section>
            </div>
          </div>
        </template>
      </div>
    </div>

    <BaseDialog :open="profileOpen" title="Ubah profil" variant="panel" @close="profileOpen = false">
      <!-- v-if supaya isian kembali ke data terbaru setiap panel dibuka. -->
      <ProfileForm v-if="profileOpen" @saved="onProfileSaved" />
    </BaseDialog>
    <MoveDialog
      :task="moving"
      :initial-target="movingTarget"
      @close="closeMove"
      @moved="onMoved"
      @stale="onStale"
    />
    <TaskDetailPanel
      :task-id="detailId"
      :labels="labels"
      @close="detailId = null"
      @changed="onChanged"
      @edit="onEdit"
    />

    <TaskFormPanel
      v-if="canCreateTask"
      :open="formOpen"
      :task="editing"
      :users="isManager ? users : people"
      :labels="labels"
      @close="formOpen = false"
      @saved="onSaved"
    />
  </section>
</template>
