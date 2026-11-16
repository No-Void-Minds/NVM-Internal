<script setup lang="ts">
import { urgencyClass, urgencyLabel, type Urgency } from '../utils/task'

defineProps<{ urgency: Urgency }>()
</script>

<template>
  <span class="inline-flex items-center gap-1.5" :class="urgencyClass[urgency]">
    <!-- Penanda bentuk untuk High (persegi) dan Urgent (belah ketupat); katanya tetap tertulis, jadi tidak bergantung pada warna. -->
    <span
      v-if="urgency === 'high' || urgency === 'urgent'"
      aria-hidden="true"
      class="size-2 shrink-0 bg-current"
      :class="urgency === 'urgent' ? 'rotate-45' : 'rounded-[1px]'"
    />
    {{ urgencyLabel[urgency] }}
  </span>
</template>
