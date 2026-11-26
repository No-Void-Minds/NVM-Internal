<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, useId } from 'vue'

// Pemilih popover Whiteboard (revisi 52), pengganti select bawaan browser di seluruh Whiteboard supaya gayanya sama
// dengan toolbar pena. Pilihan teks tampil sebagai daftar; pilihan dengan swatch (palet warna) tampil sebagai kisi
// kotak 44 px. Setiap kotak punya nama di aria-label dan tooltip, dan pilihan aktif ditandai centang (bukan warna
// saja). Keyboard: Enter, Spasi, atau panah bawah membuka; panah (kisi: atas dan bawah per baris) pindah pilihan;
// Enter memilih; Escape atau Tab menutup dan fokus kembali ke pemicu.
export type PickOption = { value: string; label: string; swatch?: string }
export type PickExtra = { id: string; label: string; pressed?: boolean; run: () => void }

const props = withDefaults(
  defineProps<{
    label: string
    options: readonly PickOption[]
    modelValue: string | null
    id?: string
    disabled?: boolean
    // Nilai berbeda di antara kartu terpilih.
    mixed?: boolean
    // Aksi tambahan di bawah pilihan (mis. Tanpa latar, Gaya lainnya...).
    extras?: PickExtra[]
    // Pemicu ringkas: hanya swatch (bar pilihan); bawaannya label pilihan aktif.
    compact?: boolean
  }>(),
  { id: undefined, disabled: false, mixed: false, extras: () => [], compact: false },
)
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
// Kelas dari induk (lebar) dipasang di tombol pemicu; komponen ini punya dua akar (pemicu dan popover).
defineOptions({ inheritAttrs: false })

const uid = useId()
const open = ref(false)
const trigger = ref<HTMLButtonElement | null>(null)
const pop = ref<HTMLElement | null>(null)
const pos = ref({ left: 0, top: 0 })
const swatches = computed(() => props.options.some((o) => o.swatch !== undefined))
const cols = computed(() => (swatches.value ? Math.min(4, props.options.length) : 1))
const current = computed(() => props.options.find((o) => o.value === props.modelValue))
const currentLabel = computed(() => (props.mixed ? 'campuran' : (current.value?.label ?? 'belum dipilih')))

function items() {
  return [...(pop.value?.querySelectorAll<HTMLElement>('[data-pick]') ?? [])]
}

// Popover fixed di bawah pemicu (di atas bila di bawah tidak cukup ruang), dijepit 8 px di dalam layar. Fixed juga
// bekerja di dalam dialog modal (top layer), tempat Teleport ke body tidak terlihat.
async function show() {
  if (props.disabled) return
  open.value = true
  await nextTick()
  const t = trigger.value?.getBoundingClientRect()
  const p = pop.value
  if (!t || !p) return
  const vw = document.documentElement.clientWidth
  const vh = window.innerHeight
  let top = t.bottom + 4
  if (top + p.offsetHeight > vh - 8) top = Math.max(8, t.top - p.offsetHeight - 4)
  pos.value = { left: Math.max(8, Math.min(t.left, vw - p.offsetWidth - 8)), top }
  const list = items()
  const at = list.findIndex((el) => el.dataset.pick === props.modelValue)
  ;(list[at >= 0 ? at : 0] ?? list[0])?.focus()
  document.addEventListener('pointerdown', onOutside, true)
}

function hide(refocus: boolean) {
  open.value = false
  document.removeEventListener('pointerdown', onOutside, true)
  if (refocus) trigger.value?.focus()
}
onBeforeUnmount(() => document.removeEventListener('pointerdown', onOutside, true))

function onOutside(e: PointerEvent) {
  const t = e.target as Node
  if (!pop.value?.contains(t) && !trigger.value?.contains(t)) hide(false)
}

function choose(v: string) {
  emit('update:modelValue', v)
  hide(true)
}
function runExtra(x: PickExtra) {
  hide(true)
  x.run()
}

function onTriggerKey(e: KeyboardEvent) {
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    void show()
  }
}

function onPopKey(e: KeyboardEvent) {
  e.stopPropagation()
  const list = items()
  const i = list.indexOf(document.activeElement as HTMLElement)
  let next = -1
  const n = list.length
  const optCount = props.options.length
  switch (e.key) {
    case 'Escape':
    case 'Tab':
      e.preventDefault()
      hide(true)
      return
    case 'ArrowRight':
      next = (i + 1) % n
      break
    case 'ArrowLeft':
      next = (i - 1 + n) % n
      break
    case 'ArrowDown':
      next =
        i < optCount && i + cols.value < optCount ? i + cols.value : i < optCount ? optCount % n : (i + 1) % n
      break
    case 'ArrowUp':
      next = i >= cols.value && i < optCount ? i - cols.value : (i - 1 + n) % n
      break
    case 'Home':
      next = 0
      break
    case 'End':
      next = n - 1
      break
    default:
      return
  }
  e.preventDefault()
  list[next]?.focus()
}

