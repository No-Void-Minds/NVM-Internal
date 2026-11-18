<script setup lang="ts">
import PersonChip from './PersonChip.vue'

// Baris "Sub task: <nama>" di Arsip: semua PIC sub task task itu (revisi 22, sebelumnya hanya yang cocok
// dengan filter PIC). Kartu papan memakai AvatarGroup.
defineProps<{ people: { id: number; name: string }[] }>()
</script>

<template>
  <p v-if="people.length" class="text-sm text-ink-muted">
    Sub task:
    <template v-for="(p, i) in people" :key="p.id"
      ><PersonChip :user-id="p.id" :name="p.name">{{ p.name }}</PersonChip
      >{{ i < people.length - 1 ? ', ' : '' }}</template
    >
  </p>
</template>
