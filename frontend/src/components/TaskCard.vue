<script setup lang="ts">
import RecurrenceBadge from './RecurrenceBadge.vue'
import AvatarGroup from './AvatarGroup.vue'
import { computed } from 'vue'
import type { BoardTask } from '../api/board'
import { formatDay, formatMoment, isOverdue, statusLabel } from '../utils/task'
import { readyBlock } from '../utils/subtaskGate'
import UrgencyLabel from './UrgencyLabel.vue'

const props = defineProps<{ task: BoardTask }>()
defineEmits<{ open: [id: string]; move: [task: BoardTask] }>()

const overdue = computed(() => isOverdue(props.task.due_date, props.task.status))
const moveLabel = computed(() =>
  props.task.move_to.length === 1 ? `Pindahkan ke ${statusLabel[props.task.move_to[0]]}` : 'Pindahkan',
)
// Selama ada sub task belum tercentang, tombol di kartu membuka detail (tempat mencentang), bukan dialog pindah.
const blocked = computed(() => readyBlock(props.task) !== null && props.task.move_to.includes('ready'))
const percent = computed(() =>
  props.task.subtask_total ? Math.round((props.task.subtask_done / props.task.subtask_total) * 100) : 0,
)
</script>

<template>
  <!-- Judul adalah tombol yang membentang ke seluruh kartu (after:inset-0), sehingga di langkah 4 tombol
       Pindahkan bisa ditaruh di dalam kartu tanpa menjadi tombol di dalam tombol. -->
  <article
    class="relative rounded-card border border-line bg-surface p-4 transition-colors duration-150 hover:border-ink has-[h3_button:focus-visible]:outline-2 has-[h3_button:focus-visible]:outline-offset-2 has-[h3_button:focus-visible]:outline-mark"
  >
    <div class="flex flex-wrap items-center justify-between gap-2 text-sm">
      <span class="flex flex-wrap items-center gap-2">
        <span class="specimen-tag">{{ task.id }}</span>
        <!-- Revisi 45: task ini ada di Task saya karena kamu yang meminta, bukan karena kamu PIC-nya. -->
        <span
          v-if="task.requester_only"
          class="rounded-chip border border-line bg-canvas px-2 py-0.5 text-ink"
          data-testid="requester-badge"
        >
          Kamu peminta
        </span>
      </span>
      <span class="flex items-center gap-3">
        <RecurrenceBadge v-if="task.series_freq" :freq="task.series_freq" />
        <UrgencyLabel :urgency="task.urgency" />
      </span>
    </div>

    <h3 class="mt-2 font-medium">
      <button
        type="button"
        class="text-left after:absolute after:inset-0 after:content-[''] focus-visible:outline-none"
        @click="$emit('open', task.id)"
      >
        {{ task.title }}
      </button>
    </h3>

    <div v-if="task.labels.length" class="mt-2 flex flex-wrap gap-1">
      <span
        v-for="l in task.labels"
        :key="l.id"
        class="rounded-chip px-2 py-0.5 text-sm"
        :style="{ backgroundColor: l.color, color: 'var(--color-tag-ink)' }"
      >
        {{ l.name }}
      </span>
    </div>

    <div v-if="task.subtask_total > 0" class="mt-2">
      <p class="text-sm text-ink-muted">
        <span class="font-mono">{{ task.subtask_done }}/{{ task.subtask_total }}</span> sub task tercentang
      </p>
      <!-- Dekoratif: angka di atasnya yang dibaca pembaca layar. -->
      <div class="mt-1 h-1.5 overflow-hidden rounded-[2px] border border-line bg-canvas" aria-hidden="true">
        <div class="h-full bg-mark transition-[width] duration-200" :style="{ width: `${percent}%` }" />
      </div>
    </div>

    <!-- Revisi 22: dua kelompok berlabel, PIC task dan PIC sub task. Kelompok Sub task hanya ada bila ada sub
         task berPIC. -->
    <div class="mt-3 flex flex-wrap items-end justify-between gap-x-4 gap-y-2 border-t border-line pt-2">
      <div class="flex flex-wrap gap-x-5 gap-y-2">
        <AvatarGroup v-if="task.assignees.length" label="PIC" :people="task.assignees" />
        <AvatarGroup v-if="task.subtask_pics.length" label="Sub task" :people="task.subtask_pics" />
      </div>

      <span v-if="task.status === 'completed' && task.solved_date" class="text-sm text-ink-muted">
        Selesai {{ formatMoment(task.solved_date) }}
      </span>
      <span
        v-else-if="task.due_date"
        class="inline-flex items-center gap-1 text-sm"
        :class="overdue ? 'font-medium text-urgent' : 'text-ink-muted'"
      >
        <!-- Terlambat ditandai ikon jam dan kata, bukan hanya warna. -->
        <svg
          v-if="overdue"
          aria-hidden="true"
          viewBox="0 0 24 24"
          class="size-4 shrink-0"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <circle cx="12" cy="12" r="9" />
          <path d="M12 7v5l3 2" />
        </svg>
        {{ overdue ? 'Terlambat, ' : 'Deadline ' }}{{ formatDay(task.due_date) }}
      </span>
    </div>
    <!-- relative z-10 supaya tombol ini berada di atas lapisan klik judul yang membentang ke seluruh kartu. -->
    <button
      v-if="task.move_to.length"
      type="button"
      class="relative z-10 mt-2 inline-flex min-h-11 w-full items-center justify-center rounded-card border border-line text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark active:translate-y-px"
      @click="blocked ? $emit('open', task.id) : $emit('move', task)"
    >
      {{ blocked ? 'Lihat sub task' : moveLabel }}<span class="sr-only"> {{ task.id }}</span>
    </button>
  </article>
</template>
