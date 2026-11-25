<script setup lang="ts">
import { PEN_WIDTHS } from '../../utils/whiteboard'
import { penColors } from './wbColors'

// Toolbar pena melayang di tepi kanvas selama mode pena aktif (revisi A).
defineProps<{ color: string; width: number; eraser: boolean; canUndo: boolean }>()
const emit = defineEmits<{
  color: [value: string]
  width: [value: number]
  eraser: [on: boolean]
  undo: []
  done: []
}>()

const btn =
  'inline-flex size-11 shrink-0 items-center justify-center rounded-card hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark aria-pressed:bg-accent aria-pressed:text-on-accent disabled:opacity-50'
const textBtn =
  'inline-flex min-h-11 shrink-0 items-center justify-center rounded-card px-3 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark aria-pressed:bg-accent aria-pressed:text-on-accent disabled:opacity-50'
</script>

<template>
  <div
    role="toolbar"
    aria-label="Alat pena"
    class="absolute top-2 left-1/2 z-30 flex max-w-[calc(100%-1rem)] -translate-x-1/2 flex-wrap items-center justify-center gap-0.5 rounded-card border border-line bg-surface p-1 shadow-menu"
    data-testid="wb-penbar"
  >
    <button
      v-for="c in penColors"
      :key="c.value"
      type="button"
      :class="btn"
      :aria-pressed="!eraser && color === c.value ? 'true' : 'false'"
      :aria-label="`Pena ${c.label}`"
      :title="`Pena ${c.label}`"
      @click="(emit('color', c.value), emit('eraser', false))"
    >
      <span
        aria-hidden="true"
        class="size-5 rounded-full border border-line"
        :style="{ background: c.css }"
      />
    </button>
    <span aria-hidden="true" class="mx-0.5 h-6 w-px bg-line" />
    <button
      v-for="w in PEN_WIDTHS"
      :key="w"
      type="button"
      :class="btn"
      :aria-pressed="width === w ? 'true' : 'false'"
      :aria-label="`Ketebalan ${w}`"
      :title="`Ketebalan ${w}`"
      @click="emit('width', w)"
    >
      <span aria-hidden="true" class="w-5 rounded-full bg-current" :style="{ height: `${w}px` }" />
    </button>
    <span aria-hidden="true" class="mx-0.5 h-6 w-px bg-line" />
    <button
      type="button"
      :class="textBtn"
      :aria-pressed="eraser ? 'true' : 'false'"
      @click="emit('eraser', !eraser)"
    >
      Penghapus
    </button>
    <button type="button" :class="textBtn" :disabled="!canUndo" @click="emit('undo')">
      Hapus goresan terakhir
    </button>
    <button type="button" :class="textBtn" class="border border-line" @click="emit('done')">Selesai</button>
  </div>
</template>
