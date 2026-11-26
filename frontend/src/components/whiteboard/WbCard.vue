<script setup lang="ts">
import { computed, defineAsyncComponent, nextTick, ref, watch } from 'vue'
import WbRichText from './WbRichText'
import WbChecklist from './WbChecklist.vue'
import WbTable from './WbTable.vue'
import WbShape from './WbShape.vue'
import WbRichField, { activateField } from './WbRichField.vue'
import { fileUrl, type ChecklistOp, type TableOp } from '../../api/whiteboard'
import { formatBytes } from '../../utils/attachment'
import type { CaretHint } from './caret'
import { HANDLES, handleSign, type Handle, type ShapeContent } from '../../utils/wbShapes'
import { styleClasses, styleOpacity } from '../../utils/wbStyle'
import { cropBox } from '../../utils/wbCrop'
import {
  LOD_ZOOM,
  cardLabel,
  cardTitle,
  refSummary,
  type CaptionContent,
  type Card,
  type ChecklistContent,
  type ColumnContent,
  type HeadingContent,
  type LinkContent,
  type NoteContent,
  type PMNode,
  type TableContent,
} from '../../utils/whiteboard'

// TipTap hanya dimuat saat ada kartu catatan yang dibuka untuk diedit.
const WbNoteEditor = defineAsyncComponent(() => import('./WbNoteEditor.vue'))

const props = defineProps<{
  card: Card
  selected: boolean
  editing: boolean
  canWrite: boolean
  focusable: boolean
  zoom: number
  // Geser sementara saat kartu ini ikut diseret; ditambahkan ke posisi tersimpan.
  dx: number
  dy: number
  dw: number
  dh: number
  // Orang lain yang sedang memilih, menyeret, atau mengedit kartu ini (W2): warna kolaborasi 1 sampai 8 dan label.
  remote?: { color: number; label: string; strong: boolean } | null
  // Posisi kursor dari dobel klik (atau dobel ketuk), untuk editor yang baru dibuka.
  caret?: CaretHint | null
  // Kartu dikunci orang lain dan dibuka hanya baca: nama pemegangnya.
  readonlyBy?: string | null
  // Sudut sementara selama pegangan putar diseret (W5); null memakai sudut tersimpan.
  rotation?: number | null
}>()
const emit = defineEmits<{
  grab: [e: PointerEvent]
  resize: [e: PointerEvent, handle?: Handle]
  rotate: [e: PointerEvent]
  live: [content: Card['content']]
  close: []
  note: [doc: PMNode]
  heading: [v: { text: string; doc: PMNode | null }]
  rename: [title: string]
  draft: [v: { text: string; doc: PMNode | null }]
  checklist: [op: ChecklistOp]
  table: [op: TableOp]
  content: [content: Card['content']]
  comments: []
}>()

const root = ref<HTMLElement | null>(null)
defineExpose({ el: root })

// Warna memakai token latar yang sudah ada (docs/whiteboard.md bagian 8), dengan teks tinta di semua warna.
const colorClass: Record<string, string> = {
  abu: 'bg-requested-bg',
  hijau: 'bg-progress-bg',
  kuning: 'bg-ready-bg',
  biru: 'bg-completed-bg',
  krem: 'bg-tag text-tag-ink wb-card-krem',
  merah: 'bg-danger-bg',
}

// Level of detail (W4, rancangan bagian 7): di bawah zoom 40% isi tidak terbaca, jadi kartu hanya menampilkan
// judul ringkas dan warnanya. Ukuran huruf dibagi zoom supaya di layar tetap sekitar 14 px, dan baris dipotong di
// tinggi kartu. Gambar tetap menampilkan pratinjaunya; kartu yang sedang diedit selalu utuh.
const lod = computed(
  () => props.zoom < LOD_ZOOM && !props.editing && props.card.kind !== 'image' && props.card.kind !== 'shape',
)
const lodStyle = computed(() => ({
  fontSize: `${14 / props.zoom}px`,
  maxHeight: `${Math.max(20, props.card.h + props.dh - 24)}px`,
}))

