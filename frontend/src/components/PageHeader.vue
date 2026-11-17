<script setup lang="ts">
defineProps<{ title?: string; description?: string }>()
</script>

<template>
  <!-- Kepala semua halaman: ukuran judul, jarak, dan posisi aksi sama di setiap rute (Sprint 11c). -->
  <div class="flex flex-wrap items-start justify-between gap-x-4 gap-y-3">
    <div class="min-w-0">
      <h1 class="text-2xl font-bold break-words">
        <slot name="title">{{ title }}</slot>
      </h1>
      <p v-if="description" class="mt-1 text-ink-muted">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="flex flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
