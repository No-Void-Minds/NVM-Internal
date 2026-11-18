<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  dueOffset,
  freqLabel,
  isoWeekday,
  nextOccurrence,
  officeToday,
  previewSentence,
  weekdayShort,
  type Freq,
  type ScheduleForm,
} from '../utils/recurrence'

// Bagian Jadwal di form Buat task dan Ubah task (revisi 19).
const props = withDefaults(
  defineProps<{
    // Tanggal mulai dan tenggat yang sedang diisi di form, untuk kalimat pratinjau.
    startDate: string
    dueDate: string
    // Task ini kemunculan seri yang masih aktif (form Ubah).
    inSeries?: boolean
    seriesActive?: boolean
    // Tanggal kemunculan task ini, untuk menghitung kemunculan berikutnya di form Ubah.
    occurrenceDate?: string | null
    stopping?: boolean
  }>(),
  { inSeries: false, seriesActive: false, occurrenceDate: null, stopping: false },
)
const form = defineModel<ScheduleForm>({ required: true })
const apply = defineModel<boolean>('apply', { default: true })
const emit = defineEmits<{ stop: [] }>()

function set<K extends keyof ScheduleForm>(key: K, value: ScheduleForm[K]) {
  form.value = { ...form.value, [key]: value }
}

const choices: { value: ScheduleForm['freq']; label: string }[] = [
  { value: 'none', label: 'Tidak berulang' },
  ...(Object.keys(freqLabel) as Freq[]).map((f) => ({ value: f, label: freqLabel[f] })),
]

// Kemunculan pertama adalah task ini sendiri, pada tanggal mulai atau hari ini (sama dengan backend).
const first = computed(() => props.occurrenceDate || props.startDate || officeToday())

function choose(value: ScheduleForm['freq']) {
  const next = { ...form.value, freq: value }
  // Bawaan mingguan: hari dari tanggal mulai. Bawaan bulanan: tanggal dari tanggal mulai.
  if (value === 'weekly' && next.weekdays.length === 0) next.weekdays = [isoWeekday(first.value)]
  if (value === 'monthly' && !next.month_day) next.month_day = Number(first.value.slice(8, 10))
  form.value = next
}

function toggleDay(day: number) {
  const has = form.value.weekdays.includes(day)
  // Minimal satu hari: hari terakhir yang menyala tidak bisa dimatikan.
  if (has && form.value.weekdays.length === 1) return
  set(
    'weekdays',
    has ? form.value.weekdays.filter((d) => d !== day) : [...form.value.weekdays, day].sort((a, b) => a - b),
  )
}

const preview = computed(() => {
  if (form.value.freq === 'none') return ''
  const today = officeToday()
  const after = first.value > today ? first.value : today
  const offset = dueOffset(props.dueDate, first.value)
  return previewSentence(nextOccurrence(form.value, after, first.value), offset)
})

const confirmStop = ref(false)

const option =
  'flex min-h-11 cursor-pointer items-center justify-center rounded-card border border-line px-3 text-center text-sm font-medium hover:bg-canvas has-[:checked]:border-mark has-[:checked]:bg-progress-bg has-[:checked]:text-progress-fg has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-mark'
const field = 'mt-1 block min-h-11 w-full rounded-card border border-line bg-surface px-3'
const secondary =
  'inline-flex min-h-11 items-center rounded-card border border-line px-4 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark active:translate-y-px disabled:opacity-60'
</script>

<template>
  <fieldset class="space-y-3">
    <legend class="text-sm font-medium">Jadwal</legend>

    <p v-if="inSeries && !seriesActive" class="text-sm text-ink-muted">
      Pengulangan task ini sudah dihentikan. Kemunculan yang sudah ada tidak berubah.
    </p>

    <template v-else>
      <!-- Radio bawaan: panah kiri dan kanan memindah pilihan, Tab masuk dan keluar grup. -->
      <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
        <label v-for="c in choices" :key="c.value" :class="option">
          <input
            type="radio"
            name="task-schedule"
            class="sr-only"
            :value="c.value"
            :checked="form.freq === c.value"
            @change="choose(c.value)"
          />
          {{ c.label }}
        </label>
      </div>

      <label v-if="form.freq === 'daily'" class="flex min-h-11 cursor-pointer items-center gap-3 text-sm">
        <input
          type="checkbox"
          class="size-5 accent-mark"
          :checked="form.skip_weekend"
          @change="set('skip_weekend', ($event.target as HTMLInputElement).checked)"
        />
        Lewati Sabtu dan Minggu
      </label>

      <div
        v-if="form.freq === 'weekly'"
        role="group"
        aria-label="Hari pengulangan"
        class="flex flex-wrap gap-2"
      >
        <button
          v-for="w in weekdayShort"
          :key="w.day"
          type="button"
          :aria-pressed="form.weekdays.includes(w.day)"
          :aria-label="w.long"
          class="inline-flex size-11 items-center justify-center rounded-card border text-sm font-medium focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
          :class="
            form.weekdays.includes(w.day)
              ? 'border-mark bg-progress-bg text-progress-fg'
              : 'border-line hover:bg-canvas'
          "
          @click="toggleDay(w.day)"
        >
          {{ w.short }}
        </button>
      </div>

      <div v-if="form.freq === 'monthly'">
        <label for="schedule-month-day" class="block text-sm">Tanggal</label>
        <select
          id="schedule-month-day"
          :value="form.month_day"
          :class="field"
          aria-describedby="schedule-month-hint"
          @change="set('month_day', Number(($event.target as HTMLSelectElement).value))"
        >
          <option v-for="d in 31" :key="d" :value="d">{{ d }}</option>
        </select>
        <p id="schedule-month-hint" class="mt-1 text-sm text-ink-muted">
          Kalau bulan itu lebih pendek, task muncul di hari terakhir bulan.
        </p>
      </div>

      <div v-if="form.freq !== 'none'">
        <label for="schedule-end" class="block text-sm">Berhenti setelah (opsional)</label>
        <input
          id="schedule-end"
          type="date"
          :value="form.end_date"
          :min="first"
          :class="field"
          @input="set('end_date', ($event.target as HTMLInputElement).value)"
        />
      </div>

      <p v-if="preview" class="text-sm" aria-live="polite">{{ preview }}</p>

      <label v-if="inSeries" class="flex min-h-11 cursor-pointer items-center gap-3 text-sm">
        <input v-model="apply" type="checkbox" class="size-5 accent-mark" />
        Pakai perubahan ini untuk kemunculan berikutnya
      </label>

      <p class="text-sm text-ink-muted">
        Lampiran brief tidak ikut diulang. Pakai tautan acuan untuk berkas tetap.
      </p>

      <div v-if="inSeries">
        <button v-if="!confirmStop" type="button" :class="secondary" @click="confirmStop = true">
          Hentikan pengulangan
        </button>
        <div v-else role="group" aria-label="Konfirmasi hentikan pengulangan" class="space-y-2">
          <p class="text-sm">Hentikan pengulangan? Kemunculan yang sudah ada tidak dihapus.</p>
          <div class="flex flex-wrap gap-2">
            <button
              type="button"
              :class="[secondary, 'border-danger text-danger']"
              :disabled="stopping"
              @click="emit('stop')"
            >
              {{ stopping ? 'Menghentikan...' : 'Ya, hentikan' }}
            </button>
            <button type="button" :class="secondary" @click="confirmStop = false">Batal</button>
          </div>
        </div>
      </div>
    </template>
  </fieldset>
</template>
