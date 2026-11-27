<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import {
  ERD_HEAD,
  ERD_KEYS,
  ERD_MAX_ROWS,
  ERD_ROW,
  keyLabel,
  newRowId,
  type ErdKey,
  type ErdRow,
  type ShapeContent,
} from '../../utils/wbShapes'

// Tabel ERD (revisi 57a). Tampilan memakai angka ERD_HEAD dan ERD_ROW yang sama dengan port garis per baris, jadi
// garis selalu tepat di tengah barisnya. Mode edit membuka editor di atas tabel dengan ukuran layar tetap (baris
// 44 px pada zoom berapa pun), karena tombol kunci, urutan, dan hapus per baris tidak muat 44 px di baris 28 px.
// Setiap aksi dikirim sebagai simpan sendiri, jadi halaman mencatatnya satu langkah undo per aksi.
const props = defineProps<{
  content: ShapeContent
  w: number
  editing: boolean
  zoom?: number
}>()
const emit = defineEmits<{
  save: [c: ShapeContent]
  live: [c: ShapeContent]
  close: []
  notice: [msg: string]
  rowPort: [row: string, side: 'left' | 'right', e: PointerEvent]
}>()

const rows = computed(() => props.content.rows ?? [])
const z = computed(() => props.zoom || 1)

// Draf ketikan; aksi struktur (tambah, hapus, urutan, kunci) dibangun dari draf lalu disimpan, dan draf diisi ulang
// dari isi kartu sesudahnya. Bila halaman menolak (kartu terkunci), isi kartu tidak berubah dan draf tetap sah.
const clone = (c: ShapeContent) => ({
  name: c.name ?? '',
  rows: (c.rows ?? []).map((r) => ({ ...r, keys: [...r.keys] })),
})
const draft = ref(clone(props.content))
let dirty = false
watch(
  () => props.content,
  (c) => {
    if (!dirty) draft.value = clone(c)
  },
)
watch(
  () => props.editing,
  (on) => {
    dirty = false
    draft.value = clone(props.content)
    shift.value = { x: 0, y: 0 }
    if (on)
      void nextTick(() => {
        fitScreen()
        focusField(draft.value.rows[0]?.id ?? 'name', 'name')
      })
  },
)

// Editor dibuka di tabel, lalu digeser (dalam piksel layar) supaya seluruhnya di dalam layar, juga di 360 px. Yang
// lebih tinggi dari layar mulai 8 px dari atas; sisanya dijangkau dengan menggeser kanvas (roda tetikus tidak
// ditahan editor).
const shift = ref({ x: 0, y: 0 })
function fitScreen() {
  const b = root.value?.getBoundingClientRect()
  if (!b) return
  const room = (start: number, end: number, size: number) =>
    end > size - 8 ? Math.max(8 - start, size - 8 - end) : start < 8 ? 8 - start : 0
  shift.value = { x: room(b.left, b.right, window.innerWidth), y: room(b.top, b.bottom, window.innerHeight) }
}

const root = ref<HTMLElement | null>(null)
function focusField(row: string, field: 'name' | 'type') {
  const sel = row === 'name' ? '[data-table-name-input]' : `[data-edit-row="${row}"] [data-field="${field}"]`
  const el = root.value?.querySelector<HTMLInputElement>(sel)
  el?.focus()
  el?.select()
}
let pendingFocus: { row: string; field: 'name' | 'type' } | null = null
watch(
  () => props.content.rows,
  () => {
    if (!pendingFocus) return
    const f = pendingFocus
    pendingFocus = null
    void nextTick(() => focusField(f.row, f.field))
  },
)

const content = (name: string, list: ErdRow[]): ShapeContent => ({ ...props.content, name, rows: list })
function commit(list = draft.value.rows) {
  dirty = false
  const next = content(draft.value.name.trim(), list)
  if (JSON.stringify(next.rows) !== JSON.stringify(props.content.rows) || next.name !== props.content.name)
    emit('save', next)
}
function type(i: number | 'name', field: 'name' | 'type', e: Event) {
  const v = (e.target as HTMLInputElement).value
  dirty = true
  if (i === 'name') draft.value.name = v
  else draft.value.rows[i]![field] = v
  emit('live', content(draft.value.name, draft.value.rows))
}
const trimmed = () => draft.value.rows.map((r) => ({ ...r, name: r.name.trim(), type: r.type.trim() }))

