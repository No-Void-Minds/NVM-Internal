<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { inkColor } from './wbColors'
import { openShape, shapePath, type ShapeContent } from '../../utils/wbShapes'

// Isi kartu bentuk (W5): SVG dari angka (tanpa markup pengguna) dan teks lewat interpolasi Vue. Putaran diterapkan
// WbCard ke seluruh kartu, jadi di sini bentuk digambar tegak di kotak w x h.
const props = defineProps<{ content: ShapeContent; w: number; h: number; editing: boolean }>()
const emit = defineEmits<{ save: [c: ShapeContent]; live: [c: ShapeContent]; close: [] }>()

const fillVar: Record<string, string> = {
  putih: 'var(--color-surface)',
  abu: 'var(--color-requested-bg)',
  hijau: 'var(--color-progress-bg)',
  kuning: 'var(--color-ready-bg)',
  biru: 'var(--color-completed-bg)',
  krem: 'var(--color-tag)',
  merah: 'var(--color-danger-bg)',
}

const sw = computed(() => props.content.stroke_width)
// Garis tepi digambar di dalam kotak: path dikecilkan setengah tebal garis supaya tidak terpotong tepi SVG.
const inset = computed(() => sw.value / 2)
const d = computed(() =>
  shapePath(props.content.shape, Math.max(1, props.w - sw.value), Math.max(1, props.h - sw.value)),
)
const fill = computed(() =>
  openShape(props.content.shape) ? 'none' : (fillVar[props.content.fill] ?? 'none'),
)
const stroke = computed(() => inkColor(props.content.stroke))
const isClass = computed(() => props.content.shape === 'uml_class')

// Area teks di dalam bentuk: bentuk runcing (belah ketupat, segitiga, bintang) memberi ruang lebih sempit.
const textInset = computed(() => {
  switch (props.content.shape) {
    case 'diamond':
    case 'decision':
    case 'star':
      return { top: '28%', bottom: '28%', left: '24%', right: '24%' }
    case 'triangle':
      return { top: '45%', bottom: '8%', left: '24%', right: '24%' }
    case 'ellipse':
      return { top: '16%', bottom: '16%', left: '14%', right: '14%' }
    case 'uml_actor':
      return { top: '102%', bottom: '-30%', left: '-40%', right: '-40%' }
    case 'uml_package':
      return { top: '22%', bottom: '8px', left: '8px', right: '8px' }
    case 'database':
      return { top: '24%', bottom: '10%', left: '8px', right: '8px' }
    case 'arrow_right':
    case 'arrow_left':
    case 'arrow_both':
      return { top: '26%', bottom: '26%', left: '14%', right: '14%' }
    default:
      return { top: '8px', bottom: '8px', left: '10px', right: '10px' }
  }
})
const textStyle = computed(() => ({
  ...textInset.value,
  fontSize: `${props.content.font_size}px`,
  textAlign: props.content.align,
  justifyContent:
    props.content.align === 'left' ? 'flex-start' : props.content.align === 'right' ? 'flex-end' : 'center',
}))
const divider = computed(() => ({ borderTop: `${sw.value}px solid ${stroke.value}` }))

// Draf lokal selama mengedit; siaran orang lain tidak menimpa ketikan (sama dengan drafts.ts untuk checklist).
const draft = ref<ShapeContent>({ ...props.content })
watch(
  () => props.editing,
  async (on) => {
    if (!on) return
    draft.value = { ...props.content }
    await nextTick()
    const el = first.value
    el?.focus()
    if (el && 'setSelectionRange' in el) el.setSelectionRange(el.value.length, el.value.length)
  },
  { immediate: true },
)
const first = ref<HTMLTextAreaElement | HTMLInputElement | null>(null)
const editor = ref<HTMLElement | null>(null)

function input() {
  emit('live', { ...draft.value })
}
function save() {
  const c = props.content
  const d2 = draft.value
  if (d2.text !== c.text || d2.name !== c.name || d2.attributes !== c.attributes || d2.methods !== c.methods)
    emit('save', { ...d2 })
}
// Simpan saat fokus keluar dari seluruh editor (bukan antar isian kelas UML).
function onFocusOut(e: FocusEvent) {
  if (editor.value?.contains(e.relatedTarget as Node)) return
  save()
  emit('close')
}
function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    e.stopPropagation()
    save()
    emit('close')
  }
}
const field =
  'block w-full resize-none bg-transparent outline-none placeholder:text-ink-muted focus-visible:bg-surface/60'
</script>

<template>
  <div class="relative size-full" :class="content.fill === 'krem' ? 'wb-card-krem' : ''">
    <svg :width="w" :height="h" class="absolute inset-0 overflow-visible" aria-hidden="true">
      <path
        :d="d"
        :transform="`translate(${inset} ${inset})`"
        :fill="fill"
        :stroke-width="sw"
        :stroke-dasharray="content.stroke_style === 'dashed' ? `${sw * 4} ${sw * 3}` : undefined"
        stroke-linejoin="round"
        stroke-linecap="round"
        :style="{ stroke }"
      />
    </svg>

    <!-- Kelas UML: tiga bagian bergaris pemisah. -->
    <div
      v-if="isClass"
      ref="editor"
      class="absolute inset-0 flex flex-col overflow-hidden text-ink"
      :style="{ fontSize: `${content.font_size}px` }"
      @focusout="editing && onFocusOut($event)"
      @keydown="editing && onKey($event)"
    >
      <template v-if="editing">
        <input
          ref="first"
          v-model="draft.name"
          aria-label="Nama kelas"
          maxlength="200"
          :class="field"
          class="px-2 py-1 text-center font-bold"
          @pointerdown.stop
          @input="input"
        />
        <textarea
          v-model="draft.attributes"
          aria-label="Atribut kelas, satu per baris"
          maxlength="2000"
          rows="3"
          :class="field"
          :style="divider"
          class="flex-1 px-2 py-1 font-mono"
          @pointerdown.stop
          @input="input"
        />
        <textarea
          v-model="draft.methods"
          aria-label="Metode kelas, satu per baris"
          maxlength="2000"
          rows="3"
          :class="field"
          :style="divider"
          class="flex-1 px-2 py-1 font-mono"
          @pointerdown.stop
          @input="input"
        />
      </template>
      <template v-else>
        <p class="px-2 py-1 text-center font-bold break-words" data-caret>{{ content.name || 'Kelas' }}</p>
        <p class="min-h-6 flex-1 px-2 py-1 font-mono break-words whitespace-pre-wrap" :style="divider">
          {{ content.attributes }}
        </p>
        <p class="min-h-6 flex-1 px-2 py-1 font-mono break-words whitespace-pre-wrap" :style="divider">
          {{ content.methods }}
        </p>
      </template>
    </div>

    <div
      v-else
      ref="editor"
      class="absolute flex items-center text-ink"
      :style="textStyle"
      @focusout="editing && onFocusOut($event)"
      @keydown="editing && onKey($event)"
    >
      <textarea
        v-if="editing"
        ref="first"
        v-model="draft.text"
        aria-label="Teks bentuk"
        maxlength="2000"
        :class="field"
        class="h-full leading-snug"
        :style="{ textAlign: content.align }"
        @pointerdown.stop
        @input="input"
      />
      <p
        v-else-if="content.text"
        data-caret
        class="max-h-full w-full overflow-hidden leading-snug break-words whitespace-pre-wrap [overflow-wrap:anywhere]"
      >
        {{ content.text }}
      </p>
    </div>
  </div>
</template>
