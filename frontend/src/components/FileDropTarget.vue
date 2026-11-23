<script setup lang="ts">
import { ref } from 'vue'
import { pastedFiles } from '../utils/pasteFiles'

// Satu bagian lampiran yang menerima Ctrl+V dan seret-lepas berkas (revisi 36), mis. Arahan atau Hasil sub
// task. Tempelan berkas dari isian mana pun di dalam bagian ini (termasuk Catatan hasil) masuk ke bagian ini;
// tempelan teks dibiarkan menjadi teks. stopPropagation menjaga bagian luar tidak ikut menerima berkas yang sama.
// Ini cara tambahan saja: tombol Tambah berkas tetap ada di dalam bagiannya.
const props = withDefaults(defineProps<{ enabled: boolean; as?: string }>(), { as: 'div' })
const emit = defineEmits<{ files: [files: File[]] }>()

const over = ref(false)

function onDragOver(e: DragEvent) {
  if (!props.enabled || !e.dataTransfer?.types.includes('Files')) return
  e.preventDefault()
  over.value = true
}

function onDragLeave(e: DragEvent) {
  // dragleave juga terpicu saat kursor pindah ke elemen di dalam bagian; hanya keluar bagian yang dihitung.
  if (!(e.currentTarget as HTMLElement).contains(e.relatedTarget as Node | null)) over.value = false
}

function onDrop(e: DragEvent) {
  over.value = false
  // Dropzone lampiran task di dalam bagian ini sudah menangani drop-nya sendiri (preventDefault).
  if (!props.enabled || e.defaultPrevented) return
  const files = Array.from(e.dataTransfer?.files ?? [])
  if (!files.length) return
  e.preventDefault()
  e.stopPropagation()
  emit('files', files)
}

function onPaste(e: ClipboardEvent) {
  if (!props.enabled) return
  const files = pastedFiles(e.clipboardData, new Date())
  if (!files.length) return
  e.preventDefault()
  e.stopPropagation()
  emit('files', files)
}
</script>

<template>
  <component
    :is="as"
    class="rounded-card"
    :class="over ? 'bg-canvas outline-2 outline-offset-4 outline-mark' : ''"
    @dragover="onDragOver"
    @dragleave="onDragLeave"
    @drop="onDrop"
    @paste="onPaste"
  >
    <slot />
  </component>
</template>
