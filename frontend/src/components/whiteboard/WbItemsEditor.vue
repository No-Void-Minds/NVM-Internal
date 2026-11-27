<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import {
  MAX_EVENTS,
  MAX_STAGES,
  newEventAt,
  newRowId,
  type ShapeContent,
  type TimelineEvent,
} from '../../utils/wbShapes'

// Editor daftar (revisi 57c): tahap corong dan chevron, atau judul dan peristiwa garis waktu. Seperti editor tabel ERD:
// ukuran layar tetap lewat skala 1/zoom (tombol 44 px pada zoom berapa pun), setiap aksi disimpan sendiri sehingga
// halaman mencatatnya satu langkah undo, Enter di baris terakhir menambah baris, Alt dan panah memindah tahap, Escape
// menyimpan lalu keluar.
const props = defineProps<{ content: ShapeContent; w: number; zoom?: number }>()
const emit = defineEmits<{
  save: [c: ShapeContent]
  live: [c: ShapeContent]
  close: []
  notice: [msg: string]
}>()

const events = computed(() => props.content.shape === 'timeline')
const z = computed(() => props.zoom || 1)
type Draft = { text: string; stages: { label: string }[]; events: TimelineEvent[] }
// Corong dan chevron lama (sebelum revisi 57c) belum punya tahap: teksnya menjadi tahap pertama.
const clone = (c: ShapeContent): Draft => ({
  text: c.text,
  stages: (c.stages?.length ? c.stages : [{ label: c.text }]).map((s) => ({ ...s })),
  events: (c.events ?? []).map((e) => ({ ...e })),
})
const draft = ref(clone(props.content))
let dirty = false
// Fokus sesudah aksi struktur menunggu isi kartu diperbarui (baris baru belum ada sebelum itu).
let pending: number | null = null
watch(
  () => props.content,
  (c) => {
    if (!dirty) draft.value = clone(c)
    if (pending !== null) {
      // Target fokus dipegang sampai barisnya benar-benar ada (isi kartu bisa berubah dua kali: lokal lalu jawaban
      // server), supaya ketikan sesudah Enter tidak jatuh ke luar editor.
      const i = pending
      void nextTick(() => {
        if (focusRow(i)) pending = null
      })
    }
  },
)

const root = ref<HTMLElement | null>(null)
const shift = ref({ x: 0, y: 0 })
function fitScreen() {
  const b = root.value?.getBoundingClientRect()
  if (!b) return
  const room = (start: number, end: number, size: number) =>
    end > size - 8 ? Math.max(8 - start, size - 8 - end) : start < 8 ? 8 - start : 0
  shift.value = { x: room(b.left, b.right, window.innerWidth), y: room(b.top, b.bottom, window.innerHeight) }
}
function focusRow(i: number, field = 'label'): boolean {
  const el = root.value?.querySelector<HTMLInputElement>(`[data-item="${i}"] [data-field="${field}"]`)
  el?.focus()
  el?.select()
  return !!el
}
void nextTick(() => {
  fitScreen()
  focusRow(0)
})

const content = (d: Draft): ShapeContent =>
  events.value
    ? { ...props.content, text: d.text.trim(), doc: null, events: d.events }
    : { ...props.content, text: '', doc: null, stages: d.stages }
