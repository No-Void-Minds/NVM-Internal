<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { avatarUrl } from '../api/profile'
import { initials } from '../utils/person'

const props = withDefaults(
  defineProps<{
    userId: number
    name: string
    avatarId?: string | null
    size?: 'sm' | 'md' | 'lg' | 'xl'
    // true bila nama sudah tertulis di sebelahnya, supaya pembaca layar tidak membacanya dua kali.
    decorative?: boolean
  }>(),
  { avatarId: null, size: 'md', decorative: false },
)

// Foto yang gagal dimuat (berkas hilang, jaringan putus) kembali ke inisial, bukan ikon gambar rusak.
const failed = ref(false)
watch(
  () => props.avatarId,
  () => (failed.value = false),
)

const src = computed(() => (props.avatarId && !failed.value ? avatarUrl(props.userId, props.avatarId) : null))

const sizeClass = {
  sm: 'size-7 text-xs',
  md: 'size-8 text-xs',
  lg: 'size-12 text-base',
  xl: 'size-20 text-2xl',
}
</script>

<template>
  <img
    v-if="src"
    :src="src"
    :alt="decorative ? '' : name"
    loading="lazy"
    decoding="async"
    class="shrink-0 rounded-full border border-line bg-canvas object-cover"
    :class="sizeClass[size]"
    @error="failed = true"
  />
  <span
    v-else
    class="flex shrink-0 items-center justify-center rounded-full border border-line bg-canvas font-medium text-ink"
    :class="sizeClass[size]"
    :role="decorative ? undefined : 'img'"
    :aria-label="decorative ? undefined : name"
    :aria-hidden="decorative ? 'true' : undefined"
    >{{ initials(name) }}</span
  >
</template>