function addRow(at: number) {
  if (draft.value.rows.length >= ERD_MAX_ROWS) {
    emit('notice', `Tabel paling banyak ${ERD_MAX_ROWS} kolom.`)
    return
  }
  const id = newRowId(draft.value.rows.map((r) => r.id))
  const list = trimmed()
  list.splice(at, 0, { id, name: '', type: '', keys: [], nullable: false })
  pendingFocus = { row: id, field: 'name' }
  commit(list)
}
function removeRow(i: number) {
  const list = trimmed()
  const next = list[i + 1] ?? list[i - 1]
  list.splice(i, 1)
  pendingFocus = next ? { row: next.id, field: 'name' } : null
  commit(list)
  if (!next) void nextTick(() => root.value?.querySelector<HTMLButtonElement>('[data-add-row]')?.focus())
}
function moveRow(i: number, to: number) {
  const list = trimmed()
  if (to < 0 || to >= list.length || to === i) return
  const [r] = list.splice(i, 1)
  list.splice(to, 0, r!)
  commit(list)
  void nextTick(() =>
    root.value?.querySelector<HTMLElement>(`[data-edit-row="${r!.id}"] [data-grip]`)?.focus(),
  )
}
function toggleKey(i: number, k: ErdKey) {
  const list = trimmed()
  const r = list[i]!
  const on = !r.keys.includes(k)
  r.keys = ERD_KEYS.filter((x) => (x === k ? on : r.keys.includes(x)))
  // Kunci primer tidak boleh kosong, jadi PK mematikan nullable.
  if (k === 'pk' && on) r.nullable = false
  commit(list)
}
function toggleNull(i: number) {
  const list = trimmed()
  list[i]!.nullable = !list[i]!.nullable
  if (list[i]!.nullable) list[i]!.keys = list[i]!.keys.filter((k) => k !== 'pk')
  commit(list)
}

// Keyboard: Enter di baris terakhir menambah baris, di baris lain pindah ke baris berikutnya; Alt+panah atas atau
// bawah memindah baris; Escape menyimpan lalu keluar. Tombol tidak diteruskan ke pintasan kanvas.
function onKey(e: KeyboardEvent) {
  e.stopPropagation()
  const t = e.target as HTMLElement
  const rowId = t.closest<HTMLElement>('[data-edit-row]')?.dataset.editRow
  const i = rowId ? draft.value.rows.findIndex((r) => r.id === rowId) : -1
  if (e.key === 'Escape') {
    e.preventDefault()
    if (dirty) commit(trimmed())
    emit('close')
    return
  }
  if (t.tagName !== 'INPUT') return
  if (e.key === 'Enter') {
    e.preventDefault()
    if (i < 0) {
      if (dirty) commit(trimmed())
      if (draft.value.rows.length) focusField(draft.value.rows[0]!.id, 'name')
      else addRow(0)
    } else if (i === draft.value.rows.length - 1) addRow(i + 1)
    else {
      if (dirty) commit(trimmed())
      focusField(draft.value.rows[i + 1]!.id, 'name')
    }
  } else if (e.altKey && (e.key === 'ArrowUp' || e.key === 'ArrowDown') && i >= 0) {
    e.preventDefault()
    const field = (t.dataset.field as 'name' | 'type') ?? 'name'
    const id = rowId!
    moveRow(i, i + (e.key === 'ArrowUp' ? -1 : 1))
    pendingFocus = { row: id, field }
  }
}
function onGripKey(i: number, e: KeyboardEvent) {
  if (e.key !== 'ArrowUp' && e.key !== 'ArrowDown') return
  e.preventDefault()
  e.stopPropagation()
  moveRow(i, i + (e.key === 'ArrowUp' ? -1 : 1))
}

