<script setup lang="ts">
import { computed } from 'vue'
import type { ChecklistOp } from '../../api/whiteboard'
import {
  cardLines,
  cardTitle,
  kindLabel,
  readingOrder,
  type Card,
  type ChecklistContent,
} from '../../utils/whiteboard'

// Tampilan Daftar (docs/whiteboard.md bagian 8): kartu berurutan, dikelompokkan per kolom, dengan aksi yang sama
// dengan kanvas. Jalur utama untuk pembaca layar dan cadangan di ponsel kecil. Aksinya dijalankan halaman, supaya
// aturan simpan, undo, dan Tempat sampah tetap satu.
const props = defineProps<{ cards: Card[]; canWrite: boolean; boardTitle: string }>()
const emit = defineEmits<{
  show: [id: string]
  edit: [id: string]
  open: [id: string]
  move: [id: string]
  remove: [id: string]
  checklist: [id: string, op: ChecklistOp]
}>()

const groups = computed(() => {
  const loose = props.cards.filter((c) => !c.column_id && c.kind !== 'column')
  const cols = readingOrder(props.cards.filter((c) => c.kind === 'column'))
  return [
    ...cols.map((col) => ({
      id: col.id,
      title: `Kolom: ${cardTitle(col)}`,
      column: col,
      items: props.cards
        .filter((c) => c.column_id === col.id)
        .sort((a, b) => (a.col_order ?? 0) - (b.col_order ?? 0)),
    })),
    {
      id: 'lepas',
      title: cols.length ? 'Di luar kolom' : 'Semua kartu',
      column: null,
      items: readingOrder(loose),
    },
  ].filter((g) => g.items.length || g.column)
})

// Ringkasan isi: paling banyak empat baris, supaya daftar tetap bisa dipindai.
function preview(c: Card): string[] {
  if (c.kind === 'checklist') return []
  return cardLines(c)
    .slice(c.kind === 'heading' || c.kind === 'column' ? 1 : 0)
    .map((l) => l.text)
    .filter(Boolean)
    .slice(0, 4)
}

const editable = (c: Card) => c.kind !== 'drawing' && c.kind !== 'link'

const btn =
  'inline-flex min-h-11 items-center justify-center rounded-card border border-line bg-surface px-3 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
</script>

<template>
  <section
    class="h-full overflow-y-auto bg-canvas p-3 sm:p-4"
    :aria-label="`Daftar kartu papan ${boardTitle}`"
  >
    <p v-if="cards.length === 0" class="rounded-card border border-line bg-surface p-4 text-ink-muted">
      Papan ini masih kosong.
    </p>
    <div v-for="g in groups" :key="g.id" class="mx-auto mb-6 max-w-3xl">
      <h2 class="mb-2 font-bold break-words">{{ g.title }}</h2>
      <p v-if="g.items.length === 0" class="text-sm text-ink-muted">Kolom ini kosong.</p>
      <ol class="flex flex-col gap-2">
        <li
          v-for="c in g.items"
          :key="c.id"
          class="rounded-card border border-line bg-surface p-3"
          data-testid="wb-list-item"
        >
          <p class="text-sm text-ink-muted">{{ kindLabel[c.kind] }}</p>
          <h3 class="font-semibold break-words [overflow-wrap:anywhere]">{{ cardTitle(c) }}</h3>
          <p
            v-for="(line, i) in preview(c)"
            :key="i"
            class="text-sm break-words whitespace-pre-line [overflow-wrap:anywhere]"
          >
            {{ line }}
          </p>
          <ul v-if="c.kind === 'checklist'" class="mt-1">
            <li v-for="item in (c.content as ChecklistContent).items" :key="item.id">
              <label class="flex min-h-11 items-center gap-2" :class="canWrite ? 'cursor-pointer' : ''">
                <input
                  type="checkbox"
                  class="size-5 shrink-0 accent-[var(--color-mark)]"
                  :checked="item.done"
                  :disabled="!canWrite"
                  @change="emit('checklist', c.id, { op: 'set', item_id: item.id, done: !item.done })"
                />
                <span class="min-w-0 break-words" :class="item.done ? 'text-ink-muted line-through' : ''">
                  {{ item.text || 'Butir kosong' }}
                </span>
              </label>
            </li>
          </ul>
          <!-- aria-label diawali teks yang terlihat (label in name), lalu judul kartunya supaya tiap tombol unik. -->
          <div class="mt-2 flex flex-wrap gap-2">
            <button
              type="button"
              :class="btn"
              :aria-label="`Lihat di kanvas: ${cardTitle(c)}`"
              @click="emit('show', c.id)"
            >
              Lihat di kanvas
            </button>
            <button
              v-if="c.kind === 'board'"
              type="button"
              :class="btn"
              :aria-label="`Buka papan: ${cardTitle(c)}`"
              @click="emit('open', c.id)"
            >
              Buka papan
            </button>
            <template v-if="canWrite">
              <button
                v-if="editable(c)"
                type="button"
                :class="btn"
                :aria-label="`Ubah: ${cardTitle(c)}`"
                @click="emit('edit', c.id)"
              >
                Ubah
              </button>
              <button
                type="button"
                :class="btn"
                :aria-label="`Pindah ke papan lain: ${cardTitle(c)}`"
                @click="emit('move', c.id)"
              >
                Pindah ke papan lain
              </button>
              <button
                type="button"
                :class="btn"
                :aria-label="`Hapus: ${cardTitle(c)}`"
                @click="emit('remove', c.id)"
              >
                Hapus
              </button>
            </template>
          </div>
        </li>
      </ol>
      <div v-if="g.column && canWrite" class="mt-2 flex flex-wrap gap-2">
        <button
          type="button"
          :class="btn"
          :aria-label="`Ubah judul kolom: ${cardTitle(g.column)}`"
          @click="emit('edit', g.column.id)"
        >
          Ubah judul kolom
        </button>
        <button
          type="button"
          :class="btn"
          :aria-label="`Hapus kolom ${cardTitle(g.column)} beserta isinya`"
          @click="emit('remove', g.column.id)"
        >
          Hapus kolom
        </button>
      </div>
    </div>
  </section>
</template>
