<script setup lang="ts">
import type { TimelineEvent } from '../utils/timeline'
import { initials } from '../utils/person'

// Penanda kategori berupa bentuk, bukan warna, dan selalu berdampingan dengan teks (judul, nama, atau label
// kategori): Launching kotak penuh, Produksi kotak garis, Kehadiran bulatan inisial orangnya.
defineProps<{ event: Pick<TimelineEvent, 'category' | 'user'> }>()
</script>

<template>
  <span
    v-if="event.category === 'kehadiran'"
    aria-hidden="true"
    class="inline-flex size-5 shrink-0 items-center justify-center rounded-full bg-tag text-[0.625rem] leading-none font-semibold text-tag-ink"
  >
    {{ initials(event.user?.name ?? '?') }}
  </span>
  <span v-else class="inline-flex size-5 shrink-0 items-center justify-center" aria-hidden="true">
    <span
      class="size-3 rounded-[1px]"
      :class="event.category === 'launching' ? 'bg-ink' : 'border-2 border-ink'"
    />
  </span>
</template>
