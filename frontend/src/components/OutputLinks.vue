<script setup lang="ts">
import { computed } from 'vue'
import LinkKindIcon from './LinkKindIcon.vue'
import { outputParts } from '../utils/outputLink'

// Output Aktivitas sebagai teks dan tombol tautan (revisi 35), dipakai tabel /aktivitas, dialog detail jam, dan
// kartu Aktivitas di papan, termasuk jam lanjutan. Teks tetap teks biasa (tanpa v-html), dan href hanya http
// atau https dari outputParts. layout list (dialog) menaruh setiap tautan di barisnya sendiri dengan URL
// lengkap di bawahnya, karena di layar sentuh title tidak pernah terlihat.
const props = withDefaults(defineProps<{ output: string; layout?: 'inline' | 'list' }>(), {
  layout: 'inline',
})

const parts = computed(() => outputParts(props.output))

const chip =
  'inline-flex min-h-11 max-w-full items-center gap-1 rounded-chip border border-line px-2 align-middle hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
</script>

<template>
  <div v-if="layout === 'list'" class="grid min-w-0 gap-2">
    <template v-for="(p, i) in parts" :key="i">
      <p v-if="p.type === 'text'" class="min-w-0 whitespace-pre-line [overflow-wrap:anywhere]">
        {{ p.text.trim() }}
      </p>
      <div v-else class="grid min-w-0 justify-items-start gap-1">
        <a
          :href="p.href"
          :title="p.url"
          :aria-label="`${p.label}, buka di tab baru`"
          target="_blank"
          rel="noopener noreferrer"
          :class="chip"
        >
          <LinkKindIcon :kind="p.kind" class="size-5 shrink-0" />
          <span class="truncate font-medium">{{ p.label }}</span>
        </a>
        <span class="max-w-full text-sm text-ink-muted [overflow-wrap:anywhere]">{{ p.url }}</span>
      </div>
    </template>
  </div>
  <p v-else class="min-w-0 whitespace-pre-line [overflow-wrap:anywhere]">
    <template v-for="(p, i) in parts" :key="i">
      <template v-if="p.type === 'text'">{{ p.text }}</template>
      <a
        v-else
        :href="p.href"
        :title="p.url"
        :aria-label="`${p.label}, buka di tab baru`"
        target="_blank"
        rel="noopener noreferrer"
        :class="[chip, 'relative z-10 my-0.5 mr-1']"
      >
        <LinkKindIcon :kind="p.kind" class="size-4 shrink-0" />
        <span class="truncate">{{ p.label }}</span>
      </a>
    </template>
  </p>
</template>
