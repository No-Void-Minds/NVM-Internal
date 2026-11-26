<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import WbPicker from './WbPicker.vue'
import { penColors } from './wbColors'
import { FONT_SIZES, SHAPES, shapeInfo, type ShapeContent } from '../../utils/wbShapes'

// Panel gaya bentuk (W5). Juga jalur keyboard untuk putar dan ukuran, yang di kanvas hanya lewat pegangan.
// Setiap perubahan langsung disimpan halaman (isi lewat content_version, ukuran lewat tata letak).
const props = defineProps<{
  open: boolean
  content: ShapeContent | null
  w: number
  h: number
  canWrite: boolean
}>()
const emit = defineEmits<{ close: []; content: [c: ShapeContent]; size: [s: { w: number; h: number }] }>()

const uid = useId()
const fills = [
  { value: '', label: 'Tanpa isian' },
  { value: 'putih', label: 'Putih' },
  { value: 'abu', label: 'Abu' },
  { value: 'hijau', label: 'Hijau' },
  { value: 'kuning', label: 'Kuning' },
  { value: 'biru', label: 'Biru' },
  { value: 'krem', label: 'Krem' },
  { value: 'merah', label: 'Merah' },
]
const rot = ref(0)
const width = ref(0)
const height = ref(0)
watch(
  () => [props.open, props.content?.rotation, props.w, props.h] as const,
  () => {
    rot.value = props.content?.rotation ?? 0
    width.value = Math.round(props.w)
    height.value = Math.round(props.h)
  },
  { immediate: true },
)

function set<K extends keyof ShapeContent>(key: K, value: ShapeContent[K]) {
  if (!props.content) return
  emit('content', { ...props.content, [key]: value })
}
// Varian (revisi 54) mengikuti jenis bentuk: ganti jenis memakai varian pertama jenis baru, atau tanpa varian.
const variants = computed(() => (props.content ? (shapeInfo(props.content.shape)?.variants ?? []) : []))
function setShape(shape: ShapeContent['shape']) {
  if (!props.content) return
  const next: ShapeContent = { ...props.content, shape }
  const first = shapeInfo(shape)?.variants?.[0]?.id
  if (first) next.variant = first
  else delete next.variant
  emit('content', next)
}
function setRotation() {
  const v = Math.round(Number(rot.value))
  if (Number.isFinite(v)) set('rotation', ((v % 360) + 360) % 360)
}
function setSize() {
  const w = Math.round(Number(width.value))
  const h = Math.round(Number(height.value))
  if (w >= 40 && w <= 4000 && h >= 40 && h <= 4000) emit('size', { w, h })
}
const control = 'min-h-11 w-full rounded-card border border-line bg-surface px-2 disabled:opacity-60'
const lbl = 'text-sm font-semibold'
</script>

