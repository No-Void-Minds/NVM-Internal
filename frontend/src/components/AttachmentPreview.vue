<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import BaseDialog from './BaseDialog.vue'
import { attachmentUrl } from '../api/attachments'
import { isImage, isPdf } from '../utils/attachment'
import { isVideo } from '../utils/activityFile'

// Cukup tiga kolom ini, supaya lampiran task dan lampiran Aktivitas Harian bisa memakai pratinjau yang sama.
type PreviewItem = { id: string; name: string; mime_type: string }

// Pratinjau gambar, PDF, dan video. Berkasnya diambil sebagai blob, bukan dibuka lewat URL unduhan,
// karena backend mengirim Content-Disposition attachment untuk lampiran task (NFR-SEC-09 d).
// src dipakai lampiran yang bukan milik task, mis. lampiran Aktivitas Harian (revisi 24); kalau diisi,
// taskId tidak dipakai sama sekali.
const props = defineProps<{ taskId?: string; item: PreviewItem | null; src?: string }>()
const emit = defineEmits<{ close: [] }>()

const url = ref('')
const loading = ref(false)
const error = ref<string | null>(null)

function fileUrl(a: PreviewItem): string {
  return props.src ?? attachmentUrl(props.taskId ?? '', a.id)
}

function release() {
  if (url.value) URL.revokeObjectURL(url.value)
  url.value = ''
}

watch(
  () => props.item,
  async (a) => {
    release()
    error.value = null
    if (!a) return
    loading.value = true
    try {
      const res = await fetch(fileUrl(a), { credentials: 'same-origin' })
      if (!res.ok) throw new Error(String(res.status))
      url.value = URL.createObjectURL(await res.blob())
    } catch {
      error.value = 'Pratinjau gagal dimuat. Coba unduh berkasnya.'
    } finally {
      loading.value = false
    }
  },
)

onBeforeUnmount(release)

const buttonSecondary =
  'inline-flex min-h-11 items-center rounded-card border border-line px-4 font-medium hover:bg-canvas active:translate-y-px'
</script>

<template>
  <BaseDialog wide :open="item !== null" :title="item?.name ?? ''" @close="emit('close')">
    <p v-if="loading" class="text-sm text-ink-muted">Memuat pratinjau...</p>
    <p v-else-if="error" role="alert" class="rounded-card bg-danger-bg p-3 text-sm text-danger">
      {{ error }}
    </p>
    <img
      v-else-if="url && item && isImage(item.mime_type)"
      :src="url"
      :alt="`Pratinjau ${item.name}`"
      class="mx-auto max-h-[70vh] w-auto max-w-full rounded-card"
    />
    <!-- preload="metadata": hanya durasi dan bingkai pertama yang diambil, bukan seluruh video 95 MB.
         Sumbernya blob, jadi lompat ke tengah berkas dilayani browser dari memori. -->
    <video
      v-else-if="url && item && isVideo(item.mime_type)"
      :src="url"
      controls
      preload="metadata"
      class="mx-auto max-h-[70vh] w-auto max-w-full rounded-card"
    ></video>
    <!-- sandbox tanpa allow-same-origin: penampil PDF tetap jalan, tapi isi berkas tidak bisa
         menyentuh sesi NVM, pengganti header CSP sandbox yang hilang saat blob dibuat. -->
    <iframe
      v-else-if="url && item && isPdf(item.mime_type)"
      :src="url"
      :title="`Pratinjau ${item.name}`"
      sandbox="allow-scripts"
      class="h-[70vh] w-full rounded-card border border-line"
    ></iframe>
    <template #footer>
      <a v-if="item" :href="fileUrl(item)" :class="buttonSecondary" :aria-label="`Unduh ${item.name}`">
        Unduh
      </a>
      <button type="button" :class="buttonSecondary" @click="emit('close')">Tutup</button>
    </template>
  </BaseDialog>
</template>
