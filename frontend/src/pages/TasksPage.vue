<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import TaskDetailPanel from '../components/TaskDetailPanel.vue'
import TaskFormPanel from '../components/TaskFormPanel.vue'
import { ApiError } from '../api/client'
import { listTasks, type TaskDetail, type TaskSummary } from '../api/tasks'
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

const detailId = ref<string | null>(null)
const formOpen = ref(false)
const editing = ref<TaskDetail | null>(null)

// quiet dipakai setelah aksi, supaya daftar tidak berkedip ke tampilan memuat.
async function load(quiet = false) {
  if (!quiet) loadState.value = 'loading'
  try {
    tasks.value = await listTasks(scope.value)
    loadState.value = 'ready'
  } catch (err) {
    loadError.value =
      err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
    loadState.value = 'error'
  }
}
watch(scope, () => load())

onMounted(async () => {
  void load()
  try {
    labels.value = (await listLabels()).labels
    // Daftar akun hanya bisa dibaca pengelola, dan hanya dibutuhkan Manajer untuk memilih PIC.
    if (isManager.value) users.value = await listUsers()
  } catch {
    notice.value = 'Daftar label atau PIC gagal dimuat. Muat ulang halaman sebelum mengubah task.'
  }
})

function openCreate() {
  editing.value = null
  formOpen.value = true
}

// Panel ubah menggantikan panel detail, lalu detail dibuka lagi setelah tersimpan.
function onEdit(t: TaskDetail) {
  editing.value = t
  detailId.value = null
  formOpen.value = true
}

async function onSaved(id: string, message: string) {
  formOpen.value = false
  notice.value = message
  await load(true)
  detailId.value = id
}

async function onChanged(message: string) {
  notice.value = message
  await load(true)
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
        @click="openCreate"
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

    <div aria-live="polite">
      <p
        v-if="notice"
        role="status"
        class="mt-4 rounded-card border border-line bg-surface p-3 text-sm text-progress-fg"
      >
        {{ notice }}
      </p>
    </div>

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
        @click="load()"
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
      <li v-for="t in tasks" :key="t.id">
        <button
          type="button"
          class="block w-full rounded-card border border-line bg-surface p-4 text-left hover:border-accent active:bg-canvas"
          :aria-label="`Buka ${t.id}: ${t.title}`"
          @click="detailId = t.id"
        >
          <span class="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
            <span class="font-mono text-ink-muted">{{ t.id }}</span>
            <span class="rounded-chip px-2 py-0.5 font-medium" :class="statusClass[t.status]">{{
              statusLabel[t.status]
            }}</span>
            <span :class="urgencyClass[t.urgency]">{{ urgencyLabel[t.urgency] }}</span>
          </span>
          <span class="mt-1 block font-medium">{{ t.title }}</span>
          <span v-if="t.labels.length" class="mt-2 flex flex-wrap gap-1">
            <span
              v-for="l in t.labels"
              :key="l.id"
              class="rounded-chip px-2 py-0.5 text-sm"
              :style="{ backgroundColor: l.color }"
            >
              {{ l.name }}
            </span>
          </span>
          <span class="mt-2 block text-sm text-ink-muted">
            {{ t.assignees.map((a) => a.name).join(', ') }} · {{ t.division.name }}
          </span>
          <span
            v-if="t.due_date"
            class="mt-1 block text-sm"
            :class="isOverdue(t.due_date, t.status) ? 'font-medium text-urgent' : 'text-ink-muted'"
          >
            {{ isOverdue(t.due_date, t.status) ? 'Terlambat, ' : 'Tenggat ' }}{{ formatDay(t.due_date) }}
          </span>
        </button>
      </li>
    </ul>

    <TaskDetailPanel
      :task-id="detailId"
      :labels="labels"
      @close="detailId = null"
      @changed="onChanged"
      @edit="onEdit"
    />

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