// Posisi lewat CSSOM (:style), yang tidak diatur CSP style-src. Garis pilihan dibagi zoom supaya tetap 2 px di layar.
// Bentuk (W5) diputar utuh terhadap pusatnya, termasuk garis pilihan dan pegangannya.
const isShape = computed(() => props.card.kind === 'shape')
const shape = computed(() => props.card.content as ShapeContent)
const deg = computed(() => (isShape.value ? (props.rotation ?? shape.value.rotation ?? 0) : 0))
const style = computed(() => ({
  transform: `translate(${props.card.x + props.dx}px, ${props.card.y + props.dy}px)${deg.value ? ` rotate(${deg.value}deg)` : ''}`,
  height: isShape.value ? `${props.card.h + props.dh}px` : undefined,
  width: `${props.card.w + props.dw}px`,
  minHeight: `${props.card.h + props.dh}px`,
  zIndex: props.editing ? 100000 : Math.round(props.card.z),
  '--wb-ring': `${2 / props.zoom}px`,
  opacity: styleOpacity(props.card.style, props.card.kind),
}))

// Gaya kartu (revisi 50). Kartu tanpa latar tidak memakai warna kartu, juga tidak memaksa token terang kartu krem:
// teksnya di atas kanvas, jadi memakai token tema yang lolos AA terhadap kanvas terang dan gelap.
const clear = computed(
  () => props.card.style?.bg === 'transparan' && !isShape.value && props.card.kind !== 'image',
)
const cardClasses = computed(() => [
  ...styleClasses(props.card.style, props.card.kind),
  isShape.value ? 'wb-card-shape' : '',
])
// Pegangan ubah ukuran dan putar tidak tampil untuk kartu terkunci (server juga menolaknya).
const canResize = computed(() => props.selected && props.canWrite && !props.editing && !props.card.locked)
const lockStyle = computed(() => ({ width: `${22 / props.zoom}px`, height: `${22 / props.zoom}px` }))
const crop = computed(() =>
  props.card.kind === 'image' ? ((props.card.content as CaptionContent).crop ?? null) : null,
)

// Pegangan bentuk: gambar 10 px dan area sentuh 44 px di layar pada zoom berapa pun.
const handleBox = computed(() => 44 / props.zoom)
function handleStyle(h: Handle) {
  const [sx, sy] = handleSign(h)
  const s = handleBox.value
  return {
    width: `${s}px`,
    height: `${s}px`,
    left: `calc(${((sx + 1) / 2) * 100}% - ${s / 2}px)`,
    top: `calc(${((sy + 1) / 2) * 100}% - ${s / 2}px)`,
    cursor:
      h === 'n' || h === 's'
        ? 'ns-resize'
        : h === 'e' || h === 'w'
          ? 'ew-resize'
          : h === 'ne' || h === 'sw'
            ? 'nesw-resize'
            : 'nwse-resize',
  }
}
const dot = computed(() => ({
  width: `${10 / props.zoom}px`,
  height: `${10 / props.zoom}px`,
  borderWidth: `${2 / props.zoom}px`,
}))
const rotateStyle = computed(() => {
  const s = handleBox.value
  return {
    width: `${s}px`,
    height: `${s}px`,
    left: `calc(50% - ${s / 2}px)`,
    // Di atas pegangan port atas (48 px di luar tepi), supaya area sentuh keduanya tidak bertumpuk.
    top: `${-100 / props.zoom - s / 2}px`,
  }
})

const label = computed(
  () =>
    cardLabel(props.card) +
    (isShape.value && deg.value ? `, diputar ${deg.value} derajat` : '') +
    (props.card.locked ? ', posisi terkunci' : '') +
    (props.card.group_ids?.length ? ', dalam grup' : '') +
    (props.selected ? ', terpilih' : '') +
    (props.remote ? `, ${props.remote.label}` : ''),
)

// Kartu judul dan judul kolom: satu field berformat (revisi 49), dibuka langsung saat kartu diedit. Isinya dikirim
// saat Enter atau editor ditutup (field dilepas dari DOM).
const fieldKey = computed(() => `${props.card.id}:teks`)
watch(
  () => props.editing,
  (on) => {
    if (on && (props.card.kind === 'heading' || props.card.kind === 'column')) activateField(fieldKey.value)
  },
  { immediate: true },
)
const caretIndex = computed(() => props.caret?.textIndex ?? props.caret?.offset ?? null)

