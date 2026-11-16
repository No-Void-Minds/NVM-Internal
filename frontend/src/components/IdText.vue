<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ text: string }>()

// ID task di dalam kalimat (mis. "Memindahkan T-0012 ke Ready") tampil sebagai label spesimen, sama dengan di kartu.
const parts = computed(() => props.text.split(/(\bT-\d{4,}\b)/).filter(Boolean))
</script>

<template>
  <span
    ><template v-for="(p, i) in parts" :key="i"
      ><span v-if="/^T-\d{4,}$/.test(p)" class="specimen-tag">{{ p }}</span
      ><template v-else>{{ p }}</template></template
    ></span
  >
</template>
