<script setup lang="ts">
import { ref, useId } from 'vue'
import UserAvatar from './UserAvatar.vue'
import type { TaskCounts } from '../api/profile'
import { useSessionStore } from '../stores/session'
import { statusClass, statusLabel, type TaskStatus } from '../utils/task'

defineProps<{
  // card: kolom kiri di layar lebar. bar: ringkas di atas papan, isinya dibuka dengan tombol.
  variant: 'card' | 'bar'
  counts: TaskCounts | null
  countsState: 'loading' | 'ready' | 'error'
  // Admin tidak pernah menjadi PIC, jadi angkanya selalu nol dan tidak ditampilkan.
  showCounts: boolean
}>()
const emit = defineEmits<{ edit: []; pick: [status: TaskStatus]; retry: [] }>()

const session = useSessionStore()
const open = ref(false)
const uid = useId()
const order: TaskStatus[] = ['requested', 'on_progress', 'ready', 'completed']

const focusRing = 'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent'
const buttonSecondary =
  'inline-flex min-h-11 items-center justify-center rounded-card border border-line bg-surface px-4 font-medium hover:bg-canvas active:translate-y-px'
</script>

<template>
  <section
    v-if="session.user"
    :aria-label="`Profil ${session.user.name}`"
    class="rounded-card border border-line bg-surface"
    :class="variant === 'card' ? 'p-4' : ''"
  >
    <button
      v-if="variant === 'bar'"
      type="button"
      class="flex min-h-14 w-full min-w-0 items-center gap-3 rounded-card px-3 py-2 text-left hover:bg-canvas"
      :class="focusRing"
      :aria-expanded="open"
      :aria-controls="`${uid}-body`"
      @click="open = !open"
    >
      <UserAvatar
        :user-id="session.user.id"
        :name="session.user.name"
        :avatar-id="session.user.avatar_id"
        size="md"
        decorative
      />
      <span class="min-w-0 flex-1">
        <span class="block truncate font-medium">{{ session.user.name }}</span>
        <span class="block truncate text-sm text-ink-muted">
          {{ session.user.job_title || 'Jabatan belum diisi' }} · {{ session.user.division_name }}
        </span>
      </span>
      <span class="shrink-0 text-sm text-ink-muted">{{ open ? 'Tutup' : 'Profil saya' }}</span>
      <svg
        aria-hidden="true"
        viewBox="0 0 24 24"
        class="size-5 shrink-0 text-ink-muted"
        :class="open ? 'rotate-180' : ''"
        fill="none"
        stroke="currentColor"
        stroke-width="1.75"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="m6 9 6 6 6-6" />
      </svg>
    </button>

    <div
      v-if="variant === 'card' || open"
      :id="`${uid}-body`"
      :class="variant === 'bar' ? 'border-t border-line p-3' : ''"
    >
      <div v-if="variant === 'card'" class="flex flex-col items-center text-center">
        <UserAvatar
          :user-id="session.user.id"
          :name="session.user.name"
          :avatar-id="session.user.avatar_id"
          size="xl"
          decorative
        />
        <p class="mt-3 font-semibold break-words">{{ session.user.name }}</p>
        <p v-if="session.user.job_title" class="text-sm break-words">{{ session.user.job_title }}</p>
        <p class="text-sm text-ink-muted">{{ session.user.division_name }}</p>
      </div>

      <!-- Jabatan kosong tampil sebagai ajakan mengisi, bukan teks contoh yang terlihat seperti data. -->
      <button
        v-if="!session.user.job_title"
        type="button"
        class="mt-2 inline-flex min-h-11 w-full items-center justify-center rounded-card border border-dashed border-line px-3 text-sm text-ink-muted hover:bg-canvas hover:text-ink"
        :class="focusRing"
        @click="emit('edit')"
      >
        Jabatanmu belum diisi. Tambahkan jabatan
      </button>

      <button type="button" :class="[buttonSecondary, focusRing]" class="mt-3 w-full" @click="emit('edit')">
        Ubah profil
      </button>

      <div v-if="showCounts" class="mt-4 border-t border-line pt-3">
        <h2 :id="`${uid}-counts`" class="text-sm font-semibold">Task saya</h2>
        <p v-if="countsState === 'loading' && !counts" class="mt-2 text-sm text-ink-muted">
          Memuat jumlah task...
        </p>
        <div v-else-if="countsState === 'error' && !counts" role="alert" class="mt-2 text-sm text-danger">
          <p>Jumlah task gagal dimuat.</p>
          <button type="button" :class="[buttonSecondary, focusRing]" class="mt-2" @click="emit('retry')">
            Coba lagi
          </button>
        </div>
        <ul v-else-if="counts" class="mt-2 space-y-1" :aria-labelledby="`${uid}-counts`">
          <li v-for="s in order" :key="s">
            <button
              type="button"
              class="flex min-h-11 w-full items-center justify-between gap-2 rounded-card px-2 text-left hover:bg-canvas"
              :class="focusRing"
              @click="emit('pick', s)"
            >
              <span class="rounded-chip px-2 py-0.5 text-sm font-medium" :class="statusClass[s]">
                {{ statusLabel[s] }}
              </span>
              <span class="font-mono font-medium">
                {{ counts.counts[s] ?? 0 }}<span class="sr-only"> task, tampilkan di papan</span>
              </span>
            </button>
          </li>
        </ul>
        <p v-if="counts" class="mt-2 text-xs text-ink-muted">
          Completed hanya {{ counts.completed_days }} hari terakhir.
        </p>
      </div>
    </div>
  </section>
</template>
