<script setup lang="ts">
import { computed, ref } from 'vue'
import { cardLabel, type Card, type DrawingContent, type Stroke } from '../../utils/whiteboard'
import { inkColor } from './wbColors'

// Objek gambar bebas (revisi A): goresan tanpa bingkai dan tanpa latar, di atas kartu. Elemennya tidak menangkap
// penunjuk (klik tembus ke kartu di bawahnya); hanya garisnya, dengan area klik selebar tebal pena plus 8 px, yang
// memilih objek. Objek tetap bisa difokus lewat keyboard (panah antar kartu) dan dihapus dengan Delete.
const props = defineProps<{
  card: Card
  selected: boolean
  focusable: boolean
  zoom: number
  dx: number
  dy: number
  // Mode pena: garis tidak menangkap penunjuk, supaya pena bisa mencoret di atas gambar lain.
  pen: boolean
  remote?: { color: number; label: string; strong: boolean } | null
}>()
const emit = defineEmits<{ grab: [e: PointerEvent] }>()

const root = ref<HTMLElement | null>(null)
defineExpose({ el: root })

const strokes = computed(() => (props.card.content as DrawingContent).strokes)
const style = computed(() => ({
  transform: `translate(${props.card.x + props.dx}px, ${props.card.y + props.dy}px)`,
  width: `${props.card.w}px`,
  height: `${props.card.h}px`,
  '--wb-ring': `${2 / props.zoom}px`,
  '--wb-remote': props.remote ? `var(--color-collab-${props.remote.color})` : undefined,
}))

function path(s: Stroke) {
  const p = s.points
  let d = ''
  for (let i = 0; i + 1 < p.length; i += 2) d += `${i ? 'L' : 'M'}${p[i]} ${p[i + 1]}`
  // Titik tunggal tetap terlihat sebagai noktah.
  return p.length === 2 ? `${d}h0.1` : d
}
</script>

<template>
  <div
    ref="root"
    :data-card="card.id"
    :tabindex="focusable ? 0 : -1"
    :aria-label="cardLabel(card) + (selected ? ', terpilih' : '') + (remote ? `, ${remote.label}` : '')"
    aria-roledescription="gambar bebas"
    class="wb-card wb-ink pointer-events-none absolute top-0 left-0 z-[1000000001] rounded-card"
    :class="[selected ? 'wb-ink-selected' : '', !selected && remote ? 'wb-card-remote' : '']"
    :style="style"
  >
    <svg :width="card.w" :height="card.h" class="overflow-visible" aria-hidden="true">
      <g v-for="(s, i) in strokes" :key="i">
        <path
          :d="path(s)"
          fill="none"
          :stroke-width="s.width"
          stroke-linecap="round"
          stroke-linejoin="round"
          :style="{ stroke: inkColor(s.color) }"
        />
        <path
          v-if="!pen"
          :d="path(s)"
          fill="none"
          stroke="transparent"
          :stroke-width="s.width + 8 / zoom"
          stroke-linecap="round"
          pointer-events="stroke"
          class="cursor-pointer"
          @pointerdown="emit('grab', $event)"
        />
      </g>
    </svg>
    <span
      v-if="remote"
      aria-hidden="true"
      class="wb-remote-label pointer-events-none absolute -top-7 left-0 rounded-chip px-1.5 py-0.5 text-xs font-medium whitespace-nowrap"
    >
      {{ remote.label }}
    </span>
  </div>
</template>
