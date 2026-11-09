<script setup lang="ts">
import { nextTick, ref } from 'vue'
import PicPicker from './PicPicker.vue'
import type { UserItem } from '../api/users'
import { emptyDraft, maxSubtasks, type SubtaskDraft } from '../utils/subtask'

defineProps<{ candidates: UserItem[]; names: Map<number, string> }>()
const rows = defineModel<SubtaskDraft[]>({ required: true })
// Pembaca layar tidak melihat baris berpindah tempat, jadi setiap perubahan urutan atau hapus diumumkan.
const announce = ref('')

function focusId(id: string) {
  void nextTick(() => document.getElementById(id)?.focus())
}

function addAfter(index: number) {
  if (rows.value.length >= maxSubtasks) return
  const draft = emptyDraft()
  const next = [...rows.value]
  next.splice(index + 1, 0, draft)
  rows.value = next
  focusId(`subtask-title-${draft.key}`)
}

function setTitle(i: number, title: string) {
  rows.value = rows.value.map((r, k) => (k === i ? { ...r, title } : r))
}

function setPics(i: number, picIds: number[]) {
  rows.value = rows.value.map((r, k) => (k === i ? { ...r, picIds } : r))
}

// Enter di nama sub task membuat baris berikutnya, bukan menyimpan form. Baris kosong diabaikan.
function onEnter(i: number) {
  if (rows.value[i]?.title.trim()) addAfter(i)
}

function remove(i: number) {
  rows.value = rows.value.filter((_, k) => k !== i)
  announce.value = `Sub task ${i + 1} dihapus.`
  const left = rows.value
  if (left.length) focusId(`subtask-title-${left[Math.min(i, left.length - 1)].key}`)
  else focusId('subtask-add')
}

// Tombol yang ditekan ikut pindah bersama barisnya. Fokus dikembalikan ke tombol yang sama, atau ke
// lawannya kalau sudah di ujung (tombolnya jadi nonaktif), supaya pengguna keyboard tidak kehilangan posisi.
function move(i: number, delta: -1 | 1) {
  const j = i + delta
  if (j < 0 || j >= rows.value.length) return
  const next = [...rows.value]
  ;[next[i], next[j]] = [next[j], next[i]]
  rows.value = next
  const key = next[j].key
  announce.value = `Sub task dipindah ke urutan ${j + 1} dari ${next.length}.`
  void nextTick(() => {
    const same = document.getElementById(
      `subtask-${delta < 0 ? 'up' : 'down'}-${key}`,
    ) as HTMLButtonElement | null
    const other = document.getElementById(
      `subtask-${delta < 0 ? 'down' : 'up'}-${key}`,
    ) as HTMLButtonElement | null
    ;(same && !same.disabled ? same : other)?.focus()
  })
}

const iconButton =
  'inline-flex size-11 items-center justify-center rounded-card hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent active:translate-y-px disabled:cursor-not-allowed disabled:opacity-40'
</script>

<template>
  <fieldset>
    <legend class="text-sm font-medium">Sub task (opsional)</legend>
    <p class="mt-1 text-sm text-ink-muted">
      Tiap sub task punya nama dan PIC sendiri, satu orang atau lebih. Task tanpa sub task tidak punya PIC,
      jadi hanya Manajer yang bisa memindahkannya.
    </p>

    <ol v-if="rows.length" class="mt-2 space-y-3">
      <li v-for="(r, i) in rows" :key="r.key" class="rounded-card border border-line p-3">
        <div class="flex items-start gap-2">
          <span class="mt-2.5 w-6 shrink-0 text-sm font-medium text-ink-muted" aria-hidden="true">{{
            i + 1
          }}</span>
          <div class="min-w-0 flex-1">
            <label :for="`subtask-title-${r.key}`" class="sr-only">Nama sub task {{ i + 1 }}</label>
            <input
              :id="`subtask-title-${r.key}`"
              :value="r.title"
              type="text"
              maxlength="200"
              autocomplete="off"
              placeholder="Nama sub task"
              class="block min-h-11 w-full rounded-card border border-line bg-surface px-3"
              @input="setTitle(i, ($event.target as HTMLInputElement).value)"
              @keydown.enter.prevent="onEnter(i)"
            />
          </div>
        </div>

        <div class="mt-2 pl-8">
          <PicPicker
            :model-value="r.picIds"
            :candidates="candidates"
            :names="names"
            :subject="`sub task ${i + 1}`"
            @update:model-value="setPics(i, $event)"
          />
        </div>

        <div class="mt-2 flex justify-end gap-1 border-t border-line pt-2">
          <button
            :id="`subtask-up-${r.key}`"
            type="button"
            :class="iconButton"
            :disabled="i === 0"
            :aria-label="`Naikkan sub task ${i + 1}`"
            @click="move(i, -1)"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 24 24"
              class="size-5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M12 19V5M5 12l7-7 7 7" />
            </svg>
          </button>
          <button
            :id="`subtask-down-${r.key}`"
            type="button"
            :class="iconButton"
            :disabled="i === rows.length - 1"
            :aria-label="`Turunkan sub task ${i + 1}`"
            @click="move(i, 1)"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 24 24"
              class="size-5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M12 5v14M19 12l-7 7-7-7" />
            </svg>
          </button>
          <button
            type="button"
            :class="[iconButton, 'text-urgent']"
            :aria-label="`Hapus sub task ${i + 1}`"
            @click="remove(i)"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 24 24"
              class="size-5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M4 7h16M10 11v6M14 11v6M6 7l1 12h10l1-12M9 7V4h6v3" />
            </svg>
          </button>
        </div>
      </li>
    </ol>
    <p v-else class="mt-2 rounded-card border border-dashed border-line p-3 text-sm text-ink-muted">
      Belum ada sub task.
    </p>

    <button
      id="subtask-add"
      type="button"
      :disabled="rows.length >= maxSubtasks"
      class="mt-3 inline-flex min-h-11 items-center gap-2 rounded-card border border-line px-4 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60"
      @click="addAfter(rows.length - 1)"
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
      Tambah sub task
    </button>
    <p v-if="rows.length >= maxSubtasks" class="mt-1 text-sm text-ink-muted">
      Maksimal {{ maxSubtasks }} sub task.
    </p>
    <p class="sr-only" role="status" aria-live="polite">{{ announce }}</p>
  </fieldset>
</template>
