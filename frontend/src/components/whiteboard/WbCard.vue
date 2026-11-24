<script setup lang="ts">
import { computed, defineAsyncComponent, nextTick, ref, watch } from 'vue'
import WbRichText from './WbRichText'
import WbChecklist from './WbChecklist.vue'
import type { ChecklistOp } from '../../api/whiteboard'
import {
  cardLabel,
  refSummary,
  type Card,
  type ChecklistContent,
  type HeadingContent,
  type NoteContent,
  type PMNode,
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
}>()
const emit = defineEmits<{
  grab: [e: PointerEvent]
  resize: [e: PointerEvent]
  edit: []
  close: []
  note: [doc: PMNode]
  heading: [text: string]
  rename: [title: string]
  draft: [text: string]
  checklist: [op: ChecklistOp]
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

// Posisi lewat CSSOM (:style), yang tidak diatur CSP style-src. Garis pilihan dibagi zoom supaya tetap 2 px di layar.
const style = computed(() => ({
  transform: `translate(${props.card.x + props.dx}px, ${props.card.y + props.dy}px)`,
  width: `${props.card.w + props.dw}px`,
  minHeight: `${props.card.h + props.dh}px`,
  zIndex: props.editing ? 100000 : Math.round(props.card.z),
  '--wb-ring': `${2 / props.zoom}px`,
  '--wb-remote': props.remote ? `var(--color-collab-${props.remote.color})` : undefined,
}))

const label = computed(
  () =>
    cardLabel(props.card) +
    (props.selected ? ', terpilih' : '') +
    (props.remote ? `, ${props.remote.label}` : ''),
)

const headingDraft = ref('')
const headingInput = ref<HTMLInputElement | null>(null)
// Teks terakhir yang sudah dikirim, supaya Enter lalu blur tidak menyimpan dua kali.
let lastSent = ''
watch(
  () => props.editing,
  async (on, was) => {
    if (props.card.kind !== 'heading') return
    // Editor bisa ditutup tanpa blur (klik di kanvas melepas input dari DOM), jadi teksnya disimpan di sini juga.
    if (was && !on) {
      saveHeading()
      return
    }
    if (!on) return
    headingDraft.value = (props.card.content as HeadingContent).text
    lastSent = headingDraft.value
    await nextTick()
    headingInput.value?.focus()
    headingInput.value?.select()
  },
  { immediate: true },
)

function saveHeading() {
  const text = headingDraft.value.trim()
  if (text === lastSent) return
  lastSent = text
  emit('heading', text)
}

function onHeadingKey(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    saveHeading()
    emit('close')
  } else if (e.key === 'Escape') {
    e.preventDefault()
    e.stopPropagation()
    emit('close')
  }
}

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
    aria-roledescription="kartu"
    class="wb-card absolute top-0 left-0 flex flex-col rounded-card border border-line p-3 text-ink shadow-sm select-none"
    :class="[
      card.color ? colorClass[card.color] : 'bg-surface',
      selected ? 'wb-card-selected' : '',
      !selected && remote ? (remote.strong ? 'wb-card-remote-strong' : 'wb-card-remote') : '',
      editing ? 'cursor-auto select-text' : canWrite ? 'cursor-grab' : 'cursor-default',
    ]"
    :style="style"
    @pointerdown="onPointerDown"
    @dblclick="canWrite && !editing && emit('edit')"
  >
    <template v-if="card.kind === 'note'">
      <WbNoteEditor
        v-if="editing"
        :doc="(card.content as NoteContent).doc"
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

    <template v-else-if="card.kind === 'heading'">
      <template v-if="editing">
        <label :for="`h-${card.id}`" class="sr-only">Teks judul</label>
        <input
          :id="`h-${card.id}`"
          ref="headingInput"
          v-model="headingDraft"
          maxlength="200"
          class="min-h-11 w-full rounded-card border border-line bg-surface px-2 text-lg font-bold text-ink"
          @pointerdown.stop
          @keydown="onHeadingKey"
          @input="emit('draft', headingDraft)"
          @blur="saveHeading"
        />
      </template>
      <p v-else class="text-lg leading-snug font-bold break-words">
        {{ (card.content as HeadingContent).text || 'Judul kosong' }}
      </p>
    </template>

    <WbChecklist
      v-else
      :content="card.content as ChecklistContent"
      :editing="editing"
      :can-write="canWrite"
      @op="emit('checklist', $event)"
      @close="emit('close')"
    />

    <!-- Label orang lain di atas kartu; warnanya warna orang itu, teksnya on-collab (kontras dihitung di style.css). -->
    <span
      v-if="remote"
      aria-hidden="true"
      class="wb-remote-label pointer-events-none absolute -top-7 left-0 rounded-chip px-1.5 py-0.5 text-xs font-medium whitespace-nowrap"
    >
      {{ remote.label }}
    </span>
    <!-- Pegangan ubah ukuran: gambarnya kecil, area sentuhnya 44x44 px. -->
    <div
      v-if="selected && canWrite && !editing"
      aria-hidden="true"
      class="absolute -right-5 -bottom-5 flex size-11 cursor-nwse-resize items-center justify-center touch-none"
      @pointerdown.stop="emit('resize', $event)"
    >
      <span class="size-3 rounded-chip border-2 border-mark bg-surface" />
    </div>
  </article>
</template>
