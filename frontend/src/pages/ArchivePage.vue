<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import KeywordSearch from '../components/KeywordSearch.vue'
import MoveDialog from '../components/MoveDialog.vue'
import TaskDetailPanel from '../components/TaskDetailPanel.vue'
import TaskFormPanel from '../components/TaskFormPanel.vue'
import { ApiError } from '../api/client'
import { getArchive, type ArchiveQuery, type ArchiveResult } from '../api/archive'
import { listPeople, type BoardTask, type Person } from '../api/board'
import type { TaskDetail } from '../api/tasks'
import { listLabels, type LabelItem } from '../api/labels'
import { listDivisions, type DivisionItem } from '../api/divisions'
import { listUsers, type UserItem } from '../api/users'
import { useSessionStore } from '../stores/session'
import { hasArchiveFilters, parseArchiveQuery, sortOptions, toArchiveRouteQuery } from '../utils/archiveQuery'
import { normalizeBoardQuery, sameRouteQuery } from '../utils/boardQuery'
import { formatMoment, urgencies, urgencyClass, urgencyLabel, type Urgency } from '../utils/task'
import type { ArchiveSort } from '../api/archive'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const isManager = computed(() => session.user?.role === 'manager')
const isStaff = computed(() => session.user?.role === 'staff')
const staffUsername = computed(() => (isStaff.value ? (session.user?.username ?? null) : null))

// URL adalah sumber filter, sama seperti papan: bisa di-refresh, dibagikan, dan tombol Back tetap bekerja.
const query = computed(() => parseArchiveQuery(route.query, staffUsername.value))
const filtered = computed(() => hasArchiveFilters(query.value))
const showPicFilter = computed(() => !isStaff.value || query.value.scope === 'all')
const queryKey = computed(() => new URLSearchParams(toArchiveRouteQuery(query.value)).toString())

// Filter apa pun yang berubah kembali ke halaman 1; hanya tombol halaman yang mengirim nomor halaman.
function setQuery(patch: Partial<ArchiveQuery>) {
  void router.push({
    query: toArchiveRouteQuery(
      normalizeBoardQuery({ ...query.value, page: undefined, ...patch }, staffUsername.value),
    ),
  })
}

watch(
  () => route.query,
  (q) => {
    const canonical = toArchiveRouteQuery(query.value)
    if (!sameRouteQuery(canonical, q)) void router.replace({ query: canonical })
  },
  { immediate: true },
)

function selected(e: Event): string {
  return (e.target as HTMLSelectElement | HTMLInputElement).value
}

function clearFilters() {
  setQuery({
    q: undefined,
    pic: undefined,
    division: undefined,
    label: undefined,
    urgency: undefined,
    from: undefined,
    to: undefined,
  })
}

const result = ref<ArchiveResult | null>(null)
const loadState = ref<'loading' | 'ready' | 'error'>('loading')
const loadError = ref('')
const notice = ref<{ kind: 'success' | 'error'; text: string } | null>(null)

function messageOf(err: unknown): string {
  return err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
}

let controller: AbortController | null = null

async function load(quiet = false) {
  // Permintaan lama dibatalkan saat filter berganti, supaya jawaban yang terlambat tidak menimpa yang baru.
  controller?.abort()
  const c = new AbortController()
  controller = c
  if (!quiet) loadState.value = 'loading'
  try {
    const res = await getArchive(query.value, c.signal)
    result.value = res
    loadState.value = 'ready'
    // Halaman yang diminta sudah lewat akhir daftar dan server mengembalikan halaman 1, jadi URL diluruskan.
    if ((query.value.page ?? 1) !== res.page) {
      void router.replace({
        query: toArchiveRouteQuery({ ...query.value, page: res.page > 1 ? res.page : undefined }),
      })
    }
  } catch (err) {
    if (c.signal.aborted) return
    if (quiet && result.value) {
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
    // Daftar akun hanya bisa dibaca pengelola, dan hanya Manajer yang butuh untuk memilih PIC di form.
    if (isManager.value) users.value = await listUsers()
  } catch {
    notice.value = {
      kind: 'error',
      text: 'Pilihan filter gagal dimuat. Muat ulang halaman untuk memakai filter.',
    }
  }
})

const pageCount = computed(() =>
  result.value ? Math.max(1, Math.ceil(result.value.total / result.value.page_size)) : 1,
)
const rangeText = computed(() => {
  const r = result.value
  if (!r || r.total === 0) return ''
  const start = (r.page - 1) * r.page_size + 1
  return `${start} sampai ${start + r.tasks.length - 1} dari ${r.total} task`
})

const resultsHeading = ref<HTMLElement | null>(null)
function goPage(p: number) {
  void router.push({ query: toArchiveRouteQuery({ ...query.value, page: p > 1 ? p : undefined }) })
  // Fokus ke judul daftar supaya pengguna keyboard dan pembaca layar mulai dari awal halaman baru.
  resultsHeading.value?.focus()
}

function names(t: BoardTask): string {
  return t.assignees.map((a) => a.name).join(', ') || 'Tanpa PIC'
}

