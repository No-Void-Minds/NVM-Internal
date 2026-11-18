<script setup lang="ts">
import type { LinkKind } from '../utils/outputLink'

// Ikon garis sederhana per jenis tautan output, sengaja bukan logo Google. Selalu dekoratif: label jenisnya
// ditulis di sebelahnya.
defineProps<{ kind: LinkKind }>()
</script>

<template>
  <svg
    aria-hidden="true"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.75"
    stroke-linecap="round"
    stroke-linejoin="round"
  >
    <template v-if="kind === 'drive-file'">
      <path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z" />
      <path d="M14 3v5h5" />
    </template>
    <path
      v-else-if="kind === 'drive-folder'"
      d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"
    />
    <template v-else-if="kind === 'docs'">
      <path d="M14 3H7a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8z" />
      <path d="M14 3v5h5M9 13h6M9 17h6" />
    </template>
    <template v-else-if="kind === 'sheets'">
      <rect x="4" y="4" width="16" height="16" rx="2" />
      <path d="M4 10h16M4 15h16M10 4v16" />
    </template>
    <template v-else-if="kind === 'slides'">
      <rect x="3" y="5" width="18" height="12" rx="2" />
      <path d="M12 17v3M8 20h8" />
    </template>
    <path v-else d="M10 13a5 5 0 0 0 7 0l3-3a5 5 0 0 0-7-7l-1 1M14 11a5 5 0 0 0-7 0l-3 3a5 5 0 0 0 7 7l1-1" />
  </svg>
</template>
