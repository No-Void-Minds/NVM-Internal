<script setup lang="ts">
import { liveDisconnected } from '../utils/taskLive'

// what: yang dimuat ulang setelah tersambung, mis. "Papan" atau "Daftar".
defineProps<{ what: string }>()
</script>

<template>
  <!-- Revisi 47: hanya tampil saat koneksi realtime terputus, gaya sama dengan penanda Whiteboard. Data tetap bisa
       dipakai lewat REST selama terputus; yang hilang hanya pembaruan otomatis. -->
  <p
    v-if="liveDisconnected"
    role="status"
    data-testid="live-offline"
    class="fixed bottom-4 left-1/2 z-30 max-w-[calc(100%-2rem)] -translate-x-1/2 rounded-card border border-line bg-surface px-3 py-2 text-sm text-ink shadow-menu"
  >
    Terputus, menyambung ulang. {{ what }} dimuat ulang setelah tersambung.
  </p>
</template>
