<script setup lang="ts">
import { nextTick, ref, useId, watch } from 'vue'
import type { TableOp } from '../../api/whiteboard'
import { MAX_TABLE_COLS, MAX_TABLE_ROWS, type TableContent } from '../../utils/whiteboard'
import { placeCaret, type CaretHint } from './caret'

// Tabel teks polos. Setiap perubahan dikirim sebagai operasi per sel, baris, atau kolom; selama diedit kartu
// dikunci (kunci lunak W2), jadi indeks tidak bergeser oleh orang lain.
const props = defineProps<{ content: TableContent; editing: boolean; caret?: CaretHint | null }>()
const emit = defineEmits<{ op: [op: TableOp]; close: [] }>()

const uid = useId()
const root = ref<HTMLElement | null>(null)

// Draf lokal: jawaban server untuk operasi lain (mis. judul atau tambah baris) merender ulang tabel; tanpa draf,
// Vue menimpa nilai input dan ketikan yang belum dikirim hilang. Sel yang belum diubah pengguna ikut nilai server.
const draft = ref<string[][]>([])
const titleDraft = ref('')
let known: string[][] = []
let knownTitle = ''
watch(
  () => props.content,
  (c) => {
    draft.value = c.rows.map((row, r) =>
      row.map((v, i) => {
        const d = draft.value[r]?.[i]
        return d !== undefined && d !== known[r]?.[i] ? d : v
      }),
    )
    known = c.rows.map((row) => [...row])
    if (titleDraft.value === knownTitle) titleDraft.value = c.title ?? ''
    knownTitle = c.title ?? ''
  },
  { immediate: true },
)

// Editor ditutup tanpa blur: sel yang berubah dikirim sebelum input hilang dari DOM (sama dengan checklist).
watch(
  () => props.editing,
  (on, was) => {
    if (!was || on) return
    for (const el of root.value?.querySelectorAll<HTMLInputElement>('[data-cell]') ?? []) setCell(el)
    const title = root.value?.querySelector<HTMLInputElement>('[data-title]')
    if (title) setTitle(title)
  },
)

// Dobel klik pada sel: kursor diletakkan di sel itu, di posisi klik.
watch(
  () => props.editing,
  async (on) => {
    if (!on || !props.caret?.cell) return
    await nextTick()
    placeCaret(
      root.value?.querySelector<HTMLInputElement>(`[data-cell="${props.caret.cell}"]`),
      props.caret.offset,
    )
  },
)

function setTitle(el: HTMLInputElement) {
  const text = el.value.trim()
  if (text !== (props.content.title ?? '')) emit('op', { op: 'title', text })
}

function onTitleKey(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    setTitle(e.target as HTMLInputElement)
    focusCell(0, 0)
  } else if (e.key === 'Escape') {
    e.preventDefault()
    e.stopPropagation()
    setTitle(e.target as HTMLInputElement)
    emit('close')
  }
}

function setCell(el: HTMLInputElement) {
  const [r, c] = (el.dataset.cell ?? '').split(',').map(Number)
  const text = el.value.trim()
  if (r === undefined || c === undefined || props.content.rows[r]?.[c] === undefined) return
  if (text !== props.content.rows[r]![c]) emit('op', { op: 'set', row: r, col: c, text })
}

async function focusCell(r: number, c: number) {
  await nextTick()
  // Baris atau kolom baru muncul setelah server menjawab.
  const find = (tries: number) => {
    const el = root.value?.querySelector<HTMLInputElement>(`[data-cell="${r},${c}"]`)
    if (el) el.focus()
    else if (tries > 0) setTimeout(() => find(tries - 1), 100)
  }
  find(20)
}

function onCellKey(e: KeyboardEvent, r: number, c: number) {
  if (e.key === 'Enter') {
    e.preventDefault()
    setCell(e.target as HTMLInputElement)
    if (r + 1 < props.content.rows.length) focusCell(r + 1, c)
    else if (props.content.rows.length < MAX_TABLE_ROWS) {
      emit('op', { op: 'add_row' })
      focusCell(r + 1, c)
    }
  } else if (e.key === 'Escape') {
    e.preventDefault()
    e.stopPropagation()
    setCell(e.target as HTMLInputElement)
    ;(e.target as HTMLInputElement).blur()
    emit('close')
  }
}

