<script setup lang="ts">
import { computed, nextTick, ref, useId, watch } from 'vue'
import type { PMNode } from '../../utils/whiteboard'
import WbRichField, { activateField } from './WbRichField.vue'
import WbRichText from './WbRichText'
import WbErdTable from './WbErdTable.vue'
import WbItemsEditor from './WbItemsEditor.vue'
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
  defaultText,
  eventX,
  isMindmap,
  stageBoxes,
  staged,
  timed,
  TIMELINE_LINE,
  type TimelineEvent,
  frameBox,
  frameImage,
  isFrame,
  type FrameImage,
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
  // Revisi 56: nama orang lain yang sedang mengedit kartu ini (kunci lunak), dan permintaan edit nama lane dari
  // menu klik kanan halaman (nonce membedakan dua permintaan untuk lane yang sama).
  lockedBy?: string | null
  laneEdit?: { index: number | 'title'; nonce: number } | null
  // Revisi 57b: alamat gambar bingkai, dan mode geser posisi gambar.
  image?: string | null
  panning?: boolean
}>()
const emit = defineEmits<{
  save: [c: ShapeContent]
  live: [c: ShapeContent]
  close: []
  addLane: []
  notice: [msg: string]
  rowPort: [row: string, side: 'left' | 'right', e: PointerEvent]
  panDone: []
}>()

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
  shapePath(
    props.content.shape,
    Math.max(1, props.w - sw.value),
    Math.max(1, props.h - sw.value),
    props.content,
  ),
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
const isTable = computed(() => props.content.shape === 'erd_table')
// Revisi 57c: garis waktu, corong, dan chevron diedit lewat editor daftar; mind map satu baris (Enter selesai, lalu
// Enter lagi menambah cabang sejajar).
const listEdit = computed(() => ['timeline', 'funnel', 'chevron'].includes(props.content.shape))
const isMind = computed(() => isMindmap(props.content.shape))
const stages = computed(() =>
  staged(props.content)
    ? stageBoxes(props.content.shape, props.w, props.h, props.content.stages!.length)
    : [],
)
const lineY = computed(() => props.h * TIMELINE_LINE)
const evDraft = ref<{ id: string; at: number } | null>(null)
const evAt = (e: TimelineEvent) => (evDraft.value?.id === e.id ? evDraft.value.at : e.at)
const evStyle = (e: TimelineEvent, part: 'label' | 'date') => {
  const x = Math.min(Math.max(eventX(props.w, evAt(e)), 0), props.w)
  const w = 140
  const left = Math.min(Math.max(x - w / 2, -20), props.w - w + 20)
  return part === 'label'
    ? { left: `${left}px`, width: `${w}px`, top: '24px', height: `${Math.max(16, lineY.value - 36)}px` }
    : {
        left: `${left}px`,
        width: `${w}px`,
        top: `${lineY.value + 12}px`,
        height: `${Math.max(16, props.h - lineY.value - 12)}px`,
      }
}
const dotsOn = computed(() => timed(props.content) && props.selected && props.canWrite && !props.editing)
function dotStyle(e: TimelineEvent) {
  const s = 44 / (props.zoom || 1)
  return {
    left: `${eventX(props.w, evAt(e)) - s / 2}px`,
    top: `${lineY.value - s / 2}px`,
    width: `${s}px`,
    height: `${s}px`,
  }
}
const pct1 = (v: number) => Math.round(Math.min(100, Math.max(0, v)) * 10) / 10
function saveEvent(id: string, at: number) {
  evDraft.value = null
  const list = (props.content.events ?? [])
    .map((e) => (e.id === id ? { ...e, at } : e))
    .sort((a, b) => a.at - b.at)
  if (JSON.stringify(list) !== JSON.stringify(props.content.events))
    emit('save', { ...props.content, events: list })
}
function onDotDown(e: TimelineEvent, ev: PointerEvent) {
  if (ev.button !== 0) return
  const x0 = ev.clientX
  const start = e.at
  const z = props.zoom || 1
  const move = (m: PointerEvent) => {
    evDraft.value = { id: e.id, at: pct1(start + ((m.clientX - x0) / z / props.w) * 100) }
  }
  const up = () => {
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', up)
    window.removeEventListener('pointercancel', up)
    if (evDraft.value) saveEvent(e.id, evDraft.value.at)
  }
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', up)
  window.addEventListener('pointercancel', up)
}
function onDotKey(e: TimelineEvent, ev: KeyboardEvent) {
  const d =
    ev.key === 'ArrowLeft'
      ? -1
      : ev.key === 'ArrowRight'
        ? 1
        : ev.key === 'Home'
          ? -100
          : ev.key === 'End'
            ? 100
            : 0
  if (!d) return
  ev.preventDefault()
  ev.stopPropagation()
  saveEvent(e.id, pct1(e.at + d * (ev.shiftKey && Math.abs(d) === 1 ? 5 : 1)))
}
// Gambar bingkai (revisi 57b): di layar bingkai dengan object-fit dan object-position dari isi bentuk. Mode geser:
// seret gambar atau panah (Shift 1 persen) mengubah posisi, satu langkah undo per seret atau per tombol; Enter atau
// Escape selesai.
const framed = computed(() => !!props.image && isFrame(props.content.shape))
const imageDraft = ref<FrameImage | null>(null)
const im = computed(() => imageDraft.value ?? frameImage(props.content))
const frameStyle = computed(() => {
  const b = frameBox(props.content.shape, props.w, props.h, sw.value)
  return { left: `${b.x}px`, top: `${b.y}px`, width: `${b.w}px`, height: `${b.h}px` }
})
const hideText = computed(() => framed.value && props.content.text === defaultText(props.content.shape))
const panEl = ref<HTMLElement | null>(null)
watch(
  () => props.panning,
  (on) => {
    if (on) void nextTick(() => panEl.value?.focus())
  },
)
const clampPct = (v: number) => Math.round(Math.min(100, Math.max(0, v)) * 10) / 10
function saveImage(next: FrameImage) {
  imageDraft.value = null
  const cur = frameImage(props.content)
  if (next.x !== cur.x || next.y !== cur.y || next.fit !== cur.fit)
    emit('save', { ...props.content, image: next })
}
function onPanDown(e: PointerEvent) {
  if (e.button !== 0) return
  const b = frameBox(props.content.shape, props.w, props.h, sw.value)
  const z = props.zoom || 1
  const start = { ...im.value }
  const x0 = e.clientX
  const y0 = e.clientY
  const move = (ev: PointerEvent) => {
    // Seret ke kanan memperlihatkan bagian kiri gambar, jadi posisi berkurang.
    imageDraft.value = {
      ...start,
      x: clampPct(start.x - ((ev.clientX - x0) / z / b.w) * 100),
      y: clampPct(start.y - ((ev.clientY - y0) / z / b.h) * 100),
    }
  }
  const up = () => {
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', up)
    window.removeEventListener('pointercancel', up)
    if (imageDraft.value) saveImage(imageDraft.value)
  }
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', up)
  window.addEventListener('pointercancel', up)
}
function onPanKey(e: KeyboardEvent) {
  e.stopPropagation()
  if (e.key === 'Escape' || e.key === 'Enter') {
    e.preventDefault()
    emit('panDone')
    return
  }
  const step = e.shiftKey ? 1 : 5
  const d = { ArrowLeft: [step, 0], ArrowRight: [-step, 0], ArrowUp: [0, step], ArrowDown: [0, -step] }[e.key]
  if (!d) return
  e.preventDefault()
  const cur = im.value
  saveImage({ ...cur, x: clampPct(cur.x + d[0]!), y: clampPct(cur.y + d[1]!) })
}
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

