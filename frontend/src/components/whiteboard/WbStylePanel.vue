<script setup lang="ts">
import { computed, useId } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import { penColors } from './wbColors'
import { STYLE_OPTIONS, styleKeys, type CardStyle, type StyleKey } from '../../utils/wbStyle'

// Panel gaya kartu (revisi 50) untuk semua kartu terpilih sekaligus. Hanya bagian yang berlaku untuk semua jenis
// kartu terpilih yang ditampilkan (bentuk: bayangan dan transparansi; gambar: tanpa latar dan jarak dalam). Nilai
// yang berbeda antar kartu tampil sebagai "Campuran". Setiap perubahan langsung disimpan halaman, dengan undo.
const props = defineProps<{
  open: boolean
  cards: { kind: string; style?: CardStyle }[]
  canWrite: boolean
}>()
const emit = defineEmits<{ close: []; apply: [patch: Partial<Record<StyleKey, string | number>>] }>()

const uid = useId()
const keys = computed(() => {
  const all = new Set<StyleKey>([
    'bg',
    'border',
    'border_color',
    'border_style',
    'radius',
    'shadow',
    'opacity',
    'padding',
  ])
  for (const c of props.cards) {
    const k = new Set(styleKeys(c.kind))
    for (const x of [...all]) if (!k.has(x)) all.delete(x)
  }
  return all
})

const MIXED = '__campuran'
function valueOf(k: StyleKey): string {
  const vals = props.cards.map((c) => String(c.style?.[k] ?? (k === 'opacity' ? 100 : '')))
  return vals.length && vals.every((v) => v === vals[0]) ? (vals[0] ?? '') : MIXED
}
// Bawaan "tipis" disimpan kosong; pilihan Tipis di daftar bernilai "".
function pick(k: StyleKey, e: Event) {
  const v = (e.target as HTMLSelectElement | HTMLInputElement).value
  if (v === MIXED) return
  emit('apply', { [k]: k === 'opacity' ? Number(v) : v })
}

const borderColors = computed(() => [
  { value: '', label: 'Garis bawaan' },
  ...penColors.map((p) => ({ value: p.value, label: p.label })),
])
const opacity = computed(() => (valueOf('opacity') === MIXED ? 100 : Number(valueOf('opacity'))))
const groups: { key: StyleKey; label: string; options: readonly { value: string; label: string }[] }[] = [
  { key: 'bg', label: 'Latar', options: STYLE_OPTIONS.bg },
  { key: 'border', label: 'Garis tepi', options: STYLE_OPTIONS.border },
  { key: 'border_style', label: 'Gaya garis tepi', options: STYLE_OPTIONS.border_style },
  { key: 'radius', label: 'Sudut', options: STYLE_OPTIONS.radius },
  { key: 'shadow', label: 'Bayangan', options: STYLE_OPTIONS.shadow },
  { key: 'padding', label: 'Jarak dalam', options: STYLE_OPTIONS.padding },
]
// Bentuk tanpa bayangan bawaan (bayangan mengikuti garis bentuk), jadi pilihannya Tanpa dan Tegas saja.
const allShapes = computed(() => props.cards.length > 0 && props.cards.every((c) => c.kind === 'shape'))
function optionsFor(g: (typeof groups)[number]) {
  return g.key === 'shadow' && allShapes.value
    ? [
        { value: '', label: 'Tanpa' },
        { value: 'tegas', label: 'Tegas' },
      ]
    : g.options
}
const control = 'min-h-11 w-full rounded-card border border-line bg-surface px-2 disabled:opacity-60'
const lbl = 'text-sm font-semibold'
</script>

<template>
  <BaseDialog :open="open" title="Gaya kartu" @close="emit('close')">
    <form
      class="grid grid-cols-[repeat(auto-fit,minmax(10rem,1fr))] gap-3 px-5 py-4"
      novalidate
      @submit.prevent
    >
      <p class="col-span-full text-sm text-ink-muted">
        Berlaku untuk {{ cards.length }} kartu terpilih. Teks di kartu tanpa latar tetap terbaca di atas
        kanvas terang dan gelap.
      </p>
      <template v-for="g in groups" :key="g.key">
        <div v-if="keys.has(g.key)" class="flex flex-col gap-1">
          <label :for="`${uid}-${g.key}`" :class="lbl">{{ g.label }}</label>
          <select
            :id="`${uid}-${g.key}`"
            :class="control"
            :value="valueOf(g.key)"
            :disabled="!canWrite"
            @change="pick(g.key, $event)"
          >
            <option v-if="valueOf(g.key) === MIXED" :value="MIXED" disabled>Campuran</option>
            <option v-for="o in optionsFor(g)" :key="o.value" :value="o.value">{{ o.label }}</option>
          </select>
        </div>
        <div v-if="g.key === 'border' && keys.has('border_color')" class="flex flex-col gap-1">
          <label :for="`${uid}-border_color`" :class="lbl">Warna garis tepi</label>
          <select
            :id="`${uid}-border_color`"
            :class="control"
            :value="valueOf('border_color')"
            :disabled="!canWrite"
            @change="pick('border_color', $event)"
          >
            <option v-if="valueOf('border_color') === MIXED" :value="MIXED" disabled>Campuran</option>
            <option v-for="o in borderColors" :key="o.value" :value="o.value">{{ o.label }}</option>
          </select>
        </div>
      </template>
      <div v-if="keys.has('opacity')" class="col-span-full flex flex-col gap-1">
        <label :for="`${uid}-opacity`" :class="lbl">
          Transparansi: {{ valueOf('opacity') === MIXED ? 'campuran' : `${opacity}%` }}
        </label>
        <input
          :id="`${uid}-opacity`"
          type="range"
          min="25"
          max="100"
          step="5"
          class="min-h-11 w-full accent-[var(--color-mark)]"
          :value="opacity"
          :aria-valuetext="`${opacity} persen`"
          :disabled="!canWrite"
          @change="pick('opacity', $event)"
        />
      </div>
    </form>
  </BaseDialog>
</template>
