<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import TaskCard from '../components/TaskCard.vue'
import TaskDetailPanel from '../components/TaskDetailPanel.vue'
import TaskFormPanel from '../components/TaskFormPanel.vue'
import { ApiError } from '../api/client'
import { getBoard, listPeople, type Board, type BoardQuery, type Person } from '../api/board'
import type { TaskDetail } from '../api/tasks'
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
import { statusClass, statusLabel, urgencies, urgencyLabel, type Urgency } from '../utils/task'
import type { DueFilter } from '../api/board'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const isManager = computed(() => session.user?.role === 'manager')
const isStaff = computed(() => session.user?.role === 'staff')

// URL adalah sumber filter: bisa di-refresh, dibagikan, dan tombol Back browser tetap bekerja.
const staffUsername = computed(() => (isStaff.value ? (session.user?.username ?? null) : null))

// URL adalah sumber filter: bisa di-refresh, dibagikan, dan tombol Back browser tetap bekerja.
const query = computed(() => parseBoardQuery(route.query, staffUsername.value))
const filtered = computed(() => hasFilters(query.value))
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
  setQuery({ pic: undefined, division: undefined, label: undefined, urgency: undefined, due: undefined })
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

const labels = ref<LabelItem[]>([])
const divisions = ref<DivisionItem[]>([])
const people = ref<Person[]>([])
const users = ref<UserItem[]>([])

onMounted(async () => {
  void load()
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

async function onSaved(id: string, message: string) {
  formOpen.value = false
  notice.value = { kind: 'success', text: message }
  await load(true)
  detailId.value = id
}

async function onChanged(message: string) {
  notice.value = { kind: 'success', text: message }
  await load(true)
}

const selectClass = 'mt-1 block min-h-11 w-full min-w-0 rounded-card border border-line bg-surface px-3'
const scopeButton =
  'inline-flex min-h-11 items-center px-4 font-medium first:rounded-l-card last:rounded-r-card aria-pressed:bg-accent aria-pressed:text-white'
const buttonSecondary =
  'inline-flex min-h-11 items-center rounded-card border border-line bg-surface px-4 font-medium hover:bg-canvas active:translate-y-px'
</script>

<template>
  <section>
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="text-2xl font-semibold">Papan Task</h1>
      <button
        v-if="isManager"
        type="button"
        class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-white active:translate-y-px"
        @click="openCreate"
      >
        Buat task
      </button>
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

    <div
      class="mt-4 grid items-end gap-3 [grid-template-columns:repeat(auto-fit,minmax(10rem,1fr))]"
      role="group"
      aria-label="Filter papan"
    >
      <div v-if="showPicFilter" class="min-w-0">
        <label for="filter-pic" class="text-sm font-medium">PIC</label>
        <select
          id="filter-pic"
          :class="selectClass"
          :value="query.pic ?? ''"
          @change="setQuery({ pic: selected($event) || undefined })"
        >
          <option value="">Semua orang</option>
          <option v-for="p in people" :key="p.id" :value="p.username">
            {{ p.name }} ({{ p.username === staffUsername ? 'kamu' : p.division_name }})
          </option>
        </select>
      </div>
      <div class="min-w-0">
        <label for="filter-division" class="text-sm font-medium">Divisi</label>
        <select
          id="filter-division"
          :class="selectClass"
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
        <label for="filter-label" class="text-sm font-medium">Label</label>
        <select
          id="filter-label"
          :class="selectClass"
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
        <label for="filter-urgency" class="text-sm font-medium">Urgency</label>
        <select
          id="filter-urgency"
          :class="selectClass"
          :value="query.urgency ?? ''"
          @change="setQuery({ urgency: (selected($event) || undefined) as Urgency | undefined })"
        >
          <option value="">Semua urgency</option>
          <option v-for="u in [...urgencies].reverse()" :key="u" :value="u">{{ urgencyLabel[u] }}</option>
        </select>
      </div>
      <div class="min-w-0">
        <label for="filter-due" class="text-sm font-medium">Tenggat</label>
        <select
          id="filter-due"
          :class="selectClass"
          :value="query.due ?? ''"
          @change="setQuery({ due: (selected($event) || undefined) as DueFilter | undefined })"
        >
          <option value="">Semua tenggat</option>
          <option v-for="d in dueOptions" :key="d.value" :value="d.value">{{ d.label }}</option>
        </select>
      </div>
      <div v-if="filtered">
        <button type="button" :class="buttonSecondary" @click="clearFilters">Hapus filter</button>
      </div>
    </div>

    <div aria-live="polite">
      <p
        v-if="notice"
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
    </div>

    <p v-if="loadState === 'loading'" class="mt-6 text-ink-muted">Memuat papan...</p>

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
      <div v-if="total === 0" class="mt-6 rounded-card border border-line bg-surface p-6 text-center">
        <template v-if="filtered">
          <p>Tidak ada task yang cocok dengan filter ini.</p>
          <button type="button" :class="buttonSecondary" class="mt-3" @click="clearFilters">
            Hapus filter
          </button>
        </template>
        <template v-else-if="query.scope === 'mine'">
          <p>Belum ada task untukmu. Task baru dari Manajer akan muncul di sini.</p>
          <button type="button" :class="buttonSecondary" class="mt-3" @click="setQuery({ scope: 'all' })">
            Lihat semua task
          </button>
        </template>
        <p v-else>Belum ada task aktif.</p>
      </div>

      <!-- Container query, bukan breakpoint layar: lebar papan bergantung pada sidebar AppShell, bukan lebar jendela. -->
      <div class="@container mt-6">
        <div class="grid items-start gap-4 @2xl:grid-cols-2 @4xl:grid-cols-4">
          <section
            v-for="col in board.columns"
            :key="col.status"
            class="min-w-0 rounded-card border border-line bg-canvas p-2"
            :aria-labelledby="`kolom-${col.status}`"
          >
            <h2 :id="`kolom-${col.status}`" class="flex min-h-11 items-center gap-2 px-1">
              <span class="rounded-chip px-2 py-0.5 text-sm font-medium" :class="statusClass[col.status]">
                {{ statusLabel[col.status] }}
              </span>
              <span class="rounded-chip border border-line bg-surface px-2 font-mono text-sm">
                {{ col.count }}<span class="sr-only"> task</span>
              </span>
            </h2>
            <p v-if="col.status === 'completed'" class="px-1 pb-2 text-sm text-ink-muted">
              Hanya yang selesai {{ board.completed_days }} hari terakhir.
            </p>
            <ul v-if="col.tasks.length" class="space-y-2">
              <li v-for="t in col.tasks" :key="t.id">
                <TaskCard :task="t" @open="detailId = $event" />
              </li>
            </ul>
            <p v-else class="px-1 py-4 text-center text-sm text-ink-muted">Tidak ada task.</p>
          </section>
        </div>
      </div>
    </template>

    <TaskDetailPanel
      :task-id="detailId"
      :labels="labels"
      @close="detailId = null"
      @changed="onChanged"
      @edit="onEdit"
    />

    <TaskFormPanel
      v-if="isManager"
      :open="formOpen"
      :task="editing"
      :users="users"
      :labels="labels"
      @close="formOpen = false"
      @saved="onSaved"
    />
  </section>
</template>
