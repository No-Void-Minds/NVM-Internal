<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageHeader from '../components/PageHeader.vue'
import LoadingSkeleton from '../components/LoadingSkeleton.vue'
import PersonChip from '../components/PersonChip.vue'
import UserAvatar from '../components/UserAvatar.vue'
import ActivityCellFields from '../components/ActivityCellFields.vue'
import BaseDialog from '../components/BaseDialog.vue'
import { ApiError } from '../api/client'
import { activityExportUrl, getActivityDay, type ActivityPerson } from '../api/activities'
import { listDivisions, type DivisionItem } from '../api/divisions'
import { useSessionStore } from '../stores/session'
import { useActivitySave } from '../composables/useActivitySave'
import { filterControl } from '../utils/ui'
import { formatLong, officeToday } from '../utils/recurrence'

// Halaman Aktivitas Harian (revisi 21): satu kolom per orang untuk satu tanggal. Semua peran boleh membaca,
// hanya Manajer yang boleh mengubah (can_edit dari server, bukan dihitung di sini).
const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const isManager = computed(() => session.user?.role === 'manager')

const dateParam = computed(() => {
  const v = route.query.tanggal
  const raw = Array.isArray(v) ? v[0] : v
  return typeof raw === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(raw) ? raw : officeToday()
})

const slots = ref<string[]>([])
const people = ref<ActivityPerson[]>([])
const canEdit = ref(false)
const divisions = ref<DivisionItem[]>([])
const divisionFilter = ref('')
const loadState = ref<'loading' | 'ready' | 'error'>('loading')
const loadError = ref('')
const exportOpen = ref(false)
const exportFrom = ref('')
const exportTo = ref('')
const exportError = ref('')

const visible = computed(() =>
  divisionFilter.value ? people.value.filter((p) => p.division_name === divisionFilter.value) : people.value,
)

const { status, message, save, retry } = useActivitySave((userId, slot, cell) => {
  const person = people.value.find((p) => p.id === userId)
  const i = person?.cells.findIndex((c) => c.slot === slot) ?? -1
  if (!person || i === -1) return
  person.cells[i] = cell
  person.filled = person.cells.some((c) => c.activity !== '' || c.output !== '')
})

// Nilai terakhir dari server per sel, supaya sel yang tidak berubah tidak ikut dikirim saat kehilangan fokus.
const saved = ref<Record<string, { activity: string; output: string }>>({})

function remember(list: ActivityPerson[]) {
  const snap: Record<string, { activity: string; output: string }> = {}
  for (const p of list) {
    for (const c of p.cells) snap[`${p.id}:${c.slot}`] = { activity: c.activity, output: c.output }
  }
  saved.value = snap
  people.value = list
}

let controller: AbortController | null = null

async function load() {
  controller?.abort()
  const c = new AbortController()
  controller = c
  loadState.value = 'loading'
  try {
    const res = await getActivityDay(dateParam.value, c.signal)
    slots.value = res.slots
    canEdit.value = res.can_edit
    remember(res.people)
    loadState.value = 'ready'
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') return
    loadError.value = err instanceof ApiError ? err.message : 'Tidak bisa memuat aktivitas tanggal ini.'
    loadState.value = 'error'
  }
}

watch(dateParam, () => void load(), { immediate: true })

// Daftar divisi hanya pelengkap filter; kalau gagal, tabelnya tetap tampil.
void listDivisions()
  .then((d) => (divisions.value = d))
  .catch(() => (divisions.value = []))

function goto(date: string) {
  void router.push({ query: { ...route.query, tanggal: date } })
}

function shift(days: number) {
  const d = new Date(`${dateParam.value}T00:00:00Z`)
  d.setUTCDate(d.getUTCDate() + days)
  goto(d.toISOString().slice(0, 10))
}

function blur(person: ActivityPerson, slot: number) {
  const cell = person.cells[slot - 1]
  if (!cell) return
  const before = saved.value[`${person.id}:${slot}`]
  if (before && before.activity === cell.activity && before.output === cell.output) return
  saved.value[`${person.id}:${slot}`] = { activity: cell.activity, output: cell.output }
  void save(person.id, dateParam.value, slot, cell.activity, cell.output)
}

function openExport() {
  exportError.value = ''
  // Bawaan: bulan ini sampai tanggal yang sedang dibuka, rentang yang paling sering diminta Manajer.
  exportFrom.value = `${dateParam.value.slice(0, 7)}-01`
  exportTo.value = dateParam.value
  exportOpen.value = true
}

function thisWeek() {
  const d = new Date(`${dateParam.value}T00:00:00Z`)
  // getUTCDay: 0 Minggu. Minggu kerja dihitung mulai Senin.
  const back = (d.getUTCDay() + 6) % 7
  const start = new Date(d)
  start.setUTCDate(d.getUTCDate() - back)
  const end = new Date(start)
  end.setUTCDate(start.getUTCDate() + 6)
  exportFrom.value = start.toISOString().slice(0, 10)
  exportTo.value = end.toISOString().slice(0, 10)
}

