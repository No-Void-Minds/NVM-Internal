<script setup lang="ts">
import { computed } from 'vue'
import type { BoardTask } from '../api/board'
import { formatDay, formatMoment, isOverdue, statusLabel, urgencyClass, urgencyLabel } from '../utils/task'

const props = defineProps<{ task: BoardTask }>()
defineEmits<{ open: [id: string]; move: [task: BoardTask] }>()

const overdue = computed(() => isOverdue(props.task.due_date, props.task.status))
const moveLabel = computed(() =>
  props.task.move_to.length === 1 ? `Pindahkan ke ${statusLabel[props.task.move_to[0]]}` : 'Pindahkan',
)

function initials(name: string): string {
  return name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() ?? '')
    .join('')
}
</script>

<template>
  <!-- Judul adalah tombol yang membentang ke seluruh kartu (after:inset-0), sehingga di langkah 4 tombol
       Pindahkan bisa ditaruh di dalam kartu tanpa menjadi tombol di dalam tombol. -->
  <article
    class="relative rounded-card border border-line bg-surface p-3 hover:border-ink has-[h3_button:focus-visible]:outline-2 has-[h3_button:focus-visible]:outline-offset-2 has-[h3_button:focus-visible]:outline-accent"
  >
    <div class="flex items-center gap-2 text-sm">
      <span class="rounded-chip bg-canvas px-1.5 font-mono text-ink-muted">{{ task.id }}</span>
      <span :class="urgencyClass[task.urgency]">{{ urgencyLabel[task.urgency] }}</span>
    </div>

    <h3 class="mt-1 font-medium">
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
        :style="{ backgroundColor: l.color }"
      >
        {{ l.name }}
      </span>
    </div>

    <div class="mt-3 flex items-center justify-between gap-2 border-t border-line pt-2">
      <ul class="flex -space-x-1.5">
        <li
          v-for="p in task.assignees"
          :key="p.id"
          :title="p.name"
          class="flex size-8 items-center justify-center rounded-full border-2 border-surface bg-canvas text-xs font-medium"
        >
          <span aria-hidden="true">{{ initials(p.name) }}</span>
          <span class="sr-only">{{ p.name }}</span>
        </li>
      </ul>

      <span v-if="task.status === 'completed' && task.solved_date" class="text-sm text-ink-muted">
        Selesai {{ formatMoment(task.solved_date) }}
      </span>
      <span
        v-else-if="task.due_date"
        class="text-sm"
        :class="overdue ? 'font-medium text-urgent' : 'text-ink-muted'"
      >
        {{ overdue ? 'Terlambat, ' : 'Tenggat ' }}{{ formatDay(task.due_date) }}
      </span>
    </div>
    <!-- relative z-10 supaya tombol ini berada di atas lapisan klik judul yang membentang ke seluruh kartu. -->
    <button
      v-if="task.move_to.length"
      type="button"
      class="relative z-10 mt-2 inline-flex min-h-11 w-full items-center justify-center rounded-card border border-line text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent active:translate-y-px"
      @click="$emit('move', task)"
    >
      {{ moveLabel }}<span class="sr-only"> {{ task.id }}</span>
    </button>
  </article>
</template>
