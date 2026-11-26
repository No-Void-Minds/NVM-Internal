<script setup lang="ts">
import { useId } from 'vue'
import { useDrafts } from './drafts'
import BaseDialog from '../BaseDialog.vue'
import WbPicker from './WbPicker.vue'
import type { Connector, ConnectorGeometry, ConnectorStyle } from '../../utils/whiteboard'
import { END_KINDS, LINE_PRESETS, portsFor } from '../../utils/wbShapes'
import { penColors } from './wbColors'

// Panel gaya garis (W5): tiga label, tujuh jenis ujung per sisi, tiga gaya garis, warna, jalur, port per ujung, dan
// hapus. Dibuka dari dobel klik garis di kanvas atau dari bar pilihan kartu (jalur keyboard), jadi semua garis bisa
// diubah tanpa penunjuk.
export type ConnectorUpdate = {
  style?: ConnectorStyle
  label?: string
  label_start?: string
  label_end?: string
  geometry?: ConnectorGeometry
  z?: number
  z_auto?: boolean
}
const props = defineProps<{
  open: boolean
  connectors: Connector[]
  nameOf: (id: string) => string
  isShape: (id: string) => boolean
  canWrite: boolean
}>()
const emit = defineEmits<{
  close: []
  update: [id: string, patch: ConnectorUpdate]
  remove: [id: string]
  restack: [id: string, kind: 'front' | 'forward' | 'backward' | 'back' | 'auto']
}>()
const uid = useId()
const labels = useDrafts(() => Object.fromEntries(props.connectors.map((c) => [c.id, c.label])))
const starts = useDrafts(() => Object.fromEntries(props.connectors.map((c) => [c.id, c.label_start])))
const ends = useDrafts(() => Object.fromEntries(props.connectors.map((c) => [c.id, c.label_end])))

const portLabel: Record<string, string> = {
  auto: 'Otomatis (sisi terdekat)',
  top: 'Atas',
  right: 'Kanan',
  bottom: 'Bawah',
  left: 'Kiri',
  top_left: 'Sudut kiri atas',
  top_right: 'Sudut kanan atas',
  bottom_left: 'Sudut kiri bawah',
  bottom_right: 'Sudut kanan bawah',
}

function setStyle(c: Connector, key: keyof ConnectorStyle, v: string) {
  emit('update', c.id, { style: { ...c.style, [key]: v } })
}
function setLabel(c: Connector, key: 'label' | 'label_start' | 'label_end', e: Event) {
  const v = (e.target as HTMLInputElement).value.trim()
  if (v !== c[key]) emit('update', c.id, { [key]: v })
}
function setRoute(c: Connector, v: string) {
  const route = v as ConnectorGeometry['route']
  emit('update', c.id, { geometry: { ...c.geometry, route } })
}
function setPort(c: Connector, side: 'from' | 'to', port: string) {
  emit('update', c.id, { geometry: { ...c.geometry, [side]: { ...c.geometry[side], port } } })
}
// Preset garis per jenis diagram (LINE_PRESETS di utils/wbShapes.ts): gaya garis dan ujung, plus label tengah
// bila presetnya punya (include, extend); label tetap bisa diubah sesudahnya.
const presetOpts = LINE_PRESETS.map((p) => ({ value: p.id, label: `${p.group}: ${p.label}` }))
function preset(c: Connector, id: string) {
  const p = LINE_PRESETS.find((x) => x.id === id)
  if (!p) return
  emit('update', c.id, {
    style: { ...c.style, line: p.line, start: p.start, end: p.end },
    ...(p.text !== undefined ? { label: p.text } : {}),
  })
}
const ORDER = [
  { kind: 'front', label: 'Paling depan' },
  { kind: 'forward', label: 'Maju' },
  { kind: 'backward', label: 'Mundur' },
  { kind: 'back', label: 'Paling belakang' },
] as const

function clearBends(c: Connector) {
  emit('update', c.id, { geometry: { ...c.geometry, points: [] } })
}

const field = 'min-h-11 w-full rounded-card border border-line bg-surface px-2'
const LINES = [
  { value: 'solid', label: 'Utuh' },
  { value: 'dashed', label: 'Putus-putus' },
  { value: 'dotted', label: 'Titik-titik' },
]
const ROUTES = [
  { value: 'straight', label: 'Lurus' },
  { value: 'orthogonal', label: 'Siku' },
  { value: 'curved', label: 'Melengkung' },
  { value: 'bebas', label: 'Bebas (sudut apa pun)' },
]
const endOpts = END_KINDS.map((k) => ({ value: k.id, label: k.label }))
const colorOpts = penColors.map((p) => ({
  value: p.value === 'tinta' ? '' : p.value,
  label: p.label,
  swatch: p.css,
}))
const portOpts = (shape: boolean) =>
  ['auto', ...portsFor(shape)].map((p) => ({ value: p, label: portLabel[p] ?? p }))
const cell = 'flex min-w-0 flex-col gap-1 text-sm'
const btn =
  'inline-flex min-h-11 items-center justify-center rounded-card border border-line bg-surface px-3 font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
</script>

