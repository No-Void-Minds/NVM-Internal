<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import WbPicker from './WbPicker.vue'
import { penColors } from './wbColors'
import { FONT_SIZES, SHAPES, laneOrient, shapeInfo, type ShapeContent } from '../../utils/wbShapes'
import type { Lane } from '../../utils/wbContainers'

// Panel gaya bentuk (W5). Juga jalur keyboard untuk putar dan ukuran, yang di kanvas hanya lewat pegangan.
// Setiap perubahan langsung disimpan halaman (isi lewat content_version, ukuran lewat tata letak).
const props = defineProps<{
  open: boolean
  content: ShapeContent | null
  w: number
  h: number
  canWrite: boolean
}>()
const emit = defineEmits<{
  close: []
  content: [c: ShapeContent]
  size: [s: { w: number; h: number }]
  addLane: []
}>()

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
// Lane pool dan swimlane (revisi 54b): label dan ukuran (bobot, px saat ukuran bentuk sekarang) per lane.
const isTable = computed(() => props.content?.shape === 'erd_table')
const hasLanes = computed(() => !!props.content && !!laneOrient(props.content.shape))
function setLane(i: number, patch: Partial<Lane>) {
  if (!props.content?.lanes) return
  const lanes = props.content.lanes.map((l, j) => (j === i ? { ...l, ...patch } : l))
  emit('content', { ...props.content, lanes })
}
function setLaneLabel(i: number, e: Event) {
  const v = (e.target as HTMLInputElement).value.trim()
  if (v !== props.content?.lanes?.[i]?.label) setLane(i, { label: v })
}
function setLaneSize(i: number, e: Event) {
  const v = Math.round(Number((e.target as HTMLInputElement).value))
  if (Number.isFinite(v) && v >= 20 && v <= 4000) setLane(i, { size: v })
}
function removeLane(i: number) {
  if (!props.content?.lanes || props.content.lanes.length < 2) return
  emit('content', { ...props.content, lanes: props.content.lanes.filter((_, j) => j !== i) })
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
        :options="
          SHAPES.filter((s) => isTable || s.id !== 'erd_table').map((s) => ({
            value: s.id,
            label: `${s.group}: ${s.label}`,
          }))
        "
        :model-value="content.shape"
        :disabled="!canWrite || isTable"
        @update:model-value="setShape($event as ShapeContent['shape'])"
      />
      <!-- Revisi 57a: isi tabel (nama dan kolom) tidak punya padanan di bentuk lain, jadi jenisnya tetap. -->
      <p v-if="isTable" class="col-span-2 text-sm text-ink-muted">
        Jenis tabel tidak bisa diganti. Tinggi tabel mengikuti jumlah kolom; kolom diedit dengan dobel klik
        tabel.
      </p>
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
        <!-- Revisi 54f: enum menyembunyikan bagian metode; isinya tidak dibuang. -->
        <p
          v-if="content.shape === 'uml_class' && content.variant === 'enum' && (content.methods ?? '').trim()"
          class="col-span-2 text-sm text-ink-muted"
          role="status"
          data-testid="wb-enum-note"
        >
          Bagian metode disembunyikan untuk enum. Isinya tetap tersimpan dan muncul lagi kalau varian diganti
          kembali.
        </p>
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
      <div v-if="!isTable">
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
      <div v-if="!isTable">
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
          :disabled="!canWrite || isTable"
          @change="setSize"
        />
      </div>
      <fieldset v-if="hasLanes && content.lanes" class="col-span-2 flex min-w-0 flex-col gap-2">
        <legend :class="lbl">Lane</legend>
        <div
          v-for="(l, i) in content.lanes"
          :key="i"
          class="grid grid-cols-[minmax(0,1fr)_6rem_auto] items-end gap-2"
          data-testid="wb-lane-row"
        >
          <div class="flex min-w-0 flex-col gap-1">
            <label :for="`${uid}-lane-${i}`" class="text-sm">Label lane {{ i + 1 }}</label>
            <input
              :id="`${uid}-lane-${i}`"
              :value="l.label"
              maxlength="200"
              :class="control"
              :disabled="!canWrite"
              @change="setLaneLabel(i, $event)"
            />
          </div>
          <div class="flex flex-col gap-1">
            <label :for="`${uid}-lane-size-${i}`" class="text-sm">Ukuran</label>
            <input
              :id="`${uid}-lane-size-${i}`"
              :value="l.size"
              type="number"
              min="20"
              max="4000"
              inputmode="numeric"
              :class="control"
              :disabled="!canWrite"
              @change="setLaneSize(i, $event)"
            />
          </div>
          <button
            type="button"
            class="min-h-11 rounded-card border border-line bg-surface px-3 text-urgent hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-60"
            :disabled="!canWrite || content.lanes.length < 2"
            :aria-label="`Hapus lane ${i + 1}`"
            @click="removeLane(i)"
          >
            Hapus
          </button>
        </div>
        <button
          type="button"
          class="min-h-11 self-start rounded-card border border-line bg-surface px-3 font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-60"
          :disabled="!canWrite || content.lanes.length >= 20"
          @click="emit('addLane')"
        >
          Tambah lane
        </button>
      </fieldset>
    </form>
  </BaseDialog>
</template>
