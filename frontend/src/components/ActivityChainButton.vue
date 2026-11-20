<script setup lang="ts">
import type { ActivityCell } from '../api/activities'

// Tombol rantai sambung jam (revisi 33, tampilan revisi 33b). Duduk di celah antara dua isian Aktivitas yang
// disambungnya: ikon kecil tanpa bingkai, kotak tombolnya sendiri yang 44x44 px. Pemanggil hanya merender
// tombol ini kalau jamnya memang bisa disambung atau sudah tersambung, jadi di sini tidak ada keadaan mati.
const props = defineProps<{ cell: ActivityCell; busy?: boolean }>()
const emit = defineEmits<{ toggle: [] }>()

function click() {
  if (!props.busy) emit('toggle')
}
</script>

<template>
  <button
    type="button"
    class="inline-flex size-11 shrink-0 items-center justify-center rounded-card focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
    :class="[
      cell.linked ? 'text-accent hover:text-accent-hover' : 'text-ink-muted hover:text-ink',
      busy ? 'opacity-60' : '',
    ]"
    :aria-pressed="cell.linked"
    :title="cell.linked ? 'Lepas sambungan' : 'Sambungkan ke jam sebelumnya'"
    :aria-label="
      cell.linked ? `Lepas sambungan jam ${cell.time}` : `Sambungkan jam ${cell.time} ke jam sebelumnya`
    "
    @click="click"
  >
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      class="size-4"
      fill="none"
      stroke="currentColor"
      :stroke-width="cell.linked ? 2.75 : 2"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
      <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
    </svg>
  </button>
</template>
