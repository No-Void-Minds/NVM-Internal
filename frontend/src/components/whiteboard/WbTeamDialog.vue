<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import UserAvatar from '../UserAvatar.vue'
import { ApiError } from '../../api/client'
import { listPeople } from '../../api/whiteboard'
import type { PersonInfo } from '../../utils/wbPerson'

// Alat Tim (revisi 51): pilih anggota tim untuk dijadikan kartu orang. Pengguna nonaktif ikut tampil dengan penanda
// teks "Nonaktif". Hanya nama, foto, jabatan, dan divisi yang ditampilkan.
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: []; add: [people: PersonInfo[]] }>()

const uid = useId()
const state = ref<'loading' | 'error' | 'ok'>('loading')
const error = ref('')
const people = ref<PersonInfo[]>([])
const query = ref('')
const division = ref<string | null>(null)
const picked = ref<Set<number>>(new Set())

async function load() {
  state.value = 'loading'
  try {
    people.value = (await listPeople()).people
    state.value = 'ok'
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : 'Daftar anggota tim gagal dimuat. Periksa koneksi.'
    state.value = 'error'
  }
}
watch(
  () => props.open,
  (on) => {
    if (!on) return
    picked.value = new Set()
    query.value = ''
    division.value = null
    void load()
  },
)

const divisions = computed(() =>
  [...new Set(people.value.map((p) => p.division))].sort((a, b) => a.localeCompare(b)),
)
const shown = computed(() => {
  const q = query.value.trim().toLowerCase()
  return people.value.filter(
    (p) => (!division.value || p.division === division.value) && (!q || p.name.toLowerCase().includes(q)),
  )
})

function toggle(id: number) {
  const next = new Set(picked.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  picked.value = next
}
// Pilih satu divisi: semua orang di divisi yang sedang disaring ikut terpilih.
function pickDivision() {
  if (!division.value) return
  const next = new Set(picked.value)
  for (const p of people.value) if (p.division === division.value) next.add(p.id)
  picked.value = next
}
function submit() {
  const list = people.value.filter((p) => picked.value.has(p.id))
  if (list.length) emit('add', list)
}

const chip =
  'inline-flex min-h-11 items-center rounded-card border border-line px-3 text-sm hover:bg-canvas aria-pressed:bg-accent aria-pressed:text-on-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
const btn =
  'inline-flex min-h-11 items-center rounded-card border border-line bg-surface px-3 font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-50'
</script>

<template>
  <BaseDialog :open="open" title="Tambah anggota tim" wide @close="emit('close')">
    <div class="flex flex-col gap-3 px-5 py-4">
      <div class="flex flex-col gap-1">
        <label :for="`${uid}-q`" class="text-sm font-semibold">Cari nama</label>
        <input
          :id="`${uid}-q`"
          v-model="query"
          type="search"
          class="min-h-11 rounded-card border border-line bg-surface px-2"
          placeholder="Ketik nama"
        />
      </div>
      <div role="group" aria-label="Filter divisi" class="flex flex-wrap gap-2">
        <button
          type="button"
          :class="chip"
          :aria-pressed="division === null ? 'true' : 'false'"
          @click="division = null"
        >
          Semua divisi
        </button>
        <button
          v-for="d in divisions"
          :key="d"
          type="button"
          :class="chip"
          :aria-pressed="division === d ? 'true' : 'false'"
          @click="division = d"
        >
          {{ d }}
        </button>
      </div>
      <div class="flex flex-wrap gap-2">
        <button type="button" :class="btn" :disabled="!division" @click="pickDivision">
          Pilih satu divisi{{ division ? ` (${division})` : '' }}
        </button>
      </div>

      <p v-if="state === 'loading'" role="status" class="text-ink-muted">Memuat daftar anggota tim...</p>
      <div v-else-if="state === 'error'" role="alert" class="flex flex-wrap items-center gap-2 text-danger">
        {{ error }}
        <button type="button" :class="btn" class="text-ink" @click="load">Coba lagi</button>
      </div>
      <p v-else-if="shown.length === 0" class="text-ink-muted">Tidak ada nama yang cocok.</p>
      <ul
        v-else
        class="flex max-h-[50dvh] flex-col overflow-y-auto rounded-card border border-line"
        aria-label="Anggota tim"
      >
        <li v-for="p in shown" :key="p.id" class="border-b border-line last:border-b-0">
          <label class="flex min-h-11 cursor-pointer items-center gap-3 px-3 py-1.5 hover:bg-canvas">
            <input
              type="checkbox"
              class="size-5 shrink-0 accent-[var(--color-mark)]"
              :checked="picked.has(p.id)"
              @change="toggle(p.id)"
            />
            <UserAvatar :user-id="p.id" :name="p.name" :src="p.avatar_url" size="md" decorative />
            <span class="min-w-0 flex-1">
              <span class="block font-medium break-words">{{ p.name }}</span>
              <span class="block text-sm break-words text-ink-muted">
                {{ [p.job_title, p.division].filter(Boolean).join(', ') }}
              </span>
            </span>
            <span
              v-if="!p.active"
              class="shrink-0 rounded-chip border border-line px-1.5 text-xs font-semibold text-ink"
            >
              Nonaktif
            </span>
          </label>
        </li>
      </ul>

      <div class="flex flex-wrap items-center justify-end gap-2">
        <span class="mr-auto text-sm text-ink-muted">{{ picked.size }} dipilih</span>
        <button type="button" :class="btn" @click="emit('close')">Batal</button>
        <button
          type="button"
          class="inline-flex min-h-11 items-center rounded-card bg-accent px-3 font-medium text-on-accent hover:bg-accent-hover disabled:opacity-50"
          :disabled="picked.size === 0"
          @click="submit"
        >
          Tambahkan {{ picked.size || '' }} orang
        </button>
      </div>
    </div>
  </BaseDialog>
</template>
