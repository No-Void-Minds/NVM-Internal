<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, useId } from 'vue'

export type MenuItem = { label: string; danger?: boolean; action: () => void }

defineProps<{ items: MenuItem[]; label: string }>()

const open = ref(false)
const root = ref<HTMLElement | null>(null)
const menuId = useId()

function menuButtons(): HTMLButtonElement[] {
  return [...(root.value?.querySelectorAll<HTMLButtonElement>('[role="menuitem"]') ?? [])]
}

async function show() {
  open.value = true
  document.addEventListener('pointerdown', onOutside)
  await nextTick()
  menuButtons()[0]?.focus()
}

function close(returnFocus = false) {
  open.value = false
  document.removeEventListener('pointerdown', onOutside)
  if (returnFocus) root.value?.querySelector<HTMLButtonElement>('[aria-haspopup]')?.focus()
}

function onOutside(e: PointerEvent) {
  if (!root.value?.contains(e.target as Node)) close()
}

// Menu ditutup saat fokus keluar dengan Tab, supaya tidak tertinggal terbuka di belakang elemen lain.
function onFocusOut(e: FocusEvent) {
  if (open.value && !root.value?.contains(e.relatedTarget as Node | null)) close()
}

function choose(action: () => void) {
  close(true)
  action()
}

function onKeydown(e: KeyboardEvent) {
  if (!open.value) return
  if (e.key === 'Escape') {
    e.preventDefault()
    close(true)
    return
  }
  if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return
  e.preventDefault()
  const buttons = menuButtons()
  const i = buttons.indexOf(document.activeElement as HTMLButtonElement)
  const next = e.key === 'ArrowDown' ? (i + 1) % buttons.length : (i - 1 + buttons.length) % buttons.length
  buttons[next]?.focus()
}

onBeforeUnmount(() => document.removeEventListener('pointerdown', onOutside))
</script>

<template>
  <div ref="root" class="relative" @keydown="onKeydown" @focusout="onFocusOut">
    <button
      type="button"
      aria-haspopup="menu"
      :aria-expanded="open"
      :aria-controls="open ? menuId : undefined"
      :aria-label="label"
      class="inline-flex size-11 items-center justify-center rounded-card text-ink-muted hover:bg-canvas hover:text-ink active:bg-canvas"
      @click="open ? close() : show()"
    >
      <svg aria-hidden="true" viewBox="0 0 24 24" class="size-5" fill="currentColor">
        <circle cx="5" cy="12" r="1.75" />
        <circle cx="12" cy="12" r="1.75" />
        <circle cx="19" cy="12" r="1.75" />
      </svg>
    </button>
    <!-- Bayangan dipakai hanya di sini untuk memisahkan menu yang melayang dari isi di bawahnya. -->
    <div
      v-if="open"
      :id="menuId"
      role="menu"
      class="absolute right-0 z-20 mt-1 min-w-52 rounded-card border border-line bg-surface py-1 shadow-md"
    >
      <button
        v-for="item in items"
        :key="item.label"
        type="button"
        role="menuitem"
        class="flex min-h-11 w-full items-center px-3.5 text-left hover:bg-canvas focus-visible:bg-canvas active:bg-canvas"
        :class="item.danger ? 'font-medium text-urgent' : ''"
        @click="choose(item.action)"
      >
        {{ item.label }}
      </button>
    </div>
  </div>
</template>
