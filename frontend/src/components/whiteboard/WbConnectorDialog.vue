<script setup lang="ts">
import { useId } from 'vue'
import { useDrafts } from './drafts'
import BaseDialog from '../BaseDialog.vue'
import type { Connector, ConnectorGeometry, ConnectorStyle } from '../../utils/whiteboard'
import { END_KINDS, portsFor } from '../../utils/wbShapes'
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

function setStyle(c: Connector, key: keyof ConnectorStyle, e: Event) {
  emit('update', c.id, { style: { ...c.style, [key]: (e.target as HTMLSelectElement).value } })
}
function setLabel(c: Connector, key: 'label' | 'label_start' | 'label_end', e: Event) {
  const v = (e.target as HTMLInputElement).value.trim()
  if (v !== c[key]) emit('update', c.id, { [key]: v })
}
function setRoute(c: Connector, e: Event) {
  const route = (e.target as HTMLSelectElement).value as ConnectorGeometry['route']
  emit('update', c.id, { geometry: { ...c.geometry, route } })
}
function setPort(c: Connector, side: 'from' | 'to', e: Event) {
  const port = (e.target as HTMLSelectElement).value
  emit('update', c.id, { geometry: { ...c.geometry, [side]: { ...c.geometry[side], port } } })
}
function clearBends(c: Connector) {
  emit('update', c.id, { geometry: { ...c.geometry, points: [] } })
}

const field = 'min-h-11 w-full rounded-card border border-line bg-surface px-2'
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
              <select
                :id="`${uid}-${i}-start`"
                :class="field"
                :value="c.style.start"
                @change="setStyle(c, 'start', $event)"
              >
                <option v-for="k in END_KINDS" :key="k.id" :value="k.id">{{ k.label }}</option>
              </select>
            </div>
            <div :class="cell">
              <label :for="`${uid}-${i}-end`">Ujung akhir</label>
              <select
                :id="`${uid}-${i}-end`"
                :class="field"
                :value="c.style.end"
                @change="setStyle(c, 'end', $event)"
              >
                <option v-for="k in END_KINDS" :key="k.id" :value="k.id">{{ k.label }}</option>
              </select>
            </div>
            <div :class="cell">
              <label :for="`${uid}-${i}-line`">Garis</label>
              <select
                :id="`${uid}-${i}-line`"
                :class="field"
                :value="c.style.line"
                @change="setStyle(c, 'line', $event)"
              >
                <option value="solid">Utuh</option>
                <option value="dashed">Putus-putus</option>
                <option value="dotted">Titik-titik</option>
              </select>
            </div>
            <div :class="cell">
              <label :for="`${uid}-${i}-color`">Warna</label>
              <select
                :id="`${uid}-${i}-color`"
                :class="field"
                :value="c.style.color"
                @change="setStyle(c, 'color', $event)"
              >
                <option v-for="p in penColors" :key="p.value" :value="p.value === 'tinta' ? '' : p.value">
                  {{ p.label }}
                </option>
              </select>
            </div>
            <div :class="cell">
              <label :for="`${uid}-${i}-route`">Jalur</label>
              <select
                :id="`${uid}-${i}-route`"
                :class="field"
                :value="c.geometry.route || 'straight'"
                @change="setRoute(c, $event)"
              >
                <option value="straight">Lurus</option>
                <option value="orthogonal">Siku</option>
                <option value="curved">Melengkung</option>
              </select>
            </div>
            <div v-if="c.from" :class="cell">
              <label :for="`${uid}-${i}-pf`">Menempel di awal</label>
              <select
                :id="`${uid}-${i}-pf`"
                :class="field"
                :value="c.geometry.from.port || 'auto'"
                @change="setPort(c, 'from', $event)"
              >
                <option v-for="p in ['auto', ...portsFor(isShape(c.from))]" :key="p" :value="p">
                  {{ portLabel[p] }}
                </option>
              </select>
            </div>
            <div v-if="c.to" :class="cell">
              <label :for="`${uid}-${i}-pt`">Menempel di akhir</label>
              <select
                :id="`${uid}-${i}-pt`"
                :class="field"
                :value="c.geometry.to.port || 'auto'"
                @change="setPort(c, 'to', $event)"
              >
                <option v-for="p in ['auto', ...portsFor(isShape(c.to))]" :key="p" :value="p">
                  {{ portLabel[p] }}
                </option>
              </select>
            </div>
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
