<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import type { PMNode } from '../../utils/whiteboard'
import WbRichField, { activateField } from './WbRichField.vue'
import WbRichText from './WbRichText'
import { inkColor } from './wbColors'
import {
  openShape,
  shapeInset,
  shapeMarks,
  shapePath,
  FRAGMENT_TAB,
  classStereotype,
  guardLanes,
  underlineText,
  laneBandOf,
  laneOrient,
  shapeValign,
  solidShape,
  strokeScale,
  type ShapeContent,
} from '../../utils/wbShapes'
import { BAND, laneBoxes, resizeLane, type Lane } from '../../utils/wbContainers'

// Isi kartu bentuk (W5): SVG dari angka (tanpa markup pengguna) dan teks lewat interpolasi Vue. Putaran diterapkan
// WbCard ke seluruh kartu, jadi di sini bentuk digambar tegak di kotak w x h.
const props = defineProps<{
  content: ShapeContent
  w: number
  h: number
  editing: boolean
  selected?: boolean
  canWrite?: boolean
  zoom?: number
}>()
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

// Tebal garis tepi efektif: event selesai BPMN memakai garis tebal (pengali di definisi bentuk).
const sw = computed(() => props.content.stroke_width * strokeScale(props.content.shape))
// Garis tepi digambar di dalam kotak: path dikecilkan setengah tebal garis supaya tidak terpotong tepi SVG.
const inset = computed(() => sw.value / 2)
const d = computed(() =>
  shapePath(props.content.shape, Math.max(1, props.w - sw.value), Math.max(1, props.h - sw.value)),
)
const fill = computed(() =>
  openShape(props.content.shape)
    ? 'none'
    : solidShape(props.content.shape)
      ? inkColor(props.content.stroke)
      : (fillVar[props.content.fill] ?? 'none'),
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
// Varian kelas (revisi 54e): stereotip di atas nama, nama miring untuk kelas abstrak, enum tanpa bagian metode.
const stereo = computed(() => classStereotype(props.content.variant))
const abstract = computed(() => props.content.variant === 'abstract')
const isEnum = computed(() => props.content.variant === 'enum')
const underline = computed(() => underlineText(props.content.shape))

// Area teks dan tanda di dalam bentuk dari definisi bentuk (utils/wbShapes.ts), sama dengan ekspor.
const textInset = computed(() => shapeInset(props.content.shape))

// Pool dan swimlane (revisi 54b): judul di pita kiri (ditulis tegak) atau atas, label tiap lane, dan pembatas lane
// yang bisa diseret atau digeser dengan panah saat bentuk terpilih. Ukuran lane draf selama diseret.
const orient = computed(() => laneOrient(props.content.shape))
const laneDraft = ref<Lane[] | null>(null)
const lanesShown = computed(() => laneDraft.value ?? props.content.lanes ?? [])
// Fragmen sequence (revisi 54d): operand tanpa pita, syaratnya [kondisi] di kiri atas tiap operand, operator di tab.
const guards = computed(() => guardLanes(props.content.shape))
const band = computed(() => laneBandOf(props.content.shape))
const lanes = computed(() =>
  orient.value ? laneBoxes(lanesShown.value, props.w, props.h, orient.value, band.value) : [],
)
const pool = computed(() => orient.value === 'rows' && !guards.value)
const bandStyle = computed(() =>
  orient.value === 'rows'
    ? { top: '0px', bottom: '0px', left: '0px', width: `${BAND}px` }
    : { top: '0px', left: '0px', right: '0px', height: `${BAND}px` },
)
const span = () => (orient.value === 'rows' ? props.h : props.w)
function laneLabelStyle(b: { x: number; y: number; w: number; h: number }, i: number) {
  if (guards.value)
    return { left: '8px', top: `${b.y + (i ? 4 : FRAGMENT_TAB.h + 4)}px`, maxWidth: `${b.w - 16}px` }
  return orient.value === 'rows'
    ? { left: `${BAND}px`, top: `${b.y}px`, width: `${BAND}px`, height: `${b.h}px` }
    : { left: `${b.x}px`, top: `${BAND}px`, width: `${b.w}px`, height: `${BAND}px` }
}
// Area pembatas 44 px di layar pada zoom berapa pun.
function dividerStyle(i: number) {
  const b = lanes.value[i + 1]!
  const t = 44 / (props.zoom || 1)
  return orient.value === 'rows'
    ? { left: `${band.value}px`, right: '0px', top: `${b.y - t / 2}px`, height: `${t}px` }
    : { top: `${band.value}px`, bottom: '0px', left: `${b.x - t / 2}px`, width: `${t}px` }
}
const canDivide = computed(() => !!orient.value && props.selected && props.canWrite && !props.editing)
function commitLanes(next: Lane[]) {
  laneDraft.value = null
  if (JSON.stringify(next) !== JSON.stringify(props.content.lanes))
    emit('save', { ...props.content, lanes: next })
}
function onDividerDown(i: number, e: PointerEvent) {
  const start = props.content.lanes ?? []
  const s0 = orient.value === 'rows' ? e.clientY : e.clientX
  const z = props.zoom || 1
  const move = (ev: PointerEvent) => {
    const d = ((orient.value === 'rows' ? ev.clientY : ev.clientX) - s0) / z
    laneDraft.value = resizeLane(start, i, d, span())
  }
  const up = () => {
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', up)
    window.removeEventListener('pointercancel', up)
    if (laneDraft.value) commitLanes(laneDraft.value)
  }
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', up)
  window.addEventListener('pointercancel', up)
}
function onDividerKey(i: number, e: KeyboardEvent) {
  const back = orient.value === 'rows' ? 'ArrowUp' : 'ArrowLeft'
  const fwd = orient.value === 'rows' ? 'ArrowDown' : 'ArrowRight'
  if (e.key !== back && e.key !== fwd) return
  e.preventDefault()
  e.stopPropagation()
  const step = e.shiftKey ? 40 : 8
  commitLanes(resizeLane(props.content.lanes ?? [], i, e.key === fwd ? step : -step, span()))
}
const top = computed(() => shapeValign(props.content.shape) === 'top')
const marks = computed(() =>
  shapeMarks(props.content, Math.max(1, props.w - sw.value), Math.max(1, props.h - sw.value)),
)
const markWidth = computed(() => props.content.stroke_width)
const paint = (v: 'line' | 'surface' | 'none' | undefined, fallback: string) =>
  v === 'line' ? stroke.value : v === 'surface' ? 'var(--color-surface)' : v === 'none' ? 'none' : fallback
const textStyle = computed(() => ({
  ...(orient.value && !guards.value ? bandStyle.value : textInset.value),
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
      <path
        v-for="(m, i) in marks"
        :key="i"
        :d="m.d"
        :transform="pathTransform"
        :style="{ fill: paint(m.fill, 'none'), stroke: paint(m.stroke, stroke) }"
        :stroke-width="markWidth * (m.width ?? 1)"
        :stroke-dasharray="m.dash ? `${markWidth * 3} ${markWidth * 3}` : undefined"
        stroke-linejoin="round"
        stroke-linecap="round"
        data-mark
      />
    </svg>

    <!-- Kelas UML: tiga bagian bergaris pemisah. -->
    <div
      v-if="isClass"
      class="absolute inset-0 flex flex-col overflow-hidden text-ink"
      :style="{ fontSize: `${content.font_size}px` }"
    >
      <p v-if="stereo" class="px-2 pt-1 text-center text-sm leading-tight" data-stereotype>{{ stereo }}</p>
      <template v-if="editing">
        <WbRichField
          :field-key="key('name')"
          :text="content.name ?? ''"
          :doc="content.name_doc"
          label="Nama kelas"
          placeholder="Kelas"
          :class="[field, abstract ? 'italic' : '']"
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
          :label="isEnum ? 'Nilai enum, satu per baris' : 'Atribut kelas, satu per baris'"
          :single-line="false"
          :class="field"
          :style="divider"
          class="flex-1 px-2 py-1 font-mono"
          @input="input('attributes', $event)"
          @commit="save('attributes', $event)"
          @close="emit('close')"
        />
        <WbRichField
          v-if="!isEnum"
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
        <WbRichText
          v-if="content.name_doc"
          :doc="content.name_doc"
          class="px-2 py-1 text-center font-bold"
          :class="abstract ? 'italic' : ''"
        />
        <p
          v-else
          class="px-2 py-1 text-center font-bold break-words"
          :class="abstract ? 'italic' : ''"
          data-caret
          data-class-name
        >
          {{ content.name || 'Kelas' }}
        </p>
        <div class="min-h-6 flex-1 px-2 py-1 font-mono break-words whitespace-pre-wrap" :style="divider">
          <WbRichText v-if="content.attributes_doc" :doc="content.attributes_doc" />
          <template v-else>{{ content.attributes }}</template>
        </div>
        <div
          v-if="!isEnum"
          class="min-h-6 flex-1 px-2 py-1 font-mono break-words whitespace-pre-wrap"
          :style="divider"
        >
          <WbRichText v-if="content.methods_doc" :doc="content.methods_doc" />
          <template v-else>{{ content.methods }}</template>
        </div>
      </template>
    </div>

    <div
      v-else
      class="absolute flex text-ink"
      :class="[
        top ? 'items-start font-semibold' : 'items-center',
        orient && !guards ? 'justify-center px-1 font-semibold' : '',
        pool ? 'wb-vertical' : '',
        underline ? 'underline' : '',
      ]"
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

    <!-- Operator fragmen di tabnya (alt, opt, loop, par). -->
    <p
      v-if="guards"
      class="absolute top-0 left-0 flex items-center px-2 text-sm font-semibold text-ink"
      :style="{ width: `${FRAGMENT_TAB.w}px`, height: `${FRAGMENT_TAB.h}px` }"
      data-fragment-op
    >
      {{ content.variant || 'alt' }}
    </p>
    <!-- Label lane (revisi 54b) atau syarat operand fragmen: diubah lewat Gaya bentuk; di sini hanya tampil. -->
    <p
      v-for="(b, i) in lanes"
      :key="`lane-${i}`"
      class="absolute overflow-hidden text-sm leading-tight break-words text-ink"
      :class="
        guards
          ? 'px-1'
          : ['flex items-center justify-center px-1 text-center', orient === 'rows' ? 'wb-vertical' : '']
      "
      :style="laneLabelStyle(b, i)"
      data-lane
    >
      {{ guards ? (b.label ? `[${b.label}]` : '') : b.label }}
    </p>
    <template v-if="canDivide">
      <div
        v-for="i in lanes.length - 1"
        :key="`batas-${i}`"
        role="separator"
        tabindex="0"
        :aria-orientation="orient === 'rows' ? 'horizontal' : 'vertical'"
        :aria-label="`Batas ${lanes[i - 1]!.label || `lane ${i}`} dan ${lanes[i]!.label || `lane ${i + 1}`}, panah untuk mengubah ukuran`"
        class="wb-lane-divider absolute"
        :class="orient === 'rows' ? 'cursor-row-resize' : 'cursor-col-resize'"
        :style="dividerStyle(i - 1)"
        data-lane-divider
        @pointerdown.stop.prevent="onDividerDown(i - 1, $event)"
        @keydown="onDividerKey(i - 1, $event)"
      />
    </template>
  </div>
</template>