// Kartu papan: judulnya judul papan anak. Saat kartu baru dibuat, kartunya langsung dibuka untuk diberi judul.
const titleDraft = ref('')
const titleInput = ref<HTMLInputElement | null>(null)
let titleSent = ''
watch(
  () => props.editing,
  async (on, was) => {
    if (props.card.kind !== 'board') return
    if (was && !on) {
      saveTitle()
      return
    }
    if (!on) return
    titleDraft.value = props.card.ref?.title ?? ''
    titleSent = titleDraft.value
    await nextTick()
    titleInput.value?.focus()
    titleInput.value?.select()
  },
  { immediate: true },
)

function saveTitle() {
  const t = titleDraft.value.trim()
  if (!t || t === titleSent) return
  titleSent = t
  emit('rename', t)
}

function onTitleKey(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    saveTitle()
    emit('close')
  } else if (e.key === 'Escape') {
    e.preventDefault()
    e.stopPropagation()
    emit('close')
  }
}

// Keterangan gambar dan berkas, serta judul kolom: satu isian teks, disimpan saat Enter, blur, atau editor ditutup.
const textDraft = ref('')
const textInput = ref<HTMLInputElement | null>(null)
let textSent = ''
const textKind = computed(() => (props.card.kind === 'image' || props.card.kind === 'file' ? 'caption' : ''))
watch(
  () => props.editing,
  async (on, was) => {
    if (!textKind.value) return
    if (was && !on) {
      saveText()
      return
    }
    if (!on) return
    textDraft.value = (props.card.content as CaptionContent).caption ?? ''
    textSent = textDraft.value
    await nextTick()
    textInput.value?.focus()
  },
  { immediate: true },
)

function saveText() {
  const t = textDraft.value.trim()
  if (t === textSent) return
  textSent = t
  emit('content', { caption: t })
}

function onTextKey(e: KeyboardEvent) {
  if (e.key === 'Enter' || e.key === 'Escape') {
    e.preventDefault()
    e.stopPropagation()
    saveText()
    emit('close')
  }
}

const caption = computed(() => (props.card.content as CaptionContent).caption ?? '')
const imageStyle = computed(() => ({
  width: 'calc(100% + 1.5rem)',
  height: `${Math.max(40, props.card.h + props.dh - (props.editing || caption.value ? 72 : 24))}px`,
}))
const linkHref = computed(() => (props.card.kind === 'link' ? (props.card.content as LinkContent).url : ''))
const linkHost = computed(() => {
  try {
    return new URL(linkHref.value).host
  } catch {
    return linkHref.value
  }
})

function onPointerDown(e: PointerEvent) {
  if (props.editing) return
  emit('grab', e)
}
</script>

