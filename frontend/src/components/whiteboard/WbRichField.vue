<script lang="ts">
import { ref } from 'vue'

// Field berformat yang sedang aktif di seluruh halaman. Hanya field ini yang memasang TipTap (revisi 49); field lain
// di kartu yang sama tampil dari JSON dan menjadi aktif saat difokus (klik atau Tab).
export const richActive = ref<string | null>(null)
export function activateField(key: string | null) {
  richActive.value = key
}
</script>

<script setup lang="ts">
import { computed, defineAsyncComponent, onBeforeUnmount, watch } from 'vue'
import type { PMNode } from '../../utils/whiteboard'
import WbRichText from './WbRichText'
import { canonDoc, docText, richValue, textDoc } from '../../utils/wbRich'

const WbNoteEditor = defineAsyncComponent(() => import('./WbNoteEditor.vue'))

const props = withDefaults(
  defineProps<{
    fieldKey: string
    text: string
    doc?: PMNode | null
    label: string
    singleLine?: boolean
    placeholder?: string
    // Indeks karakter dari dobel klik, dipakai sekali saat field ini dibuka.
    caretIndex?: number | null
    selectAll?: boolean
    editorClass?: string
  }>(),
  { singleLine: true, doc: null, placeholder: '', caretIndex: null, editorClass: 'min-h-6' },
)
const emit = defineEmits<{
  commit: [v: { text: string; doc: PMNode | null }]
  input: [v: { text: string; doc: PMNode | null }]
  enter: []
  close: []
}>()

const source = () => props.doc ?? textDoc(props.text)
const sig = (d: PMNode) => JSON.stringify(canonDoc(d))

// Draf lokal: siaran orang lain atau jawaban operasi lain tidak menimpa ketikan yang belum dikirim (sama dengan
// drafts.ts). Draf yang belum diubah ikut nilai server.
const cur = ref<PMNode>(source())
let known = sig(cur.value)
watch(
  () => [props.text, props.doc],
  () => {
    const next = source()
    if (sig(cur.value) === known) cur.value = next
    known = sig(next)
  },
)

const active = computed(() => richActive.value === props.fieldKey)
let caretOnce = props.caretIndex
const caret = computed(() => (active.value && caretOnce !== null ? { textIndex: caretOnce } : null))

function onInput(d: PMNode) {
  cur.value = d
  caretOnce = null
  emit('input', richValue(d))
}

// Dikirim saat field lain diaktifkan atau editor kartu ditutup (field dilepas dari DOM tanpa blur).
function commit() {
  const s = sig(cur.value)
  if (s === known) return
  known = s
  emit('commit', richValue(cur.value))
}
watch(active, (on, was) => {
  if (was && !on) commit()
})
onBeforeUnmount(() => {
  commit()
  if (active.value) richActive.value = null
})

const empty = computed(
  () => !docText(cur.value) && !(cur.value.content ?? []).some((b) => b.type === 'horizontalRule'),
)
</script>

<template>
  <div :data-field="fieldKey" class="min-w-0">
    <WbNoteEditor
      v-if="active"
      :doc="cur"
      :label="label"
      :caret="caret"
      :single-line="singleLine"
      :select-all="selectAll"
      :editor-class="editorClass"
      @input="onInput"
      @enter="emit('enter')"
      @close="emit('close')"
    />
    <div
      v-else
      role="textbox"
      tabindex="0"
      :aria-label="label"
      :aria-multiline="singleLine ? 'false' : 'true'"
      class="cursor-text"
      :class="editorClass"
      @focus="activateField(fieldKey)"
      @pointerdown.stop
    >
      <WbRichText v-if="!empty" :doc="cur" />
      <span v-else class="text-ink-muted">{{ placeholder }}</span>
    </div>
  </div>
</template>
