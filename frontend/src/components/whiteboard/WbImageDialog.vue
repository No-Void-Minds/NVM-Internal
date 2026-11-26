<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import { downloadUrl, fileUrl } from '../../api/whiteboard'
import type { Card, CaptionContent, Crop } from '../../utils/whiteboard'
import { clampCrop, cropBox, frameCrop, frameOf } from '../../utils/wbCrop'

// Gambar kartu (revisi 50): lihat ukuran penuh, potong, dan atur posisi dan zoom di dalam bingkai. Potongan disimpan
// sebagai parameter di isi kartu (berkas asli tidak diubah). Semua kontrol juga bisa lewat keyboard: isian persen
// untuk potongan, slider untuk zoom dan posisi.
const props = defineProps<{ mode: 'view' | 'crop' | 'frame' | null; card: Card | null }>()
const emit = defineEmits<{ close: []; save: [crop: Crop | null] }>()

const uid = useId()
const file = computed(() => props.card?.file ?? null)
const caption = computed(() => (props.card?.content as CaptionContent | undefined)?.caption ?? '')
const iw = computed(() => file.value?.width || 1)
const ih = computed(() => file.value?.height || 1)
const fa = computed(() =>
  props.card ? props.card.w / Math.max(1, props.card.h - (caption.value ? 48 : 0)) : 1,
)

// Potong: persegi panjang bebas dalam persen.
const crop = ref<Crop>({ x: 0, y: 0, w: 1, h: 1 })
// Posisi dan zoom.
const zoom = ref(1)
const px = ref(0.5)
const py = ref(0.5)
watch(
  () => [props.mode, props.card?.id] as const,
  () => {
    const cur = (props.card?.content as CaptionContent | undefined)?.crop ?? null
    crop.value = cur ? { ...cur } : { x: 0, y: 0, w: 1, h: 1 }
    const f = frameOf(iw.value, ih.value, fa.value, cur)
    zoom.value = f.zoom
    px.value = f.px
    py.value = f.py
  },
  { immediate: true },
)

const pct = (v: number) => Math.round(v * 1000) / 10
function setPct(key: keyof Crop, e: Event) {
  const v = Number((e.target as HTMLInputElement).value)
  if (!Number.isFinite(v)) return
  const c = { ...crop.value, [key]: v / 100 }
  // Kiri dan atas yang diisi tetap berlaku: lebar atau tingginya menyusut supaya potongan tetap di dalam gambar.
  if (key === 'x') c.w = Math.min(c.w, 1 - c.x)
  if (key === 'y') c.h = Math.min(c.h, 1 - c.y)
  crop.value = clampCrop(c)
}

// Seret persegi potongan (geser) atau pegangan kanan bawahnya (ubah ukuran) di atas gambar utuh.
const stage = ref<HTMLElement | null>(null)
let drag: { mode: 'move' | 'size'; sx: number; sy: number; start: Crop } | null = null
function onDown(mode: 'move' | 'size', e: PointerEvent) {
  ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  drag = { mode, sx: e.clientX, sy: e.clientY, start: { ...crop.value } }
}
function onDrag(e: PointerEvent) {
  if (!drag || !stage.value) return
  const r = stage.value.getBoundingClientRect()
  const dx = (e.clientX - drag.sx) / r.width
  const dy = (e.clientY - drag.sy) / r.height
  const s = drag.start
  crop.value = clampCrop(
    drag.mode === 'move' ? { ...s, x: s.x + dx, y: s.y + dy } : { ...s, w: s.w + dx, h: s.h + dy },
  )
}
function onUp() {
  drag = null
}

const framed = computed(() => frameCrop(iw.value, ih.value, fa.value, zoom.value, px.value, py.value))
const frameStyle = computed(() => ({ aspectRatio: String(fa.value) }))
const imgBox = computed(() => cropBox(framed.value))
const rectStyle = computed(() => ({
  left: `${crop.value.x * 100}%`,
  top: `${crop.value.y * 100}%`,
  width: `${crop.value.w * 100}%`,
  height: `${crop.value.h * 100}%`,
}))

const title = computed(() =>
  props.mode === 'view'
    ? 'Gambar ukuran penuh'
    : props.mode === 'crop'
      ? 'Potong gambar'
      : 'Atur posisi dan zoom',
)
const btn = 'min-h-11 rounded-card border border-line bg-surface px-3 font-medium hover:bg-canvas'
const primary = 'min-h-11 rounded-card bg-accent px-3 font-medium text-on-accent hover:bg-accent-hover'
const lbl = 'text-sm font-semibold'
</script>

