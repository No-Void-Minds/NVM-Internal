<script setup lang="ts">
import { computed } from 'vue'
import type { Port, Pt } from '../../utils/wbShapes'

// Pegangan diagram (W5) di atas kartu: port kartu terpilih untuk menarik garis baru, titik belok garis terpilih
// (seret untuk memindah, dobel klik untuk membuang), titik sisip di tengah tiap bagian, dan garis sementara selama
// menarik. Area sentuh 44 px di layar pada zoom berapa pun; gambarnya kecil supaya tidak menutupi bentuk.
const props = defineProps<{
  zoom: number
  // p: titik sambung di tepi kartu; h: letak pegangan di luar tepi.
  ports: { card: string; port: Port; p: Pt; h: Pt }[]
  stops: Pt[] | null
  temp: { a: Pt; b: Pt } | null
  // Pesan sequence: satu pegangan tinggi pesan, tanpa titik sisip.
  noInsert?: boolean
}>()
const emit = defineEmits<{
  port: [card: string, port: Port, e: PointerEvent]
  bend: [index: number, e: PointerEvent]
  insert: [index: number, e: PointerEvent]
  unbend: [index: number]
}>()

const hit = computed(() => 22 / props.zoom)
const dot = computed(() => 5 / props.zoom)
const ring = computed(() => 2 / props.zoom)
// Titik belok adalah stops tanpa ujung awal dan akhir; titik sisip di tengah setiap pasangan stops berurutan.
const bends = computed(() => (props.stops ?? []).slice(1, -1))
const inserts = computed(() => {
  if (props.noInsert) return []
  const s = props.stops ?? []
  return s.slice(1).map((p, i) => ({ x: (p.x + s[i]!.x) / 2, y: (p.y + s[i]!.y) / 2 }))
})
</script>

<template>
  <svg
    aria-hidden="true"
    class="pointer-events-none absolute top-0 left-0 z-[1000000003] overflow-visible"
    width="1"
    height="1"
    data-testid="wb-diagram-handles"
  >
    <line
      v-if="temp"
      :x1="temp.a.x"
      :y1="temp.a.y"
      :x2="temp.b.x"
      :y2="temp.b.y"
      class="stroke-mark"
      :stroke-width="2 / zoom"
      :stroke-dasharray="`${6 / zoom} ${4 / zoom}`"
    />
    <g v-for="pt in ports" :key="pt.card + pt.port" :data-port="pt.port">
      <line
        :x1="pt.p.x"
        :y1="pt.p.y"
        :x2="pt.h.x"
        :y2="pt.h.y"
        class="stroke-mark"
        :stroke-width="ring / 2"
      />
      <circle :cx="pt.h.x" :cy="pt.h.y" :r="dot" class="fill-surface stroke-mark" :stroke-width="ring" />
      <circle
        :cx="pt.h.x"
        :cy="pt.h.y"
        :r="hit"
        fill="transparent"
        pointer-events="all"
        class="cursor-crosshair"
        @pointerdown.stop.prevent="emit('port', pt.card, pt.port, $event)"
      />
    </g>
    <g v-for="(p, i) in bends" :key="'b' + i" data-bend>
      <rect
        :x="p.x - dot"
        :y="p.y - dot"
        :width="dot * 2"
        :height="dot * 2"
        class="fill-mark stroke-surface"
        :stroke-width="ring"
      />
      <circle
        :cx="p.x"
        :cy="p.y"
        :r="hit"
        fill="transparent"
        pointer-events="all"
        class="cursor-move"
        @pointerdown.stop.prevent="emit('bend', i, $event)"
        @dblclick.stop="emit('unbend', i)"
      />
    </g>
    <g v-for="(p, i) in inserts" :key="'i' + i" data-insert>
      <circle :cx="p.x" :cy="p.y" :r="dot * 0.8" class="fill-surface stroke-mark" :stroke-width="ring" />
      <circle
        :cx="p.x"
        :cy="p.y"
        :r="hit"
        fill="transparent"
        pointer-events="all"
        class="cursor-copy"
        @pointerdown.stop.prevent="emit('insert', i, $event)"
      />
    </g>
  </svg>
</template>
