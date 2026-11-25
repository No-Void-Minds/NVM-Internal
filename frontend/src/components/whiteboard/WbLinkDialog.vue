<script setup lang="ts">
import { ref, useId, watch } from 'vue'
import BaseDialog from '../BaseDialog.vue'

// Alamat untuk kartu tautan. Pratinjau diambil server (dengan pemeriksaan alamat internal); kalau gagal, kartu
// tetap dibuat dengan alamatnya saja.
const props = defineProps<{ open: boolean; busy: boolean; error: string }>()
const emit = defineEmits<{ close: []; submit: [url: string] }>()

const uid = useId()
const url = ref('')
watch(
  () => props.open,
  (o) => o && (url.value = ''),
)

const btn =
  'inline-flex min-h-11 items-center justify-center rounded-card border border-line bg-surface px-3 font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-60'
</script>

<template>
  <BaseDialog :open="open" title="Tambah tautan" @close="emit('close')">
    <form class="flex flex-col gap-2 px-5 py-4" novalidate @submit.prevent="emit('submit', url.trim())">
      <label :for="`${uid}-url`" class="font-medium">Alamat (http:// atau https://)</label>
      <input
        :id="`${uid}-url`"
        v-model="url"
        type="url"
        inputmode="url"
        maxlength="2048"
        placeholder="https://"
        :aria-invalid="error ? 'true' : undefined"
        :aria-describedby="error ? `${uid}-err` : undefined"
        class="min-h-11 rounded-card border border-line bg-surface px-3"
      />
      <p v-if="error" :id="`${uid}-err`" role="alert" class="text-sm text-danger">{{ error }}</p>
      <p v-if="busy" role="status" class="text-sm text-ink-muted">Mengambil pratinjau...</p>
      <div class="mt-2 flex flex-wrap justify-end gap-2">
        <button type="button" :class="btn" @click="emit('close')">Batal</button>
        <button type="submit" :class="btn" :disabled="busy || !url.trim()">Tambah tautan</button>
      </div>
    </form>
  </BaseDialog>
</template>
