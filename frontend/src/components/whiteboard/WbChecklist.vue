<script setup lang="ts">
import { useId, watch } from 'vue'
import type { ChecklistOp } from '../../api/whiteboard'
import { newId, type ChecklistContent } from '../../utils/whiteboard'
import type { CaretHint } from './caret'
import WbRichField, { activateField } from './WbRichField.vue'
import WbRichText from './WbRichText'

// Checklist di kartu. Setiap perubahan dikirim sebagai operasi per butir (centang, teks, tambah, hapus), bukan isi
// utuh, jadi dua orang yang mengubah butir berbeda tidak saling menimpa. Centang boleh tanpa membuka editor. Judul
// dan teks butir berformat (revisi 49): satu field aktif sekaligus, dikirim saat pindah field atau editor ditutup.
const props = defineProps<{
  content: ChecklistContent
  editing: boolean
  canWrite: boolean
  caret?: CaretHint | null
}>()
const emit = defineEmits<{ op: [op: ChecklistOp]; close: [] }>()

const uid = useId()
const key = (id: string) => `${uid}:${id}`

// Dobel klik pada teks butir: butir itu dibuka dengan kursor di posisi klik.
watch(
  () => props.editing,
  (on) => {
    if (on && props.caret?.item) activateField(key(props.caret.item))
  },
  { immediate: true },
)
const caretFor = (id: string) =>
  props.caret?.item === id ? (props.caret.textIndex ?? props.caret.offset ?? null) : null

function add(index?: number) {
  const id = newId()
  emit('op', { op: 'add', item_id: id, text: '', index })
  // Butir baru muncul setelah server menjawab; field itu aktif begitu terpasang.
  activateField(key(id))
}

function onTitleEnter() {
  const first = props.content.items[0]
  if (first) activateField(key(first.id))
  else add()
}
</script>

<template>
  <div class="flex flex-col gap-1">
    <WbRichField
      v-if="editing"
      :field-key="key('title')"
      :text="content.title"
      :doc="content.title_doc"
      label="Judul checklist"
      placeholder="Judul checklist"
      class="rounded-card border border-line bg-surface px-2 py-2 font-semibold text-ink"
      @commit="emit('op', { op: 'title', ...$event })"
      @enter="onTitleEnter"
      @close="emit('close')"
    />
    <WbRichText v-else-if="content.title_doc" :doc="content.title_doc" class="font-semibold" />
    <p v-else-if="content.title" class="font-semibold break-words">{{ content.title }}</p>
    <p v-else class="text-ink-muted">Checklist tanpa judul</p>

    <ul class="flex flex-col">
      <li v-for="(item, i) in content.items" :key="item.id" class="flex min-h-11 items-center gap-1">
        <label
          class="inline-flex size-11 shrink-0 cursor-pointer items-center justify-center"
          @pointerdown.stop
        >
          <input
            type="checkbox"
            class="size-5 accent-[var(--color-mark)]"
            :checked="item.done"
            :disabled="!canWrite"
            :aria-label="item.text ? `Selesai: ${item.text}` : `Selesai: butir ${i + 1}`"
            @change="
              emit('op', { op: 'set', item_id: item.id, done: ($event.target as HTMLInputElement).checked })
            "
          />
        </label>
        <template v-if="editing">
          <WbRichField
            :field-key="key(item.id)"
            :text="item.text"
            :doc="item.doc"
            :label="`Teks butir ${i + 1}`"
            :caret-index="caretFor(item.id)"
            :data-item="item.id"
            class="flex-1 rounded-card border border-line bg-surface px-2 py-2 text-ink"
            @commit="emit('op', { op: 'set', item_id: item.id, ...$event })"
            @enter="add(i + 1)"
            @close="emit('close')"
          />
          <button
            type="button"
            class="inline-flex size-11 shrink-0 items-center justify-center rounded-card text-ink-muted hover:bg-canvas hover:text-ink"
            :aria-label="`Hapus butir ${item.text || i + 1}`"
            @pointerdown.stop
            @click="emit('op', { op: 'remove', item_id: item.id })"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 24 24"
              class="size-4"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
            >
              <path d="M6 6l12 12M18 6L6 18" />
            </svg>
          </button>
        </template>
        <span
          v-else
          class="min-w-0 flex-1 break-words"
          :class="item.done ? 'text-ink-muted line-through' : ''"
          :data-item-text="item.id"
          :data-caret="item.text ? '' : undefined"
        >
          <WbRichText v-if="item.doc" :doc="item.doc" />
          <template v-else>{{ item.text || 'Butir kosong' }}</template>
        </span>
      </li>
    </ul>
    <p v-if="!editing && content.items.length === 0" class="text-sm text-ink-muted">Belum ada butir.</p>
    <button
      v-if="editing"
      type="button"
      class="inline-flex min-h-11 items-center gap-1 self-start rounded-card px-2 font-medium hover:bg-canvas"
      @pointerdown.stop
      @click="add()"
    >
      <svg
        aria-hidden="true"
        viewBox="0 0 24 24"
        class="size-4"
        fill="none"
        stroke="currentColor"
        stroke-width="1.75"
        stroke-linecap="round"
      >
        <path d="M12 5v14M5 12h14" />
      </svg>
      Tambah butir
    </button>
  </div>
</template>
