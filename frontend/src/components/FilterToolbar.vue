<script setup lang="ts">
import { ref, useId } from 'vue'

defineProps<{ label: string; activeCount: number; canClear: boolean }>()
const emit = defineEmits<{ clear: [] }>()

const open = ref(false)
const id = useId()
</script>

<template>
  <!-- Satu pola cari dan filter untuk papan, Arsip, Pengguna, dan Audit Log (Sprint 11c). Di layar kecil filter
       dilipat ke tombol Filter dengan jumlah filter aktif, supaya tidak memakan tinggi tetapi tetap terjangkau. -->
  <div class="mt-4 flex flex-wrap items-start gap-2">
    <div v-if="$slots.search" class="min-w-0 flex-1 basis-72">
      <slot name="search" />
    </div>
    <button
      type="button"
      class="inline-flex min-h-11 items-center gap-2 rounded-card border border-line bg-surface px-4 font-medium hover:bg-canvas md:hidden"
      :aria-expanded="open"
      :aria-controls="id"
      @click="open = !open"
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
        <path d="M3 5h18l-7 8v6l-4 2v-8z" />
      </svg>
      Filter<span v-if="activeCount" class="font-mono">({{ activeCount }})</span>
    </button>
    <div
      :id="id"
      role="group"
      :aria-label="label"
      class="w-full gap-2 sm:grid-cols-2 md:flex md:w-auto md:flex-wrap md:items-start"
      :class="open ? 'grid grid-cols-1' : 'hidden'"
    >
      <slot />
      <button
        v-if="canClear"
        type="button"
        class="inline-flex min-h-11 items-center justify-center rounded-card border border-line bg-surface px-4 font-medium hover:bg-canvas"
        @click="emit('clear')"
      >
        Hapus filter
      </button>
    </div>
  </div>
</template>
