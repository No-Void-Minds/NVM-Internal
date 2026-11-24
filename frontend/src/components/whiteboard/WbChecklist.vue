<script setup lang="ts">
import { nextTick, ref, useId, watch } from 'vue'
import type { ChecklistOp } from '../../api/whiteboard'
import { newId, type ChecklistContent } from '../../utils/whiteboard'

// Checklist di kartu. Setiap perubahan dikirim sebagai operasi per butir (centang, teks, tambah, hapus), bukan isi
// utuh, jadi dua orang yang mengubah butir berbeda tidak saling menimpa. Centang boleh tanpa membuka editor.
const props = defineProps<{ content: ChecklistContent; editing: boolean; canWrite: boolean }>()
const emit = defineEmits<{ op: [op: ChecklistOp]; close: [] }>()

const uid = useId()
const root = ref<HTMLElement | null>(null)

// Editor ditutup tanpa blur (klik di kanvas melepas input dari DOM, jadi change tidak terpanggil): isian yang
// berubah dikirim sebelum DOM diperbarui. Watch bawaan berjalan sebelum render, saat input masih ada.
watch(
  () => props.editing,
  (on, was) => {
    if (!was || on) return
    const title = root.value?.querySelector<HTMLInputElement>(`#${CSS.escape(`${uid}-title`)}`)
    if (title && title.value.trim() !== props.content.title)
      emit('op', { op: 'title', text: title.value.trim() })
    for (const el of root.value?.querySelectorAll<HTMLInputElement>('[data-item]') ?? []) {
      const cur = props.content.items.find((i) => i.id === el.dataset.item)
      if (cur && el.value.trim() !== cur.text)
        emit('op', { op: 'set', item_id: cur.id, text: el.value.trim() })
    }
  },
)

function setTitle(e: Event) {
  const text = (e.target as HTMLInputElement).value.trim()
  if (text !== props.content.title) emit('op', { op: 'title', text })
}

function setText(id: string, e: Event) {
  const text = (e.target as HTMLInputElement).value.trim()
  const cur = props.content.items.find((i) => i.id === id)
  if (cur && text !== cur.text) emit('op', { op: 'set', item_id: id, text })
}

async function add(index?: number) {
  const id = newId()
  emit('op', { op: 'add', item_id: id, text: '', index })
  await nextTick()
  // Butir baru muncul setelah server menjawab; fokus dipindah begitu inputnya ada.
  const focusNew = (tries: number) => {
    const el = root.value?.querySelector<HTMLInputElement>(`[data-item="${id}"]`)
    if (el) el.focus()
    else if (tries > 0) setTimeout(() => focusNew(tries - 1), 100)
  }
  focusNew(20)
}

function onItemKey(e: KeyboardEvent, index: number) {
  if (e.key === 'Enter') {
    e.preventDefault()
    ;(e.target as HTMLInputElement).blur()
    add(index + 1)
  }
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.editing) {
    e.preventDefault()
    e.stopPropagation()
    ;(document.activeElement as HTMLElement | null)?.blur()
    emit('close')
  }
}
</script>

<template>
  <div ref="root" class="flex flex-col gap-1" @keydown="onKeydown">
    <template v-if="editing">
      <label :for="`${uid}-title`" class="sr-only">Judul checklist</label>
      <input
        :id="`${uid}-title`"
        :value="content.title"
        maxlength="200"
        placeholder="Judul checklist"
        class="min-h-11 rounded-card border border-line bg-surface px-2 font-semibold text-ink"
        @pointerdown.stop
        @change="setTitle"
      />
    </template>
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
          <label :for="`${uid}-${item.id}`" class="sr-only">Teks butir {{ i + 1 }}</label>
          <input
            :id="`${uid}-${item.id}`"
            :data-item="item.id"
            :value="item.text"
            maxlength="500"
            class="min-h-11 min-w-0 flex-1 rounded-card border border-line bg-surface px-2 text-ink"
            @pointerdown.stop
            @change="setText(item.id, $event)"
            @keydown="onItemKey($event, i)"
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
        >
          {{ item.text || 'Butir kosong' }}
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