<template>
  <BaseDialog :open="open" title="Gaya bentuk" @close="emit('close')">
    <form v-if="content" class="grid grid-cols-2 gap-3 px-5 py-4" novalidate @submit.prevent>
      <label :for="`${uid}-shape`" :class="lbl" class="col-span-2">Jenis bentuk</label>
      <WbPicker
        :id="`${uid}-shape`"
        label="Jenis bentuk"
        class="col-span-2 w-full"
        :options="SHAPES.map((s) => ({ value: s.id, label: `${s.group}: ${s.label}` }))"
        :model-value="content.shape"
        :disabled="!canWrite"
        @update:model-value="setShape($event as ShapeContent['shape'])"
      />
      <template v-if="variants.length">
        <label :for="`${uid}-variant`" :class="lbl" class="col-span-2">Varian</label>
        <WbPicker
          :id="`${uid}-variant`"
          label="Varian"
          class="col-span-2 w-full"
          :options="variants.map((v) => ({ value: v.id, label: v.label }))"
          :model-value="content.variant || variants[0]!.id"
          :disabled="!canWrite"
          @update:model-value="set('variant', $event)"
        />
      </template>
      <div>
        <label :for="`${uid}-fill`" :class="lbl">Isian</label>
        <WbPicker
          :id="`${uid}-fill`"
          label="Isian"
          class="w-full"
          :options="fills"
          :model-value="content.fill"
          :disabled="!canWrite"
          @update:model-value="set('fill', $event)"
        />
      </div>
      <div>
        <label :for="`${uid}-stroke`" :class="lbl">Warna garis</label>
        <WbPicker
          :id="`${uid}-stroke`"
          label="Warna garis"
          class="w-full"
          :options="penColors.map((c) => ({ value: c.value, label: c.label, swatch: c.css }))"
          :model-value="content.stroke"
          :disabled="!canWrite"
          @update:model-value="set('stroke', $event)"
        />
      </div>
      <div>
        <label :for="`${uid}-sw`" :class="lbl">Tebal garis</label>
        <WbPicker
          :id="`${uid}-sw`"
          label="Tebal garis"
          class="w-full"
          :options="Array.from({ length: 8 }, (_, i) => ({ value: String(i + 1), label: `${i + 1} px` }))"
          :model-value="String(content.stroke_width)"
          :disabled="!canWrite"
          @update:model-value="set('stroke_width', Number($event))"
        />
      </div>
      <div>
        <label :for="`${uid}-ss`" :class="lbl">Gaya garis</label>
        <WbPicker
          :id="`${uid}-ss`"
          label="Gaya garis"
          class="w-full"
          :options="[
            { value: 'solid', label: 'Utuh' },
            { value: 'dashed', label: 'Putus-putus' },
          ]"
          :model-value="content.stroke_style"
          :disabled="!canWrite"
          @update:model-value="set('stroke_style', $event as 'solid' | 'dashed')"
        />
      </div>
      <div>
        <label :for="`${uid}-fs`" :class="lbl">Ukuran huruf</label>
        <WbPicker
          :id="`${uid}-fs`"
          label="Ukuran huruf"
          class="w-full"
          :options="FONT_SIZES.map((n) => ({ value: String(n), label: `${n} px` }))"
          :model-value="String(content.font_size)"
          :disabled="!canWrite"
          @update:model-value="set('font_size', Number($event))"
        />
      </div>
      <div>
        <label :for="`${uid}-al`" :class="lbl">Perataan teks</label>
        <WbPicker
          :id="`${uid}-al`"
          label="Perataan teks"
          class="w-full"
          :options="[
            { value: 'left', label: 'Kiri' },
            { value: 'center', label: 'Tengah' },
            { value: 'right', label: 'Kanan' },
          ]"
          :model-value="content.align"
          :disabled="!canWrite"
          @update:model-value="set('align', $event as ShapeContent['align'])"
        />
      </div>
      <div>
        <label :for="`${uid}-rot`" :class="lbl">Putar (derajat)</label>
        <input
          :id="`${uid}-rot`"
          v-model="rot"
          type="number"
          min="0"
          max="359"
          step="15"
          inputmode="numeric"
          :class="control"
          :disabled="!canWrite"
          @change="setRotation"
        />
      </div>
      <div />
      <div>
        <label :for="`${uid}-w`" :class="lbl">Lebar (px)</label>
        <input
          :id="`${uid}-w`"
          v-model="width"
          type="number"
          min="40"
          max="4000"
          inputmode="numeric"
          :class="control"
          :disabled="!canWrite"
          @change="setSize"
        />
      </div>
      <div>
        <label :for="`${uid}-h`" :class="lbl">Tinggi (px)</label>
        <input
          :id="`${uid}-h`"
          v-model="height"
          type="number"
          min="40"
          max="4000"
          inputmode="numeric"
          :class="control"
          :disabled="!canWrite"
          @change="setSize"
        />
      </div>
    </form>
  </BaseDialog>
</template>
