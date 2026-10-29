<script setup lang="ts">
import { computed, onMounted, ref, useId, watch } from 'vue'

const props = withDefaults(defineProps<{ open: boolean; title: string; variant?: 'modal' | 'panel' }>(), {
  variant: 'modal',
})
const emit = defineEmits<{ close: [] }>()

const el = ref<HTMLDialogElement | null>(null)
const titleId = useId()

// Dialog bawaan browser dipakai karena showModal sudah mengurus fokus, tombol Escape,
// dan membuat isi halaman di belakangnya tidak bisa diklik.
function sync() {
  const d = el.value
  if (!d) return
  if (props.open && !d.open) d.showModal()
  if (!props.open && d.open) d.close()
}
watch(() => props.open, sync)
onMounted(sync)

// Escape memicu event cancel; ditangkap supaya status buka-tutup tetap dikendalikan parent.
function onCancel(e: Event) {
  e.preventDefault()
  emit('close')
}

const shape = computed(() =>
  props.variant === 'panel'
    ? 'm-0 ml-auto h-dvh max-h-dvh w-full max-w-md open:flex open:flex-col'
    : 'm-auto w-[calc(100%-2rem)] max-w-md rounded-card',
)
</script>

<template>
  <dialog
    ref="el"
    :aria-labelledby="titleId"
    :class="shape"
    class="bg-surface p-0 text-ink backdrop:bg-ink/45"
    @cancel="onCancel"
  >
    <div class="flex items-center justify-between gap-4 border-b border-line px-5 py-3">
      <h2 :id="titleId" class="text-lg font-semibold">{{ title }}</h2>
      <button
        type="button"
        aria-label="Tutup"
        class="inline-flex size-11 shrink-0 items-center justify-center rounded-card text-ink-muted hover:bg-canvas hover:text-ink active:bg-canvas"
        @click="emit('close')"
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
    <div class="min-h-0 flex-1 overflow-y-auto px-5 py-4">
      <slot />
    </div>
    <div v-if="$slots.footer" class="flex justify-end gap-2 border-t border-line px-5 py-3">
      <slot name="footer" />
    </div>
  </dialog>
</template>