// Revisi 56: ganti nama lane, syarat operand, dan nama pool atau swimlane langsung di kanvas. Dobel klik (atau fokus
// lalu Enter) membuka isian di tempat; Enter atau klik di luar menyimpan (satu langkah undo lewat simpan isi bentuk),
// Escape membatalkan. Ditolak bila orang lain sedang mengedit kartu ini.
const nameEdit = ref<number | 'title' | null>(null)
const nameDraft = ref('')
const nameInput = ref<HTMLInputElement | null>(null)
const laneWord = computed(() => (guards.value ? 'operand' : 'lane'))
function startName(which: number | 'title') {
  if (!props.canWrite || !orient.value) return
  if (props.lockedBy) {
    emit('notice', `${props.lockedBy} sedang mengedit kartu ini. Coba lagi sesudah selesai.`)
    return
  }
  nameEdit.value = which
  nameDraft.value = which === 'title' ? props.content.text : (props.content.lanes?.[which]?.label ?? '')
  void nextTick(() => {
    nameInput.value?.focus()
    nameInput.value?.select()
  })
}
function commitName() {
  const which = nameEdit.value
  if (which === null) return
  nameEdit.value = null
  const v = nameDraft.value.replace(/\s+/g, ' ').trim()
  if (which === 'title') {
    if (v !== props.content.text) emit('save', { ...props.content, text: v, doc: null })
    return
  }
  const lanes = props.content.lanes ?? []
  if (!lanes[which] || lanes[which]!.label === v) return
  emit('save', { ...props.content, lanes: lanes.map((l, j) => (j === which ? { ...l, label: v } : l)) })
}
function cancelName() {
  nameEdit.value = null
}
function onNameKey(e: KeyboardEvent) {
  e.stopPropagation()
  if (e.key === 'Enter') {
    e.preventDefault()
    commitName()
  } else if (e.key === 'Escape') {
    e.preventDefault()
    cancelName()
  }
}
function onLabelKey(which: number | 'title', e: KeyboardEvent) {
  if (e.key !== 'Enter' && e.key !== 'F2') return
  e.preventDefault()
  e.stopPropagation()
  startName(which)
}
function nameInputStyle() {
  const which = nameEdit.value
  if (which === 'title')
    return orient.value === 'rows'
      ? {
          left: '0px',
          top: `${props.h / 2 - 18}px`,
          width: `${Math.min(220, Math.max(140, props.w - BAND))}px`,
        }
      : { left: '4px', top: `${BAND / 2 - 18}px`, width: `${Math.max(120, props.w - 8)}px` }
  const b = which === null ? null : lanes.value[which]
  if (!b) return {}
  if (guards.value)
    return {
      left: '8px',
      top: `${b.y + (which ? 2 : FRAGMENT_TAB.h + 2)}px`,
      width: `${Math.max(140, b.w - 16)}px`,
    }
  return orient.value === 'rows'
    ? {
        left: `${band.value}px`,
        top: `${b.y + b.h / 2 - 18}px`,
        width: `${Math.min(220, Math.max(140, b.w))}px`,
      }
    : { left: `${b.x + 4}px`, top: `${band.value}px`, width: `${Math.max(120, b.w - 8)}px` }
}
// Lane baru dari tombol tambah langsung masuk mode edit nama.
let pendingLane: number | null = null
function onAddLane() {
  pendingLane = props.content.lanes?.length ?? 0
  emit('addLane')
}
watch(
  () => props.content.lanes?.length ?? 0,
  (n) => {
    if (pendingLane !== null && n > pendingLane) {
      const i = pendingLane
      pendingLane = null
      startName(i)
    }
  },
)
watch(
  () => props.laneEdit,
  (r) => {
    if (r) startName(r.index)
  },
)
// Tombol tambah 44 px di layar pada zoom berapa pun, di tengah tepi bawah (pool, fragmen) atau kanan (swimlane).
function addStyle() {
  const s = 44 / (props.zoom || 1)
  const gap = 6 / (props.zoom || 1)
  return orient.value === 'cols'
    ? { left: `${props.w + gap}px`, top: `${props.h / 2 - s / 2}px`, width: `${s}px`, height: `${s}px` }
    : { left: `${props.w / 2 - s / 2}px`, top: `${props.h + gap}px`, width: `${s}px`, height: `${s}px` }
}
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
    if (!on || isTable.value || listEdit.value) return
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
        v-for="(m, i) in framed && content.shape === 'wf_image' ? [] : marks"
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

    <!-- Gambar bingkai (revisi 57b). -->
    <div v-if="framed" class="absolute overflow-hidden" :style="frameStyle" data-frame-image>
      <img
        :src="image!"
        alt="Gambar di bingkai"
        draggable="false"
        class="size-full max-w-none"
        :style="{ objectFit: im.fit, objectPosition: `${im.x}% ${im.y}%` }"
      />
      <div
        v-if="panning"
        ref="panEl"
        tabindex="0"
        class="absolute inset-0 cursor-move shadow-[inset_0_0_0_2px_var(--color-mark),inset_0_0_0_4px_var(--color-surface)] outline-none"
        :aria-label="`Posisi gambar ${Math.round(im.x)} persen dari kiri, ${Math.round(im.y)} persen dari atas. Seret atau tekan panah untuk menggeser, Enter atau Escape selesai.`"
        data-frame-pan
        @pointerdown.stop.prevent="onPanDown"
        @dblclick.stop
        @keydown="onPanKey"
        @blur="emit('panDone')"
      />
    </div>

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

    <!-- Tabel ERD (revisi 57a): kepala nama tabel dan baris kolom, editor di tempat saat mode edit. -->
    <WbErdTable
      v-else-if="isTable"
      :content="content"
      :w="w"
      :editing="editing"
      :zoom="zoom"
      @save="emit('save', $event)"
      @live="emit('live', $event)"
      @close="emit('close')"
      @notice="emit('notice', $event)"
      @row-port="(r, s, e) => emit('rowPort', r, s, e)"
    />

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
      :role="orient && !guards && canDivide ? 'button' : undefined"
      :tabindex="orient && !guards && canDivide ? 0 : undefined"
      :aria-label="orient && !guards && canDivide ? `Ganti nama ${content.text || 'bentuk'}` : undefined"
      :data-lane-title="orient && !guards ? '' : undefined"
      @keydown="orient && !guards && canDivide ? onLabelKey('title', $event) : undefined"
    >
      <WbRichField
        v-if="editing && !listEdit"
        :field-key="key('text')"
        :text="content.text"
        :doc="content.doc"
        label="Teks bentuk"
        :single-line="isMind"
        class="max-h-full w-full overflow-auto leading-snug"
        @input="input('text', $event)"
        @commit="save('text', $event)"
        @enter="isMind && emit('close')"
        @close="emit('close')"
      />
      <WbRichText
        v-else-if="content.doc"
        :doc="content.doc"
        class="max-h-full w-full overflow-hidden leading-snug [overflow-wrap:anywhere]"
      />
      <p
        v-else-if="content.text && !hideText && !stages.length"
        data-caret
        class="max-h-full w-full overflow-hidden leading-snug break-words whitespace-pre-wrap [overflow-wrap:anywhere]"
      >
        {{ content.text }}
      </p>
    </div>

    <!-- Revisi 57c: label tahap corong dan chevron, label dan tanggal peristiwa garis waktu. -->
    <p
      v-for="(b, i) in stages"
      :key="`tahap-${i}`"
      class="absolute flex items-center justify-center overflow-hidden px-1 text-center text-sm leading-tight break-words text-ink"
      :style="{ left: `${b.x}px`, top: `${b.y}px`, width: `${b.w}px`, height: `${b.h}px` }"
      data-stage
    >
      {{ content.stages![i]!.label }}
    </p>
    <template v-if="timed(content)">
      <template v-for="e in content.events" :key="e.id">
        <p
          class="absolute flex items-end justify-center overflow-hidden text-center text-sm leading-tight font-semibold break-words text-ink"
          :style="evStyle(e, 'label')"
          :data-event="e.id"
        >
          {{ e.label }}
        </p>
        <p
          class="absolute flex items-start justify-center overflow-hidden text-center text-xs leading-tight text-ink-muted"
          :style="evStyle(e, 'date')"
          :data-event-date="e.id"
        >
          {{ e.date }}
        </p>
      </template>
    </template>
    <template v-if="dotsOn">
      <button
        v-for="e in content.events"
        :key="`titik-${e.id}`"
        type="button"
        role="slider"
        aria-valuemin="0"
        aria-valuemax="100"
        :aria-valuenow="evAt(e)"
        :aria-label="`Posisi ${e.label || 'peristiwa'}, panah kiri atau kanan untuk menggeser`"
        class="absolute flex cursor-ew-resize items-center justify-center focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
        :style="dotStyle(e)"
        :data-event-dot="e.id"
        @pointerdown.stop.prevent="onDotDown(e, $event)"
        @dblclick.stop
        @keydown="onDotKey(e, $event)"
      >
        <span class="size-1/3 rounded-full border-2 border-mark bg-surface" />
      </button>
    </template>
    <WbItemsEditor
      v-if="editing && listEdit"
      :content="content"
      :w="w"
      :zoom="zoom"
      @save="emit('save', $event)"
      @live="emit('live', $event)"
      @close="emit('close')"
      @notice="emit('notice', $event)"
    />

    <!-- Operator fragmen di tabnya (alt, opt, loop, par). -->
    <p
      v-if="guards"
      class="absolute top-0 left-0 flex items-center px-2 text-sm font-semibold text-ink"
      :style="{ width: `${FRAGMENT_TAB.w}px`, height: `${FRAGMENT_TAB.h}px` }"
      data-fragment-op
    >
      {{ content.variant || 'alt' }}
    </p>
    <!-- Label lane (revisi 54b) atau syarat operand fragmen: dobel klik atau Enter untuk mengganti nama (revisi 56);
         juga lewat Gaya bentuk. -->
    <p
      v-for="(b, i) in lanes"
      :key="`lane-${i}`"
      :role="canDivide ? 'button' : undefined"
      :tabindex="canDivide ? 0 : undefined"
      :aria-label="canDivide ? `Ganti nama ${laneWord} ${i + 1}${b.label ? `, ${b.label}` : ''}` : undefined"
      :data-lane-index="i"
      class="absolute overflow-hidden text-sm leading-tight break-words text-ink"
      :class="
        guards
          ? 'px-1'
          : ['flex items-center justify-center px-1 text-center', orient === 'rows' ? 'wb-vertical' : '']
      "
      :style="laneLabelStyle(b, i)"
      data-lane
      @keydown="canDivide && onLabelKey(i, $event)"
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
      <button
        type="button"
        class="wb-lane-add absolute flex items-center justify-center rounded-full border border-line bg-surface text-ink shadow-menu hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
        :style="addStyle()"
        :aria-label="`Tambah ${laneWord}`"
        :title="`Tambah ${laneWord}`"
        data-lane-add
        @pointerdown.stop
        @keydown.enter.stop
        @keydown.space.stop
        @dblclick.stop
        @click.stop="onAddLane"
      >
        <svg
          aria-hidden="true"
          viewBox="0 0 16 16"
          class="size-1/2"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
        >
          <path d="M8 3v10M3 8h10" />
        </svg>
      </button>
    </template>
    <input
      v-if="nameEdit !== null"
      ref="nameInput"
      v-model="nameDraft"
      type="text"
      maxlength="200"
      class="absolute z-10 h-9 rounded-card border border-mark bg-surface px-2 text-sm text-ink"
      :style="nameInputStyle()"
      :aria-label="nameEdit === 'title' ? 'Nama' : `Nama ${laneWord} ${(nameEdit as number) + 1}`"
      data-lane-input
      @pointerdown.stop
      @dblclick.stop
      @keydown="onNameKey"
      @blur="commitName"
    />
  </div>
</template>
