<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, useId } from 'vue'
import TodoCard from './TodoCard.vue'
import { useTodos } from '../composables/useTodos'

// Tombol "To Do (n)" di bar papan, dengan panelnya (revisi 28). Di bawah 640 px panelnya jadi lembar dari
// bawah, pola yang sama dengan popover profil, supaya di ponsel tidak ada panel melayang yang terpotong.
const uid = useId()
const { openCount, load } = useTodos()

const open = ref(false)
const sheet = ref(false)
const trigger = ref<HTMLButtonElement | null>(null)
const panel = ref<HTMLElement | null>(null)

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') close()
}

function onPointerDown(e: Event) {
  const t = e.target as Node
  if (panel.value?.contains(t) || trigger.value?.contains(t)) return
  close()
}

function listen(on: boolean) {
  const fn = on ? document.addEventListener : document.removeEventListener
  fn('keydown', onKeydown as EventListener)
  fn('pointerdown', onPointerDown)
}

async function toggle() {
  if (open.value) {
    close()
    return
  }
  sheet.value = !window.matchMedia('(min-width: 640px)').matches
  open.value = true
  void load()
  listen(true)
  await nextTick()
  panel.value?.querySelector<HTMLElement>('input, button')?.focus()
}

function close() {
  if (!open.value) return
  open.value = false
  listen(false)
  // Fokus kembali ke pemicu, supaya Escape tidak membuang posisi keyboard.
  trigger.value?.focus()
}

onBeforeUnmount(() => listen(false))
</script>

<template>
  <div class="relative">
    <button
      ref="trigger"
      type="button"
      class="inline-flex min-h-11 items-center gap-2 rounded-card border border-line bg-surface px-3 font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
      :aria-expanded="open"
      :aria-controls="`${uid}-panel`"
      @click="toggle"
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
        <path d="M4 7h3l2 2 5-5M4 12h16M4 17h16" />
      </svg>
      To Do<span class="font-mono">({{ openCount }})</span>
    </button>

    <!-- Lembar dari bawah di ponsel, panel melayang mulai 640 px. -->
    <div v-if="open && sheet" class="fixed inset-0 z-40 bg-scrim" aria-hidden="true" @click="close" />
    <div
      v-if="open"
      :id="`${uid}-panel`"
      ref="panel"
      role="dialog"
      aria-label="To Do"
      class="border-line bg-surface shadow-dialog"
      :class="
        sheet
          ? 'fixed inset-x-0 bottom-0 z-50 max-h-[85dvh] overflow-y-auto rounded-t-card border-t px-3 pt-2 pb-[calc(1rem+env(safe-area-inset-bottom))]'
          : 'absolute top-full left-0 z-40 mt-1 w-80 max-w-[calc(100vw-2rem)] rounded-card border p-2'
      "
    >
      <!-- Judulnya ada di kepala kartu, jadi di sini hanya tombol tutup. -->
      <div class="mb-1 flex justify-end">
        <button
          type="button"
          aria-label="Tutup To Do"
          class="inline-flex size-11 items-center justify-center rounded-card text-ink-muted hover:bg-canvas hover:text-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
          @click="close"
        >
          <svg
            aria-hidden="true"
            viewBox="0 0 24 24"
            class="size-5"
            fill="none"
            stroke="currentColor"
            stroke-width="1.75"
            stroke-linecap="round"
          >
            <path d="M6 6l12 12M18 6L6 18" />
          </svg>
        </button>
      </div>
      <TodoCard />
    </div>
  </div>
</template>
