<script setup lang="ts">
// Kerangka memuat yang mengikuti bentuk konten, supaya halaman tidak melompat saat data datang.
// Sengaja diam (tanpa pulse): DESIGN.md dan Sprint 11 melarang animasi berulang tanpa pemicu.
withDefaults(defineProps<{ label: string; shape?: 'board' | 'rows' | 'detail'; rows?: number }>(), {
  shape: 'rows',
  rows: 5,
})
</script>

<template>
  <div role="status">
    <span class="sr-only">{{ label }}</span>
    <div v-if="shape === 'board'" aria-hidden="true" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <div v-for="c in 4" :key="c" class="rounded-card border border-line bg-canvas p-2">
        <div class="m-1 h-6 w-28 rounded-chip bg-tag" />
        <div v-for="k in 2" :key="k" class="mt-2 rounded-card border border-line bg-surface p-4">
          <div class="h-5 w-16 rounded-chip bg-tag" />
          <div class="mt-3 h-4 w-4/5 rounded-chip bg-tag" />
          <div class="mt-2 h-4 w-1/2 rounded-chip bg-tag" />
        </div>
      </div>
    </div>
    <div v-else-if="shape === 'detail'" aria-hidden="true" class="space-y-3">
      <div class="h-6 w-24 rounded-chip bg-tag" />
      <div class="h-7 w-3/4 rounded-chip bg-tag" />
      <div v-for="k in 4" :key="k" class="h-4 rounded-chip bg-tag" :class="k % 2 ? 'w-full' : 'w-2/3'" />
    </div>
    <ul v-else aria-hidden="true" class="divide-y divide-line rounded-card border border-line bg-surface">
      <li v-for="k in rows" :key="k" class="flex min-h-13 items-center gap-4 px-4">
        <div class="h-4 w-1/4 rounded-chip bg-tag" />
        <div class="h-4 flex-1 rounded-chip bg-tag" />
        <div class="hidden h-4 w-1/6 rounded-chip bg-tag sm:block" />
      </li>
    </ul>
  </div>
</template>
