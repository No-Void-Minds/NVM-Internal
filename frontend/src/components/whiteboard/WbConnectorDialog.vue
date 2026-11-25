<script setup lang="ts">
import { useId } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import type { Connector, ConnectorStyle } from '../../utils/whiteboard'
import { penColors } from './wbColors'

// Dialog garis penghubung: label, ujung, jenis garis, warna, dan hapus. Dibuka dari garis di kanvas atau dari bar
// pilihan kartu (jalur keyboard), jadi semua garis bisa diubah tanpa penunjuk.
defineProps<{ open: boolean; connectors: Connector[]; nameOf: (id: string) => string; canWrite: boolean }>()
const emit = defineEmits<{
  close: []
  update: [id: string, patch: { style?: ConnectorStyle; label?: string }]
  remove: [id: string]
}>()
const uid = useId()

function setStyle(c: Connector, key: keyof ConnectorStyle, e: Event) {
  emit('update', c.id, { style: { ...c.style, [key]: (e.target as HTMLSelectElement).value } })
}

function setLabel(c: Connector, e: Event) {
  const label = (e.target as HTMLInputElement).value.trim()
  if (label !== c.label) emit('update', c.id, { label })
}

const field = 'min-h-11 w-full rounded-card border border-line bg-surface px-2'
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
          <label :for="`${uid}-${i}-label`" class="text-sm">Label</label>
          <input
            :id="`${uid}-${i}-label`"
            :class="field"
            :value="c.label"
            maxlength="200"
            @change="setLabel(c, $event)"
          />
          <div class="grid grid-cols-[repeat(auto-fit,minmax(8rem,1fr))] gap-2">
            <div class="flex min-w-0 flex-col gap-1 text-sm">
              <label :for="`${uid}-${i}-start`">Ujung awal</label>
              <select
                :id="`${uid}-${i}-start`"
                :class="field"
                :value="c.style.start"
                @change="setStyle(c, 'start', $event)"
              >
                <option value="none">Polos</option>
                <option value="arrow">Panah</option>
              </select>
            </div>
            <div class="flex min-w-0 flex-col gap-1 text-sm">
              <label :for="`${uid}-${i}-end`">Ujung akhir</label>
              <select
                :id="`${uid}-${i}-end`"
                :class="field"
                :value="c.style.end"
                @change="setStyle(c, 'end', $event)"
              >
                <option value="none">Polos</option>
                <option value="arrow">Panah</option>
              </select>
            </div>
            <div class="flex min-w-0 flex-col gap-1 text-sm">
              <label :for="`${uid}-${i}-line`">Garis</label>
              <select
                :id="`${uid}-${i}-line`"
                :class="field"
                :value="c.style.line"
                @change="setStyle(c, 'line', $event)"
              >
                <option value="solid">Utuh</option>
                <option value="dashed">Putus-putus</option>
              </select>
            </div>
            <div class="flex min-w-0 flex-col gap-1 text-sm">
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
          </div>
          <button type="button" :class="btn" class="self-start text-urgent" @click="emit('remove', c.id)">
            Hapus garis
          </button>
        </template>
        <p v-else-if="c.label" class="text-sm">Label: {{ c.label }}</p>
      </section>
    </div>
  </BaseDialog>
</template>
