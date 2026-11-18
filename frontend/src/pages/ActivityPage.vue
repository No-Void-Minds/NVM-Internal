<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageHeader from '../components/PageHeader.vue'
import LoadingSkeleton from '../components/LoadingSkeleton.vue'
import PersonChip from '../components/PersonChip.vue'
import UserAvatar from '../components/UserAvatar.vue'
import BaseDialog from '../components/BaseDialog.vue'
import ActivityFiles from '../components/ActivityFiles.vue'
import ActivityCellView from '../components/ActivityCellView.vue'
import ActivityOutputView from '../components/ActivityOutputView.vue'
import { ApiError } from '../api/client'
import {
  activityExportUrl,
  getActivityRange,
  saveActivity,
  type ActivityFile,
  type ActivityPerson,
} from '../api/activities'
import { listDivisions, type DivisionItem } from '../api/divisions'
import { useSessionStore } from '../stores/session'
import {
  dayLabel,
  isDay,
  monthRange,
  rangeLabel,
  shiftDay,
  shortDayLabel,
  weekDays,
  weekStart,
  type ActivityMode,
} from '../utils/activityWeek'
import { officeToday } from '../utils/recurrence'

// Halaman Aktivitas Harian (revisi 21): satu kolom per orang, bawaan satu minggu kerja. Semua peran boleh
// membaca, hanya Manajer yang boleh mengubah (can_edit dari server, bukan dihitung di sini).
const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const isManager = computed(() => session.user?.role === 'manager')

function queryValue(key: string): string {
  const v = route.query[key]
  const raw = Array.isArray(v) ? v[0] : v
  return typeof raw === 'string' ? raw : ''
}

// Mode dan tanggal ada di URL supaya tampilan yang sama bisa dibagikan, mis. ?mode=minggu&tanggal=2026-11-16.
const mode = computed<ActivityMode>(() => (queryValue('mode') === 'hari' ? 'hari' : 'minggu'))
const anchor = computed(() => {
  const v = queryValue('tanggal')
  return isDay(v) ? v : officeToday()
})
const divisionFilter = computed(() => queryValue('divisi'))
const personFilter = computed(() =>
  queryValue('orang')
    .split(',')
    .map(Number)
    .filter((n) => Number.isInteger(n) && n > 0),
)

const dates = ref<string[]>([])
const slots = ref<string[]>([])
const people = ref<ActivityPerson[]>([])
const canEdit = ref(false)
const today = ref(officeToday())
const divisions = ref<DivisionItem[]>([])
const loadState = ref<'loading' | 'ready' | 'error'>('loading')
const loadError = ref('')

const label = computed(() =>
  mode.value === 'hari' ? dayLabel(anchor.value) : rangeLabel(weekStart(anchor.value), weekEnd.value),
)
const weekEnd = computed(() => shiftDay(weekStart(anchor.value), 4))
const visible = computed(() =>
  people.value.filter(
    (p) =>
      (!divisionFilter.value || p.division_name === divisionFilter.value) &&
      (personFilter.value.length === 0 || personFilter.value.includes(p.id)),
  ),
)

function setQuery(patch: Record<string, string | undefined>) {
  const next: Record<string, string> = {}
  for (const [k, v] of Object.entries({ ...route.query, ...patch })) {
    if (typeof v === 'string' && v !== '') next[k] = v
  }
  void router.push({ query: next })
}

let controller: AbortController | null = null

async function load() {
  controller?.abort()
  const c = new AbortController()
  controller = c
  loadState.value = 'loading'
  try {
    const res = await getActivityRange(
      mode.value === 'hari' ? { date: anchor.value } : { week: anchor.value },
      c.signal,
    )
    dates.value = res.dates
    slots.value = res.slots
    people.value = res.people
    canEdit.value = res.can_edit
    today.value = res.today
    loadState.value = 'ready'
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') return
    loadError.value = err instanceof ApiError ? err.message : 'Tidak bisa memuat aktivitas.'
    loadState.value = 'error'
  }
}

watch([mode, anchor], () => void load(), { immediate: true })