// Seret pegangan baris: posisi sisip dari tengah baris lain di bawah kursor.
const dragFrom = ref<number | null>(null)
const dragTo = ref<number | null>(null)
function onGripDown(i: number, e: PointerEvent) {
  if (e.button !== 0) return
  e.preventDefault()
  e.stopPropagation()
  dragFrom.value = i
  dragTo.value = i
  const move = (ev: PointerEvent) => {
    const els = [...(root.value?.querySelectorAll<HTMLElement>('[data-edit-row]') ?? [])]
    let to = els.length - 1
    for (let k = 0; k < els.length; k++) {
      const b = els[k]!.getBoundingClientRect()
      if (ev.clientY < b.top + b.height / 2) {
        to = k
        break
      }
    }
    dragTo.value = to
  }
  const up = () => {
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', up)
    window.removeEventListener('pointercancel', up)
    const from = dragFrom.value
    const to = dragTo.value
    dragFrom.value = dragTo.value = null
    if (from !== null && to !== null && to !== from) moveRow(from, to)
  }
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', up)
  window.addEventListener('pointercancel', up)
}

const editorWidth = computed(() => Math.min(Math.max(props.w * z.value, 600), window.innerWidth - 16))
const toggle =
  'flex size-11 shrink-0 items-center justify-center rounded-card border border-line text-xs font-semibold hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark aria-pressed:bg-accent aria-pressed:text-on-accent aria-pressed:hover:bg-accent-hover'
const input =
  'h-11 min-w-0 rounded-card border border-line bg-surface px-2 text-sm text-ink focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-mark'
</script>

