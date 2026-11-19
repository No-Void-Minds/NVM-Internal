<script setup lang="ts">
import { computed } from 'vue'
import TimelineMarker from './TimelineMarker.vue'
import { dayLabel, shiftDay } from '../utils/activityWeek'
import { eventCovers, eventSummary, shiftMonth, type TimelineEvent } from '../utils/timeline'

// Pengganti grid di bawah 768 px: tujuh kolom tidak terbaca di 360 px. Hanya tanggal yang punya entri, plus hari
// ini, dikelompokkan per tanggal. Kepala tiap kelompok membuka popover yang sama dengan grid.
const props = defineProps<{ month: string; today: string; events: TimelineEvent[] }>()
const emit = defineEmits<{ open: [date: string, trigger: HTMLElement] }>()

const groups = computed(() => {
  const out: { date: string; events: TimelineEvent[] }[] = []
  const end = `${shiftMonth(props.month, 1)}-01`
  for (let d = `${props.month}-01`; d < end; d = shiftDay(d, 1)) {
    const list = props.events.filter((e) => eventCovers(e, d))
    if (list.length || d === props.today) out.push({ date: d, events: list })
  }
  return out
})
</script>

<template>
  <ol class="space-y-3" aria-label="Agenda bulan ini">
    <li v-for="g in groups" :key="g.date" class="rounded-card border border-line bg-surface">
      <button
        type="button"
        :data-date="g.date"
        aria-haspopup="dialog"
        class="flex min-h-11 w-full items-center gap-2 border-b border-line px-3 py-2 text-left font-semibold hover:bg-canvas focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-mark"
        @click="emit('open', g.date, $event.currentTarget as HTMLElement)"
      >
        <span v-if="g.date === today" class="rounded-chip bg-ink px-1.5 text-xs font-semibold text-surface">
          Hari ini
        </span>
        <span class="min-w-0 flex-1">{{ dayLabel(g.date) }}</span>
        <svg
          aria-hidden="true"
          viewBox="0 0 24 24"
          class="size-5 shrink-0 text-ink-muted"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M9 6l6 6-6 6" />
        </svg>
      </button>
      <ul v-if="g.events.length" class="divide-y divide-line/40">
        <li v-for="ev in g.events" :key="ev.id" class="flex min-w-0 items-start gap-2 px-3 py-2 text-sm">
          <TimelineMarker :event="ev" />
          <span class="min-w-0 [overflow-wrap:anywhere]">{{ eventSummary(ev) }}</span>
        </li>
      </ul>
      <p v-else class="px-3 py-2 text-sm text-ink-muted">Tidak ada entri hari ini.</p>
    </li>
  </ol>
</template>