const btn =
  'inline-flex min-h-11 items-center gap-1 rounded-card px-2 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-mark disabled:opacity-50'
</script>

<template>
  <div ref="root" class="flex min-w-0 flex-col gap-1">
    <template v-if="editing">
      <label :for="`${uid}-title`" class="sr-only">Judul tabel</label>
      <input
        :id="`${uid}-title`"
        v-model="titleDraft"
        data-title
        maxlength="200"
        placeholder="Judul tabel"
        class="min-h-11 rounded-card border border-line bg-surface px-2 font-semibold text-ink"
        @pointerdown.stop
        @change="setTitle($event.target as HTMLInputElement)"
        @keydown="onTitleKey"
      />
    </template>
    <p v-else-if="content.title" class="font-semibold break-words">{{ content.title }}</p>
    <div class="min-w-0 overflow-x-auto" @pointerdown="editing && $event.stopPropagation()">
      <table class="w-full border-collapse text-sm">
        <caption class="sr-only">
          {{
            content.title || 'Tabel'
          }}
        </caption>
        <tbody>
          <tr
            v-for="(row, r) in content.rows"
            :key="r"
            :class="r === 0 && content.header ? 'bg-canvas font-semibold' : ''"
          >
            <component
              :is="r === 0 && content.header ? 'th' : 'td'"
              v-for="(cell, c) in row"
              :key="c"
              :scope="r === 0 && content.header ? 'col' : undefined"
              class="border border-line p-0 text-left align-top"
            >
              <template v-if="editing">
                <label :for="`${uid}-${r}-${c}`" class="sr-only">Baris {{ r + 1 }}, kolom {{ c + 1 }}</label>
                <input
                  :id="`${uid}-${r}-${c}`"
                  v-model="draft[r]![c]"
                  :data-cell="`${r},${c}`"
                  maxlength="500"
                  class="min-h-11 w-full min-w-24 bg-surface px-2 text-ink"
                  @change="setCell($event.target as HTMLInputElement)"
                  @keydown="onCellKey($event, r, c)"
                />
              </template>
              <span
                v-else
                class="block min-h-8 px-2 py-1.5 break-words"
                :data-cell-text="`${r},${c}`"
                :data-caret="cell ? '' : undefined"
                >{{ cell }}</span
              >
            </component>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-if="editing" class="flex flex-wrap gap-1" @pointerdown.stop>
      <button
        type="button"
        :class="btn"
        :disabled="content.rows.length >= MAX_TABLE_ROWS"
        @click="emit('op', { op: 'add_row' })"
      >
        Tambah baris
      </button>
      <button
        type="button"
        :class="btn"
        :disabled="(content.rows[0]?.length ?? 0) >= MAX_TABLE_COLS"
        @click="emit('op', { op: 'add_col' })"
      >
        Tambah kolom
      </button>
      <button
        type="button"
        :class="btn"
        :disabled="content.rows.length <= 1"
        @click="emit('op', { op: 'remove_row', index: content.rows.length - 1 })"
      >
        Hapus baris terakhir
      </button>
      <button
        type="button"
        :class="btn"
        :disabled="(content.rows[0]?.length ?? 0) <= 1"
        @click="emit('op', { op: 'remove_col', index: (content.rows[0]?.length ?? 1) - 1 })"
      >
        Hapus kolom terakhir
      </button>
      <label class="inline-flex min-h-11 items-center gap-2 px-2 text-sm">
        <input
          type="checkbox"
          class="size-5 accent-[var(--color-mark)]"
          :checked="content.header"
          @change="emit('op', { op: 'header', on: ($event.target as HTMLInputElement).checked })"
        />
        Baris pertama judul
      </label>
    </div>
  </div>
</template>
