<script setup lang="ts">
import { computed, nextTick, ref, useId, watch } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import { useSessionStore } from '../../stores/session'
import {
  PALETTE,
  SHAPE_GROUPS,
  openShape,
  pushRecent,
  shapeMarks,
  shapePath,
  solidShape,
  type PaletteEntry,
} from '../../utils/wbShapes'

// Palet bentuk (W5, revisi 54): kotak cari sebagai combobox yang mengendalikan listbox. Panah atas dan bawah, Home,
// dan End berpindah opsi; Enter menaruh bentuk di tengah layar (atau di titik lepas alat); Escape menutup lewat
// BaseDialog. Pencarian lintas kelompok, kelompok bisa dilipat (opsi di kelompok terlipat tidak ikut navigasi),
// "Terakhir dipakai" paling atas, dan pratinjau besar untuk opsi aktif (dari hover maupun panah).
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: []; pick: [entry: PaletteEntry] }>()

const uid = useId()
const session = useSessionStore()
const q = ref('')
const active = ref(0)
const input = ref<HTMLInputElement | null>(null)
const folded = ref<Set<string>>(new Set())
const RECENT = 'Terakhir dipakai'

// Terakhir dipakai per pengguna di peramban ini; localStorage bisa tidak tersedia (mode privat, kebijakan), jadi
// galatnya diabaikan dan daftar kosong.
const storeKey = computed(() => `nvm-wb-bentuk-terakhir:${session.user?.id ?? 0}`)
const recent = ref<string[]>([])
function loadRecent() {
  try {
    const v = JSON.parse(localStorage.getItem(storeKey.value) ?? '[]')
    recent.value = Array.isArray(v) ? v.filter((k) => typeof k === 'string').slice(0, 12) : []
  } catch {
    recent.value = []
  }
}
function remember(key: string) {
  recent.value = pushRecent(recent.value, key)
  try {
    localStorage.setItem(storeKey.value, JSON.stringify(recent.value))
  } catch {
    // Lihat komentar loadRecent.
  }
}

const byKey = new Map(PALETTE.map((e) => [e.key, e]))
const matches = computed(() => {
  const t = q.value.trim().toLowerCase()
  return PALETTE.filter((e) => !t || e.label.toLowerCase().includes(t) || e.group.toLowerCase().includes(t))
})
// Kelompok tampil; saat mencari semua kelompok terbuka supaya hasil tidak tersembunyi.
const groups = computed(() => {
  const searching = !!q.value.trim()
  const out: { g: string; items: PaletteEntry[]; open: boolean }[] = []
  const rec = recent.value.flatMap((k) => byKey.get(k) ?? [])
  if (!searching && rec.length) out.push({ g: RECENT, items: rec, open: !folded.value.has(RECENT) })
  for (const g of SHAPE_GROUPS) {
    const items = matches.value.filter((e) => e.group === g)
    if (items.length) out.push({ g, items, open: searching || !folded.value.has(g) })
  }
  return out
})
// Opsi yang bisa dinavigasi: id unik per kelompok, karena satu bentuk bisa ada di Terakhir dipakai dan kelompoknya.
const flat = computed(() =>
  groups.value.flatMap((g) => (g.open ? g.items.map((e) => ({ e, id: optId(g.g, e.key) })) : [])),
)
const current = computed(() => flat.value[active.value] ?? null)
function optId(g: string, key: string) {
  return `${uid}-opt-${g === RECENT ? 'r' : 'g'}-${key.replace(/[^a-z0-9_-]/gi, '-')}`
}

watch(
  () => props.open,
  async (o) => {
    if (!o) return
    q.value = ''
    active.value = 0
    loadRecent()
    await nextTick()
    input.value?.focus()
  },
)
watch(q, () => (active.value = 0))

function toggle(g: string) {
  const next = new Set(folded.value)
  if (next.has(g)) next.delete(g)
  else next.add(g)
  folded.value = next
  active.value = Math.min(active.value, Math.max(0, flat.value.length - 1))
}

// mousemove, bukan mouseenter: hasil cari yang bergeser ke bawah kursor yang diam tidak boleh merebut opsi aktif
// pilihan keyboard.
function hover(id: string) {
  const i = flat.value.findIndex((x) => x.id === id)
  if (i >= 0 && i !== active.value) active.value = i
}

function pick(e: PaletteEntry) {
  remember(e.key)
  emit('pick', e)
}

function onKey(ev: KeyboardEvent) {
  const n = flat.value.length
  if (!n) return
  if (ev.key === 'ArrowDown') active.value = (active.value + 1) % n
  else if (ev.key === 'ArrowUp') active.value = (active.value - 1 + n) % n
  else if (ev.key === 'Home') active.value = 0
  else if (ev.key === 'End') active.value = n - 1
  else if (ev.key === 'Enter') pick(flat.value[active.value]!.e)
  else return
  ev.preventDefault()
  document.getElementById(flat.value[active.value]!.id)?.scrollIntoView({ block: 'nearest' })
}

// Digambar di ukuran bawaan lalu diskalakan (ikon task berukuran px tetap); garis tidak ikut mengecil.
function preview(e: PaletteEntry, size: number) {
  const k = (size - 4) / Math.max(e.w, e.h)
  return {
    d: shapePath(e.shape, e.w, e.h),
    marks: shapeMarks({ shape: e.shape, variant: e.variant }, e.w, e.h),
    t: `translate(${(size - e.w * k) / 2} ${(size - e.h * k) / 2}) scale(${k})`,
    open: openShape(e.shape),
    solid: solidShape(e.shape),
  }
}
</script>

