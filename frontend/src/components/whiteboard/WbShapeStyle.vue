<script setup lang="ts">
import { ref, useId, watch } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import { penColors } from './wbColors'
import { FONT_SIZES, SHAPES, type ShapeContent } from '../../utils/wbShapes'

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
      <select
        :id="`${uid}-shape`"
        :class="control"
        class="col-span-2"
        :value="content.shape"
        :disabled="!canWrite"
        @change="set('shape', ($event.target as HTMLSelectElement).value as ShapeContent['shape'])"
      >
        <option v-for="s in SHAPES" :key="s.id" :value="s.id">{{ s.group }}: {{ s.label }}</option>
      </select>
      <div>
        <label :for="`${uid}-fill`" :class="lbl">Isian</label>
        <select
          :id="`${uid}-fill`"
          :class="control"
          :value="content.fill"
          :disabled="!canWrite"
          @change="set('fill', ($event.target as HTMLSelectElement).value)"
        >
          <option v-for="f in fills" :key="f.value" :value="f.value">{{ f.label }}</option>
        </select>
      </div>
      <div>
        <label :for="`${uid}-stroke`" :class="lbl">Warna garis</label>
        <select
          :id="`${uid}-stroke`"
          :class="control"
          :value="content.stroke"
          :disabled="!canWrite"
          @change="set('stroke', ($event.target as HTMLSelectElement).value)"
        >
          <option v-for="c in penColors" :key="c.value" :value="c.value">{{ c.label }}</option>
        </select>
      </div>
      <div>
        <label :for="`${uid}-sw`" :class="lbl">Tebal garis</label>
        <select
          :id="`${uid}-sw`"
          :class="control"
          :value="content.stroke_width"
          :disabled="!canWrite"
          @change="set('stroke_width', Number(($event.target as HTMLSelectElement).value))"
        >
          <option v-for="n in 8" :key="n" :value="n">{{ n }} px</option>
        </select>
      </div>
      <div>
        <label :for="`${uid}-ss`" :class="lbl">Gaya garis</label>
        <select
          :id="`${uid}-ss`"
          :class="control"
          :value="content.stroke_style"
          :disabled="!canWrite"
          @change="set('stroke_style', ($event.target as HTMLSelectElement).value as 'solid' | 'dashed')"
        >
          <option value="solid">Utuh</option>
          <option value="dashed">Putus-putus</option>
        </select>
      </div>
      <div>
        <label :for="`${uid}-fs`" :class="lbl">Ukuran huruf</label>
        <select
          :id="`${uid}-fs`"
          :class="control"
          :value="content.font_size"
          :disabled="!canWrite"
          @change="set('font_size', Number(($event.target as HTMLSelectElement).value))"
        >
          <option v-for="n in FONT_SIZES" :key="n" :value="n">{{ n }} px</option>
        </select>
      </div>
      <div>
        <label :for="`${uid}-al`" :class="lbl">Perataan teks</label>
        <select
          :id="`${uid}-al`"
          :class="control"
          :value="content.align"
          :disabled="!canWrite"
          @change="set('align', ($event.target as HTMLSelectElement).value as ShapeContent['align'])"
        >
          <option value="left">Kiri</option>
          <option value="center">Tengah</option>
          <option value="right">Kanan</option>
        </select>
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
