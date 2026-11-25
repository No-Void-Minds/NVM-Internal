<script setup lang="ts">
import { computed, nextTick, ref, useId, watch } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import { SHAPE_GROUPS, SHAPES, openShape, shapePath, type ShapeType } from '../../utils/wbShapes'

// Palet bentuk (W5): kotak cari sebagai combobox yang mengendalikan listbox. Panah atas dan bawah, Home, dan End
// berpindah opsi; Enter menaruh bentuk di tengah layar (atau di titik lepas alat); Escape menutup lewat BaseDialog.
const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: []; pick: [shape: ShapeType] }>()

const uid = useId()
const q = ref('')
const active = ref(0)
const input = ref<HTMLInputElement | null>(null)

const list = computed(() => {
  const t = q.value.trim().toLowerCase()
  return SHAPES.filter((s) => !t || s.label.toLowerCase().includes(t) || s.group.toLowerCase().includes(t))
})
const groups = computed(() =>
  SHAPE_GROUPS.map((g) => ({ g, items: list.value.filter((s) => s.group === g) })).filter(
    (x) => x.items.length,
  ),
)
const optId = (id: string) => `${uid}-opt-${id}`

watch(
  () => props.open,
  async (o) => {
    if (!o) return
    q.value = ''
    active.value = 0
    await nextTick()
    input.value?.focus()
  },
)
watch(q, () => (active.value = 0))

function onKey(e: KeyboardEvent) {
  const n = list.value.length
  if (!n) return
  if (e.key === 'ArrowDown') active.value = (active.value + 1) % n
  else if (e.key === 'ArrowUp') active.value = (active.value - 1 + n) % n
  else if (e.key === 'Home') active.value = 0
  else if (e.key === 'End') active.value = n - 1
  else if (e.key === 'Enter') emit('pick', list.value[active.value]!.id)
  else return
  e.preventDefault()
  document.getElementById(optId(list.value[active.value]!.id))?.scrollIntoView({ block: 'nearest' })
}

function preview(id: ShapeType) {
  const s = SHAPES.find((x) => x.id === id)!
  const k = 36 / Math.max(s.w, s.h)
  return { d: shapePath(id, s.w * k, s.h * k), w: s.w * k, h: s.h * k, open: openShape(id) }
}
</script>

<template>
  <BaseDialog :open="open" title="Pilih bentuk" @close="emit('close')">
    <div class="flex flex-col gap-3 px-5 py-4">
      <label :for="`${uid}-q`" class="text-sm font-semibold">Cari bentuk</label>
      <input
        :id="`${uid}-q`"
        ref="input"
        v-model="q"
        type="text"
        role="combobox"
        aria-autocomplete="list"
        aria-expanded="true"
        :aria-controls="`${uid}-list`"
        :aria-activedescendant="list[active] ? optId(list[active]!.id) : undefined"
        autocomplete="off"
        class="min-h-11 rounded-card border border-line bg-surface px-3"
        placeholder="Contoh: keputusan, kelas, panah"
        @keydown="onKey"
      />
      <p v-if="list.length === 0" role="status" class="text-sm text-ink-muted">
        Tidak ada bentuk dengan nama itu.
      </p>
      <div :id="`${uid}-list`" role="listbox" aria-label="Bentuk" class="max-h-[50vh] overflow-y-auto">
        <div v-for="g in groups" :key="g.g" role="group" :aria-labelledby="`${uid}-g-${g.g}`" class="mb-2">
          <p :id="`${uid}-g-${g.g}`" class="px-1 py-1 text-sm font-semibold text-ink-muted">{{ g.g }}</p>
          <div class="grid grid-cols-[repeat(auto-fill,minmax(6.5rem,1fr))] gap-1">
            <div
              v-for="s in g.items"
              :id="optId(s.id)"
              :key="s.id"
              role="option"
              :aria-selected="list[active]?.id === s.id ? 'true' : 'false'"
              class="flex min-h-11 cursor-pointer flex-col items-center gap-1 rounded-card border border-transparent p-2 text-center text-sm hover:bg-canvas aria-selected:border-mark aria-selected:bg-canvas"
              :data-shape="s.id"
              @click="emit('pick', s.id)"
              @mouseenter="active = list.findIndex((x) => x.id === s.id)"
            >
              <svg aria-hidden="true" width="40" height="40" viewBox="-2 -2 40 40">
                <path
                  :d="preview(s.id).d"
                  :transform="`translate(${(36 - preview(s.id).w) / 2} ${(36 - preview(s.id).h) / 2})`"
                  :fill="preview(s.id).open ? 'none' : 'var(--color-surface)'"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linejoin="round"
                />
              </svg>
              {{ s.label }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </BaseDialog>
</template>