// Daftar divisi hanya pelengkap filter; kalau gagal, tabelnya tetap tampil.
void listDivisions()
  .then((d) => (divisions.value = d))
  .catch(() => (divisions.value = []))

function step(arah: number) {
  setQuery({ tanggal: shiftDay(anchor.value, mode.value === 'hari' ? arah : arah * 7) })
}

function summary(p: ActivityPerson): string {
  return p.filled === 0 ? 'Belum diisi' : `${p.filled} dari ${p.total} jam terisi`
}

// Filter orang: daftar centang dengan pencarian, dibuka dari satu tombol (pola disclosure, cukup Tab,
// Enter atau Space, dan Escape).
const peopleOpen = ref(false)
const peopleSearch = ref('')
const peopleChoices = computed(() => {
  const q = peopleSearch.value.trim().toLowerCase()
  return people.value.filter((p) => !q || p.name.toLowerCase().includes(q))
})

const peopleBox = ref<HTMLElement | null>(null)
const peopleButton = ref<HTMLButtonElement | null>(null)

function closePeople(returnFocus: boolean) {
  peopleOpen.value = false
  if (returnFocus) peopleButton.value?.focus()
}

// Klik di luar tombol dan panel menutup panel. Fokus tidak dipindah, karena pengguna sedang menuju tempat lain.
function onOutsidePointer(e: PointerEvent) {
  if (peopleBox.value && !peopleBox.value.contains(e.target as Node)) closePeople(false)
}
watch(peopleOpen, (open) => {
  if (open) document.addEventListener('pointerdown', onOutsidePointer)
  else document.removeEventListener('pointerdown', onOutsidePointer)
})
onBeforeUnmount(() => document.removeEventListener('pointerdown', onOutsidePointer))

function togglePerson(id: number) {
  const next = personFilter.value.includes(id)
    ? personFilter.value.filter((v) => v !== id)
    : [...personFilter.value, id]
  setQuery({ orang: next.join(',') })
}

// Dialog sel: selalu dibuka dalam mode lihat untuk semua peran. Manajer (can_edit dari server) bisa pindah ke
// mode ubah lewat tombol Ubah; Batal dan simpan yang berhasil kembali ke mode lihat dengan isi terbaru.
type CellDialog = {
  personId: number
  personName: string
  date: string
  slot: number
  time: string
  activity: string
  output: string
  updatedBy: string
  files: ActivityFile[]
}
const cellDialog = ref<CellDialog | null>(null)
const cellSaving = ref(false)
const cellError = ref('')
const editing = ref(false)
const draft = ref({ activity: '', output: '' })
const editButton = ref<HTMLButtonElement | null>(null)
const activityInput = ref<HTMLTextAreaElement | null>(null)
let lastTrigger: HTMLElement | null = null

const dirty = computed(
  () =>
    !!cellDialog.value &&
    (draft.value.activity !== cellDialog.value.activity || draft.value.output !== cellDialog.value.output),
)
// Petunjuk ke papan task hanya berguna bagi Staf yang melihat isiannya sendiri.
const ownCell = computed(
  () => session.user?.role === 'staff' && cellDialog.value?.personId === session.user.id,
)

function openCell(p: ActivityPerson, date: string, slot: number, e: Event) {
  const cell = p.days[date]?.[slot - 1]
  if (!cell) return
  lastTrigger = e.currentTarget instanceof HTMLElement ? e.currentTarget : null
  cellError.value = ''
  editing.value = false
  cellDialog.value = {
    personId: p.id,
    personName: p.name,
    date,
    slot,
    time: cell.time,
    activity: cell.activity,
    output: cell.output,
    updatedBy: cell.updated_by,
    files: cell.files,
  }
}

// Lampiran berubah di dalam dialog (revisi 24): tabelnya dimuat ulang, lalu daftar di dialog disegarkan dari
// data baru supaya jumlah di sel dan isi dialog tidak pernah berbeda.
async function refreshFiles() {
  const d = cellDialog.value
  await load()
  if (!d || !cellDialog.value) return
  const cell = people.value.find((p) => p.id === d.personId)?.days[d.date]?.[d.slot - 1]
  cellDialog.value.files = cell?.files ?? []
}