function canReopen(t: BoardTask): boolean {
  return t.move_to.includes('on_progress')
}

// Buka kembali lewat MoveDialog yang sama dengan papan: tujuannya satu, jadi sudah terpilih, dan
// dialog menjelaskan bahwa tanggal selesai akan dikosongkan sebelum pengguna menekan tombol.
const moving = ref<BoardTask | null>(null)

async function onMoved(message: string) {
  moving.value = null
  notice.value = { kind: 'success', text: message }
  await load(true)
}

async function onStale(message: string) {
  moving.value = null
  notice.value = { kind: 'error', text: message }
  await load(true)
}

const detailId = ref<string | null>(null)
const formOpen = ref(false)
const editing = ref<TaskDetail | null>(null)

async function onChanged(message: string) {
  notice.value = { kind: 'success', text: message }
  await load(true)
}

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

const selectClass = 'mt-1 block min-h-11 w-full min-w-0 rounded-card border border-line bg-surface px-3'
const scopeButton =
  'inline-flex min-h-11 items-center px-4 font-medium first:rounded-l-card last:rounded-r-card aria-pressed:bg-accent aria-pressed:text-white'
const buttonSecondary =
  'inline-flex min-h-11 items-center rounded-card border border-line bg-surface px-4 font-medium hover:bg-canvas active:translate-y-px disabled:opacity-60'
</script>