const triggerCls =
  'inline-flex min-h-11 min-w-11 shrink-0 items-center gap-2 rounded-card border border-line bg-surface px-2 text-left text-sm text-ink hover:bg-canvas aria-expanded:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-60'
const optCls =
  'relative inline-flex min-h-11 items-center gap-2 rounded-card px-3 text-left text-sm text-ink hover:bg-canvas aria-pressed:bg-accent aria-pressed:text-on-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
const swatchCls =
  'relative inline-flex size-11 items-center justify-center rounded-card hover:bg-canvas aria-pressed:outline-2 aria-pressed:outline-mark focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
</script>

<template>
  <button
    v-bind="$attrs"
    :id="id"
    ref="trigger"
    type="button"
    :class="[triggerCls, compact ? 'justify-center px-0' : '']"
    :aria-label="`${label}: ${currentLabel}`"
    :title="`${label}: ${currentLabel}`"
    aria-haspopup="true"
    :aria-expanded="open"
    :aria-controls="open ? `${uid}-pop` : undefined"
    :disabled="disabled"
    :data-picker="label"
    @click="open ? hide(false) : show()"
    @keydown="onTriggerKey"
  >
    <span
      v-if="current?.swatch !== undefined && !mixed"
      aria-hidden="true"
      class="size-5 shrink-0 rounded-full border border-line"
      :style="{ background: current.swatch }"
    />
    <span v-if="!compact" class="min-w-0 flex-1 truncate">{{
      mixed ? 'Campuran' : (current?.label ?? '')
    }}</span>
    <svg
      v-if="!compact || current?.swatch === undefined"
      aria-hidden="true"
      viewBox="0 0 16 16"
      class="size-4 shrink-0 text-ink-muted"
      fill="none"
      stroke="currentColor"
      stroke-width="1.75"
    >
      <path d="M4 6l4 4 4-4" />
    </svg>
  </button>
  <div
    v-if="open"
    :id="`${uid}-pop`"
    ref="pop"
    role="group"
    :aria-label="label"
    data-testid="wb-picker"
    class="fixed z-[80] flex max-h-[min(60dvh,24rem)] max-w-[calc(100vw-1rem)] flex-col gap-1 overflow-y-auto rounded-card border border-line bg-surface p-1 shadow-menu"
    :style="{ left: `${pos.left}px`, top: `${pos.top}px` }"
    @keydown="onPopKey"
    @pointerdown.stop
  >
    <div v-if="swatches" class="grid gap-0.5" :style="{ gridTemplateColumns: `repeat(${cols}, 2.75rem)` }">
      <button
        v-for="o in options"
        :key="o.value"
        type="button"
        :class="swatchCls"
        :data-pick="o.value"
        :aria-label="o.label"
        :title="o.label"
        :aria-pressed="!mixed && o.value === modelValue ? 'true' : 'false'"
        @click="choose(o.value)"
      >
        <span
          aria-hidden="true"
          class="size-7 rounded-full border border-line"
          :style="{ background: o.swatch }"
        />
        <svg
          v-if="!mixed && o.value === modelValue"
          aria-hidden="true"
          viewBox="0 0 16 16"
          class="absolute right-0.5 bottom-0.5 size-4 rounded-full bg-surface text-ink"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path d="M4 8.5l2.5 2.5L12 5.5" />
        </svg>
      </button>
    </div>
    <template v-else>
      <button
        v-for="o in options"
        :key="o.value"
        type="button"
        :class="optCls"
        :data-pick="o.value"
        :aria-pressed="!mixed && o.value === modelValue ? 'true' : 'false'"
        @click="choose(o.value)"
      >
        {{ o.label }}
      </button>
    </template>
    <button
      v-for="x in extras"
      :key="x.id"
      type="button"
      :class="optCls"
      :data-pick="`extra-${x.id}`"
      :aria-pressed="x.pressed === undefined ? undefined : x.pressed ? 'true' : 'false'"
      @click="runExtra(x)"
    >
      {{ x.label }}
    </button>
  </div>
</template>