function startEdit() {
  const d = cellDialog.value
  if (!d) return
  draft.value = { activity: d.activity, output: d.output }
  cellError.value = ''
  editing.value = true
  void nextTick(() => activityInput.value?.focus())
}

function stopEdit() {
  editing.value = false
  cellError.value = ''
  void nextTick(() => editButton.value?.focus())
}

// Escape bertahap di mode ubah: perubahan yang belum disimpan dibuang dulu dengan kembali ke mode lihat, baru
// Escape berikutnya menutup dialog. Tombol Tutup selalu langsung menutup.
function onCellClose(reason?: 'escape') {
  if (reason === 'escape' && editing.value && dirty.value) {
    stopEdit()
    return
  }
  closeCell()
}

function closeCell() {
  cellDialog.value = null
  editing.value = false
  // Fokus kembali ke sel yang dibuka, supaya posisi di tabel tidak hilang.
  void nextTick(() => lastTrigger?.focus())
}

async function saveCell() {
  const d = cellDialog.value
  if (!d) return
  cellSaving.value = true
  cellError.value = ''
  try {
    const res = await saveActivity({
      user_id: d.personId,
      date: d.date,
      slot: d.slot,
      activity: draft.value.activity,
      output: draft.value.output,
    })
    const person = people.value.find((p) => p.id === d.personId)
    const cells = person?.days[d.date]
    const fresh = res.cells[d.slot - 1]
    if (person && cells && fresh) {
      const terisiSebelum = cells[d.slot - 1].activity !== '' || cells[d.slot - 1].output !== ''
      const terisiSesudah = fresh.activity !== '' || fresh.output !== ''
      cells[d.slot - 1] = fresh
      // Ringkasan kolom ikut berubah tanpa memuat ulang seluruh tabel.
      if (terisiSebelum !== terisiSesudah) person.filled += terisiSesudah ? 1 : -1
    }
    if (fresh) {
      d.activity = fresh.activity
      d.output = fresh.output
      d.updatedBy = fresh.updated_by
    }
    stopEdit()
  } catch (err) {
    cellError.value = err instanceof ApiError ? err.message : 'Tidak bisa menyimpan. Coba lagi.'
  } finally {
    cellSaving.value = false
  }
}

// Ekspor Excel: rentang bawaannya mengikuti tampilan yang sedang dibuka.
const exportOpen = ref(false)
const exportFrom = ref('')
const exportTo = ref('')
const exportError = ref('')

const exportDays = computed(() => {
  if (!isDay(exportFrom.value) || !isDay(exportTo.value)) return 0
  const from = new Date(`${exportFrom.value}T00:00:00Z`).getTime()
  const to = new Date(`${exportTo.value}T00:00:00Z`).getTime()
  return Math.round((to - from) / 86_400_000) + 1
})

function openExport() {
  exportError.value = ''
  if (mode.value === 'hari') {
    exportFrom.value = anchor.value
    exportTo.value = anchor.value
  } else {
    exportFrom.value = weekStart(anchor.value)
    exportTo.value = weekEnd.value
  }
  exportOpen.value = true
}

function thisMonth() {
  const r = monthRange(anchor.value)
  exportFrom.value = r.from
  exportTo.value = r.to
}

function download() {
  if (exportDays.value < 1) {
    exportError.value = 'Tanggal akhir tidak boleh lebih awal dari tanggal awal.'
    return
  }
  if (exportDays.value > 31) {
    exportError.value = 'Rentang ekspor maksimal 31 hari.'
    return
  }
  exportError.value = ''
  window.location.assign(activityExportUrl(exportFrom.value, exportTo.value))
  exportOpen.value = false
}

const navButton =
  'inline-flex min-h-11 items-center rounded-card border border-line bg-surface px-4 font-medium hover:bg-canvas active:translate-y-px focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
// Pola segmented sama dengan "Task saya" dan "Semua task" di papan (Sprint 11c).
const segment =
  'inline-flex min-h-11 items-center px-4 font-medium first:rounded-l-card last:rounded-r-card aria-pressed:bg-accent aria-pressed:text-on-accent aria-pressed:outline aria-pressed:outline-1 aria-pressed:-outline-offset-1 aria-pressed:outline-mark focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
