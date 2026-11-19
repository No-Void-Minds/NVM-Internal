<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import TimelineMarker from './TimelineMarker.vue'
import { dayLabel, shiftDay, weekStart } from '../utils/activityWeek'
import {
  eventCovers,
  eventSummary,
  monthGrid,
  monthLabel,
  weekdayIndex,
  type TimelineEvent,
} from '../utils/timeline'

// Grid bulan (mulai 768 px). Satu tombol per tanggal dengan roving tabindex: hanya satu tanggal yang masuk
// urutan Tab, panah memindahkan fokus, dan panah di tepi bulan pindah ke bulan sebelah.
const props = defineProps<{
  month: string
  today: string
  // Event yang sudah disaring kategori; null saat memuat atau galat, grid tetap tampil tanpa isi.
  events: TimelineEvent[] | null
}>()
const emit = defineEmits<{ open: [date: string, trigger: HTMLElement]; month: [month: string] }>()

const MAX_IN_CELL = 3
const weekdayNames = ['Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab', 'Min']
const weekdayLong = ['Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu', 'Minggu']

const days = computed(() => monthGrid(props.month))
const weeks = computed(() => {
  const out: string[][] = []
  for (let i = 0; i < days.value.length; i += 7) out.push(days.value.slice(i, i + 7))
  return out
})

const byDate = computed(() => {
  const map = new Map<string, TimelineEvent[]>()
  if (!props.events) return map
  for (const d of days.value) {
    if (!d.startsWith(props.month)) continue
    map.set(
      d,
      props.events.filter((e) => eventCovers(e, d)),
    )
  }
  return map
})

const active = ref('')
function defaultActive() {
  return props.today.startsWith(props.month) ? props.today : `${props.month}-01`
}
watch(
  () => props.month,
  () => {
    if (!active.value.startsWith(props.month)) active.value = defaultActive()
  },
  { immediate: true },
)

const grid = ref<HTMLElement | null>(null)
function cell(date: string) {
  return grid.value?.querySelector<HTMLButtonElement>(`[data-date="${date}"]`) ?? null
}

// Dipanggil halaman setelah panel atau popover ditutup, supaya fokus kembali ke tanggal yang sedang dikerjakan.
async function focusDate(date: string) {
  if (!date.startsWith(props.month)) emit('month', date.slice(0, 7))
  active.value = date
  await nextTick()
  cell(date)?.focus()
}
defineExpose({ focusDate })

function onKeydown(e: KeyboardEvent, date: string) {
  let next: string | null = null
  switch (e.key) {
    case 'ArrowLeft':
      next = shiftDay(date, -1)
      break
    case 'ArrowRight':
      next = shiftDay(date, 1)
      break
    case 'ArrowUp':
      next = shiftDay(date, -7)
      break
    case 'ArrowDown':
      next = shiftDay(date, 7)
      break
    case 'Home':
      next = weekStart(date)
      break
    case 'End':
      next = shiftDay(weekStart(date), 6)
      break
    default:
      return
  }
  e.preventDefault()
  focusDate(next)
}

function onActivate(date: string, e: MouseEvent) {
  // Tanggal bulan sebelah hanya membawa ke bulannya; isinya belum dimuat, jadi popover-nya tidak dibuka.
  if (!date.startsWith(props.month)) {
    focusDate(date)
    return
  }
  active.value = date
  emit('open', date, e.currentTarget as HTMLElement)
}

function cellLabel(date: string) {
  const list = byDate.value.get(date) ?? []
  const parts = [dayLabel(date)]
  if (date === props.today) parts.push('hari ini')
  if (!date.startsWith(props.month)) parts.push('bulan lain')
  else if (props.events) parts.push(list.length ? `${list.length} entri` : 'tanpa entri')
  return parts.join(', ')
}
</script>

<template>
  <div
    ref="grid"
    role="grid"
    :aria-label="`Kalender ${monthLabel(month)}`"
    class="overflow-hidden rounded-card border border-line"
  >
    <div role="row" class="grid grid-cols-7 border-b border-line bg-surface">
      <div
        v-for="(d, i) in weekdayNames"
        :key="d"
        role="columnheader"
        :aria-label="weekdayLong[i]"
        class="px-2 py-2 text-sm font-medium text-ink-muted"
      >
        {{ d }}
      </div>
    </div>
    <div
      v-for="(week, wi) in weeks"
      :key="week[0]"
      role="row"
      class="grid grid-cols-7"
      :class="wi < weeks.length - 1 ? 'border-b border-line' : ''"
    >
      <div
        v-for="date in week"
        :key="date"
        role="gridcell"
        class="min-w-0"
        :class="[
          weekdayIndex(date) >= 5 ? 'bg-canvas' : 'bg-surface',
          weekdayIndex(date) < 6 ? 'border-r border-line' : '',
        ]"
      >
        <button
          type="button"
          :data-date="date"
          :tabindex="date === active ? 0 : -1"
          :aria-label="cellLabel(date)"
          aria-haspopup="dialog"
          class="flex h-full min-h-28 w-full min-w-0 flex-col items-stretch gap-1 p-1.5 text-left hover:bg-ink/5 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-mark"
          @click="onActivate(date, $event)"
          @keydown="onKeydown($event, date)"
          @focus="active = date"
        >
          <span
            class="inline-flex size-7 shrink-0 items-center justify-center rounded-full text-sm"
            :class="[
              date === today ? 'bg-ink font-semibold text-surface' : '',
              date.startsWith(month) ? '' : 'text-ink-muted',
            ]"
          >
            {{ Number(date.slice(8)) }}
          </span>
          <template v-if="byDate.get(date)?.length">
            <span
              v-for="ev in byDate.get(date)!.slice(0, MAX_IN_CELL)"
              :key="ev.id"
              class="flex min-w-0 items-center gap-1 text-xs"
            >
              <TimelineMarker :event="ev" />
              <span class="min-w-0 truncate">{{ eventSummary(ev) }}</span>
            </span>
            <span v-if="byDate.get(date)!.length > MAX_IN_CELL" class="text-xs font-medium text-ink-muted">
              +{{ byDate.get(date)!.length - MAX_IN_CELL }} lainnya
            </span>
          </template>
        </button>
      </div>
    </div>
  </div>
</template>
