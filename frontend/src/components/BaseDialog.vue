<script setup lang="ts">
import { computed, onMounted, ref, useId, watch } from 'vue'

const props = withDefaults(
  // tag: judul berupa kode (mis. ID task), ditampilkan sebagai label spesimen.
  // wide: isi yang butuh ruang baca, mis. pratinjau gambar dan PDF.
  defineProps<{
    open: boolean
    title: string
    variant?: 'modal' | 'panel'
    tag?: boolean
    wide?: boolean
  }>(),
  { variant: 'modal', tag: false, wide: false },
)
// reason 'escape' membedakan tombol Escape dari tombol Tutup, untuk dialog yang Escape-nya bertahap.
const emit = defineEmits<{ close: [reason?: 'escape'] }>()

const el = ref<HTMLDialogElement | null>(null)
const titleId = useId()

// Dialog bawaan browser dipakai karena showModal sudah mengurus fokus, tombol Escape,
// dan membuat isi halaman di belakangnya tidak bisa diklik.
function sync() {
  const d = el.value
  if (!d) return
  if (props.open && !d.open) d.showModal()
  if (!props.open && d.open) d.close()
}
watch(() => props.open, sync)
onMounted(sync)

// Escape memicu event cancel; ditangkap supaya status buka-tutup tetap dikendalikan parent.
// Hanya cancel milik dialog ini sendiri: input berkas juga mengirim event cancel yang bubble saat pemilih berkas
// ditutup tanpa memilih, dan dulu itu ikut menutup panel (revisi 26). Dialog bersarang juga tidak saling tutup.
function onCancel(e: Event) {
  if (e.target !== e.currentTarget) return
  e.preventDefault()
  emit('close', 'escape')
}

const shape = computed(() =>
  props.variant === 'panel'
    ? // overflow-hidden wajib: <dialog> bawaan browser punya overflow auto, jadi panelnya sendiri ikut
      // tergulir dan menyeret area gulir plus bar aksi naik keluar layar (terukur 431 px di Chrome).
      // Isinya sudah diatur flex: header tetap, isian yang menggulir sendiri.
      'dialog-panel m-0 ml-auto h-dvh max-h-dvh w-full max-w-md overflow-hidden border-l border-line shadow-dialog open:flex open:flex-col'
    : `dialog-modal m-auto w-[calc(100%-2rem)] ${props.wide ? 'max-w-3xl' : 'max-w-md'} rounded-card border border-line shadow-dialog`,
)
</script>

<template>
  <dialog
    ref="el"
    :aria-labelledby="titleId"
    :class="shape"
    class="bg-surface p-0 text-ink backdrop:bg-scrim"
    @cancel="onCancel"
  >
    <div class="flex items-center justify-between gap-4 border-b border-line px-5 py-3">
      <h2 :id="titleId" class="text-lg font-semibold">
        <span v-if="tag" class="specimen-tag">{{ title }}</span>
        <template v-else>{{ title }}</template>
      </h2>
      <div class="flex shrink-0 items-center gap-1">
        <slot name="actions" />
        <button
          type="button"
          aria-label="Tutup"
          class="inline-flex size-11 shrink-0 items-center justify-center rounded-card text-ink-muted hover:bg-canvas hover:text-ink active:bg-canvas"
          @click="emit('close')"
        >
          <svg
            aria-hidden="true"
            viewBox="0 0 24 24"
            class="size-5"
            fill="none"
            stroke="currentColor"
            stroke-width="1.75"
            stroke-linecap="round"
          >
            <path d="M6 6l12 12M18 6L6 18" />
          </svg>
        </button>
      </div>
    </div>
    <!-- relative membuat wadah gulir ini induk elemen absolute di dalamnya (input berkas dan radio sr-only).
         Tanpa itu induknya <dialog>, posisinya tidak ikut tergulir, dan fokus ke input itu menggulir dialognya
         sendiri walau overflow-hidden: terukur dialog scrollTop 48 ke 287 saat pilih berkas diklik (revisi 26). -->
    <div class="relative min-h-0 flex-1 overflow-y-auto px-5 py-4">
      <slot />
    </div>
    <div v-if="$slots.footer" class="flex justify-end gap-2 border-t border-line px-5 py-3">
      <slot name="footer" />
    </div>
  </dialog>
</template>
