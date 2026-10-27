<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <section class="mx-auto max-w-md py-16 text-center">
    <h1 class="text-2xl font-semibold">Halaman tidak ditemukan</h1>
    <p class="mt-2 text-ink-muted">
      Alamat ini tidak ada di Sistem Internal NVM. Periksa lagi tautannya, atau kembali ke halaman awal.
    </p>
    <RouterLink
      to="/"
      class="mt-6 inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-white hover:underline active:translate-y-px"
    >
      Kembali ke halaman awal
    </RouterLink>
  </section>
</template>
