<script setup lang="ts">
import { computed } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AppShell from './components/AppShell.vue'

const route = useRoute()
const authLayout = computed(() => route.meta.layout === 'auth')
</script>

<template>
  <RouterView v-if="authLayout" />
  <AppShell v-else>
    <RouterView />
  </AppShell>
</template>