<template>
  <section>
    <h1 class="text-2xl font-semibold">Arsip</h1>
    <p class="mt-1 text-ink-muted">
      Semua task yang sudah Completed, termasuk yang tidak lagi tampil di papan.
    </p>

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

    <div class="mt-4 max-w-xl">
      <KeywordSearch id="archive-search" :value="query.q" @search="setQuery({ q: $event })" />
    </div>

    <div
      class="mt-4 grid items-end gap-3 [grid-template-columns:repeat(auto-fit,minmax(10rem,1fr))]"
      role="group"
      aria-label="Filter arsip"
    >
      <div v-if="showPicFilter" class="min-w-0">
        <label for="archive-pic" class="text-sm font-medium">PIC</label>
        <select
          id="archive-pic"
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
        <label for="archive-division" class="text-sm font-medium">Divisi</label>
        <select
          id="archive-division"
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
        <label for="archive-label" class="text-sm font-medium">Label</label>
        <select
          id="archive-label"
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
        <label for="archive-urgency" class="text-sm font-medium">Urgency</label>
        <select
          id="archive-urgency"
          :class="selectClass"
          :value="query.urgency ?? ''"
          @change="setQuery({ urgency: (selected($event) || undefined) as Urgency | undefined })"
        >
          <option value="">Semua urgency</option>
          <option v-for="u in [...urgencies].reverse()" :key="u" :value="u">{{ urgencyLabel[u] }}</option>
        </select>
      </div>
      <div class="min-w-0">
        <label for="archive-from" class="text-sm font-medium">Selesai dari</label>
        <input
          id="archive-from"
          type="date"
          :class="selectClass"
          :value="query.from ?? ''"
          :max="query.to"
          @change="setQuery({ from: selected($event) || undefined })"
        />
      </div>
      <div class="min-w-0">
        <label for="archive-to" class="text-sm font-medium">Selesai sampai</label>
        <input
          id="archive-to"
          type="date"
          :class="selectClass"
          :value="query.to ?? ''"
          :min="query.from"
          @change="setQuery({ to: selected($event) || undefined })"
        />
      </div>
      <div class="min-w-0">
        <label for="archive-sort" class="text-sm font-medium">Urutkan</label>
        <select
          id="archive-sort"
          :class="selectClass"
          :value="query.sort ?? 'newest'"
          @change="setQuery({ sort: selected($event) as ArchiveSort })"
        >
          <option v-for="s in sortOptions" :key="s.value" :value="s.value">{{ s.label }}</option>
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

    <h2 ref="resultsHeading" tabindex="-1" class="sr-only">Daftar task selesai</h2>

    <p v-if="loadState === 'loading'" class="mt-6 text-ink-muted">Memuat arsip...</p>

    <div
      v-else-if="loadState === 'error'"
      role="alert"
      class="mt-6 rounded-card bg-danger-bg p-4 text-danger"
    >
      <p class="font-medium">{{ loadError }}</p>
      <div class="mt-3 flex flex-wrap gap-2">
        <button
          type="button"
          class="inline-flex min-h-11 items-center rounded-card border border-danger px-4 font-medium"
          @click="load()"
        >
          Muat ulang
        </button>
        <button v-if="filtered" type="button" :class="buttonSecondary" @click="clearFilters">
          Hapus filter
        </button>
      </div>
    </div>

    <template v-else-if="result">
      <div v-if="result.total === 0" class="mt-6 rounded-card border border-line bg-surface p-6 text-center">
        <template v-if="filtered">
          <p>Tidak ada task selesai yang cocok dengan pencarian atau filter ini.</p>
          <button type="button" :class="buttonSecondary" class="mt-3" @click="clearFilters">
            Hapus filter
          </button>
        </template>
        <template v-else-if="query.scope === 'mine'">
          <p>Belum ada task selesai yang PIC-nya kamu.</p>
          <button type="button" :class="buttonSecondary" class="mt-3" @click="setQuery({ scope: 'all' })">
            Lihat semua task
          </button>
        </template>
        <p v-else>Belum ada task yang selesai.</p>
      </div>

      <div v-else class="@container mt-6">
        <p class="text-sm text-ink-muted">{{ rangeText }}</p>

        <!-- Tabel hanya bila kontainer cukup lebar; di bawah itu kartu, supaya tidak ada gulir horizontal. -->
        <table class="mt-2 hidden w-full text-sm @3xl:table">
          <caption class="sr-only">
            Task selesai,
            {{
              rangeText
            }}
          </caption>
          <thead class="border-b border-line text-left text-ink-muted">
            <tr>
              <th scope="col" class="py-2 pr-3 font-medium">ID</th>
              <th scope="col" class="py-2 pr-3 font-medium">Judul</th>
              <th scope="col" class="py-2 pr-3 font-medium">PIC</th>
              <th scope="col" class="py-2 pr-3 font-medium">Divisi</th>
              <th scope="col" class="py-2 pr-3 font-medium">Urgency</th>
              <th scope="col" class="py-2 pr-3 font-medium">Selesai</th>
              <th scope="col" class="py-2"><span class="sr-only">Aksi</span></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-line">
            <tr v-for="t in result.tasks" :key="t.id" class="align-top">
              <td class="py-3 pr-3 font-mono whitespace-nowrap text-ink-muted">{{ t.id }}</td>
              <td class="min-w-0 py-1 pr-3">
                <button
                  type="button"
                  class="min-h-11 rounded-card text-left font-medium break-words hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
                  @click="detailId = t.id"
                >
                  {{ t.title }}
                </button>
                <div v-if="t.labels.length" class="flex flex-wrap gap-1 pb-2">
                  <span
                    v-for="l in t.labels"
                    :key="l.id"
                    class="rounded-chip px-2 py-0.5"
                    :style="{ backgroundColor: l.color }"
                    >{{ l.name }}</span
                  >
                </div>
              </td>
              <td class="py-3 pr-3">{{ names(t) }}</td>
              <td class="py-3 pr-3">{{ t.division.name }}</td>
              <td class="py-3 pr-3" :class="urgencyClass[t.urgency]">{{ urgencyLabel[t.urgency] }}</td>
              <td class="py-3 pr-3 whitespace-nowrap">
                {{ t.solved_date ? formatMoment(t.solved_date) : '' }}
              </td>
              <td class="py-1 text-right">
                <button v-if="canReopen(t)" type="button" :class="buttonSecondary" @click="moving = t">
                  Buka kembali<span class="sr-only"> {{ t.id }}</span>
                </button>
              </td>
            </tr>
          </tbody>
        </table>

        <ul class="mt-2 space-y-2 @3xl:hidden">
          <li v-for="t in result.tasks" :key="t.id" class="rounded-card border border-line bg-surface p-3">
            <div class="flex items-center gap-2 text-sm">
              <span class="rounded-chip bg-canvas px-1.5 font-mono text-ink-muted">{{ t.id }}</span>
              <span :class="urgencyClass[t.urgency]">{{ urgencyLabel[t.urgency] }}</span>
            </div>
            <h3 class="font-medium">
              <button
                type="button"
                class="min-h-11 rounded-card text-left break-words focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
                @click="detailId = t.id"
              >
                {{ t.title }}
              </button>
            </h3>
            <dl class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 text-sm">
              <dt class="text-ink-muted">PIC</dt>
              <dd>{{ names(t) }}</dd>
              <dt class="text-ink-muted">Divisi</dt>
              <dd>{{ t.division.name }}</dd>
              <dt class="text-ink-muted">Selesai</dt>
              <dd>{{ t.solved_date ? formatMoment(t.solved_date) : '' }}</dd>
            </dl>
            <button
              v-if="canReopen(t)"
              type="button"
              :class="buttonSecondary"
              class="mt-2 w-full justify-center"
              @click="moving = t"
            >
              Buka kembali<span class="sr-only"> {{ t.id }}</span>
            </button>
          </li>
        </ul>

        <nav
          v-if="pageCount > 1"
          aria-label="Halaman arsip"
          class="mt-4 flex flex-wrap items-center justify-between gap-3"
        >
          <button
            type="button"
            :class="buttonSecondary"
            :disabled="result.page <= 1"
            @click="goPage(result.page - 1)"
          >
            Sebelumnya
          </button>
          <p class="text-sm text-ink-muted">Halaman {{ result.page }} dari {{ pageCount }}</p>
          <button
            type="button"
            :class="buttonSecondary"
            :disabled="result.page >= pageCount"
            @click="goPage(result.page + 1)"
          >
            Berikutnya
          </button>
        </nav>
      </div>
    </template>

    <MoveDialog :task="moving" @close="moving = null" @moved="onMoved" @stale="onStale" />
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
