<script setup lang="ts">
import { ref } from 'vue'
import { acceptAttr, allowedText } from '../utils/attachment'

const emit = defineEmits<{ files: [files: File[]] }>()

const dragOver = ref(false)

function onPick(e: Event) {
  const input = e.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  // Dikosongkan supaya memilih berkas yang sama lagi (misalnya setelah gagal) tetap memicu change.
  input.value = ''
  if (files.length) emit('files', files)
}

function onDragLeave(e: DragEvent) {
  // dragleave juga terpicu saat kursor masuk ke teks di dalam area; hanya keluar area yang dihitung.
  if (!(e.currentTarget as HTMLElement).contains(e.relatedTarget as Node | null)) dragOver.value = false
}

function onDrop(e: DragEvent) {
  dragOver.value = false
  const files = Array.from(e.dataTransfer?.files ?? [])
  if (files.length) emit('files', files)
}
</script>

<template>
  <!-- label membungkus input supaya area ini bisa diklik, diketuk, dan dibuka dengan Tab lalu Enter.
       Di ponsel, accept yang memuat MIME gambar membuat pemilih berkas menawarkan kamera dan galeri. -->
  <label
    class="flex min-h-24 cursor-pointer flex-col items-center justify-center gap-1 rounded-card border border-dashed p-4 text-center text-sm has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-2 has-[:focus-visible]:outline-accent"
    :class="dragOver ? 'border-accent bg-surface' : 'border-line'"
    @dragover.prevent="dragOver = true"
    @dragleave="onDragLeave"
    @drop.prevent="onDrop"
  >
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      class="size-6 text-ink-muted"
      fill="none"
      stroke="currentColor"
      stroke-width="1.75"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path d="M12 16V5" />
      <path d="m7 10 5-5 5 5" />
      <path d="M5 20h14" />
    </svg>
    <span
      >Tarik berkas ke sini atau <span class="font-medium text-accent underline">pilih berkas</span>.</span
    >
    <span class="text-ink-muted">{{ allowedText }}</span>
    <input type="file" multiple :accept="acceptAttr" class="sr-only" @change="onPick" />
  </label>
</template>
