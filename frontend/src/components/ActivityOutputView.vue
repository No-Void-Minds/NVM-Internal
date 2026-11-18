<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import LinkKindIcon from './LinkKindIcon.vue'
import { driveThumbnail, outputKind, outputLink } from '../utils/outputLink'

// Output satu sel di dialog Aktivitas, mode lihat: jenis tautan, thumbnail untuk file Drive, lalu URL lengkap.
// Output yang bukan tautan tampil sebagai teks biasa.
const props = defineProps<{ output: string }>()

const kind = computed(() => outputKind(props.output))
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
  <div v-if="kind" class="mt-1 grid gap-2">
    <p class="inline-flex items-center gap-2 font-medium">
      <LinkKindIcon :kind="kind.kind" class="size-5 shrink-0" />
      {{ kind.label }}
    </p>
    <template v-if="kind.driveFileId">
      <!-- Ruang tetap lewat aspect-ratio supaya gambar yang datang belakangan tidak menggeser isi dialog. -->
      <div v-if="!thumbFailed" class="aspect-video overflow-hidden rounded-card border border-line bg-canvas">
        <img
          :src="driveThumbnail(kind.driveFileId)"
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
    <a
      :href="outputLink(output) ?? undefined"
      target="_blank"
      rel="noopener noreferrer"
      class="break-all underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
    >
      {{ output }}
    </a>
  </div>
  <p v-else class="mt-1 break-words whitespace-pre-line">{{ output || 'Kosong' }}</p>
</template>