<template>
  <div class="absolute inset-0 overflow-hidden text-ink" data-erd-table>
    <p
      class="flex items-center justify-center truncate px-2 text-sm font-bold"
      :style="{ height: `${ERD_HEAD}px` }"
      data-table-name
    >
      {{ content.name || 'tabel' }}
    </p>
    <div
      v-for="r in rows"
      :key="r.id"
      class="flex items-center gap-1.5 px-2 text-sm"
      :style="{ height: `${ERD_ROW}px` }"
      :data-row="r.id"
    >
      <span class="flex w-16 shrink-0 items-center gap-1 text-xs font-semibold" data-keys>
        <svg
          v-if="r.keys.includes('pk')"
          aria-hidden="true"
          viewBox="0 0 16 16"
          class="size-3.5 shrink-0"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
          data-key-icon
        >
          <circle cx="5" cy="8" r="3" />
          <path d="M8 8h6.5M12 8v3M14.5 8v2" />
        </svg>
        {{ keyLabel(r.keys) }}
      </span>
      <span class="min-w-0 flex-1 truncate" :class="r.keys.includes('pk') ? 'font-semibold' : ''">{{
        r.name
      }}</span>
      <span class="max-w-[45%] shrink-0 truncate font-mono text-xs text-ink-muted">
        {{ r.type }}<template v-if="r.nullable"> null</template>
      </span>
    </div>
    <p
      v-if="!rows.length"
      class="flex items-center px-2 text-sm text-ink-muted"
      :style="{ height: `${ERD_ROW}px` }"
    >
      Belum ada kolom
    </p>
  </div>

  <!-- Editor di tempat: ukuran layar tetap lewat skala 1/zoom, menutupi tabel selama mode edit. -->
  <div
    v-if="editing"
    ref="root"
    class="absolute top-0 left-0 z-20 origin-top-left rounded-card border border-mark bg-surface p-2 text-ink shadow-menu"
    :style="{
      transform: `translate(${shift.x / z}px, ${shift.y / z}px) scale(${1 / z})`,
      width: `${editorWidth}px`,
    }"
    role="group"
    aria-label="Edit tabel"
    data-erd-editor
    @pointerdown.stop
    @dblclick.stop
    @keydown="onKey"
  >
    <label class="flex items-center gap-2">
      <span class="w-24 shrink-0 text-sm font-semibold">Nama tabel</span>
      <input
        :value="draft.name"
        type="text"
        maxlength="100"
        :class="input"
        class="flex-1 font-bold"
        data-table-name-input
        @input="type('name', 'name', $event)"
        @change="commit(trimmed())"
      />
    </label>
    <p class="mt-2 mb-1 text-xs text-ink-muted">
      Enter di kolom terakhir menambah kolom. Alt dan panah atas atau bawah memindah kolom. Escape selesai.
    </p>
    <ol class="flex flex-col gap-1" aria-label="Kolom tabel">
      <li
        v-for="(r, i) in draft.rows"
        :key="r.id"
        class="relative flex flex-wrap items-center gap-1"
        :class="dragTo === i && dragFrom !== null && dragFrom !== i ? 'border-t-2 border-mark' : ''"
        :data-edit-row="r.id"
      >
        <div
          class="absolute top-1/2 -left-14 size-11 -translate-y-1/2 cursor-crosshair"
          aria-hidden="true"
          data-row-port="left"
          @pointerdown.stop.prevent="emit('rowPort', r.id, 'left', $event)"
        >
          <span class="absolute inset-3.5 rounded-full border-2 border-mark bg-surface" />
        </div>
        <button
          type="button"
          class="flex size-11 shrink-0 cursor-grab items-center justify-center rounded-card hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
          :aria-label="`Pindah kolom ${r.name || i + 1}, panah atas atau bawah`"
          data-grip
          @pointerdown="onGripDown(i, $event)"
          @keydown="onGripKey(i, $event)"
        >
          <svg aria-hidden="true" viewBox="0 0 16 16" class="size-4" fill="currentColor">
            <circle cx="6" cy="4" r="1.25" />
            <circle cx="10" cy="4" r="1.25" />
            <circle cx="6" cy="8" r="1.25" />
            <circle cx="10" cy="8" r="1.25" />
            <circle cx="6" cy="12" r="1.25" />
            <circle cx="10" cy="12" r="1.25" />
          </svg>
        </button>
        <input
          :value="r.name"
          type="text"
          maxlength="100"
          :class="input"
          class="min-w-32 flex-1"
          placeholder="nama_kolom"
          :aria-label="`Nama kolom ${i + 1}`"
          data-field="name"
          @input="type(i, 'name', $event)"
          @change="commit(trimmed())"
        />
        <input
          :value="r.type"
          type="text"
          maxlength="60"
          :class="input"
          class="w-32 font-mono"
          placeholder="tipe"
          :aria-label="`Tipe data kolom ${r.name || i + 1}`"
          data-field="type"
          @input="type(i, 'type', $event)"
          @change="commit(trimmed())"
        />
        <button
          v-for="k in ERD_KEYS"
          :key="k"
          type="button"
          :class="toggle"
          :aria-pressed="r.keys.includes(k)"
          :aria-label="`${k.toUpperCase()} untuk kolom ${r.name || i + 1}`"
          :data-key-toggle="k"
          @click="toggleKey(i, k)"
        >
          {{ k.toUpperCase() }}
        </button>
        <button
          type="button"
          :class="toggle"
          class="w-14"
          :aria-pressed="r.nullable"
          :aria-label="`Boleh kosong (null) untuk kolom ${r.name || i + 1}`"
          data-null-toggle
          @click="toggleNull(i)"
        >
          NULL
        </button>
        <button
          type="button"
          class="flex size-11 shrink-0 items-center justify-center rounded-card text-urgent hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
          :aria-label="`Hapus kolom ${r.name || i + 1}`"
          data-remove-row
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
        <div
          class="absolute top-1/2 -right-14 size-11 -translate-y-1/2 cursor-crosshair"
          aria-hidden="true"
          data-row-port="right"
          @pointerdown.stop.prevent="emit('rowPort', r.id, 'right', $event)"
        >
          <span class="absolute inset-3.5 rounded-full border-2 border-mark bg-surface" />
        </div>
      </li>
    </ol>
    <button
      type="button"
      class="mt-2 min-h-11 rounded-card border border-line bg-surface px-3 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
      data-add-row
      @click="addRow(draft.rows.length)"
    >
      Tambah kolom
    </button>
  </div>
</template>
