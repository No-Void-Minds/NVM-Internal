<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { activityFileUrl } from '../api/activityFiles'

// Thumbnail gambar lampiran (revisi 24). Berkas aslinya tetap utuh di server; yang diperkecil hanya
// tampilannya, di browser, supaya foto 10 MB tidak dipakai sebagai gambar 40 px.
const props = defineProps<{ id: string; name: string; size?: number }>()

const url = ref('')
const failed = ref(false)

function release() {
  if (url.value) URL.revokeObjectURL(url.value)
  url.value = ''
}

async function load(id: string) {
  release()
  failed.value = false
  try {
    const res = await fetch(activityFileUrl(id), { credentials: 'same-origin' })
    if (!res.ok) throw new Error(String(res.status))
    const blob = await res.blob()
    const bitmap = await createImageBitmap(blob)
    const max = props.size ?? 96
    const scale = Math.min(1, max / Math.max(bitmap.width, bitmap.height))
    const canvas = document.createElement('canvas')
    canvas.width = Math.max(1, Math.round(bitmap.width * scale))
    canvas.height = Math.max(1, Math.round(bitmap.height * scale))
    canvas.getContext('2d')?.drawImage(bitmap, 0, 0, canvas.width, canvas.height)
    bitmap.close()
    const small = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/webp', 0.8))
    url.value = URL.createObjectURL(small ?? blob)
  } catch {
    // Gagal memperkecil bukan galat yang perlu diumumkan: ikon jenis berkas sudah cukup sebagai pengganti.
    failed.value = true
  }
}

watch(() => props.id, load, { immediate: true })
onBeforeUnmount(release)
</script>

<template>
  <img
    v-if="url"
    :src="url"
    :alt="`Thumbnail ${name}`"
    class="size-10 rounded-card border border-line object-cover"
  />
  <span
    v-else-if="!failed"
    class="size-10 rounded-card border border-line bg-canvas"
    aria-hidden="true"
  ></span>
  <svg
    v-else
    aria-hidden="true"
    viewBox="0 0 24 24"
    class="size-10 rounded-card border border-line p-2 text-ink-muted"
    fill="none"
    stroke="currentColor"
    stroke-width="1.75"
    stroke-linecap="round"
    stroke-linejoin="round"
  >
    <path d="M4 5h16v14H4zM4 15l5-5 5 5 3-3 3 3" />
    <circle cx="9" cy="9" r="1.5" />
  </svg>
</template>
