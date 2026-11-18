<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import PageHeader from '../components/PageHeader.vue'
import TaskForm from '../components/TaskForm.vue'
import LoadingSkeleton from '../components/LoadingSkeleton.vue'
import { listLabels, type LabelItem } from '../api/labels'
import { listUsers, type UserItem } from '../api/users'

const router = useRouter()
const labels = ref<LabelItem[]>([])
const users = ref<UserItem[]>([])
const state = ref<'loading' | 'ready' | 'error'>('loading')
const form = ref<InstanceType<typeof TaskForm> | null>(null)

async function load() {
  state.value = 'loading'
  try {
    const [l, u] = await Promise.all([listLabels(), listUsers()])
    labels.value = l.labels
    users.value = u
    state.value = 'ready'
  } catch {
    state.value = 'error'
  }
}
onMounted(load)

// replace: tombol Kembali dari halaman task tidak membuka form yang sudah terkirim. Pesannya ikut lewat
// history.state supaya lampiran yang gagal diunggah tetap diberitahukan di halaman task.
function onSaved(id: string, message: string, partial: boolean) {
  void router.replace({ name: 'task-detail', params: { id }, state: { notice: message, partial } })
}

const buttonSecondary =
  'inline-flex min-h-11 items-center rounded-card border border-line px-4 font-medium hover:bg-canvas active:translate-y-px'
</script>

<template>
  <section class="mx-auto max-w-2xl">
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

    <PageHeader class="mt-1" title="Buat task" />

    <LoadingSkeleton v-if="state === 'loading'" class="mt-4" label="Memuat daftar akun dan label" />
    <div
      v-else-if="state === 'error'"
      role="alert"
      class="mt-4 rounded-card bg-danger-bg p-3 text-sm text-danger"
    >
      <p>Daftar akun dan label gagal dimuat, jadi PIC belum bisa dipilih.</p>
      <button type="button" class="mt-2 bg-surface text-ink" :class="buttonSecondary" @click="load">
        Muat ulang
      </button>
    </div>
    <template v-else>
      <div class="mt-4 rounded-card border border-line bg-surface p-5">
        <TaskForm ref="form" :open="true" :task="null" :users="users" :labels="labels" @saved="onSaved" />
      </div>
      <div class="mt-4 flex flex-wrap justify-end gap-2 border-t border-line pt-4">
        <RouterLink :to="{ name: 'tasks' }" :class="buttonSecondary">Batal</RouterLink>
        <button
          type="submit"
          form="task-form"
          :disabled="form?.submitting"
          class="inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-on-accent hover:bg-accent-hover active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60"
        >
          {{ form?.submitLabel ?? 'Buat task' }}
        </button>
      </div>
    </template>
  </section>
</template>
