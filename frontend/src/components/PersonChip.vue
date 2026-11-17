<script setup lang="ts">
import { computed, ref } from 'vue'
import { usePersonPopover } from '../composables/usePersonPopover'

const props = defineProps<{ userId: number; name: string }>()
const { current, open, close } = usePersonPopover()
const el = ref<HTMLButtonElement | null>(null)
const expanded = computed(() => current.value?.trigger === el.value && el.value !== null)

function toggle() {
  if (!el.value) return
  if (expanded.value) close(true)
  else open(props.userId, props.name, el.value)
}
</script>

<template>
  <!-- draggable plus dragstart.prevent: menekan pemicu di kartu papan tidak boleh memulai drag kartunya.
       ::after memperluas area klik ke 44x44 px tanpa mengubah ukuran yang terlihat. -->
  <button
    ref="el"
    type="button"
    data-person-trigger
    draggable="true"
    aria-haspopup="dialog"
    :aria-expanded="expanded"
    :aria-label="`Lihat profil ${name}`"
    class="relative inline-flex max-w-full items-center gap-2 rounded-chip text-left after:absolute after:top-1/2 after:left-1/2 after:h-[max(100%,2.75rem)] after:w-[max(100%,2.75rem)] after:-translate-x-1/2 after:-translate-y-1/2 after:content-[''] hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
    @dragstart.stop.prevent
    @click.stop="toggle"
  >
    <slot />
  </button>
</template>
