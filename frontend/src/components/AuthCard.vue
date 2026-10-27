<script setup lang="ts">
defineProps<{ title: string; subtitle: string }>()
</script>

<template>
  <div class="flex min-h-dvh items-center justify-center px-4 py-10">
    <main class="w-full max-w-sm">
      <p class="font-semibold">NVM Internal</p>
      <div class="mt-4 rounded-card border border-line bg-surface p-6">
        <h1 class="text-xl font-semibold">{{ title }}</h1>
        <p class="mt-1 text-sm text-ink-muted">{{ subtitle }}</p>
        <slot />
      </div>
      <p class="mt-4 text-center text-sm text-ink-muted">Lupa password? Hubungi Admin atau Manajer.</p>
    </main>
  </div>
</template>