<template>
  <article
    ref="root"
    :data-card="card.id"
    :tabindex="focusable ? 0 : -1"
    :aria-label="label"
    :data-collab="remote?.color"
    aria-roledescription="kartu"
    class="wb-card absolute top-0 left-0 flex flex-col text-ink select-none"
    :class="[
      isShape ? 'origin-center' : 'rounded-card border border-line p-3 shadow-sm',
      isShape || clear ? '' : card.color ? colorClass[card.color] : 'bg-surface',
      cardClasses,
      selected ? 'wb-card-selected' : '',
      !selected && remote ? (remote.strong ? 'wb-card-remote-strong' : 'wb-card-remote') : '',
      editing ? 'cursor-auto select-text' : canWrite ? 'cursor-grab' : 'cursor-default',
    ]"
    :style="style"
    @pointerdown="onPointerDown"
  >
    <p
      v-if="lod"
      class="overflow-hidden leading-tight font-bold break-words [overflow-wrap:anywhere]"
      :style="lodStyle"
      data-testid="wb-lod"
    >
      {{ cardTitle(card) }}
    </p>
    <WbShape
      v-else-if="isShape"
      :content="shape"
      :w="card.w + dw"
      :h="card.h + dh"
      :editing="editing"
      @save="emit('content', $event)"
      @live="emit('live', $event)"
      @close="emit('close')"
    />
    <template v-else-if="card.kind === 'note'">
      <WbNoteEditor
        v-if="editing"
        :doc="(card.content as NoteContent).doc"
        :caret="caret"
        label="Isi catatan"
        @input="emit('note', $event)"
        @close="emit('close')"
      />
      <WbRichText v-else :doc="(card.content as NoteContent).doc" />
    </template>

    <template v-else-if="card.kind === 'board'">
      <div class="flex min-w-0 items-start gap-3">
        <span
          aria-hidden="true"
          class="inline-flex size-11 shrink-0 items-center justify-center rounded-card border border-line bg-canvas"
        >
          <svg
            viewBox="0 0 24 24"
            class="size-6"
            fill="none"
            stroke="currentColor"
            stroke-width="1.75"
            stroke-linejoin="round"
          >
            <path d="M4 4h16v16H4zM8 8h3v3H8zM13 8h3M13 11h3M8 14h8" />
          </svg>
        </span>
        <div class="min-w-0 flex-1">
          <template v-if="editing">
            <label :for="`t-${card.id}`" class="sr-only">Judul papan</label>
            <input
              :id="`t-${card.id}`"
              ref="titleInput"
              v-model="titleDraft"
              maxlength="120"
              class="min-h-11 w-full rounded-card border border-line bg-surface px-2 font-bold text-ink"
              @pointerdown.stop
              @keydown="onTitleKey"
              @blur="saveTitle"
            />
          </template>
          <p v-else class="font-bold break-words">{{ card.ref?.title }}</p>
          <p class="text-sm text-ink-muted">{{ refSummary(card.ref) }}</p>
        </div>
      </div>
    </template>

    <template v-else-if="card.kind === 'image' || card.kind === 'file'">
      <!-- Potongan dan posisi zoom (revisi 50): bingkai menampilkan bagian potongan saja, dari berkas asli supaya tajam. -->
      <div
        v-if="card.kind === 'image' && card.file?.thumb && crop"
        class="relative -m-3 mb-0 overflow-hidden rounded-t-card bg-canvas"
        :style="imageStyle"
        data-testid="wb-cropped"
      >
        <img
          :src="fileUrl(card.file.id)"
          :alt="caption || card.file.name"
          draggable="false"
          loading="lazy"
          class="absolute block max-w-none"
          :style="cropBox(crop)"
        />
      </div>
      <img
        v-else-if="card.kind === 'image' && card.file?.thumb"
        :src="fileUrl(card.file.id, true)"
        :alt="caption || card.file.name"
        draggable="false"
        loading="lazy"
        class="-m-3 mb-0 block max-w-none rounded-t-card bg-canvas object-contain"
        :style="imageStyle"
      />
      <div v-else class="flex min-w-0 items-start gap-3">
        <span
          aria-hidden="true"
          class="inline-flex size-11 shrink-0 items-center justify-center rounded-card border border-line bg-canvas"
        >
          <svg
            viewBox="0 0 24 24"
            class="size-6"
            fill="none"
            stroke="currentColor"
            stroke-width="1.75"
            stroke-linejoin="round"
          >
            <path d="M6 3h8l4 4v14H6zM14 3v4h4M9 13h6M9 17h6" />
          </svg>
        </span>
        <div class="min-w-0 flex-1">
          <p class="font-semibold break-all">{{ card.file?.name }}</p>
          <p class="text-sm text-ink-muted">{{ card.file ? formatBytes(card.file.size) : '' }}</p>
        </div>
      </div>
      <template v-if="editing">
        <label :for="`c-${card.id}`" class="sr-only">Keterangan</label>
        <input
          :id="`c-${card.id}`"
          ref="textInput"
          v-model="textDraft"
          maxlength="300"
          placeholder="Keterangan"
          class="mt-2 min-h-11 w-full rounded-card border border-line bg-surface px-2 text-ink"
          @pointerdown.stop
          @keydown="onTextKey"
          @blur="saveText"
        />
      </template>
      <p v-else-if="caption" class="mt-2 text-sm break-words">{{ caption }}</p>
      <a
        v-if="card.file && card.kind === 'file'"
        :href="fileUrl(card.file.id)"
        class="mt-2 inline-flex min-h-11 items-center gap-1 self-start rounded-card px-2 font-medium underline underline-offset-2 hover:bg-canvas focus-visible:outline-2 focus-visible:outline-mark"
        @pointerdown.stop
      >
        Unduh berkas<span class="sr-only"> {{ card.file.name }}</span>
      </a>
    </template>

    <template v-else-if="card.kind === 'link'">
      <img
        v-if="card.preview?.status === 'ok' && card.preview.image"
        :src="fileUrl(card.preview.image)"
        alt=""
        draggable="false"
        loading="lazy"
        class="-m-3 mb-2 block h-32 max-w-none rounded-t-card bg-canvas object-cover"
        :style="{ width: 'calc(100% + 1.5rem)' }"
      />
      <p class="font-semibold break-words">
        {{ card.preview?.status === 'ok' && card.preview.title ? card.preview.title : linkHost }}
      </p>
      <p
        v-if="card.preview?.status === 'ok' && card.preview.description"
        class="line-clamp-3 text-sm break-words"
      >
        {{ card.preview.description }}
      </p>
      <p v-else-if="card.preview && card.preview.status !== 'ok'" class="text-sm text-ink-muted">
        Pratinjau tidak tersedia.
      </p>
      <p class="text-sm break-all text-ink-muted">{{ card.preview?.site || linkHost }}</p>
      <a
        :href="linkHref"
        target="_blank"
        rel="noopener noreferrer"
        class="mt-1 inline-flex min-h-11 items-center gap-1 self-start rounded-card px-2 font-medium underline underline-offset-2 hover:bg-canvas focus-visible:outline-2 focus-visible:outline-mark"
        @pointerdown.stop
      >
        Buka tautan<span class="sr-only"> {{ linkHost }} di tab baru</span>
        <svg
          aria-hidden="true"
          viewBox="0 0 24 24"
          class="size-4"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
        >
          <path d="M14 4h6v6M20 4l-9 9M18 14v6H4V6h6" />
        </svg>
      </a>
    </template>

    <template v-else-if="card.kind === 'column'">
      <WbRichField
        v-if="editing"
        :field-key="fieldKey"
        :text="(card.content as ColumnContent).title"
        :doc="(card.content as ColumnContent).title_doc"
        label="Judul kolom"
        placeholder="Judul kolom"
        :caret-index="caretIndex"
        class="rounded-card border border-line bg-surface px-2 py-2 font-bold text-ink"
        @commit="emit('content', { title: $event.text, title_doc: $event.doc })"
        @enter="emit('close')"
        @close="emit('close')"
      />
      <WbRichText
        v-else-if="(card.content as ColumnContent).title_doc"
        :doc="(card.content as ColumnContent).title_doc!"
        class="min-h-8 font-bold"
      />
      <p v-else class="min-h-8 font-bold break-words">
        {{ (card.content as ColumnContent).title || 'Kolom tanpa judul' }}
      </p>
    </template>

    <WbTable
      v-else-if="card.kind === 'table'"
      :content="card.content as TableContent"
      :editing="editing"
      :caret="caret"
      @op="emit('table', $event)"
      @close="emit('close')"
    />

    <template v-else-if="card.kind === 'heading'">
      <WbRichField
        v-if="editing"
        :field-key="fieldKey"
        :text="(card.content as HeadingContent).text"
        :doc="(card.content as HeadingContent).doc"
        label="Teks judul"
        :caret-index="caretIndex"
        :select-all="caretIndex === null"
        class="rounded-card border border-line bg-surface px-2 py-1.5 text-lg leading-snug font-bold text-ink"
        @input="emit('draft', $event)"
        @commit="emit('heading', $event)"
        @enter="emit('close')"
        @close="emit('close')"
      />
      <WbRichText
        v-else-if="(card.content as HeadingContent).doc"
        :doc="(card.content as HeadingContent).doc!"
        class="text-lg leading-snug font-bold"
      />
      <p
        v-else-if="(card.content as HeadingContent).text"
        data-caret
        class="text-lg leading-snug font-bold break-words"
      >
        {{ (card.content as HeadingContent).text }}
      </p>
      <p v-else class="text-lg leading-snug font-bold break-words">Judul kosong</p>
    </template>

    <WbChecklist
      v-else
      :content="card.content as ChecklistContent"
      :editing="editing"
      :can-write="canWrite"
      :caret="caret"
      @op="emit('checklist', $event)"
      @close="emit('close')"
    />

    <!-- Lencana komentar menjadi tombol hanya di zoom 100% ke atas, supaya area sentuhnya tetap 44 px di layar. Di
         bawah itu hanya penanda: jumlahnya ada di label kartu, dan panelnya dibuka lewat Aksi lain di bar pilihan. -->
    <component
      :is="zoom >= 1 ? 'button' : 'span'"
      v-if="card.comments"
      :type="zoom >= 1 ? 'button' : undefined"
      class="absolute -top-4 -right-4 inline-flex size-11 items-center justify-center rounded-full focus-visible:outline-2 focus-visible:outline-mark"
      :class="zoom >= 1 ? '' : 'pointer-events-none'"
      :aria-label="zoom >= 1 ? `${card.comments} komentar, buka komentar` : undefined"
      :aria-hidden="zoom >= 1 ? undefined : 'true'"
      data-testid="wb-comment-badge"
      @pointerdown.stop
      @click="zoom >= 1 && emit('comments')"
    >
      <span
        class="inline-flex min-w-6 items-center justify-center gap-0.5 rounded-chip bg-accent px-1.5 py-0.5 text-xs font-semibold text-on-accent"
      >
        <svg
          aria-hidden="true"
          viewBox="0 0 24 24"
          class="size-3"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linejoin="round"
        >
          <path d="M4 5h16v11H9l-5 4z" />
        </svg>
        {{ card.comments }}
      </span>
    </component>
    <p
      v-if="readonlyBy && !editing"
      role="status"
      class="mt-2 rounded-chip bg-canvas px-2 py-1 text-sm text-ink"
      data-testid="wb-readonly"
    >
      Hanya baca: {{ readonlyBy }} sedang mengedit. Teksnya berubah langsung di sini.
    </p>
    <!-- Label orang lain di atas kartu; warnanya warna orang itu, teksnya on-collab per warna lewat data-collab (kontras dihitung di style.css). -->
    <span
      v-if="remote"
      aria-hidden="true"
      class="wb-remote-label pointer-events-none absolute -top-7 left-0 rounded-chip px-1.5 py-0.5 text-xs font-medium whitespace-nowrap"
    >
      {{ remote.label }}
    </span>
    <!-- Bentuk: delapan pegangan ubah ukuran (Shift menjaga proporsi) dan pegangan putar (Shift kelipatan 15 derajat).
         Jalur keyboard untuk keduanya ada di panel gaya bentuk. -->
    <template v-if="isShape && canResize">
      <div
        v-for="h in HANDLES"
        :key="h"
        aria-hidden="true"
        class="absolute flex items-center justify-center touch-none"
        :style="handleStyle(h)"
        :data-handle="h"
        @pointerdown.stop="emit('resize', $event, h)"
      >
        <span class="rounded-chip border-mark bg-surface" :style="dot" />
      </div>
      <div
        aria-hidden="true"
        class="absolute flex cursor-grab items-center justify-center touch-none"
        :style="rotateStyle"
        data-handle="rotate"
        @pointerdown.stop="emit('rotate', $event)"
      >
        <span class="rounded-full border-mark bg-surface" :style="dot" />
      </div>
    </template>
    <!-- Pegangan ubah ukuran: gambarnya kecil, area sentuhnya 44x44 px. -->
    <div
      v-else-if="canResize"
      aria-hidden="true"
      class="absolute -right-5 -bottom-5 flex size-11 cursor-nwse-resize items-center justify-center touch-none"
      @pointerdown.stop="emit('resize', $event)"
    >
      <span class="size-3 rounded-chip border-2 border-mark bg-surface" />
    </div>
    <!-- Gembok kartu terkunci (revisi 50), hanya saat dipilih; pembaca layar mendapatnya dari label kartu. -->
    <span
      v-if="card.locked && selected"
      aria-hidden="true"
      data-testid="wb-lock"
      class="absolute top-0 left-0 inline-flex -translate-1/2 items-center justify-center rounded-full border border-line bg-surface text-ink"
      :style="lockStyle"
    >
      <svg
        viewBox="0 0 24 24"
        class="size-3/5"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
      >
        <rect x="5" y="11" width="14" height="9" rx="2" />
        <path d="M8 11V8a4 4 0 0 1 8 0v3" />
      </svg>
    </span>
  </article>
</template>