<template>
  <BaseDialog :open="mode !== null && !!file" :title="title" wide @close="emit('close')">
    <div v-if="file" class="flex flex-col gap-4 px-5 py-4">
      <template v-if="mode === 'view'">
        <img
          :src="fileUrl(file.id)"
          :alt="caption || file.name"
          class="mx-auto block max-h-[70dvh] max-w-full rounded-card bg-canvas object-contain"
        />
        <div class="flex flex-wrap gap-2">
          <a :href="downloadUrl(file.id)" :class="btn" class="inline-flex items-center" download>
            Unduh berkas asli ({{ file.name }})
          </a>
        </div>
      </template>

      <template v-else-if="mode === 'crop'">
        <p class="text-sm text-ink-muted">
          Seret kotak untuk menggeser, pegangan kanan bawah untuk mengubah ukuran, atau isi persennya di
          bawah. Berkas asli tidak diubah.
        </p>
        <div
          ref="stage"
          class="relative mx-auto max-h-[55dvh] w-full max-w-2xl overflow-hidden select-none"
          @pointermove="onDrag"
          @pointerup="onUp"
        >
          <img :src="fileUrl(file.id)" :alt="caption || file.name" draggable="false" class="block w-full" />
          <div
            class="absolute cursor-move border-2 border-dashed border-mark bg-surface/10 outline outline-[100vmax] outline-ink/50"
            :style="rectStyle"
            data-testid="wb-crop-rect"
            @pointerdown.prevent="onDown('move', $event)"
          >
            <span
              class="absolute right-0 bottom-0 size-11 cursor-nwse-resize"
              aria-hidden="true"
              @pointerdown.prevent.stop="onDown('size', $event)"
            >
              <span class="absolute right-1 bottom-1 size-3 rounded-full border-2 border-surface bg-mark" />
            </span>
          </div>
        </div>
        <fieldset class="grid grid-cols-[repeat(auto-fit,minmax(7rem,1fr))] gap-3">
          <legend class="sr-only">Potongan dalam persen</legend>
          <label v-for="k in ['x', 'y', 'w', 'h'] as const" :key="k" class="flex flex-col gap-1">
            <span :class="lbl">{{ { x: 'Kiri', y: 'Atas', w: 'Lebar', h: 'Tinggi' }[k] }} (%)</span>
            <input
              :id="`${uid}-${k}`"
              type="number"
              min="0"
              max="100"
              step="1"
              inputmode="decimal"
              class="min-h-11 rounded-card border border-line bg-surface px-2"
              :value="pct(crop[k])"
              @change="setPct(k, $event)"
            />
          </label>
        </fieldset>
      </template>

      <template v-else>
        <p class="text-sm text-ink-muted">Bingkai mengikuti ukuran kartu. Berkas asli tidak diubah.</p>
        <div
          class="relative mx-auto w-full max-w-md overflow-hidden rounded-card border border-line bg-canvas"
          :style="frameStyle"
        >
          <img
            :src="fileUrl(file.id)"
            :alt="caption || file.name"
            draggable="false"
            class="absolute block max-w-none"
            :style="imgBox"
          />
        </div>
        <div class="grid grid-cols-[repeat(auto-fit,minmax(10rem,1fr))] gap-3">
          <label class="flex flex-col gap-1">
            <span :class="lbl">Zoom {{ Math.round(zoom * 100) }}%</span>
            <input
              v-model.number="zoom"
              type="range"
              min="1"
              max="5"
              step="0.05"
              class="min-h-11 accent-[var(--color-mark)]"
              :aria-valuetext="`${Math.round(zoom * 100)} persen`"
            />
          </label>
          <label class="flex flex-col gap-1">
            <span :class="lbl">Posisi mendatar</span>
            <input
              v-model.number="px"
              type="range"
              min="0"
              max="1"
              step="0.01"
              class="min-h-11 accent-[var(--color-mark)]"
              :aria-valuetext="`${Math.round(px * 100)} persen dari kiri`"
            />
          </label>
          <label class="flex flex-col gap-1">
            <span :class="lbl">Posisi tegak</span>
            <input
              v-model.number="py"
              type="range"
              min="0"
              max="1"
              step="0.01"
              class="min-h-11 accent-[var(--color-mark)]"
              :aria-valuetext="`${Math.round(py * 100)} persen dari atas`"
            />
          </label>
        </div>
      </template>

      <div v-if="mode !== 'view'" class="flex flex-wrap justify-end gap-2">
        <button type="button" :class="btn" @click="emit('save', null)">Tampilkan utuh</button>
        <button type="button" :class="btn" @click="emit('close')">Batal</button>
        <button type="button" :class="primary" @click="emit('save', mode === 'crop' ? crop : framed)">
          {{ mode === 'crop' ? 'Simpan potongan' : 'Simpan posisi dan zoom' }}
        </button>
      </div>
    </div>
  </BaseDialog>
</template>
