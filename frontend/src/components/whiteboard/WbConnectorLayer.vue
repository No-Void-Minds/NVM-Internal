<script setup lang="ts">
import { computed } from 'vue'
import { edgePoint, type Connector, type Rect } from '../../utils/whiteboard'
import { inkColor } from './wbColors'

// Garis penghubung di bawah kartu, dalam koordinat dunia. Ujung garis di tepi kartu; panah digambar sebagai
// segitiga sendiri (bukan marker) supaya warnanya selalu sama dengan garisnya di semua peramban. Garis bisa diklik
// untuk membuka dialog garis; jalur keyboard ke dialog yang sama ada di bar pilihan kartu.
const props = defineProps<{ connectors: Connector[]; geom: Map<string, Rect>; canWrite: boolean }>()
const emit = defineEmits<{ open: [id: string] }>()

function head(tip: { x: number; y: number }, from: { x: number; y: number }) {
  const len = Math.hypot(tip.x - from.x, tip.y - from.y) || 1
  const ux = (tip.x - from.x) / len
  const uy = (tip.y - from.y) / len
  const bx = tip.x - ux * 12
  const by = tip.y - uy * 12
  return `M${tip.x} ${tip.y}L${bx - uy * 6} ${by + ux * 6}L${bx + uy * 6} ${by - ux * 6}Z`
}

const lines = computed(() =>
  props.connectors.flatMap((c) => {
    // Ujung bebas (model W5) digambar sebagai titik di koordinat dunia.
    const end = (card: string | null, e: { x?: number; y?: number }) =>
      card
        ? props.geom.get(card)
        : e.x !== undefined && e.y !== undefined
          ? { x: e.x, y: e.y, w: 0, h: 0 }
          : undefined
    const a = end(c.from, c.geometry.from)
    const b = end(c.to, c.geometry.to)
    if (!a || !b) return []
    const ca = { x: a.x + a.w / 2, y: a.y + a.h / 2 }
    const cb = { x: b.x + b.w / 2, y: b.y + b.h / 2 }
    const p = edgePoint(a, cb)
    const q = edgePoint(b, ca)
    return [
      {
        id: c.id,
        d: `M${p.x} ${p.y}L${q.x} ${q.y}`,
        heads: [
          c.style.end === 'arrow' ? head(q, p) : '',
          c.style.start === 'arrow' ? head(p, q) : '',
        ].filter(Boolean),
        dash: c.style.line === 'dashed' ? '8 6' : c.style.line === 'dotted' ? '2 5' : undefined,
        color: inkColor(c.style.color || 'tinta'),
        label: c.label,
        mid: { x: (p.x + q.x) / 2, y: (p.y + q.y) / 2 },
      },
    ]
  }),
)
</script>

<template>
  <svg
    aria-hidden="true"
    class="pointer-events-none absolute top-0 left-0 overflow-visible"
    width="1"
    height="1"
  >
    <g v-for="l in lines" :key="l.id" :style="{ color: l.color }" data-testid="wb-connector">
      <path :d="l.d" fill="none" stroke="currentColor" stroke-width="2" :stroke-dasharray="l.dash" />
      <path v-for="(h, i) in l.heads" :key="i" :d="h" fill="currentColor" />
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
      <!-- Area klik selebar 16 px di sepanjang garis. -->
      <path
        v-if="canWrite"
        :d="l.d"
        fill="none"
        stroke="transparent"
        stroke-width="16"
        pointer-events="stroke"
        class="cursor-pointer"
        @pointerdown.stop
        @click="emit('open', l.id)"
      />
    </g>
  </svg>
</template>
