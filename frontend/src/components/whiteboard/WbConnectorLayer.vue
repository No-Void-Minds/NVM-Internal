<script setup lang="ts">
import type { Routed } from '../../utils/wbShapes'

// Garis penghubung di bawah kartu, dalam koordinat dunia. Jalurnya (lurus, siku, melengkung), ujung, dan posisi label
// dihitung routeConnector (utils/wbShapes.ts), sama dengan pegangan diagram dan ekspor. Ujung digambar sebagai path
// sendiri (bukan marker) supaya warnanya selalu sama dengan garisnya di semua peramban. Klik memilih garis (Delete
// menghapus), dobel klik membuka panel gaya; jalur keyboard ke panel yang sama ada di bar pilihan kartu.
export type DrawnLine = Routed & {
  color: string
  dash?: string
  label: string
  label_start: string
  label_end: string
}
defineProps<{ lines: DrawnLine[]; selected: string | null; zoom: number }>()
const emit = defineEmits<{ select: [id: string]; open: [id: string] }>()
</script>

<template>
  <svg
    aria-hidden="true"
    class="pointer-events-none absolute top-0 left-0 overflow-visible"
    width="1"
    height="1"
  >
    <g
      v-for="l in lines"
      :key="l.id"
      :style="{ color: l.color }"
      data-testid="wb-connector"
      :data-connector="l.id"
      :data-selected="selected === l.id ? 'true' : undefined"
    >
      <path
        :d="l.d"
        fill="none"
        stroke="currentColor"
        :stroke-width="selected === l.id ? 3 : 2"
        :stroke-dasharray="l.dash"
        stroke-linejoin="round"
      />
      <path
        v-for="(h, i) in l.heads"
        :key="i"
        :d="h.d"
        :fill="h.fill === 'line' ? 'currentColor' : h.fill === 'surface' ? 'var(--color-surface)' : 'none'"
        stroke="currentColor"
        stroke-width="2"
        stroke-linejoin="round"
      />
      <text
        v-if="l.label"
        :x="l.mid.x"
        :y="l.mid.y"
        text-anchor="middle"
        dominant-baseline="middle"
        class="wb-connector-label text-sm"
      >
        {{ l.label }}
      </text>
      <text
        v-if="l.label_start"
        :x="l.near.start.x"
        :y="l.near.start.y - 12"
        text-anchor="middle"
        dominant-baseline="middle"
        class="wb-connector-label text-xs"
        data-label="start"
      >
        {{ l.label_start }}
      </text>
      <text
        v-if="l.label_end"
        :x="l.near.end.x"
        :y="l.near.end.y - 12"
        text-anchor="middle"
        dominant-baseline="middle"
        class="wb-connector-label text-xs"
        data-label="end"
      >
        {{ l.label_end }}
      </text>
      <!-- Area klik 16 px di layar di sepanjang garis. -->
      <path
        :d="l.d"
        fill="none"
        stroke="transparent"
        :stroke-width="16 / zoom"
        pointer-events="stroke"
        class="cursor-pointer"
        @pointerdown.stop="emit('select', l.id)"
        @dblclick.stop="emit('open', l.id)"
      />
    </g>
  </svg>
</template>
