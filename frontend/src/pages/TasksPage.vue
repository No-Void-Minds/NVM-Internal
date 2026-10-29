<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import TaskFormPanel from '../components/TaskFormPanel.vue'
import { ApiError } from '../api/client'
import { listTasks, type TaskSummary } from '../api/tasks'
import { listUsers, type UserItem } from '../api/users'
import { listLabels, type LabelItem } from '../api/labels'
import { useSessionStore } from '../stores/session'
import { formatDay, isOverdue, statusClass, statusLabel, urgencyClass, urgencyLabel } from '../utils/task'

const session = useSessionStore()
const isManager = computed(() => session.user?.role === 'manager')
const isStaff = computed(() => session.user?.role === 'staff')

// Staf mulai dari "Task saya"; Manajer dan Admin melihat semua task.
const scope = ref<'mine' | 'all'>(isStaff.value ? 'mine' : 'all')
const tasks = ref<TaskSummary[]>([])
const loadState = ref<'loading' | 'ready' | 'error'>('loading')
const loadError = ref('')
const notice = ref<string | null>(null)

const users = ref<UserItem[]>([])
const labels = ref<LabelItem[]>([])
const panelOpen = ref(false)

async function load() {
  loadState.value = 'loading'
  try {
    tasks.value = await listTasks(scope.value)
    loadState.value = 'ready'
  } catch (err) {
    loadError.value =
      err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
    loadState.value = 'error'
  }
}
watch(scope, load)

onMounted(async () => {
  void load()
  // Daftar akun hanya bisa dibaca pengelola, dan hanya dibutuhkan untuk memilih PIC saat membuat task.
  if (isManager.value) {
    try {
      ;[users.value, labels.value] = await Promise.all([listUsers(), listLabels().then((r) => r.labels)])
    } catch {
      notice.value = 'Daftar PIC atau label gagal dimuat. Muat ulang halaman sebelum membuat task.'
    }
  }
})

async function onSaved(_id: string, message: string) {
  panelOpen.value = false
  notice.value = message
  await load()
}

const scopeButton =
  'inline-flex min-h-11 items-center px-4 font-medium first:rounded-l-card last:rounded-r-card aria-pressed:bg-accent aria-pressed:text-white'
</script>

<template>
  <section>
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="text-2xl font-semibold">Task</h1>
      <button
        v-if="isManager"
        type="button"
        class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-white active:translate-y-px"
        @click="panelOpen = true"
      >
        Buat task
      </button>
    </div>

    <div
      v-if="isStaff"
      class="mt-4 inline-flex rounded-card border border-line bg-surface"
      role="group"
      aria-label="Lingkup task"
    >
      <button type="button" :class="scopeButton" :aria-pressed="scope === 'mine'" @click="scope = 'mine'">
        Task saya
      </button>
      <button type="button" :class="scopeButton" :aria-pressed="scope === 'all'" @click="scope = 'all'">
        Semua task
      </button>
    </div>

    <p
      v-if="notice"
      role="status"
      class="mt-4 rounded-card border border-line bg-surface p-3 text-sm text-progress-fg"
    >
      {{ notice }}
    </p>

    <p v-if="loadState === 'loading'" class="mt-6 text-ink-muted">Memuat task...</p>

    <div
      v-else-if="loadState === 'error'"
      role="alert"
      class="mt-6 rounded-card bg-danger-bg p-4 text-danger"
    >
      <p class="font-medium">{{ loadError }}</p>
      <button
        type="button"
        class="mt-3 inline-flex min-h-11 items-center rounded-card border border-danger px-4 font-medium"
        @click="load"
      >
        Muat ulang
      </button>
    </div>

    <div
      v-else-if="tasks.length === 0"
      class="mt-6 rounded-card border border-line bg-surface p-6 text-center"
    >
      <p v-if="scope === 'mine'">Belum ada task untukmu. Task baru dari Manajer akan muncul di sini.</p>
      <p v-else>Belum ada task.</p>
      <button
        v-if="scope === 'mine'"
        type="button"
        class="mt-3 inline-flex min-h-11 items-center rounded-card border border-line px-4 font-medium hover:bg-canvas"
        @click="scope = 'all'"
      >
        Lihat semua task
      </button>
    </div>

    <ul v-else class="mt-4 space-y-3">
      <li v-for="t in tasks" :key="t.id" class="rounded-card border border-line bg-surface p-4">
        <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
          <span class="font-mono text-ink-muted">{{ t.id }}</span>
          <span class="rounded-chip px-2 py-0.5 font-medium" :class="statusClass[t.status]">{{
            statusLabel[t.status]
          }}</span>
          <span :class="urgencyClass[t.urgency]">{{ urgencyLabel[t.urgency] }}</span>
        </div>
        <p class="mt-1 font-medium">{{ t.title }}</p>
        <div v-if="t.labels.length" class="mt-2 flex flex-wrap gap-1">
          <span
            v-for="l in t.labels"
            :key="l.id"
            class="rounded-chip px-2 py-0.5 text-sm"
            :style="{ backgroundColor: l.color }"
          >
            {{ l.name }}
          </span>
        </div>
        <p class="mt-2 text-sm text-ink-muted">
          {{ t.assignees.map((a) => a.name).join(', ') }} · {{ t.division.name }}
        </p>
        <p
          v-if="t.due_date"
          class="mt-1 text-sm"
          :class="isOverdue(t.due_date, t.status) ? 'font-medium text-urgent' : 'text-ink-muted'"
        >
          {{ isOverdue(t.due_date, t.status) ? 'Terlambat, ' : 'Tenggat ' }}{{ formatDay(t.due_date) }}
        </p>
      </li>
    </ul>

    <TaskFormPanel
      v-if="isManager"
      :open="panelOpen"
      :task="null"
      :users="users"
      :labels="labels"
      @close="panelOpen = false"
      @saved="onSaved"
    />
  </section>
</template>
