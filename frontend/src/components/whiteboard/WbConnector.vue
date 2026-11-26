<script setup lang="ts">
import type { Routed } from '../../utils/wbShapes'

// Satu garis penghubung dalam koordinat dunia, sebagai svg sendiri dengan z-index peringkat tumpuknya (revisi 53),
// jadi garis bisa berada di antara kartu: di atas batas sistem yang dikirim ke belakang, di bawah kartu yang
// sengaja ditaruh di depannya. Jalur, ujung, dan label dari routeConnector (utils/wbShapes.ts), sama dengan pegangan
// diagram dan ekspor. Halaman memakai ulang objek garis yang tidak berubah, jadi menyeret kartu hanya me-render ulang
// garis yang menempel padanya. Ujung digambar sebagai path sendiri (bukan marker) supaya warnanya selalu sama dengan
// garisnya. Klik memilih garis, dobel klik membuka panel gaya; jalur keyboard ada di menu kartu (Garis).
export type DrawnLine = Routed & {
  color: string
  dash?: string
  label: string
  label_start: string
  label_end: string
  layer: number
}
defineProps<{ line: DrawnLine; selected: boolean; zoom: number }>()
const emit = defineEmits<{ select: [id: string]; open: [id: string] }>()
</script>

<template>
  <svg
    aria-hidden="true"
    class="pointer-events-none absolute top-0 left-0 overflow-visible"
    width="1"
    height="1"
    :style="{ zIndex: line.layer }"
    data-testid="wb-connector"
    :data-connector="line.id"
    :data-selected="selected ? 'true' : undefined"
  >
    <g :style="{ color: line.color }">
      <path
        :d="line.d"
        fill="none"
        stroke="currentColor"
        :stroke-width="selected ? 3 : 2"
        :stroke-dasharray="line.dash"
        stroke-linejoin="round"
      />
      <path
        v-for="(h, i) in line.heads"
        :key="i"
        :d="h.d"
        :fill="h.fill === 'line' ? 'currentColor' : h.fill === 'surface' ? 'var(--color-surface)' : 'none'"
        stroke="currentColor"
        stroke-width="2"
        stroke-linejoin="round"
      />
      <text
        v-if="line.label"
        :x="line.mid.x"
        :y="line.mid.y"
        text-anchor="middle"
        dominant-baseline="middle"
        class="wb-connector-label text-sm"
      >
        {{ line.label }}
      </text>
      <text
        v-if="line.label_start"
        :x="line.near.start.x"
        :y="line.near.start.y - 12"
        text-anchor="middle"
        dominant-baseline="middle"
        class="wb-connector-label text-xs"
        data-label="start"
      >
        {{ line.label_start }}
      </text>
      <text
        v-if="line.label_end"
        :x="line.near.end.x"
        :y="line.near.end.y - 12"
        text-anchor="middle"
        dominant-baseline="middle"
        class="wb-connector-label text-xs"
        data-label="end"
      >
        {{ line.label_end }}
      </text>
      <!-- Area klik 16 px di layar di sepanjang garis. -->
      <path
        :d="line.d"
        fill="none"
        stroke="transparent"
        :stroke-width="16 / zoom"
        pointer-events="stroke"
        class="cursor-pointer"
        data-hit
        @pointerdown.stop="emit('select', line.id)"
        @dblclick.stop="emit('open', line.id)"
      />
    </g>
  </svg>
</template>
