<script setup lang="ts">
import { RouterLink } from 'vue-router'

type NavItem = { to: string; label: string; iconPath: string }

// Hanya halaman yang sudah ada yang boleh masuk sini; ditambah per sprint.
const navItems: NavItem[] = [{ to: '/', label: 'Status', iconPath: 'M3 12h4l3-8 4 16 3-8h4' }]
</script>

<template>
  <div class="min-h-dvh">
    <a
      href="#konten"
      class="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50 focus:rounded-card focus:bg-surface focus:px-4 focus:py-3"
    >
      Lewati ke konten
    </a>

    <header class="border-b border-line bg-surface">
      <div class="mx-auto flex h-14 max-w-6xl items-center gap-6 px-4 md:px-8">
        <span class="font-semibold">NVM Internal</span>
        <nav aria-label="Navigasi utama" class="hidden md:flex md:gap-1">
          <RouterLink
            v-for="item in navItems"
            :key="item.to"
            :to="item.to"
            class="inline-flex min-h-11 items-center rounded-card px-3 text-ink-muted hover:bg-canvas hover:text-ink active:bg-canvas aria-[current=page]:font-medium aria-[current=page]:text-ink"
          >
            {{ item.label }}
          </RouterLink>
        </nav>
      </div>
    </header>

    <!-- Padding bawah di mobile memberi ruang untuk bottom nav dan area aman layar. -->
    <main
      id="konten"
      class="mx-auto max-w-6xl px-4 pt-6 pb-[calc(5rem+env(safe-area-inset-bottom))] md:px-8 md:pb-10"
    >
      <slot />
    </main>

    <nav
      aria-label="Navigasi utama"
      class="fixed inset-x-0 bottom-0 border-t border-line bg-surface pb-[env(safe-area-inset-bottom)] md:hidden"
    >
      <ul class="flex">
        <li v-for="item in navItems" :key="item.to" class="flex-1">
          <RouterLink
            :to="item.to"
            class="flex min-h-14 flex-col items-center justify-center gap-1 text-xs text-ink-muted active:bg-canvas aria-[current=page]:font-medium aria-[current=page]:text-accent"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 24 24"
              class="size-5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path :d="item.iconPath" />
            </svg>
            {{ item.label }}
          </RouterLink>
        </li>
      </ul>
    </nav>
  </div>
</template>