const tidy = (d: Draft): Draft => ({
  text: d.text,
  stages: d.stages.map((s) => ({ label: s.label.trim() })),
  // Peristiwa disimpan urut posisinya, jadi urutan baris sama dengan urutan di garis.
  events: d.events
    .map((e) => ({ ...e, label: e.label.trim(), date: e.date.trim() }))
    .sort((a, b) => a.at - b.at),
})
function commit(d = draft.value) {
  dirty = false
  const next = content(tidy(d))
  const cur = props.content
  if (
    JSON.stringify([next.text, next.stages ?? null, next.events ?? null]) !==
    JSON.stringify([cur.text, cur.stages ?? null, cur.events ?? null])
  )
    emit('save', next)
}
function type(e: Event, set: (v: string) => void) {
  dirty = true
  set((e.target as HTMLInputElement).value)
  emit('live', content(draft.value))
}
const rows = computed(() => (events.value ? draft.value.events.length : draft.value.stages.length))
function addRow(at: number) {
  const d = tidy(draft.value)
  if (events.value) {
    if (d.events.length >= MAX_EVENTS)
      return emit('notice', `Garis waktu paling banyak ${MAX_EVENTS} peristiwa.`)
    const pos = newEventAt(d.events)
    d.events.push({ id: newRowId(d.events.map((e) => e.id)), label: '', date: '', at: pos })
    commit(d)
    pending = tidy(d).events.findIndex((e) => e.at === pos && !e.label)
    return
  }
  if (d.stages.length >= MAX_STAGES) return emit('notice', `Tahap paling banyak ${MAX_STAGES}.`)
  d.stages.splice(at, 0, { label: '' })
  pending = at
  commit(d)
}
function removeRow(i: number) {
  const d = tidy(draft.value)
  if (events.value) d.events.splice(i, 1)
  else if (d.stages.length > 1) d.stages.splice(i, 1)
  else return
  pending = Math.max(0, Math.min(i, rows.value - 2))
  commit(d)
}
function moveStage(i: number, to: number) {
  const d = tidy(draft.value)
  if (to < 0 || to >= d.stages.length) return
  const [s] = d.stages.splice(i, 1)
  d.stages.splice(to, 0, s!)
  pending = to
  commit(d)
}
function setAt(i: number, e: Event) {
  const v = Number((e.target as HTMLInputElement).value)
  if (!Number.isFinite(v)) return
  const d = tidy(draft.value)
  const id = draft.value.events[i]!.id
  const ev = d.events.find((x) => x.id === id)!
  ev.at = Math.round(Math.min(100, Math.max(0, v)) * 10) / 10
  commit(d)
}

function onKey(e: KeyboardEvent) {
  e.stopPropagation()
  const t = e.target as HTMLElement
  const i = Number(t.closest<HTMLElement>('[data-item]')?.dataset.item ?? -1)
  if (e.key === 'Escape') {
    e.preventDefault()
    if (dirty) commit()
    emit('close')
    return
  }
  if (t.tagName !== 'INPUT') return
  if (e.key === 'Enter') {
    e.preventDefault()
    if (i < 0) {
      if (dirty) commit()
      void nextTick(() => focusRow(0))
    } else if (i === rows.value - 1) {
      // Ketikan yang belum tersimpan satu langkah undo sendiri, terpisah dari tambah baris.
      if (dirty) commit()
      addRow(i + 1)
    } else {
      if (dirty) commit()
      void nextTick(() => focusRow(i + 1))
    }
  } else if (!events.value && e.altKey && (e.key === 'ArrowUp' || e.key === 'ArrowDown') && i >= 0) {
    e.preventDefault()
    if (dirty) commit()
    moveStage(i, i + (e.key === 'ArrowUp' ? -1 : 1))
  }
}

const editorWidth = computed(() =>
  Math.min(Math.max(props.w * z.value, events.value ? 560 : 440), window.innerWidth - 16),
)
const input =
  'h-11 min-w-0 rounded-card border border-line bg-surface px-2 text-sm text-ink focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-mark'
const iconBtn =
  'flex size-11 shrink-0 items-center justify-center rounded-card hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-40'
const noun = computed(() => (events.value ? 'peristiwa' : 'tahap'))
</script>

