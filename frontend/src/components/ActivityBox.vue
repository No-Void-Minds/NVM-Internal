<script setup lang="ts">
import { computed, nextTick, onMounted, watch } from 'vue'
import OutputLinks from './OutputLinks.vue'
import { outputParts } from '../utils/outputLink'

// Satu isian di tabel Aktivitas hari ini (revisi 21, tata letak diputar): satu textarea per sel. Nama isian
// datang dari aria-label, bukan label terlihat, karena kepala baris dan kolom tabelnya sudah menyebut
// "Aktivitas" dan jamnya. Status simpannya ada di baris judul jam (ActivitySlotStatus, revisi 33b).
const props = defineProps<{
  num: number
  time: string
  kind: 'activity' | 'output'
  label: string
  placeholder: string
  // statusId menunjuk baris status di baris judul jam (ActivitySlotStatus), dipakai aria-describedby kedua
  // sel, supaya hasil simpan tetap terbaca saat fokus ada di isian Aktivitas.
  statusId: string
  // readonly: jam yang tersambung ke jam sebelumnya. Isiannya tetap isian yang sama, hanya tidak bisa
  // diubah, supaya tinggi dan patahan teksnya persis sama dengan jam kepalanya (revisi 33b lanjutan).
  readonly?: boolean
}>()
const emit = defineEmits<{ blur: [] }>()

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

// Revisi 35: Output jam lanjutan yang berisi tautan tampil sebagai tombol, bukan URL mentah di textarea.
// Bentuk kotaknya disamakan dengan textarea readonly supaya tetap terbaca menyambung dari jam kepalanya.
const linkView = computed(
  () => props.readonly && props.kind === 'output' && outputParts(value.value).some((p) => p.type === 'link'),
)
</script>

<template>
  <!-- max-h-40 plus overflow-auto: catatan panjang digulir di dalam kotaknya, tabelnya tidak ikut memanjang.
       resize-none karena tingginya sudah mengikuti isi, jadi pegangan resize hanya mengacaukan baris tabel. -->
  <!-- Jam tersambung: tepi kiri lurus tanpa garis dan latar canvas, jadi terbaca menyambung dari jam
       sebelumnya tanpa melebar menutupi celah tombol rantai. Placeholder dimatikan supaya jam kepala yang
       kosong tidak tampak berisi di jam lanjutannya. -->
  <div
    v-if="linkView"
    :data-slot="num"
    :data-kind="kind"
    role="group"
    :aria-label="`${label} ${time}`"
    :aria-describedby="statusId"
    class="block min-h-11 max-h-40 w-full min-w-0 overflow-y-auto rounded-card rounded-l-none border border-l-0 border-line bg-canvas px-3 py-1"
  >
    <OutputLinks :output="value" />
  </div>
  <textarea
    v-else
    :ref="bind"
    v-model="value"
    rows="1"
    maxlength="1000"
    :data-slot="num"
    :data-kind="kind"
    :readonly="readonly"
    :aria-label="`${label} ${time}`"
    :aria-describedby="statusId"
    :placeholder="readonly ? undefined : placeholder"
    class="field-sizing-content block min-h-11 max-h-40 w-full min-w-0 resize-none overflow-auto rounded-card border border-line px-3 py-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
    :class="readonly ? 'rounded-l-none border-l-0 bg-canvas' : 'bg-surface'"
    @input="grow($event.target)"
    @blur="emit('blur')"
  ></textarea>
</template>
