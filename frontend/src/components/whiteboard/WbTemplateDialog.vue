<script setup lang="ts">
import { ref, shallowRef, useId, watch } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import { ApiError } from '../../api/client'
import { listTemplates, type TemplateInfo } from '../../api/whiteboard'

// Pilih template (W4): papan baru berisi salinan template ditaruh di papan ini. Template yang sedang dibuka tidak
// bisa dipilih, supaya template tidak ikut membesar berisi salinan dirinya sendiri setiap kali dipakai di sana.
const props = defineProps<{ open: boolean; current: string; busy: boolean; error: string }>()
const emit = defineEmits<{ close: []; pick: [t: TemplateInfo] }>()

const uid = useId()
const list = shallowRef<TemplateInfo[]>([])
const state = ref<'loading' | 'error' | 'ok'>('loading')
const loadError = ref('')
const choice = ref('')

watch(
  () => props.open,
  async (open) => {
    if (!open) return
    choice.value = ''
    state.value = 'loading'
    try {
      list.value = (await listTemplates()).templates
      state.value = 'ok'
    } catch (e) {
      loadError.value = e instanceof ApiError ? e.message : 'Daftar template gagal dimuat.'
      state.value = 'error'
    }
  },
)

function submit() {
  const t = list.value.find((x) => x.id === choice.value)
  if (t) emit('pick', t)
}

const btn =
  'inline-flex min-h-11 items-center justify-center rounded-card border border-line bg-surface px-3 font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-60'
</script>

<template>
  <BaseDialog :open="open" title="Buat papan dari template" @close="emit('close')">
    <form class="flex flex-col gap-3 px-5 py-4" novalidate @submit.prevent="submit">
      <p v-if="state === 'loading'" role="status" class="text-ink-muted">Memuat daftar template...</p>
      <p v-else-if="state === 'error'" role="alert" class="text-danger">{{ loadError }}</p>
      <p v-else-if="list.length === 0" class="text-ink-muted">
        Belum ada template. Buka papan yang ingin dipakai ulang, lalu tekan Jadikan template di bar atasnya.
      </p>
      <fieldset v-else class="max-h-[50vh] overflow-y-auto">
        <legend class="sr-only">Template</legend>
        <label
          v-for="t in list"
          :key="t.id"
          class="flex min-h-11 items-center gap-3 rounded-card px-2 py-1.5 hover:bg-canvas"
          :class="t.id === current ? 'text-ink-muted' : 'cursor-pointer'"
        >
          <input
            v-model="choice"
            type="radio"
            :name="`${uid}-tpl`"
            :value="t.id"
            :disabled="t.id === current"
            class="size-5 shrink-0 accent-[var(--color-mark)]"
          />
          <span class="min-w-0">
            <span class="block font-semibold break-words">
              {{ t.title }}<template v-if="t.id === current"> (papan ini)</template>
            </span>
            <span class="block text-sm break-words text-ink-muted"> {{ t.path }} · {{ t.cards }} kartu </span>
          </span>
        </label>
      </fieldset>
      <p v-if="error" role="alert" class="text-sm text-danger">{{ error }}</p>
      <div class="flex flex-wrap justify-end gap-2">
        <button type="button" :class="btn" @click="emit('close')">Batal</button>
        <button type="submit" :class="btn" :disabled="!choice || busy">
          {{ busy ? 'Menyalin...' : 'Buat papan' }}
        </button>
      </div>
    </form>
  </BaseDialog>
</template>