<template>
  <div
    ref="root"
    class="absolute top-0 left-0 z-20 origin-top-left rounded-card border border-mark bg-surface p-2 text-ink shadow-menu"
    :style="{
      transform: `translate(${shift.x / z}px, ${shift.y / z}px) scale(${1 / z})`,
      width: `${editorWidth}px`,
    }"
    role="group"
    :aria-label="events ? 'Edit garis waktu' : 'Edit tahap'"
    data-items-editor
    @pointerdown.stop
    @dblclick.stop
    @keydown="onKey"
  >
    <label v-if="events" class="mb-2 flex items-center gap-2">
      <span class="w-16 shrink-0 text-sm font-semibold">Judul</span>
      <input
        :value="draft.text"
        type="text"
        maxlength="200"
        :class="input"
        class="flex-1 font-semibold"
        data-field="title"
        @input="type($event, (v) => (draft.text = v))"
        @change="commit()"
      />
    </label>
    <p class="mb-1 text-xs text-ink-muted">
      Enter di baris terakhir menambah {{ noun }}.<template v-if="!events">
        Alt dan panah atas atau bawah memindah tahap.</template
      >
      Escape selesai.
    </p>
    <ol class="flex flex-col gap-1" :aria-label="events ? 'Peristiwa' : 'Tahap'">
      <template v-if="events">
        <li
          v-for="(ev, i) in draft.events"
          :key="ev.id"
          class="flex flex-wrap items-center gap-1"
          :data-item="i"
        >
          <input
            :value="ev.label"
            type="text"
            maxlength="100"
            :class="input"
            class="min-w-32 flex-1"
            placeholder="Label"
            :aria-label="`Label peristiwa ${i + 1}`"
            data-field="label"
            @input="type($event, (v) => (ev.label = v))"
            @change="commit()"
          />
          <input
            :value="ev.date"
            type="text"
            maxlength="40"
            :class="input"
            class="w-32"
            placeholder="Tanggal"
            :aria-label="`Tanggal peristiwa ${ev.label || i + 1}`"
            data-field="date"
            @input="type($event, (v) => (ev.date = v))"
            @change="commit()"
          />
          <input
            :value="ev.at"
            type="number"
            min="0"
            max="100"
            step="1"
            inputmode="decimal"
            :class="input"
            class="w-20"
            :aria-label="`Posisi peristiwa ${ev.label || i + 1}, persen dari kiri`"
            data-field="at"
            @change="setAt(i, $event)"
          />
          <button
            type="button"
            :class="iconBtn"
            class="text-urgent"
            :aria-label="`Hapus peristiwa ${ev.label || i + 1}`"
            data-remove-item
            @click="removeRow(i)"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 16 16"
              class="size-4"
              fill="none"
              stroke="currentColor"
              stroke-width="1.5"
              stroke-linecap="round"
            >
              <path d="M3 4.5h10M6.5 4.5V3h3v1.5M4.5 4.5l.7 8.5h5.6l.7-8.5M7 7v4M9 7v4" />
            </svg>
          </button>
        </li>
      </template>
      <template v-else>
        <li v-for="(st, i) in draft.stages" :key="i" class="flex items-center gap-1" :data-item="i">
          <span class="w-8 shrink-0 text-center text-sm text-ink-muted">{{ i + 1 }}</span>
          <input
            :value="st.label"
            type="text"
            maxlength="100"
            :class="input"
            class="flex-1"
            placeholder="Label tahap"
            :aria-label="`Label tahap ${i + 1}`"
            data-field="label"
            @input="type($event, (v) => (st.label = v))"
            @change="commit()"
          />
          <button
            type="button"
            :class="iconBtn"
            :disabled="i === 0"
            :aria-label="`Naikkan tahap ${st.label || i + 1}`"
            @click="moveStage(i, i - 1)"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 16 16"
              class="size-4"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M8 13V3M4 7l4-4 4 4" />
            </svg>
          </button>
          <button
            type="button"
            :class="iconBtn"
            :disabled="i === draft.stages.length - 1"
            :aria-label="`Turunkan tahap ${st.label || i + 1}`"
            @click="moveStage(i, i + 1)"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 16 16"
              class="size-4"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M8 3v10M4 9l4 4 4-4" />
            </svg>
          </button>
          <button
            type="button"
            :class="iconBtn"
            class="text-urgent"
            :disabled="draft.stages.length <= 1"
            :aria-label="`Hapus tahap ${st.label || i + 1}`"
            data-remove-item
            @click="removeRow(i)"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 16 16"
              class="size-4"
              fill="none"
              stroke="currentColor"
              stroke-width="1.5"
              stroke-linecap="round"
            >
              <path d="M3 4.5h10M6.5 4.5V3h3v1.5M4.5 4.5l.7 8.5h5.6l.7-8.5M7 7v4M9 7v4" />
            </svg>
          </button>
        </li>
      </template>
    </ol>
    <button
      type="button"
      class="mt-2 min-h-11 rounded-card border border-line bg-surface px-3 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
      data-add-item
      @click="addRow(rows)"
    >
      {{ events ? 'Tambah peristiwa' : 'Tambah tahap' }}
    </button>
  </div>
</template>