// Tangga lapisan saat tabel digulir ke dua arah: sudut kiri atas (40) di atas kepala nama (30), kepala di atas
// kolom jam (20), kolom jam di atas label hari (10). Semuanya berlatar opak supaya isi di belakangnya tidak
// tembus. Kelas sudut ditulis sendiri: dua utility z-* di satu elemen tidak pasti yang mana yang menang.
const headCell = 'sticky top-0 z-30 bg-surface px-3 py-2 text-left align-bottom'
const cornerCell = 'sticky top-0 left-0 z-40 w-28 bg-surface px-3 py-2 text-left align-bottom'
const timeCell = 'sticky left-0 z-20 w-28 px-3 py-2 text-left font-medium whitespace-nowrap'
</script>

<template>
  <section>
    <PageHeader title="Aktivitas Harian" description="Apa yang dikerjakan tim per jam, beserta hasilnya.">
      <template v-if="isManager" #actions>
        <button
          type="button"
          class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-on-accent hover:bg-accent-hover active:translate-y-px"
          @click="openExport"
        >
          Ekspor Excel
        </button>
      </template>
    </PageHeader>

    <div class="mt-4 flex flex-wrap items-start gap-2">
      <div class="inline-flex rounded-card border border-line bg-surface" role="group" aria-label="Tampilan">
        <button
          type="button"
          :class="segment"
          :aria-pressed="mode === 'minggu'"
          @click="setQuery({ mode: 'minggu' })"
        >
          Minggu
        </button>
        <button
          type="button"
          :class="segment"
          :aria-pressed="mode === 'hari'"
          @click="setQuery({ mode: 'hari' })"
        >
          Hari
        </button>
      </div>

      <nav class="flex flex-wrap items-center gap-2" aria-label="Pindah tanggal">
        <button type="button" :class="navButton" @click="step(-1)">
          {{ mode === 'hari' ? 'Hari sebelumnya' : 'Minggu sebelumnya' }}
        </button>
        <button type="button" :class="navButton" @click="setQuery({ tanggal: today })">
          {{ mode === 'hari' ? 'Hari ini' : 'Minggu ini' }}
        </button>
        <button type="button" :class="navButton" @click="step(1)">
          {{ mode === 'hari' ? 'Hari berikutnya' : 'Minggu berikutnya' }}
        </button>
        <div class="flex min-h-11 items-center gap-2 rounded-card border border-line bg-surface pl-3">
          <label for="aktivitas-tanggal" class="text-sm text-ink-muted">Tanggal</label>
          <input
            id="aktivitas-tanggal"
            type="date"
            :value="anchor"
            class="min-h-11 min-w-0 rounded-r-card bg-transparent px-2 text-base text-ink"
            @change="setQuery({ tanggal: ($event.target as HTMLInputElement).value || today })"
          />
        </div>
      </nav>

      <div>
        <label for="aktivitas-divisi" class="sr-only">Divisi</label>
        <select
          id="aktivitas-divisi"
          class="block min-h-11 w-full min-w-0 rounded-card border border-line bg-surface px-3 md:w-auto"
          :value="divisionFilter"
          @change="setQuery({ divisi: ($event.target as HTMLSelectElement).value })"
        >
          <option value="">Semua divisi</option>
          <option v-for="d in divisions" :key="d.id" :value="d.name">{{ d.name }}</option>
        </select>
      </div>

      <!-- Escape dipasang di pembungkus, bukan di panel, supaya juga bekerja saat fokus masih di tombol. -->
      <div
        ref="peopleBox"
        class="relative w-full sm:w-auto"
        @keydown.escape="peopleOpen && closePeople(true)"
      >
        <button
          ref="peopleButton"
          type="button"
          :class="navButton"
          :aria-expanded="peopleOpen"
          aria-controls="aktivitas-orang"
          @click="peopleOpen = !peopleOpen"
        >
          {{ personFilter.length ? `Orang (${personFilter.length})` : 'Orang' }}
        </button>
        <div
          v-if="peopleOpen"
          id="aktivitas-orang"
          role="group"
          aria-label="Filter orang"
          class="z-30 mt-1 w-full rounded-card border border-line bg-surface p-3 shadow-dialog sm:absolute sm:w-72"
        >
          <label for="aktivitas-cari-orang" class="sr-only">Cari nama</label>
          <input
            id="aktivitas-cari-orang"
            v-model="peopleSearch"
            type="search"
            placeholder="Cari nama"
            class="block min-h-11 w-full rounded-card border border-line bg-surface px-3"
          />
          <ul class="mt-2 max-h-64 overflow-y-auto">
            <li v-for="p in peopleChoices" :key="p.id">
              <label class="flex min-h-11 items-center gap-2 px-1">
                <input
                  type="checkbox"
                  class="size-5"
                  :checked="personFilter.includes(p.id)"
                  @change="togglePerson(p.id)"
                />
                <span class="min-w-0 break-words">{{ p.name }}</span>
              </label>
            </li>
            <li v-if="peopleChoices.length === 0" class="px-1 py-2 text-sm text-ink-muted">
              Tidak ada nama yang cocok.
            </li>
          </ul>
          <div class="mt-2 flex justify-between gap-2">
            <button
              v-if="personFilter.length"
              type="button"
              :class="navButton"
              @click="setQuery({ orang: undefined })"
            >
              Semua orang
            </button>
            <button type="button" :class="navButton" @click="closePeople(true)">Tutup</button>
          </div>
        </div>
      </div>
    </div>

    <div
      v-if="mode === 'hari'"
      class="mt-3 flex flex-wrap gap-2"
      role="group"
      aria-label="Hari dalam minggu ini"
    >
      <button
        v-for="d in weekDays(anchor)"
        :key="d"
        type="button"
        class="inline-flex min-h-11 flex-col items-center justify-center rounded-card border border-line bg-surface px-3 py-1 leading-tight aria-pressed:bg-accent aria-pressed:text-on-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
        :aria-pressed="d === anchor"
        @click="setQuery({ tanggal: d })"
      >
        <span class="font-medium">{{ shortDayLabel(d) }}</span>
        <span class="text-sm">{{ Number(d.slice(8, 10)) }}</span>
      </button>
    </div>

    <p class="mt-3 font-medium">{{ label }}</p>
    <p v-if="loadState === 'ready' && session.user?.role === 'staff'" class="mt-1 text-sm text-ink-muted">
      Ubah isianmu lewat tabel di papan task.
    </p>

    <LoadingSkeleton v-if="loadState === 'loading'" class="mt-4" label="Memuat aktivitas..." />

    <div
      v-else-if="loadState === 'error'"
      role="alert"
      class="mt-4 rounded-card bg-danger-bg p-4 text-danger"
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

    <p
      v-else-if="visible.length === 0"
      class="mt-4 rounded-card border border-line bg-surface p-6 text-center"
    >
      {{
        divisionFilter || personFilter.length
          ? 'Tidak ada Staf yang cocok dengan filter ini.'
          : 'Belum ada akun Staf yang aktif.'
      }}
    </p>

    <!-- Bisa terjadi bila rentang di URL hanya memuat Sabtu dan Minggu yang kosong: keduanya disaring,
         jadi tidak ada baris yang bisa ditampilkan. -->
    <p v-else-if="dates.length === 0" class="mt-4 rounded-card border border-line bg-surface p-6 text-center">
      Tidak ada hari yang bisa ditampilkan di rentang ini. Sabtu dan Minggu hanya tampil kalau ada isinya.
    </p>

    <!-- Tabel bergulir di dalam wadahnya sendiri, ke samping dan ke bawah, jadi halaman tidak pernah
         bergulir horizontal. Kepala nama lengket di atas, kolom jam lengket di kiri. -->
    <!-- isolate membuat wadah tabel jadi stacking context sendiri: tangga z-index sel lengket di dalamnya
         (sampai z-40) hanya berlaku di dalam tabel, jadi panel filter Orang (z-30) tetap di atas seluruh tabel. -->
    <div
      v-else
      class="isolate mt-4 max-h-[calc(100dvh-20rem)] overflow-auto rounded-card border border-line bg-surface"
    >
      <table class="w-full border-collapse text-left align-top text-sm">
        <thead>
          <tr>
            <th scope="col" :class="cornerCell">Jam</th>
            <th
              v-for="p in visible"
              :key="p.id"
              scope="col"
              :class="[headCell, 'min-w-[16rem] border-l border-line']"
            >
              <PersonChip :user-id="p.id" :name="p.name">
                <UserAvatar :user-id="p.id" :name="p.name" :avatar-id="p.avatar_id" size="sm" decorative />
                <span class="font-medium">{{ p.name }}</span>
              </PersonChip>
              <span class="mt-1 block text-ink-muted">
                {{ p.division_name }}<template v-if="!p.is_active">, nonaktif</template>
              </span>
              <span class="mt-1 block text-ink-muted">{{ summary(p) }}</span>
              <!-- Bar ringkasan memakai token progres yang sudah ada, bukan warna baru. -->
              <span class="mt-1 block h-1 w-full rounded-chip bg-progress-bg">
                <span
                  class="block h-1 rounded-chip bg-progress-fg"
                  :style="{ width: `${p.total ? Math.round((p.filled / p.total) * 100) : 0}%` }"
                ></span>
              </span>
            </th>
          </tr>
        </thead>
        <tbody v-for="date in dates" :key="date" :class="date === today ? 'bg-canvas' : ''">
          <tr>
            <!-- Sel ini selebar tabel, jadi ia sendiri ikut bergulir ke samping. Yang lengket adalah isinya,
                 dengan latar opak yang sama dengan barisnya. -->
            <th
              :colspan="1 + visible.length"
              scope="colgroup"
              class="border-y border-line py-2 text-left font-semibold"
              :class="date === today ? 'bg-canvas' : 'bg-surface'"
            >
              <span
                class="sticky left-0 z-10 inline-block px-3"
                :class="date === today ? 'border-l-4 border-l-mark bg-canvas' : 'bg-surface'"
              >
                {{ dayLabel(date) }}
                <span v-if="date === today" class="ml-2 font-normal text-ink-muted">hari ini</span>
              </span>
            </th>
          </tr>
          <tr v-for="(time, i) in slots" :key="time" class="border-b border-line last:border-b-0">
            <th
              scope="row"
              :class="[timeCell, date === today ? 'bg-canvas' : 'bg-surface']"
              class="border-r border-line"
            >
              {{ time }}
            </th>
            <td v-for="p in visible" :key="p.id" class="relative border-l border-line p-0">
              <ActivityCellView
                v-if="p.days[date]?.[i]"
                :cell="p.days[date][i]"
                :open-label="`Lihat aktivitas ${p.name}, ${time}, ${dayLabel(date)}`"
                @open="openCell(p, date, i + 1, $event)"
              />
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <BaseDialog
      :open="!!cellDialog"
      :title="cellDialog ? `${cellDialog.personName}, ${cellDialog.time}` : ''"
      @close="onCellClose"
    >
      <div v-if="cellDialog" class="grid gap-4">
        <p class="text-sm text-ink-muted">{{ dayLabel(cellDialog.date) }}</p>
        <form v-if="editing" class="grid gap-4" @submit.prevent="saveCell">
          <div class="grid gap-2">
            <label for="sel-aktivitas" class="font-medium">Aktivitas</label>
            <textarea
              id="sel-aktivitas"
              ref="activityInput"
              v-model="draft.activity"
              rows="4"
              maxlength="1000"
              class="field-sizing-content min-h-11 rounded-card border border-line bg-surface px-3 py-2"
            ></textarea>
          </div>
          <div class="grid gap-2">
            <label for="sel-output" class="font-medium">Output</label>
            <textarea
              id="sel-output"
              v-model="draft.output"
              rows="4"
              maxlength="1000"
              class="field-sizing-content min-h-11 rounded-card border border-line bg-surface px-3 py-2"
            ></textarea>
          </div>
          <div>
            <h3 class="font-medium">Berkas</h3>
            <ActivityFiles
              class="mt-1"
              :user-id="cellDialog.personId"
              :date="cellDialog.date"
              :num="cellDialog.slot"
              :time="cellDialog.time"
              :files="cellDialog.files"
              can-manage
              @changed="refreshFiles"
            />
          </div>
          <p v-if="cellError" role="alert" class="rounded-card bg-danger-bg p-3 text-danger">
            {{ cellError }}
          </p>
          <div class="flex flex-wrap justify-end gap-2">
            <button type="button" :class="navButton" @click="stopEdit">Batal</button>
            <button
              type="submit"
              :disabled="cellSaving"
              class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-on-accent hover:bg-accent-hover active:translate-y-px disabled:opacity-60"
            >
              {{ cellSaving ? 'Menyimpan...' : 'Simpan' }}
            </button>
          </div>
        </form>
        <template v-else>
          <div>
            <h3 class="font-medium">Aktivitas</h3>
            <p class="mt-1 break-words whitespace-pre-line">{{ cellDialog.activity || 'Kosong' }}</p>
          </div>
          <div>
            <h3 class="font-medium">Output</h3>
            <ActivityOutputView :output="cellDialog.output" />
          </div>
          <div>
            <h3 class="font-medium">Berkas</h3>
            <p v-if="cellDialog.files.length === 0" class="mt-1 text-sm text-ink-muted">Belum ada berkas.</p>
            <ActivityFiles
              class="mt-1"
              :user-id="cellDialog.personId"
              :date="cellDialog.date"
              :num="cellDialog.slot"
              :time="cellDialog.time"
              :files="cellDialog.files"
              :can-manage="false"
              @changed="refreshFiles"
            />
          </div>
          <p v-if="cellDialog.updatedBy" class="text-sm text-ink-muted">diubah {{ cellDialog.updatedBy }}</p>
          <p v-if="ownCell" class="text-sm text-ink-muted">Ubah isianmu lewat tabel di papan task.</p>
          <div v-if="canEdit" class="flex flex-wrap justify-end gap-2">
            <button ref="editButton" type="button" :class="navButton" @click="startEdit">Ubah</button>
          </div>
        </template>
      </div>
    </BaseDialog>

    <BaseDialog :open="exportOpen" title="Ekspor Excel" @close="exportOpen = false">
      <form class="grid gap-4" @submit.prevent="download">
        <p class="text-sm text-ink-muted">Bawaannya mengikuti tampilan yang sedang dibuka.</p>
        <div class="flex flex-wrap gap-2">
          <button type="button" :class="navButton" @click="openExport">
            {{ mode === 'hari' ? 'Hari ini saja' : 'Minggu ini' }}
          </button>
          <button type="button" :class="navButton" @click="thisMonth">Bulan ini</button>
        </div>
        <div class="grid gap-2">
          <label for="ekspor-dari" class="font-medium">Dari tanggal</label>
          <input
            id="ekspor-dari"
            v-model="exportFrom"
            type="date"
            required
            class="min-h-11 rounded-card border border-line bg-surface px-3"
          />
        </div>
        <div class="grid gap-2">
          <label for="ekspor-sampai" class="font-medium">Sampai tanggal</label>
          <input
            id="ekspor-sampai"
            v-model="exportTo"
            type="date"
            required
            class="min-h-11 rounded-card border border-line bg-surface px-3"
          />
        </div>
        <p class="text-sm text-ink-muted">
          {{ exportDays > 0 ? `${exportDays} hari` : 'Rentang belum sah' }}. Maksimal 31 hari.
        </p>
        <p v-if="exportError" role="alert" class="rounded-card bg-danger-bg p-3 text-danger">
          {{ exportError }}
        </p>
        <div class="flex flex-wrap justify-end gap-2">
          <button type="button" :class="navButton" @click="exportOpen = false">Batal</button>
          <button
            type="submit"
            class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-on-accent hover:bg-accent-hover active:translate-y-px"
          >
            Unduh
          </button>
        </div>
      </form>
    </BaseDialog>
  </section>
</template>
