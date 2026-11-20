<script setup lang="ts">
import { computed, useId } from 'vue'
import type { ActivityCell } from '../api/activities'

// Tombol rantai sambung jam (revisi 33). Boleh tidaknya datang dari server (link_hint), frontend hanya
// menampilkannya. Saat tidak bisa dipakai, tombol tetap bisa difokus (aria-disabled, bukan disabled), supaya
// pengguna keyboard dan pembaca layar tetap mendengar alasannya.
const props = defineProps<{ cell: ActivityCell; busy?: boolean }>()
const emit = defineEmits<{ toggle: [] }>()

const id = useId()
const blocked = computed(() => !props.cell.linked && props.cell.link_hint !== '')

function click() {
  if (!blocked.value && !props.busy) emit('toggle')
}
</script>

<template>
  <button
    type="button"
    class="inline-flex size-11 shrink-0 items-center justify-center rounded-card border focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
    :class="[
      cell.linked ? 'border-accent bg-accent text-on-accent hover:bg-accent-hover' : 'border-line bg-surface',
      blocked ? 'cursor-not-allowed text-ink-muted' : cell.linked ? '' : 'text-ink hover:bg-canvas',
      busy ? 'opacity-60' : '',
    ]"
    :aria-pressed="cell.linked"
    :aria-disabled="blocked || busy ? 'true' : undefined"
    :aria-label="cell.linked ? 'Lepas sambungan' : 'Sambungkan ke jam sebelumnya'"
    :aria-describedby="blocked ? id : undefined"
    :title="blocked ? cell.link_hint : undefined"
    @click="click"
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
      <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
      <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
    </svg>
    <!-- Di dalam tombol supaya komponen ini tetap satu elemen akar dan class dari pemanggil sampai ke tombol. -->
    <span v-if="blocked" :id="id" class="sr-only">{{ cell.link_hint }}</span>
  </button>
</template>
