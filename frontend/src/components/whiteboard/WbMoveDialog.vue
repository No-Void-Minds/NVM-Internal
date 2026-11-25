<script setup lang="ts">
import { computed, ref, shallowRef, useId, watch } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import { ApiError } from '../../api/client'
import { listBoards, type BoardOption } from '../../api/whiteboard'

// Pilih papan tujuan untuk kartu terpilih. Papan ditampilkan sebagai pohon (menjorok per tingkat). Papan asal dan
// papan yang ikut dipindah (beserta isinya) tidak bisa dipilih; server tetap memeriksa siklus dan kedalaman.
const props = defineProps<{
  open: boolean
  current: string
  moving: string[]
  busy: boolean
  error: string
}>()
const emit = defineEmits<{ close: []; move: [to: string] }>()

const uid = useId()
const boards = shallowRef<BoardOption[]>([])
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
      boards.value = (await listBoards()).boards
      state.value = 'ok'
    } catch (e) {
      loadError.value = e instanceof ApiError ? e.message : 'Daftar papan gagal dimuat.'
      state.value = 'error'
    }
  },
)

const tree = computed(() => {
  const kids = new Map<string, BoardOption[]>()
  for (const b of boards.value) {
    const k = b.parent_id ?? ''
    kids.set(k, [...(kids.get(k) ?? []), b])
  }
  const blocked = new Set(props.moving)
  const out: { b: BoardOption; depth: number; disabled: boolean }[] = []
  const walk = (parent: string, depth: number, inBlocked: boolean) => {
    for (const b of kids.get(parent) ?? []) {
      const off = inBlocked || blocked.has(b.id)
      out.push({ b, depth, disabled: off || b.id === props.current })
      walk(b.id, depth + 1, off)
    }
  }
  walk('', 0, false)
  return out
})

const btn =
  'inline-flex min-h-11 items-center justify-center rounded-card border border-line bg-surface px-3 font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-60'
</script>

<template>
  <BaseDialog :open="open" title="Pindahkan ke papan" @close="emit('close')">
    <form class="flex flex-col gap-3 px-5 py-4" novalidate @submit.prevent="choice && emit('move', choice)">
      <p v-if="state === 'loading'" role="status" class="text-ink-muted">Memuat daftar papan...</p>
      <p v-else-if="state === 'error'" role="alert" class="text-danger">{{ loadError }}</p>
      <fieldset v-else class="max-h-[50vh] overflow-y-auto">
        <legend class="sr-only">Papan tujuan</legend>
        <label
          v-for="t in tree"
          :key="t.b.id"
          class="flex min-h-11 items-center gap-2 rounded-card pr-2 hover:bg-canvas"
          :class="t.disabled ? 'text-ink-muted' : 'cursor-pointer'"
          :style="{ paddingLeft: `${0.5 + t.depth * 1.25}rem` }"
        >
          <input
            v-model="choice"
            type="radio"
            :name="`${uid}-to`"
            :value="t.b.id"
            :disabled="t.disabled"
            class="size-5 accent-[var(--color-mark)]"
          />
          <span class="min-w-0 break-words">
            {{ t.b.title }}<template v-if="t.b.id === current"> (papan ini)</template>
          </span>
        </label>
      </fieldset>
      <p v-if="error" role="alert" class="text-sm text-danger">{{ error }}</p>
      <div class="flex flex-wrap justify-end gap-2">
        <button type="button" :class="btn" @click="emit('close')">Batal</button>
        <button type="submit" :class="btn" :disabled="!choice || busy">Pindahkan</button>
      </div>
    </form>
  </BaseDialog>
</template>
