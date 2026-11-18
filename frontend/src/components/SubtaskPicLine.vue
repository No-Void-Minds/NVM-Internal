<script setup lang="ts">
import PersonChip from './PersonChip.vue'

// Revisi 15: menjelaskan kenapa task masuk hasil filter PIC padahal orangnya bukan PIC task.
// overlay: di kartu papan pemicu harus di atas lapisan klik judul.
defineProps<{ people: { id: number; name: string }[]; overlay?: boolean }>()
</script>

<template>
  <p v-if="people.length" class="text-sm text-ink-muted">
    Sub task:
    <template v-for="(p, i) in people" :key="p.id"
      ><PersonChip :user-id="p.id" :name="p.name" :class="overlay ? 'z-10' : ''">{{ p.name }}</PersonChip
      >{{ i < people.length - 1 ? ', ' : '' }}</template
    >
  </p>
</template>
