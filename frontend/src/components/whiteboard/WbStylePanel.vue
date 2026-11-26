<script setup lang="ts">
import { computed, useId } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import WbPicker from './WbPicker.vue'
import { penColors } from './wbColors'
import { STYLE_OPTIONS, styleKeys, type CardStyle, type StyleKey } from '../../utils/wbStyle'
import { PERSON_LAYOUTS, PERSON_SIZES, PERSON_VIEWS, type PersonContent } from '../../utils/wbPerson'

// Panel gaya kartu (revisi 50) untuk semua kartu terpilih sekaligus. Hanya bagian yang berlaku untuk semua jenis
// kartu terpilih yang ditampilkan (bentuk: bayangan dan transparansi; gambar: tanpa latar dan jarak dalam). Nilai
// yang berbeda antar kartu tampil sebagai "Campuran". Setiap perubahan langsung disimpan halaman, dengan undo.
const props = defineProps<{
  open: boolean
  cards: { kind: string; style?: CardStyle; content?: unknown }[]
  canWrite: boolean
}>()
const emit = defineEmits<{
  close: []
  apply: [patch: Partial<Record<StyleKey, string | number>>]
  person: [patch: Partial<Omit<PersonContent, 'user_id'>>]
}>()

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
function pickValue(k: StyleKey, v: string) {
  emit('apply', { [k]: v })
}
function pick(k: StyleKey, e: Event) {
  const v = (e.target as HTMLSelectElement | HTMLInputElement).value
  if (v === MIXED) return
  emit('apply', { [k]: k === 'opacity' ? Number(v) : v })
}

const borderColors = computed(() => [
  { value: '', label: 'Garis bawaan', swatch: 'var(--color-line)' },
  ...penColors.map((p) => ({ value: p.value, label: p.label, swatch: p.css })),
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
// Tampilan kartu orang (revisi 51), untuk semua kartu orang terpilih sekaligus.
const persons = computed(() =>
  props.cards.filter((c) => c.kind === 'person').map((c) => c.content as PersonContent),
)
const personGroups = [
  { key: 'view' as const, label: 'Tampilan kartu orang', options: PERSON_VIEWS },
  { key: 'size' as const, label: 'Ukuran kartu orang', options: PERSON_SIZES },
  { key: 'layout' as const, label: 'Susunan foto', options: PERSON_LAYOUTS },
]
const personOn = (k: 'view' | 'size' | 'layout', v: string) =>
  persons.value.length > 0 && persons.value.every((p) => p[k] === v)
const chip =
  'inline-flex min-h-11 items-center rounded-card border border-line px-3 text-sm hover:bg-canvas aria-pressed:bg-accent aria-pressed:text-on-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-60'
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
          <WbPicker
            :id="`${uid}-${g.key}`"
            :label="g.label"
            class="w-full"
            :options="optionsFor(g)"
            :model-value="valueOf(g.key) === MIXED ? null : valueOf(g.key)"
            :mixed="valueOf(g.key) === MIXED"
            :disabled="!canWrite"
            @update:model-value="pickValue(g.key, $event)"
          />
        </div>
        <div v-if="g.key === 'border' && keys.has('border_color')" class="flex flex-col gap-1">
          <label :for="`${uid}-border_color`" :class="lbl">Warna garis tepi</label>
          <WbPicker
            :id="`${uid}-border_color`"
            label="Warna garis tepi"
            class="w-full"
            :options="borderColors"
            :model-value="valueOf('border_color') === MIXED ? null : valueOf('border_color')"
            :mixed="valueOf('border_color') === MIXED"
            :disabled="!canWrite"
            @update:model-value="pickValue('border_color', $event)"
          />
        </div>
      </template>
      <div
        v-for="g in persons.length ? personGroups : []"
        :key="g.key"
        role="group"
        :aria-label="g.label"
        class="col-span-full flex flex-col gap-1"
      >
        <span :class="lbl">{{ g.label }}</span>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="o in g.options"
            :key="o.value"
            type="button"
            :class="chip"
            :aria-pressed="personOn(g.key, o.value) ? 'true' : 'false'"
            :disabled="!canWrite"
            @click="emit('person', { [g.key]: o.value })"
          >
            {{ o.label }}
          </button>
        </div>
      </div>
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