<template>
  <BaseDialog :open="open" title="Garis penghubung" @close="emit('close')">
    <div class="flex max-h-[70vh] flex-col gap-4 overflow-y-auto px-5 py-4">
      <p v-if="connectors.length === 0" class="text-ink-muted">Kartu ini belum punya garis.</p>
      <section
        v-for="(c, i) in connectors"
        :key="c.id"
        class="flex flex-col gap-2 border-b border-line pb-4 last:border-b-0"
        :aria-labelledby="`${uid}-${i}`"
      >
        <h3 :id="`${uid}-${i}`" class="font-semibold break-words">
          Dari {{ c.from ? nameOf(c.from) : 'titik bebas' }} ke {{ c.to ? nameOf(c.to) : 'titik bebas' }}
        </h3>
        <template v-if="canWrite">
          <div class="grid grid-cols-[repeat(auto-fit,minmax(9rem,1fr))] gap-2">
            <div :class="cell">
              <label :for="`${uid}-${i}-label`">Label tengah</label>
              <input
                :id="`${uid}-${i}-label`"
                v-model="labels[c.id]"
                :class="field"
                maxlength="200"
                @change="setLabel(c, 'label', $event)"
              />
            </div>
            <div :class="cell">
              <label :for="`${uid}-${i}-ls`">Label dekat awal</label>
              <input
                :id="`${uid}-${i}-ls`"
                v-model="starts[c.id]"
                :class="field"
                maxlength="200"
                @change="setLabel(c, 'label_start', $event)"
              />
            </div>
            <div :class="cell">
              <label :for="`${uid}-${i}-le`">Label dekat akhir</label>
              <input
                :id="`${uid}-${i}-le`"
                v-model="ends[c.id]"
                :class="field"
                maxlength="200"
                @change="setLabel(c, 'label_end', $event)"
              />
            </div>
          </div>
          <div class="grid grid-cols-[repeat(auto-fit,minmax(9rem,1fr))] gap-2">
            <div :class="cell">
              <label :for="`${uid}-${i}-start`">Ujung awal</label>
              <WbPicker
                :id="`${uid}-${i}-start`"
                label="Ujung awal"
                class="w-full"
                :options="endOpts"
                :model-value="c.style.start"
                :disabled="!canWrite"
                @update:model-value="setStyle(c, 'start', $event)"
              />
            </div>
            <div :class="cell">
              <label :for="`${uid}-${i}-end`">Ujung akhir</label>
              <WbPicker
                :id="`${uid}-${i}-end`"
                label="Ujung akhir"
                class="w-full"
                :options="endOpts"
                :model-value="c.style.end"
                :disabled="!canWrite"
                @update:model-value="setStyle(c, 'end', $event)"
              />
            </div>
            <div :class="cell">
              <label :for="`${uid}-${i}-line`">Garis</label>
              <WbPicker
                :id="`${uid}-${i}-line`"
                label="Garis"
                class="w-full"
                :options="LINES"
                :model-value="c.style.line"
                :disabled="!canWrite"
                @update:model-value="setStyle(c, 'line', $event)"
              />
            </div>
            <div :class="cell">
              <label :for="`${uid}-${i}-color`">Warna</label>
              <WbPicker
                :id="`${uid}-${i}-color`"
                label="Warna"
                class="w-full"
                :options="colorOpts"
                :model-value="c.style.color"
                :disabled="!canWrite"
                @update:model-value="setStyle(c, 'color', $event)"
              />
            </div>
            <div :class="cell">
              <label :for="`${uid}-${i}-route`">Jalur</label>
              <WbPicker
                :id="`${uid}-${i}-route`"
                label="Jalur"
                class="w-full"
                :options="ROUTES"
                :model-value="c.geometry.route || 'straight'"
                :disabled="!canWrite"
                @update:model-value="setRoute(c, $event)"
              />
            </div>
            <div v-if="c.from" :class="cell">
              <label :for="`${uid}-${i}-pf`">Menempel di awal</label>
              <WbPicker
                :id="`${uid}-${i}-pf`"
                label="Menempel di awal"
                class="w-full"
                :options="portOpts(isShape(c.from))"
                :model-value="c.geometry.from.port || 'auto'"
                :disabled="!canWrite"
                @update:model-value="setPort(c, 'from', $event)"
              />
            </div>
            <div v-if="c.to" :class="cell">
              <label :for="`${uid}-${i}-pt`">Menempel di akhir</label>
              <WbPicker
                :id="`${uid}-${i}-pt`"
                label="Menempel di akhir"
                class="w-full"
                :options="portOpts(isShape(c.to))"
                :model-value="c.geometry.to.port || 'auto'"
                :disabled="!canWrite"
                @update:model-value="setPort(c, 'to', $event)"
              />
            </div>
          </div>
          <div :class="cell">
            <label :for="`${uid}-${i}-preset`">Preset garis</label>
            <WbPicker
              :id="`${uid}-${i}-preset`"
              label="Preset garis"
              class="w-full"
              :options="presetOpts"
              :model-value="null"
              @update:model-value="preset(c, $event)"
            />
          </div>
          <div class="flex flex-wrap items-center gap-2" role="group" :aria-label="`Urutan garis ${i + 1}`">
            <span class="text-sm">Urutan:</span>
            <button
              v-for="o in ORDER"
              :key="o.kind"
              type="button"
              :class="btn"
              @click="emit('restack', c.id, o.kind)"
            >
              {{ o.label }}
            </button>
            <button
              type="button"
              :class="btn"
              :aria-pressed="c.z === null ? 'true' : 'false'"
              class="aria-pressed:border-accent aria-pressed:bg-accent aria-pressed:text-on-accent"
              @click="emit('restack', c.id, 'auto')"
            >
              Otomatis
            </button>
          </div>
          <div class="flex flex-wrap gap-2">
            <button v-if="c.geometry.points?.length" type="button" :class="btn" @click="clearBends(c)">
              Hapus {{ c.geometry.points.length }} titik belok
            </button>
            <button type="button" :class="btn" class="text-urgent" @click="emit('remove', c.id)">
              Hapus garis
            </button>
          </div>
        </template>
        <p v-else-if="c.label" class="text-sm">Label: {{ c.label }}</p>
      </section>
    </div>
  </BaseDialog>
</template>
