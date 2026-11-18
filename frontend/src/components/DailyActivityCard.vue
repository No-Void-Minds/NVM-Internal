<script setup lang="ts">
import { computed, onMounted, ref, useId } from 'vue'
import { RouterLink } from 'vue-router'
import LoadingSkeleton from './LoadingSkeleton.vue'
import ActivityCellFields from './ActivityCellFields.vue'
import { getMyActivities, type ActivityCell } from '../api/activities'
import { ApiError } from '../api/client'
import { useActivitySave } from '../composables/useActivitySave'
import { filledHours } from '../utils/activity'
import { formatLong } from '../utils/recurrence'

// Kartu Aktivitas hari ini di papan task (revisi 21), hanya untuk Staf. Isian hari lain diubah di /aktivitas.
const id = useId()
const lipatKey = 'nvm-aktivitas-lipat'

const open = ref(true)
const date = ref('')
const cells = ref<ActivityCell[]>([])
// Nilai terakhir dari server, jadi sel yang tidak berubah tidak ikut dikirim saat kehilangan fokus.
const saved = ref<Record<number, { activity: string; output: string }>>({})
const loadState = ref<'loading' | 'ready' | 'error'>('loading')
const loadError = ref('')

const terisi = computed(() => filledHours(cells.value))
const { status, message, save, retry } = useActivitySave((_userId, slot, cell) => {
  const i = cells.value.findIndex((c) => c.slot === slot)
  if (i === -1) return
  cells.value[i] = cell
  saved.value[slot] = { activity: cell.activity, output: cell.output }
})

function remember(list: ActivityCell[]) {
  cells.value = list
  saved.value = Object.fromEntries(list.map((c) => [c.slot, { activity: c.activity, output: c.output }]))
}

async function load() {
  loadState.value = 'loading'
  try {
    const res = await getMyActivities()
    date.value = res.date
    remember(res.cells)
    loadState.value = 'ready'
  } catch (err) {
    loadError.value = err instanceof ApiError ? err.message : 'Tidak bisa memuat aktivitas hari ini.'
    loadState.value = 'error'
  }
}

// Sel disimpan saat kehilangan fokus, hanya kalau isinya memang berubah.
function blur(cell: ActivityCell) {
  const before = saved.value[cell.slot]
  if (before && before.activity === cell.activity && before.output === cell.output) return
  void save(0, date.value, cell.slot, cell.activity, cell.output)
}

function toggle() {
  open.value = !open.value
  try {
    localStorage.setItem(lipatKey, open.value ? 'buka' : 'lipat')
  } catch {
    // Mode privat atau penyimpanan diblokir: status lipat cukup berlaku untuk kunjungan ini.
  }
}

onMounted(() => {
  try {
    open.value = localStorage.getItem(lipatKey) !== 'lipat'
  } catch {
    open.value = true
  }
  void load()
})
</script>

<template>
  <section class="mt-4 rounded-card border border-line bg-surface">
    <h2>
      <button
        type="button"
        class="flex min-h-11 w-full flex-wrap items-center justify-between gap-x-3 gap-y-1 px-4 py-2 text-left font-medium focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
        :aria-expanded="open"
        :aria-controls="id"
        @click="toggle"
      >
        <span class="flex items-center gap-2">
          <svg
            aria-hidden="true"
            viewBox="0 0 24 24"
            class="size-5 shrink-0 transition-transform motion-reduce:transition-none"
            :class="open ? 'rotate-90' : ''"
            fill="none"
            stroke="currentColor"
            stroke-width="1.75"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M9 6l6 6-6 6" />
          </svg>
          Aktivitas hari ini
          <span v-if="date" class="font-normal text-ink-muted">{{ formatLong(date) }}</span>
        </span>
        <span class="text-sm font-normal text-ink-muted"
          >{{ terisi }} dari {{ cells.length || 8 }} jam terisi</span
        >
      </button>
    </h2>

    <div v-show="open" :id="id" class="border-t border-line p-4">
      <LoadingSkeleton v-if="loadState === 'loading'" label="Memuat aktivitas hari ini..." />

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

      <template v-else>
        <p v-if="terisi === 0" class="mb-3 text-sm text-ink-muted">
          Belum ada jam yang terisi. Tulis apa yang kamu kerjakan dan hasilnya, lalu klik di luar kotak untuk
          menyimpan.
        </p>
        <!-- Di bawah 640 px tiap jam jadi kartu bertumpuk; mulai 640 px barisnya sejajar jadi tiga kolom. -->
        <div
          class="mb-2 hidden gap-3 px-1 text-sm font-medium text-ink-muted sm:grid sm:grid-cols-[7rem_minmax(0,1fr)_minmax(0,1fr)]"
          aria-hidden="true"
        >
          <span>Jam</span>
          <span>Aktivitas</span>
          <span>Output</span>
        </div>
        <ul class="grid gap-3">
          <li
            v-for="c in cells"
            :key="c.slot"
            class="grid gap-2 rounded-card border border-line p-3 sm:grid-cols-[7rem_minmax(0,1fr)_minmax(0,1fr)] sm:items-start sm:gap-3"
          >
            <p class="font-medium sm:pt-2">{{ c.time }}</p>
            <ActivityCellFields
              v-model:activity="c.activity"
              v-model:output="c.output"
              :time="c.time"
              :updated-by="c.updated_by"
              :status="status[`0:${c.slot}`]"
              :message="message[`0:${c.slot}`]"
              @blur="blur(c)"
              @retry="retry(0, c.slot)"
            />
          </li>
        </ul>
        <p class="mt-3 text-sm text-ink-muted">
          Hari lain dan isian tim ada di
          <RouterLink
            class="underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
            to="/aktivitas"
          >
            Aktivitas Harian
          </RouterLink>
          .
        </p>
      </template>
    </div>
  </section>
</template>