function thisMonth() {
  const month = dateParam.value.slice(0, 7)
  const last = new Date(`${month}-01T00:00:00Z`)
  last.setUTCMonth(last.getUTCMonth() + 1)
  last.setUTCDate(0)
  exportFrom.value = `${month}-01`
  exportTo.value = last.toISOString().slice(0, 10)
}

const exportDays = computed(() => {
  if (!exportFrom.value || !exportTo.value) return 0
  const from = new Date(`${exportFrom.value}T00:00:00Z`).getTime()
  const to = new Date(`${exportTo.value}T00:00:00Z`).getTime()
  return Math.round((to - from) / 86_400_000) + 1
})

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
  'inline-flex min-h-11 items-center rounded-card border border-line bg-surface px-4 font-medium hover:bg-canvas active:translate-y-px'
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

    <div class="mt-4 flex flex-wrap items-center gap-2">
      <nav class="flex flex-wrap items-center gap-2" aria-label="Pindah tanggal">
        <button type="button" :class="navButton" @click="shift(-1)">Hari sebelumnya</button>
        <div class="flex min-h-11 items-center gap-2 rounded-card border border-line bg-surface pl-3">
          <label for="aktivitas-tanggal" class="text-sm text-ink-muted">Tanggal</label>
          <input
            id="aktivitas-tanggal"
            type="date"
            :value="dateParam"
            class="min-h-11 min-w-0 rounded-r-card bg-transparent px-2 text-base text-ink"
            @change="goto(($event.target as HTMLInputElement).value || officeToday())"
          />
        </div>
        <button type="button" :class="navButton" @click="goto(officeToday())">Hari ini</button>
        <button type="button" :class="navButton" @click="shift(1)">Hari berikutnya</button>
      </nav>
      <div>
        <label for="aktivitas-divisi" class="sr-only">Divisi</label>
        <select id="aktivitas-divisi" v-model="divisionFilter" :class="filterControl">
          <option value="">Semua divisi</option>
          <option v-for="d in divisions" :key="d.id" :value="d.name">{{ d.name }}</option>
        </select>
      </div>
    </div>

    <p class="mt-3 font-medium">{{ formatLong(dateParam) }}</p>
    <p v-if="!canEdit" class="mt-1 text-sm text-ink-muted">Ubah isianmu lewat tabel di papan task.</p>

    <LoadingSkeleton v-if="loadState === 'loading'" class="mt-6" label="Memuat aktivitas..." />

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

    <p
      v-else-if="visible.length === 0"
      class="mt-6 rounded-card border border-line bg-surface p-6 text-center"
    >
      {{ divisionFilter ? 'Tidak ada Staf di divisi ini.' : 'Belum ada akun Staf yang aktif.' }}
    </p>

    <!-- Tabel lebar bergulir di dalam wadahnya sendiri, jadi halamannya tidak pernah ikut bergulir ke samping.
         Kolom Jam lengket di kiri supaya jamnya tetap terbaca saat kolom orang digulir. -->
    <div v-else class="mt-4 overflow-x-auto rounded-card border border-line bg-surface">
      <table class="w-full border-collapse text-left align-top">
        <thead class="border-b border-line">
          <tr>
            <th scope="col" class="sticky left-0 z-10 w-28 bg-surface px-3 py-3 font-medium">Jam</th>
            <th
              v-for="p in visible"
              :key="p.id"
              scope="col"
              class="min-w-[22rem] border-l border-line px-3 py-3 font-medium"
            >
              <PersonChip :user-id="p.id" :name="p.name">
                <UserAvatar :user-id="p.id" :name="p.name" :avatar-id="p.avatar_id" size="sm" decorative />
                {{ p.name }}
              </PersonChip>
              <span class="mt-1 block text-sm font-normal text-ink-muted">
                {{ p.division_name }}<template v-if="!p.is_active">, nonaktif</template>
              </span>
              <span v-if="!p.filled" class="mt-1 block text-sm font-normal text-ink-muted">Belum diisi</span>
              <span class="mt-2 grid grid-cols-2 gap-3 text-sm font-normal text-ink-muted">
                <span>Aktivitas</span>
                <span>Output</span>
              </span>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(time, i) in slots" :key="time" class="border-b border-line last:border-b-0">
            <th
              scope="row"
              class="sticky left-0 z-10 bg-surface px-3 py-3 text-left font-medium whitespace-nowrap"
            >
              {{ time }}
            </th>
            <td v-for="p in visible" :key="p.id" class="border-l border-line px-3 py-3">
              <ActivityCellFields
                v-if="p.cells[i]"
                v-model:activity="p.cells[i].activity"
                v-model:output="p.cells[i].output"
                :time="time"
                :context="p.name"
                :updated-by="p.cells[i].updated_by"
                :readonly="!canEdit"
                :status="status[`${p.id}:${i + 1}`]"
                :message="message[`${p.id}:${i + 1}`]"
                @blur="blur(p, i + 1)"
                @retry="retry(p.id, i + 1)"
              />
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <BaseDialog :open="exportOpen" title="Ekspor Excel" @close="exportOpen = false">
      <form class="grid gap-4" @submit.prevent="download">
        <div class="flex flex-wrap gap-2">
          <button type="button" :class="navButton" @click="thisWeek">Minggu ini</button>
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
