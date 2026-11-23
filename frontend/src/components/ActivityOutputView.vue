<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import OutputLinks from './OutputLinks.vue'
import { driveThumbnail, outputParts } from '../utils/outputLink'

// Output satu sel di dialog Aktivitas, mode lihat: setiap tautan jadi tombol berlabel dengan URL lengkap di
// bawahnya (revisi 35), plus thumbnail untuk file Drive pertama. Output tanpa tautan tampil sebagai teks biasa.
// Semua teks dipatahkan di mana saja, jadi URL panjang tidak membuat dialog bisa digulir ke samping.
const props = defineProps<{ output: string }>()

const parts = computed(() => outputParts(props.output))
const driveFileId = computed(() => {
  for (const p of parts.value) if (p.type === 'link' && p.driveFileId) return p.driveFileId
  return null
})
const hasLink = computed(() => parts.value.some((p) => p.type === 'link'))
const thumbFailed = ref(false)
watch(
  () => props.output,
  () => (thumbFailed.value = false),
)

// Drive kadang mengembalikan gambar sangat kecil (ikon umum) untuk file tanpa pratinjau; itu dianggap gagal
// supaya tidak tampil kotak kosong yang besar.
function onLoad(e: Event) {
  const img = e.target as HTMLImageElement
  if (img.naturalWidth < 64 || img.naturalHeight < 32) thumbFailed.value = true
}
</script>

<template>
  <div v-if="hasLink" class="mt-1 grid min-w-0 gap-2">
    <template v-if="driveFileId">
      <!-- Ruang tetap lewat aspect-ratio supaya gambar yang datang belakangan tidak menggeser isi dialog. -->
      <div v-if="!thumbFailed" class="aspect-video overflow-hidden rounded-card border border-line bg-canvas">
        <img
          :src="driveThumbnail(driveFileId)"
          alt=""
          referrerpolicy="no-referrer"
          loading="lazy"
          class="size-full object-contain"
          @load="onLoad"
          @error="thumbFailed = true"
        />
      </div>
      <p class="text-sm text-ink-muted">
        Pratinjau muncul kalau kamu login Google dan punya akses ke file ini.
      </p>
    </template>
    <OutputLinks :output="output" layout="list" />
  </div>
  <p v-else class="mt-1 whitespace-pre-line [overflow-wrap:anywhere]">{{ output || 'Kosong' }}</p>
</template>
