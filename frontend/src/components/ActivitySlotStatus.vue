<script setup lang="ts">
import { computed } from 'vue'
import { saveLabel } from '../utils/activity'

// Satu baris kecil di baris judul jam (revisi 33b lanjutan): label "Lanjutan <jam>", status simpan, atau
// nama pengubah terakhir. Tingginya dicadangkan satu baris di semua kolom, jadi status yang datang dan
// hilang tidak mengubah tinggi baris judul, dan di bawah Output tidak ada celah kosong yang menunggunya.
const props = defineProps<{
  id: string
  time: string
  linked?: boolean
  headTime?: string
  status?: 'saving' | 'saved' | 'error'
  message?: string
  updatedBy?: string
}>()
const emit = defineEmits<{ retry: [] }>()

const error = computed(() => props.status === 'error')
const text = computed(() => {
  if (props.linked) return `Lanjutan ${props.headTime ?? ''}`
  if (props.status)
    return [saveLabel[props.status], error.value ? props.message : ''].filter(Boolean).join(' ')
  return props.updatedBy ? `diubah ${props.updatedBy}` : ''
})
</script>

<template>
  <!-- min-h sama dengan leading-nya, jadi baris ini setinggi satu baris teks kecil walau isinya kosong. -->
  <span
    :id="id"
    aria-live="polite"
    :role="error ? 'alert' : undefined"
    class="block min-h-[1.125rem] overflow-hidden text-xs leading-[1.125rem] font-normal text-ellipsis whitespace-nowrap"
    :class="error ? 'text-danger' : 'text-ink-muted'"
  >
    <!-- Galat: teksnya dipotong elipsis dengan title, isi penuhnya tetap di DOM jadi pembaca layar ikut
         membacakannya lewat role="alert". Tinggi klik 44 px datang dari padding yang ditarik balik margin
         negatif, jadi baris judul tidak ikut tinggi. -->
    <button
      v-if="error"
      type="button"
      class="-my-[13px] block max-w-full overflow-hidden py-[13px] text-left text-ellipsis whitespace-nowrap underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
      :title="text"
      @click="emit('retry')"
    >
      {{ text }}<span class="sr-only">, klik untuk menyimpan lagi</span>
    </button>
    <template v-else>{{ text }}</template>
  </span>
</template>
