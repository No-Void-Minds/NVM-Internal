<script setup lang="ts">
import { nextTick, onMounted, ref, useId, watch } from 'vue'
import { outputLink, saveLabel } from '../utils/activity'

// Dua isian satu slot jam: Aktivitas dan Output, dengan status simpan di bawahnya. Dipakai kartu di papan
// task dan halaman Aktivitas, jadi aturan simpan otomatis dan teks statusnya hanya ada di satu tempat.
defineProps<{
  time: string
  // Dibaca pembaca layar bersama nama kolom, mis. "Aktivitas 09.00-10.00, Dina".
  context?: string
  updatedBy?: string
  status?: 'saving' | 'saved' | 'error'
  message?: string
  // readonly berarti sel hanya dibaca: teks biasa dan tautan, bukan isian.
  readonly?: boolean
}>()
const emit = defineEmits<{ blur: []; retry: [] }>()

const activity = defineModel<string>('activity', { required: true })
const output = defineModel<string>('output', { required: true })

const id = useId()
const actBox = ref<HTMLTextAreaElement | null>(null)
const outBox = ref<HTMLTextAreaElement | null>(null)

// Textarea tumbuh mengikuti isinya, supaya catatan panjang tidak perlu digulir di dalam kotak kecil.
function grow(el: HTMLTextAreaElement | null) {
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${el.scrollHeight}px`
}

function growAll() {
  grow(actBox.value)
  grow(outBox.value)
}

onMounted(growAll)
// Isian yang datang dari server (muat ulang atau simpan) juga perlu diukur ulang.
watch([activity, output], () => void nextTick(growAll))

const box =
  'block min-h-11 w-full min-w-0 resize-y overflow-hidden rounded-card border border-line bg-surface px-3 py-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
const link = 'underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
</script>

<template>
  <div class="grid min-w-0 gap-2 sm:col-span-2 sm:grid-cols-2 sm:items-start">
    <template v-if="readonly">
      <p class="min-w-0 break-words whitespace-pre-line">
        {{ activity || '-' }}
      </p>
      <p class="min-w-0 break-words whitespace-pre-line">
        <a
          v-if="outputLink(output)"
          :href="outputLink(output) ?? undefined"
          target="_blank"
          rel="noopener noreferrer"
          :class="link"
        >
          {{ output }}
        </a>
        <template v-else>{{ output || '-' }}</template>
      </p>
    </template>
    <template v-else>
      <div class="min-w-0">
        <label :for="`${id}-act`" class="sr-only"
          >Aktivitas {{ time }}{{ context ? `, ${context}` : '' }}</label
        >
        <textarea
          :id="`${id}-act`"
          ref="actBox"
          v-model="activity"
          rows="2"
          maxlength="1000"
          :class="box"
          placeholder="Apa yang dikerjakan"
          @input="grow(actBox)"
          @blur="emit('blur')"
        ></textarea>
      </div>
      <div class="min-w-0">
        <label :for="`${id}-out`" class="sr-only">Output {{ time }}{{ context ? `, ${context}` : '' }}</label>
        <textarea
          :id="`${id}-out`"
          ref="outBox"
          v-model="output"
          rows="2"
          maxlength="1000"
          :class="box"
          placeholder="Hasilnya, boleh tautan"
          @input="grow(outBox)"
          @blur="emit('blur')"
        ></textarea>
      </div>
    </template>

    <p
      v-if="status || updatedBy"
      class="text-sm sm:col-span-2"
      :class="status === 'error' ? 'text-danger' : 'text-ink-muted'"
    >
      <span v-if="status" aria-live="polite">{{ saveLabel[status] }}</span>
      <template v-if="status === 'error'">
        <span v-if="message"> {{ message }}</span>
        <button
          type="button"
          class="ml-2 inline-flex min-h-11 items-center rounded-card border border-danger px-3 font-medium"
          @click="emit('retry')"
        >
          Coba lagi
        </button>
      </template>
      <span v-else-if="updatedBy"><span v-if="status"> &middot; </span>diubah {{ updatedBy }}</span>
    </p>
  </div>
</template>
