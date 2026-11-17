<script setup lang="ts">
import PageHeader from '../components/PageHeader.vue'
import { RouterLink } from 'vue-router'
</script>

<template>
  <section class="max-w-xl">
    <PageHeader
      title="Halaman tidak ditemukan"
      description="Alamat ini tidak ada di Sistem Internal NVM. Periksa lagi tautannya, atau kembali ke halaman awal."
    />
    <RouterLink
      to="/"
      class="mt-6 inline-flex min-h-11 items-center rounded-card bg-accent px-4 font-medium text-on-accent hover:bg-accent-hover active:translate-y-px"
    >
      Kembali ke halaman awal
    </RouterLink>
  </section>
</template>
