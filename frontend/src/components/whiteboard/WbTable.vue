<script setup lang="ts">
import { useId, watch } from 'vue'
import type { TableOp } from '../../api/whiteboard'
import { MAX_TABLE_COLS, MAX_TABLE_ROWS, type TableContent } from '../../utils/whiteboard'
import type { CaretHint } from './caret'
import WbRichField, { activateField } from './WbRichField.vue'
import WbRichText from './WbRichText'

// Tabel. Setiap perubahan dikirim sebagai operasi per sel, baris, atau kolom; selama diedit kartu dikunci (kunci
// lunak W2), jadi indeks tidak bergeser oleh orang lain. Judul dan sel berformat (revisi 49): satu field aktif
// sekaligus, dikirim saat pindah sel atau editor ditutup.
const props = defineProps<{ content: TableContent; editing: boolean; caret?: CaretHint | null }>()
const emit = defineEmits<{ op: [op: TableOp]; close: [] }>()

const uid = useId()
const cellKey = (r: number, c: number) => `${uid}:${r},${c}`
const docAt = (r: number, c: number) => props.content.docs?.[r]?.[c] ?? null

// Dobel klik pada sel: sel itu dibuka dengan kursor di posisi klik.
watch(
  () => props.editing,
  (on) => {
    if (on && props.caret?.cell) {
      const [r, c] = props.caret.cell.split(',').map(Number)
      activateField(cellKey(r ?? 0, c ?? 0))
    }
  },
  { immediate: true },
)
const caretFor = (r: number, c: number) =>
  props.caret?.cell === `${r},${c}` ? (props.caret.textIndex ?? props.caret.offset ?? null) : null

// Enter turun ke baris berikutnya; di baris terakhir baris baru ditambahkan (sel aktif begitu terpasang).
function onCellEnter(r: number, c: number) {
  if (r + 1 < props.content.rows.length) activateField(cellKey(r + 1, c))
  else if (props.content.rows.length < MAX_TABLE_ROWS) {
    emit('op', { op: 'add_row' })
    activateField(cellKey(r + 1, c))
  }
}

const btn =
  'inline-flex min-h-11 items-center gap-1 rounded-card px-2 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-mark disabled:opacity-50'
</script>

<template>
  <div class="flex min-w-0 flex-col gap-1">
    <WbRichField
      v-if="editing"
      :field-key="`${uid}:title`"
      :text="content.title ?? ''"
      :doc="content.title_doc"
      label="Judul tabel"
      placeholder="Judul tabel"
      data-title
      class="rounded-card border border-line bg-surface px-2 py-2 font-semibold text-ink"
      @commit="emit('op', { op: 'title', ...$event })"
      @enter="activateField(cellKey(0, 0))"
      @close="emit('close')"
    />
    <WbRichText v-else-if="content.title_doc" :doc="content.title_doc" class="font-semibold" />
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
              <WbRichField
                v-if="editing"
                :field-key="cellKey(r, c)"
                :text="cell"
                :doc="docAt(r, c)"
                :label="`Baris ${r + 1}, kolom ${c + 1}`"
                :caret-index="caretFor(r, c)"
                :data-cell="`${r},${c}`"
                class="w-full min-w-24 bg-surface px-2 py-2.5 text-ink"
                @commit="emit('op', { op: 'set', row: r, col: c, ...$event })"
                @enter="onCellEnter(r, c)"
                @close="emit('close')"
              />
              <span
                v-else
                class="block min-h-8 px-2 py-1.5 break-words"
                :data-cell-text="`${r},${c}`"
                :data-caret="cell ? '' : undefined"
                ><WbRichText v-if="docAt(r, c)" :doc="docAt(r, c)!" /><template v-else>{{
                  cell
                }}</template></span
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
