<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import type { PMNode } from '../../utils/whiteboard'
import WbRichField, { activateField } from './WbRichField.vue'
import WbRichText from './WbRichText'
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
// Balik mendatar dan tegak (revisi 50) hanya membalik gambar bentuk di kotaknya; teks tetap terbaca.
const pathTransform = computed(() => {
  const fx = props.content.flip_x ? Math.max(1, props.w - sw.value) : 0
  const fy = props.content.flip_y ? Math.max(1, props.h - sw.value) : 0
  const flip = fx || fy ? ` translate(${fx} ${fy}) scale(${fx ? -1 : 1} ${fy ? -1 : 1})` : ''
  return `translate(${inset.value} ${inset.value})${flip}`
})
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
    case 'uml_usecase':
      return { top: '16%', bottom: '16%', left: '14%', right: '14%' }
    case 'uml_system':
      return { top: '8px', bottom: 'auto', left: '10px', right: '10px' }
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

// Draf lokal selama mengedit: gabungan semua bagian yang sudah diketik, dikirim utuh sebagai draf langsung dan saat
// bagian itu selesai diedit (pindah bagian atau editor ditutup). Siaran orang lain tidak menimpa ketikan.
type Part = 'text' | 'name' | 'attributes' | 'methods'
const docKey = {
  text: 'doc',
  name: 'name_doc',
  attributes: 'attributes_doc',
  methods: 'methods_doc',
} as const
const uid = useId()
const key = (p: Part) => `${uid}:${p}`
const draft = ref<ShapeContent>({ ...props.content })
watch(
  () => props.editing,
  (on) => {
    if (!on) return
    draft.value = { ...props.content }
    activateField(key(isClass.value ? 'name' : 'text'))
  },
  { immediate: true },
)

function put(p: Part, v: { text: string; doc: PMNode | null }) {
  draft.value = { ...draft.value, [p]: v.text, [docKey[p]]: v.doc }
}
function input(p: Part, v: { text: string; doc: PMNode | null }) {
  put(p, v)
  emit('live', { ...draft.value })
}
function save(p: Part, v: { text: string; doc: PMNode | null }) {
  put(p, v)
  emit('save', { ...draft.value })
}
const field = 'block w-full'
</script>

<template>
  <div class="relative size-full" :class="content.fill === 'krem' ? 'wb-card-krem' : ''">
    <svg :width="w" :height="h" class="absolute inset-0 overflow-visible" aria-hidden="true">
      <path
        :d="d"
        :transform="pathTransform"
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
      class="absolute inset-0 flex flex-col overflow-hidden text-ink"
      :style="{ fontSize: `${content.font_size}px` }"
    >
      <template v-if="editing">
        <WbRichField
          :field-key="key('name')"
          :text="content.name ?? ''"
          :doc="content.name_doc"
          label="Nama kelas"
          placeholder="Kelas"
          :class="field"
          class="px-2 py-1 text-center font-bold"
          @input="input('name', $event)"
          @commit="save('name', $event)"
          @enter="activateField(key('attributes'))"
          @close="emit('close')"
        />
        <WbRichField
          :field-key="key('attributes')"
          :text="content.attributes ?? ''"
          :doc="content.attributes_doc"
          label="Atribut kelas, satu per baris"
          :single-line="false"
          :class="field"
          :style="divider"
          class="flex-1 px-2 py-1 font-mono"
          @input="input('attributes', $event)"
          @commit="save('attributes', $event)"
          @close="emit('close')"
        />
        <WbRichField
          :field-key="key('methods')"
          :text="content.methods ?? ''"
          :doc="content.methods_doc"
          label="Metode kelas, satu per baris"
          :single-line="false"
          :class="field"
          :style="divider"
          class="flex-1 px-2 py-1 font-mono"
          @input="input('methods', $event)"
          @commit="save('methods', $event)"
          @close="emit('close')"
        />
      </template>
      <template v-else>
        <WbRichText v-if="content.name_doc" :doc="content.name_doc" class="px-2 py-1 text-center font-bold" />
        <p v-else class="px-2 py-1 text-center font-bold break-words" data-caret>
          {{ content.name || 'Kelas' }}
        </p>
        <div class="min-h-6 flex-1 px-2 py-1 font-mono break-words whitespace-pre-wrap" :style="divider">
          <WbRichText v-if="content.attributes_doc" :doc="content.attributes_doc" />
          <template v-else>{{ content.attributes }}</template>
        </div>
        <div class="min-h-6 flex-1 px-2 py-1 font-mono break-words whitespace-pre-wrap" :style="divider">
          <WbRichText v-if="content.methods_doc" :doc="content.methods_doc" />
          <template v-else>{{ content.methods }}</template>
        </div>
      </template>
    </div>

    <div
      v-else
      class="absolute flex items-center text-ink"
      :class="content.shape === 'uml_system' ? 'font-semibold' : ''"
      :style="textStyle"
    >
      <WbRichField
        v-if="editing"
        :field-key="key('text')"
        :text="content.text"
        :doc="content.doc"
        label="Teks bentuk"
        :single-line="false"
        class="max-h-full w-full overflow-auto leading-snug"
        @input="input('text', $event)"
        @commit="save('text', $event)"
        @close="emit('close')"
      />
      <WbRichText
        v-else-if="content.doc"
        :doc="content.doc"
        class="max-h-full w-full overflow-hidden leading-snug [overflow-wrap:anywhere]"
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
