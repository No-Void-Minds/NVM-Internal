<script setup lang="ts">
import { useRouter } from 'vue-router'
import BaseDialog from './BaseDialog.vue'
import TaskDetailBody from './TaskDetailBody.vue'
import type { LabelItem } from '../api/labels'
import type { TaskDetail } from '../api/tasks'

defineProps<{ taskId: string | null; labels: LabelItem[] }>()
const emit = defineEmits<{ close: []; changed: [message: string]; edit: [task: TaskDetail] }>()

const router = useRouter()

// Tautan biasa (bukan RouterLink) karena target _blank memuat aplikasi di tab baru; sesi tetap terbawa lewat cookie.
function fullHref(id: string): string {
  return router.resolve({ name: 'task-detail', params: { id } }).href
}
</script>

<template>
  <BaseDialog :open="taskId !== null" :title="taskId ?? 'Detail task'" variant="panel" @close="emit('close')">
    <template v-if="taskId" #actions>
      <a
        :href="fullHref(taskId)"
        target="_blank"
        rel="noopener"
        :aria-label="`Buka ${taskId} di tab baru`"
        title="Buka di tab baru"
        class="inline-flex size-11 shrink-0 items-center justify-center rounded-card text-ink-muted hover:bg-canvas hover:text-ink active:bg-canvas"
      >
        <svg
          aria-hidden="true"
          viewBox="0 0 24 24"
          class="size-5"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M14 4h6v6M20 4l-9 9M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5" />
        </svg>
      </a>
    </template>

    <TaskDetailBody
      v-if="taskId"
      :task-id="taskId"
      :labels="labels"
      @close="emit('close')"
      @changed="emit('changed', $event)"
      @edit="emit('edit', $event)"
    />
  </BaseDialog>
</template>
