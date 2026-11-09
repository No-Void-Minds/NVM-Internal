<script setup lang="ts">
import { computed, nextTick, ref, useId } from 'vue'
import type { UserItem } from '../api/users'
import { initials } from '../utils/person'

const props = defineProps<{
  candidates: UserItem[]
  names: Map<number, string>
  // Dipakai di label tombol supaya pembaca layar tahu ini PIC untuk sub task yang mana, mis. "sub task 2".
  subject: string
}>()
const picked = defineModel<number[]>({ required: true })

const open = ref(false)
const query = ref('')
const uid = useId()
const toggleEl = ref<HTMLButtonElement | null>(null)
const searchEl = ref<HTMLInputElement | null>(null)

const activeIds = computed(() => new Set(props.candidates.map((u) => u.id)))
const chips = computed(() =>
  picked.value.map((id) => ({
    id,
    name: props.names.get(id) ?? `Akun #${id}`,
    active: activeIds.value.has(id),
  })),
)
const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return props.candidates
  return props.candidates.filter(
    (u) => u.name.toLowerCase().includes(q) || u.division_name.toLowerCase().includes(q),
  )
})

function toggle(id: number) {
  picked.value = picked.value.includes(id) ? picked.value.filter((p) => p !== id) : [...picked.value, id]
}

function removeChip(id: number) {
  toggle(id)
  void nextTick(() => toggleEl.value?.focus())
}

function toggleOpen() {
  open.value = !open.value
  if (open.value) void nextTick(() => searchEl.value?.focus())
  else query.value = ''
}

// Escape hanya menutup daftar ini. Tanpa preventDefault dan stopPropagation, panel form di belakangnya ikut tertutup.
function onEscape(e: KeyboardEvent) {
  if (!open.value) return
  e.preventDefault()
  e.stopPropagation()
  open.value = false
  query.value = ''
  toggleEl.value?.focus()
}

const focusRing = 'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent'
</script>

<template>
  <div @keydown.esc="onEscape">
    <ul v-if="chips.length" class="flex flex-wrap gap-2" :aria-label="`PIC ${subject}`">
      <li v-for="c in chips" :key="c.id">
        <button
          type="button"
          class="inline-flex min-h-11 items-center gap-2 rounded-chip border border-line bg-canvas pr-3 pl-1.5 text-sm hover:border-ink"
          :class="focusRing"
          :aria-label="`Lepas ${c.name} dari ${subject}`"
          @click="removeChip(c.id)"
        >
          <span
            aria-hidden="true"
            class="flex size-7 items-center justify-center rounded-full border border-line bg-surface text-xs font-medium"
            >{{ initials(c.name) }}</span
          >
          <span>{{ c.name }}<span v-if="!c.active" class="text-ink-muted"> (tidak aktif)</span></span>
          <svg
            aria-hidden="true"
            viewBox="0 0 24 24"
            class="size-4"
            fill="none"
            stroke="currentColor"
            stroke-width="1.75"
            stroke-linecap="round"
          >
            <path d="M6 6l12 12M18 6 6 18" />
          </svg>
        </button>
      </li>
    </ul>
    <p v-else class="text-sm text-ink-muted">Belum ada PIC.</p>

    <button
      ref="toggleEl"
      type="button"
      class="mt-2 inline-flex min-h-11 items-center gap-2 rounded-card border border-line px-3 text-sm font-medium hover:bg-canvas active:translate-y-px"
      :class="focusRing"
      :aria-expanded="open"
      :aria-controls="`${uid}-list`"
      @click="toggleOpen"
    >
      <svg
        aria-hidden="true"
        viewBox="0 0 24 24"
        class="size-4"
        fill="none"
        stroke="currentColor"
        stroke-width="1.75"
        stroke-linecap="round"
      >
        <path d="M12 5v14M5 12h14" />
      </svg>
      {{ open ? 'Tutup daftar' : 'Pilih PIC' }}<span class="sr-only"> untuk {{ subject }}</span>
    </button>

    <div v-if="open" :id="`${uid}-list`" class="mt-2 rounded-card border border-line">
      <div class="border-b border-line p-2">
        <label :for="`${uid}-q`" class="sr-only">Cari nama atau divisi</label>
        <!-- Enter dicegah karena input ini ada di dalam form task; tanpa ini Enter langsung menyimpan task. -->
        <input
          :id="`${uid}-q`"
          ref="searchEl"
          v-model="query"
          type="search"
          autocomplete="off"
          placeholder="Cari nama atau divisi"
          class="block min-h-11 w-full rounded-card border border-line bg-surface px-3"
          @keydown.enter.prevent
        />
      </div>
      <ul v-if="filtered.length" class="max-h-56 divide-y divide-line overflow-y-auto">
        <li v-for="u in filtered" :key="u.id">
          <label class="flex min-h-11 cursor-pointer items-center gap-3 px-3 hover:bg-canvas">
            <input
              type="checkbox"
              class="size-5 accent-accent"
              :checked="picked.includes(u.id)"
              @change="toggle(u.id)"
            />
            <span class="min-w-0 flex-1">{{ u.name }}</span>
            <span class="text-sm text-ink-muted">{{ u.division_name }}</span>
          </label>
        </li>
      </ul>
      <p v-else class="p-3 text-sm text-ink-muted">
        {{
          candidates.length ? 'Tidak ada akun yang cocok.' : 'Belum ada akun Staf atau Manajer yang aktif.'
        }}
      </p>
    </div>
  </div>
</template>
