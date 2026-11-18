<script setup lang="ts">
import { freqLabel, type Freq } from '../utils/recurrence'

// Penanda task berulang (revisi 19): ikon panah melingkar plus teks, bukan ikon saja.
defineProps<{ freq: Freq }>()
</script>

<template>
  <span class="inline-flex items-center gap-1 text-sm text-ink-muted">
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      class="size-4 shrink-0"
      fill="none"
      stroke="currentColor"
      stroke-width="1.75"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path d="M20 11a8 8 0 0 0-14.3-4.9L4 8" />
      <path d="M4 4v4h4" />
      <path d="M4 13a8 8 0 0 0 14.3 4.9L20 16" />
      <path d="M20 20v-4h-4" />
    </svg>
    <span><span class="sr-only">Task berulang </span>{{ freqLabel[freq] }}</span>
  </span>
</template>
