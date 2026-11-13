<script setup lang="ts">
import { ref, watch } from 'vue'
import { minKeyword } from '../utils/boardQuery'

const props = defineProps<{ id: string; value: string | undefined }>()
const emit = defineEmits<{ search: [value: string | undefined] }>()

const draft = ref(props.value ?? '')
const error = ref('')

// URL bisa berubah dari luar (tombol Back, Hapus filter), jadi isian ikut menyesuaikan.
watch(
  () => props.value,
  (v) => {
    draft.value = v ?? ''
    error.value = ''
  },
)

function submit() {
  const text = draft.value.trim()
  if (text && [...text].length < minKeyword) {
    error.value = `Kata kunci minimal ${minKeyword} huruf.`
    return
  }
  error.value = ''
  emit('search', text || undefined)
}

function clear() {
  draft.value = ''
  error.value = ''
  emit('search', undefined)
}

const buttonSecondary =
  'inline-flex min-h-11 shrink-0 items-center rounded-card border border-line bg-surface px-4 font-medium hover:bg-canvas active:translate-y-px'
</script>

<template>
  <form role="search" class="min-w-0" novalidate @submit.prevent="submit">
    <label :for="id" class="text-sm font-medium">Cari task</label>
    <div class="mt-1 flex flex-wrap gap-2">
      <input
        :id="id"
        v-model="draft"
        type="search"
        enterkeyhint="search"
        maxlength="100"
        :aria-invalid="error ? 'true' : undefined"
        :aria-describedby="error ? `${id}-error` : `${id}-hint`"
        class="block min-h-11 min-w-0 flex-1 basis-48 rounded-card border border-line bg-surface px-3"
      />
      <button type="submit" :class="buttonSecondary">Cari</button>
      <button v-if="value" type="button" :class="buttonSecondary" @click="clear">
        Hapus<span class="sr-only"> pencarian</span>
      </button>
    </div>
    <p v-if="error" :id="`${id}-error`" role="alert" class="mt-1 text-sm text-danger">{{ error }}</p>
    <p v-else :id="`${id}-hint`" class="mt-1 text-sm text-ink-muted">
      ID, judul, deskripsi, pemohon, PIC, sub task, label, atau nama berkas. Minimal {{ minKeyword }} huruf.
    </p>
  </form>
</template>
