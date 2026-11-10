<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import TaskDetailBody from '../components/TaskDetailBody.vue'
import TaskFormPanel from '../components/TaskFormPanel.vue'
import { listLabels, type LabelItem } from '../api/labels'
import { listUsers, type UserItem } from '../api/users'
import type { TaskDetail } from '../api/tasks'
import { useSessionStore } from '../stores/session'

const route = useRoute()
const router = useRouter()
const session = useSessionStore()
const isManager = computed(() => session.user?.role === 'manager')
const taskId = computed(() => String(route.params.id))

const labels = ref<LabelItem[]>([])
const users = ref<UserItem[]>([])
const setupError = ref('')

onMounted(async () => {
  try {
    labels.value = (await listLabels()).labels
    // Daftar akun hanya bisa dibaca pengelola, dan hanya Manajer yang butuh untuk memilih PIC di form.
    if (isManager.value) users.value = await listUsers()
  } catch {
    setupError.value = 'Daftar label dan akun gagal dimuat. Muat ulang halaman sebelum mengubah task.'
  }
})

const notice = ref<{ kind: 'success' | 'error'; text: string } | null>(null)
// Kunci berubah setelah form tersimpan, supaya isi task dimuat ulang dari server.
const reloadKey = ref(0)
const formOpen = ref(false)
const editing = ref<TaskDetail | null>(null)

function onChanged(message: string) {
  notice.value = { kind: 'success', text: message }
}

function onEdit(t: TaskDetail) {
  editing.value = t
  formOpen.value = true
}

function onSaved(_id: string, message: string, partial: boolean) {
  formOpen.value = false
  notice.value = { kind: partial ? 'error' : 'success', text: message }
  reloadKey.value++
}

// Dipicu setelah task dihapus: halamannya sudah tidak punya isi.
function backToBoard() {
  void router.push({ name: 'tasks' })
}
</script>

<template>
  <section>
    <RouterLink
      :to="{ name: 'tasks' }"
      class="-ml-2 inline-flex min-h-11 items-center gap-2 rounded-card px-2 text-sm font-medium text-ink-muted hover:bg-canvas hover:text-ink active:bg-canvas"
    >
      <svg
        aria-hidden="true"
        viewBox="0 0 24 24"
        class="size-4"
        fill="none"
        stroke="currentColor"
        stroke-width="1.75"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="M19 12H5M11 6l-6 6 6 6" />
      </svg>
      Papan Task
    </RouterLink>

    <h1 class="mt-1 font-mono text-sm text-ink-muted">{{ taskId }}</h1>

    <div aria-live="polite">
      <p v-if="setupError" role="alert" class="mt-3 rounded-card bg-danger-bg p-3 text-sm text-danger">
        {{ setupError }}
      </p>
      <p
        v-if="notice"
        :role="notice.kind === 'error' ? 'alert' : 'status'"
        class="mt-3 rounded-card p-3 text-sm"
        :class="
          notice.kind === 'error'
            ? 'bg-danger-bg text-danger'
            : 'border border-line bg-surface text-progress-fg'
        "
      >
        {{ notice.text }}
      </p>
    </div>

    <div class="mt-4">
      <TaskDetailBody
        :key="`${taskId}-${reloadKey}`"
        :task-id="taskId"
        :labels="labels"
        wide
        @close="backToBoard"
        @changed="onChanged"
        @edit="onEdit"
      />
    </div>

    <TaskFormPanel
      v-if="isManager"
      :open="formOpen"
      :task="editing"
      :users="users"
      :labels="labels"
      @close="formOpen = false"
      @saved="onSaved"
    />
  </section>
</template>
