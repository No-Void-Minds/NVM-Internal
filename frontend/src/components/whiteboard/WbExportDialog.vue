<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import { canvasBlob, renderBoard } from './exportBoard'
import { exportName, pdfFromJpeg } from '../../utils/wbExport'
import { formatBytes } from '../../utils/attachment'
import type { Card, Connector } from '../../utils/whiteboard'

// Ekspor papan atau pilihan ke PNG atau PDF (W4). Semua dikerjakan di browser; tidak ada yang dikirim ke server.
const props = defineProps<{
  open: boolean
  title: string
  cards: Card[]
  connectors: Connector[]
  selected: Set<string>
}>()
const emit = defineEmits<{ close: [] }>()

const uid = useId()
const format = ref<'png' | 'pdf'>('png')
const scope = ref<'all' | 'selection'>('all')
const busy = ref(false)
const status = ref('')
const error = ref('')

watch(
  () => props.open,
  (open) => {
    if (!open) return
    scope.value = props.selected.size ? 'selection' : 'all'
    status.value = ''
    error.value = ''
  },
)

// Kolom terpilih membawa isinya.
const chosen = computed(() => {
  if (scope.value === 'all') return props.cards
  const ids = props.selected
  return props.cards.filter((c) => ids.has(c.id) || (c.column_id && ids.has(c.column_id)))
})

function download(blob: Blob, name: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  document.body.append(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 10_000)
}

async function run() {
  if (busy.value) return
  error.value = ''
  status.value = ''
  if (chosen.value.length === 0) {
    error.value = 'Tidak ada kartu untuk diekspor.'
    return
  }
  busy.value = true
  try {
    const ids = new Set(chosen.value.map((c) => c.id))
    const lines = props.connectors.filter((l) => (!l.from || ids.has(l.from)) && (!l.to || ids.has(l.to)))
    const res = await renderBoard(chosen.value, lines)
    if (!res) throw new Error('kanvas')
    let blob: Blob | null
    if (format.value === 'png') blob = await canvasBlob(res.canvas, 'image/png')
    else {
      const jpeg = await canvasBlob(res.canvas, 'image/jpeg', 0.92)
      blob = jpeg
        ? new Blob(
            [pdfFromJpeg(new Uint8Array(await jpeg.arrayBuffer()), res.canvas.width, res.canvas.height)],
            {
              type: 'application/pdf',
            },
          )
        : null
    }
    if (!blob) throw new Error('blob')
    download(blob, exportName(props.title, format.value))
    status.value =
      `${format.value.toUpperCase()} ${formatBytes(blob.size)} diunduh (${res.canvas.width} x ${res.canvas.height} piksel).` +
      (res.reduced
        ? ` Papan ini besar, jadi diekspor dengan skala ${Math.round(res.scale * 100)}% supaya tetap bisa dibuka di semua peramban.`
        : '')
  } catch {
    error.value = 'Ekspor gagal. Coba pilih lebih sedikit kartu, atau ulangi.'
  } finally {
    busy.value = false
  }
}

const btn =
  'inline-flex min-h-11 items-center justify-center rounded-card border border-line bg-surface px-3 font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-60'
const option = 'flex min-h-11 cursor-pointer items-center gap-2 rounded-card px-2 hover:bg-canvas'
</script>

<template>
  <BaseDialog :open="open" title="Ekspor papan" @close="emit('close')">
    <form class="flex flex-col gap-4 px-5 py-4" novalidate @submit.prevent="run">
      <fieldset>
        <legend class="font-semibold">Format</legend>
        <label :class="option">
          <input
            v-model="format"
            type="radio"
            :name="`${uid}-f`"
            value="png"
            class="size-5 accent-[var(--color-mark)]"
          />
          PNG, gambar untuk dibagikan
        </label>
        <label :class="option">
          <input
            v-model="format"
            type="radio"
            :name="`${uid}-f`"
            value="pdf"
            class="size-5 accent-[var(--color-mark)]"
          />
          PDF satu halaman, untuk dicetak
        </label>
      </fieldset>
      <fieldset>
        <legend class="font-semibold">Isi</legend>
        <label :class="option">
          <input
            v-model="scope"
            type="radio"
            :name="`${uid}-s`"
            value="all"
            class="size-5 accent-[var(--color-mark)]"
          />
          Seluruh papan ({{ cards.length }} kartu)
        </label>
        <label :class="[option, selected.size ? '' : 'cursor-not-allowed text-ink-muted']">
          <input
            v-model="scope"
            type="radio"
            :name="`${uid}-s`"
            value="selection"
            :disabled="selected.size === 0"
            class="size-5 accent-[var(--color-mark)]"
          />
          Kartu terpilih ({{ selected.size }})
        </label>
      </fieldset>
      <p class="text-sm text-ink-muted">
        Teks diekspor tanpa format huruf, gambar memakai pratinjaunya, dan warna memakai tema terang.
      </p>
      <p v-if="status" role="status" class="text-sm" data-testid="wb-export-status">{{ status }}</p>
      <p v-if="error" role="alert" class="text-sm text-danger">{{ error }}</p>
      <div class="flex flex-wrap justify-end gap-2">
        <button type="button" :class="btn" @click="emit('close')">Selesai</button>
        <button type="submit" :class="btn" :disabled="busy">
          {{ busy ? 'Menyiapkan...' : `Unduh ${format.toUpperCase()}` }}
        </button>
      </div>
    </form>
  </BaseDialog>
</template>
