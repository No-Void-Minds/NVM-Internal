<script setup lang="ts">
import { nextTick, onMounted, watch } from 'vue'
import { saveLabel } from '../utils/activity'

// Satu isian di tabel Aktivitas hari ini (revisi 21, tata letak diputar): satu textarea per sel, dengan
// status simpan di bawah sel Output. Nama isian datang dari aria-label, bukan label terlihat, karena kepala
// baris dan kolom tabelnya sudah menyebut "Aktivitas" dan jamnya.
defineProps<{
  num: number
  time: string
  kind: 'activity' | 'output'
  label: string
  placeholder: string
  // statusId dipakai dua arah: id baris status di sel Output, dan aria-describedby di kedua sel, supaya
  // hasil simpan tetap terbaca saat fokus ada di isian Aktivitas.
  statusId: string
  showStatus?: boolean
  status?: 'saving' | 'saved' | 'error'
  message?: string
  updatedBy?: string
}>()
const emit = defineEmits<{ blur: []; retry: [] }>()

const value = defineModel<string>({ required: true })

// field-sizing: content membuat textarea tumbuh sendiri mengikuti isinya. Browser yang belum mendukungnya
// (mis. Firefox lama) memakai hitungan scrollHeight sebagai cadangan.
const autoGrow = typeof CSS === 'undefined' || !CSS.supports?.('field-sizing', 'content')

function grow(el: EventTarget | HTMLTextAreaElement | null) {
  if (!autoGrow || !(el instanceof HTMLTextAreaElement)) return
  el.style.height = 'auto'
  el.style.height = `${Math.min(el.scrollHeight, 160)}px`
}

let box: HTMLTextAreaElement | null = null
function bind(el: unknown) {
  box = el instanceof HTMLTextAreaElement ? el : null
  grow(box)
}

onMounted(() => grow(box))
// Isian yang datang dari server (muat ulang atau hasil simpan) juga perlu diukur ulang.
watch(value, () => void nextTick(() => grow(box)))
</script>

<template>
  <!-- max-h-40 plus overflow-auto: catatan panjang digulir di dalam kotaknya, tabelnya tidak ikut memanjang.
       resize-none karena tingginya sudah mengikuti isi, jadi pegangan resize hanya mengacaukan baris tabel. -->
  <textarea
    :ref="bind"
    v-model="value"
    rows="1"
    maxlength="1000"
    :data-slot="num"
    :data-kind="kind"
    :aria-label="`${label} ${time}`"
    :aria-describedby="statusId"
    :placeholder="placeholder"
    class="field-sizing-content block min-h-11 max-h-40 w-full min-w-0 resize-none overflow-auto rounded-card border border-line bg-surface px-3 py-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
    @input="grow($event.target)"
    @blur="emit('blur')"
  ></textarea>

  <!-- Baris status selalu ada selama showStatus, dengan tinggi satu baris, supaya munculnya "Tersimpan"
       tidak mendorong Tambah berkas ke bawah dan membuat tinggi kolom melompat (revisi 33b). -->
  <p
    v-if="showStatus"
    :id="statusId"
    aria-live="polite"
    class="mt-1 min-h-5 text-xs leading-5"
    :class="status === 'error' ? 'text-danger' : 'text-ink-muted'"
  >
    <span v-if="status">{{ saveLabel[status] }}</span>
    <template v-if="status === 'error'">
      <span v-if="message"> {{ message }}</span>
      <button
        type="button"
        class="mt-1 inline-flex min-h-11 items-center rounded-card border border-danger px-3 font-medium"
        @click="emit('retry')"
      >
        Coba lagi
      </button>
    </template>
    <span v-else-if="updatedBy"><span v-if="status"> &middot; </span>diubah {{ updatedBy }}</span>
  </p>
</template>
