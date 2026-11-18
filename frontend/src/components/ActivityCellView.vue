<script setup lang="ts">
import type { ActivityCell } from '../api/activities'
import LinkKindIcon from './LinkKindIcon.vue'
import ActivityThumb from './ActivityThumb.vue'
import { outputKind, outputLink } from '../utils/outputLink'

// Satu sel tabel Aktivitas (revisi 21): ringkasan isi, dipotong tiga baris, dengan tombol selebar sel yang
// membuka dialog. Sel kosong dibiarkan kosong secara visual, hanya pembaca layar yang diberi tahu.
const props = defineProps<{ cell: ActivityCell; openLabel: string }>()
const emit = defineEmits<{ open: [Event] }>()

const link = () => outputLink(props.cell.output)
// Thumbnail hanya untuk gambar pertama: cukup sebagai petunjuk isi, tanpa mengunduh semua lampiran.
const firstImage = () => props.cell.files.find((f) => f.mime_type.startsWith('image/'))
const kind = () => outputKind(props.cell.output)
</script>

<template>
  <div class="min-h-11 px-3 py-2">
    <span v-if="!cell.activity && !cell.output && cell.files.length === 0" class="sr-only">kosong</span>
    <p v-if="cell.activity" class="line-clamp-3 break-words whitespace-pre-line">{{ cell.activity }}</p>
    <template v-if="cell.output">
      <!-- URL panjang jadi chip: ikon dan label jenis (File Drive, Spreadsheet, atau nama host) saja, URL
           penuhnya di title dan aria-label. z-10 menaruh chip di atas tombol sel, jadi tautannya tetap bisa diklik sendiri. -->
      <a
        v-if="link()"
        :href="link() ?? undefined"
        :title="cell.output"
        :aria-label="`Buka tautan output ${cell.output}`"
        target="_blank"
        rel="noopener noreferrer"
        class="relative z-10 mt-1 inline-flex min-h-11 max-w-full items-center gap-1 rounded-chip border border-line px-2 hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
      >
        <LinkKindIcon :kind="kind()?.kind ?? 'host'" class="size-4 shrink-0" />
        <span class="truncate">{{ kind()?.label }}</span>
      </a>
      <p v-else class="mt-1 line-clamp-3 break-words whitespace-pre-line text-ink-muted">
        {{ cell.output }}
      </p>
    </template>
    <!-- Lampiran (revisi 24): ikon klip, jumlahnya, dan thumbnail gambar pertama. Isi lengkapnya di dialog. -->
    <p v-if="cell.files.length" class="mt-1 flex items-center gap-1 text-xs text-ink-muted">
      <svg
        aria-hidden="true"
        viewBox="0 0 24 24"
        class="size-4 shrink-0"
        fill="none"
        stroke="currentColor"
        stroke-width="1.75"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path
          d="M21 12.5 12.5 21a4.95 4.95 0 0 1-7-7l8.5-8.5a3.5 3.5 0 0 1 5 5l-8.5 8.5a2 2 0 0 1-3-3l7.5-7.5"
        />
      </svg>
      {{ cell.files.length }} berkas
      <ActivityThumb
        v-if="firstImage()"
        :id="firstImage()!.id"
        :name="firstImage()!.name"
        :size="40"
        class="ml-1"
      />
    </p>
    <p v-if="cell.updated_by" class="mt-1 text-xs text-ink-muted">diubah {{ cell.updated_by }}</p>
  </div>
  <!-- Tombol selebar sel, bukan sel yang diberi role: klik dan Enter sama-sama membuka dialog. -->
  <button
    type="button"
    class="absolute inset-0 focus-visible:-outline-offset-2 focus-visible:outline-2 focus-visible:outline-mark"
    :aria-label="openLabel"
    @click="emit('open', $event)"
  ></button>
</template>
