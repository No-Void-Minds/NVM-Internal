<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import BaseDialog from './BaseDialog.vue'
import TaskForm from './TaskForm.vue'
import type { TaskDetail } from '../api/tasks'
import type { UserItem } from '../api/users'
import type { LabelItem } from '../api/labels'

defineProps<{
  open: boolean
  task: TaskDetail | null
  users: UserItem[]
  labels: LabelItem[]
}>()
// partial true berarti task tersimpan, tetapi ada lampiran yang gagal diunggah.
const emit = defineEmits<{ close: []; saved: [id: string, message: string, partial: boolean] }>()

const form = ref<InstanceType<typeof TaskForm> | null>(null)
// Tab baru membuka form kosong. Isian panel tidak dibawa dan panelnya tetap terbuka (keputusan Sprint 11x).
const newHref = useRouter().resolve({ name: 'task-new' }).href
</script>

<template>
  <BaseDialog
    :open="open"
    :title="task ? `Ubah ${task.id}` : 'Buat task'"
    variant="panel"
    @close="emit('close')"
  >
    <template v-if="!task" #actions>
      <a
        :href="newHref"
        target="_blank"
        rel="noopener"
        aria-label="Buka form Buat task di tab baru"
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

    <TaskForm
      ref="form"
      :open="open"
      :task="task"
      :users="users"
      :labels="labels"
      @saved="(id, message, partial) => emit('saved', id, message, partial)"
    />

    <template #footer>
      <button
        type="button"
        class="inline-flex min-h-11 items-center rounded-card border border-line px-4 font-medium hover:bg-canvas active:translate-y-px"
        @click="emit('close')"
      >
        Batal
      </button>
      <button
        type="submit"
        form="task-form"
        :disabled="form?.submitting"
        class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-on-accent hover:bg-accent-hover active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60"
      >
        {{ form?.submitLabel ?? (task ? 'Simpan perubahan' : 'Buat task') }}
      </button>
    </template>
  </BaseDialog>
</template>