<template>
  <BaseDialog :open="open" title="Pilih bentuk" @close="emit('close')">
    <div class="flex flex-col gap-3 px-5 py-4">
      <label :for="`${uid}-q`" class="text-sm font-semibold">Cari bentuk</label>
      <input
        :id="`${uid}-q`"
        ref="input"
        v-model="q"
        type="text"
        role="combobox"
        aria-autocomplete="list"
        aria-expanded="true"
        :aria-controls="`${uid}-list`"
        :aria-activedescendant="current?.id"
        autocomplete="off"
        class="min-h-11 rounded-card border border-line bg-surface px-3"
        placeholder="Contoh: gateway, kelas, panah"
        @keydown="onKey"
      />
      <p v-if="groups.length === 0" role="status" class="text-sm text-ink-muted">
        Tidak ada bentuk dengan nama itu.
      </p>
      <!-- Pratinjau opsi aktif: dekoratif, namanya sudah terbaca dari opsi itu sendiri. -->
      <div
        v-if="current"
        aria-hidden="true"
        class="flex min-h-24 items-center gap-3 rounded-card border border-line bg-canvas p-2"
        data-testid="wb-shape-preview"
      >
        <svg width="88" height="88" viewBox="-4 -4 96 96" class="shrink-0">
          <g :transform="preview(current.e, 88).t">
            <path
              :d="preview(current.e, 88).d"
              :fill="
                preview(current.e, 88).open
                  ? 'none'
                  : preview(current.e, 88).solid
                    ? 'currentColor'
                    : 'var(--color-surface)'
              "
              stroke="currentColor"
              stroke-width="2"
              stroke-linejoin="round"
              vector-effect="non-scaling-stroke"
            />
            <path
              v-for="(m, i) in preview(current.e, 88).marks"
              :key="i"
              :d="m.d"
              :fill="
                m.fill === 'line' ? 'currentColor' : m.fill === 'surface' ? 'var(--color-surface)' : 'none'
              "
              :stroke="m.stroke === 'surface' ? 'var(--color-surface)' : 'currentColor'"
              :stroke-width="2 * (m.width ?? 1)"
              stroke-linejoin="round"
              vector-effect="non-scaling-stroke"
            />
          </g>
        </svg>
        <div class="min-w-0">
          <p class="font-semibold break-words">{{ current.e.label }}</p>
          <p class="text-sm text-ink-muted">{{ current.e.group }}</p>
        </div>
      </div>
      <div :id="`${uid}-list`" role="listbox" aria-label="Bentuk" class="max-h-[45vh] overflow-y-auto">
        <div
          v-for="(g, gi) in groups"
          :key="g.g"
          role="group"
          :aria-labelledby="`${uid}-g-${gi}`"
          class="mb-2"
        >
          <button
            :id="`${uid}-g-${gi}`"
            type="button"
            class="flex min-h-11 w-full items-center gap-2 rounded-card px-1 text-left text-sm font-semibold text-ink-muted hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
            :aria-expanded="g.open ? 'true' : 'false'"
            data-testid="wb-shape-group"
            @click="toggle(g.g)"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 16 16"
              class="size-4 shrink-0"
              :class="g.open ? '' : '-rotate-90'"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
            >
              <path d="M4 6l4 4 4-4" />
            </svg>
            {{ g.g }} <span class="font-normal">({{ g.items.length }})</span>
          </button>
          <div v-if="g.open" class="grid grid-cols-[repeat(auto-fill,minmax(6.5rem,1fr))] gap-1">
            <div
              v-for="e in g.items"
              :id="optId(g.g, e.key)"
              :key="e.key"
              role="option"
              :aria-selected="current?.id === optId(g.g, e.key) ? 'true' : 'false'"
              class="flex min-h-11 min-w-0 cursor-pointer flex-col items-center gap-1 rounded-card border border-transparent p-2 text-center text-sm break-words hover:bg-canvas aria-selected:border-mark aria-selected:bg-canvas"
              :data-shape="e.shape"
              :data-key="e.key"
              @click="pick(e)"
              @mousemove="hover(optId(g.g, e.key))"
            >
              <svg aria-hidden="true" width="40" height="40" viewBox="-2 -2 40 40">
                <g :transform="preview(e, 36).t">
                  <path
                    :d="preview(e, 36).d"
                    :fill="
                      preview(e, 36).open
                        ? 'none'
                        : preview(e, 36).solid
                          ? 'currentColor'
                          : 'var(--color-surface)'
                    "
                    stroke="currentColor"
                    stroke-width="1.75"
                    stroke-linejoin="round"
                    vector-effect="non-scaling-stroke"
                  />
                  <path
                    v-for="(m, i) in preview(e, 36).marks"
                    :key="i"
                    :d="m.d"
                    :fill="
                      m.fill === 'line'
                        ? 'currentColor'
                        : m.fill === 'surface'
                          ? 'var(--color-surface)'
                          : 'none'
                    "
                    :stroke="m.stroke === 'surface' ? 'var(--color-surface)' : 'currentColor'"
                    :stroke-width="1.25 * (m.width ?? 1)"
                    stroke-linejoin="round"
                    vector-effect="non-scaling-stroke"
                  />
                </g>
              </svg>
              {{ e.label }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </BaseDialog>
</template>
