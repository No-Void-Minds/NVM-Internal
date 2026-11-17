<script setup lang="ts">
import { computed } from 'vue'
import { useTheme } from '../composables/useTheme'

// icon: tombol ikon di top bar. row: baris di menu Lainnya ponsel.
defineProps<{ variant: 'icon' | 'row' }>()

const { theme, toggle } = useTheme()
const dark = computed(() => theme.value === 'gelap')
// Bulan saat terang (ajakan ke gelap), matahari saat gelap.
const iconPath = computed(() =>
  dark.value
    ? 'M12 3v2M12 19v2M4.2 4.2l1.4 1.4M18.4 18.4l1.4 1.4M3 12h2M19 12h2M4.2 19.8l1.4-1.4M18.4 5.6l1.4-1.4M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8'
    : 'M20 14.5A8 8 0 0 1 9.5 4a8 8 0 1 0 10.5 10.5',
)
</script>

<template>
  <!-- aria-pressed: tombol ini saklar "Mode gelap", jadi pembaca layar mendengar nyala atau mati. -->
  <button
    type="button"
    :aria-pressed="dark"
    :aria-label="variant === 'icon' ? 'Mode gelap' : undefined"
    :title="variant === 'icon' ? (dark ? 'Ganti ke mode terang' : 'Ganti ke mode gelap') : undefined"
    :class="
      variant === 'icon'
        ? 'inline-flex size-11 shrink-0 items-center justify-center rounded-card text-ink-muted hover:bg-canvas hover:text-ink'
        : 'flex min-h-11 w-full items-center gap-3 px-4 text-left text-ink hover:bg-canvas'
    "
    @click="toggle"
  >
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      class="size-5 shrink-0"
      :class="variant === 'row' ? 'text-ink-muted' : ''"
      fill="none"
      stroke="currentColor"
      stroke-width="1.75"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path :d="iconPath" />
    </svg>
    <template v-if="variant === 'row'">
      Mode gelap
      <span class="ml-auto text-sm text-ink-muted">{{ dark ? 'Nyala' : 'Mati' }}</span>
    </template>
  </button>
</template>
