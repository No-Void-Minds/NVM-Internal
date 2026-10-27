<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { useSessionStore } from '../stores/session'
import { useLogout } from '../composables/useLogout'
import { roleLabel } from '../utils/roles'

type NavItem = { to: string; label: string; iconPath: string }

// Hanya halaman yang sudah ada yang boleh masuk sini; ditambah per sprint.
const navItems: NavItem[] = [
  { to: '/status', label: 'Status', iconPath: 'M3 12h4l3-8 4 16 3-8h4' },
  { to: '/profil', label: 'Profil', iconPath: 'M20 21a8 8 0 0 0-16 0M12 13a4 4 0 1 0 0-8 4 4 0 0 0 0 8' },
]

const session = useSessionStore()
const { logout, loggingOut, logoutError } = useLogout()
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

        <div v-if="session.user" class="ml-auto hidden items-center gap-3 md:flex">
          <span class="text-right text-sm leading-tight">
            <span class="block font-medium">{{ session.user.name }}</span>
            <span class="block text-ink-muted">{{ roleLabel(session.user.role) }}</span>
          </span>
          <button
            type="button"
            :disabled="loggingOut"
            class="inline-flex min-h-11 items-center rounded-card border border-line px-3 text-sm font-medium hover:bg-canvas active:translate-y-px disabled:opacity-60"
            @click="logout"
          >
            Keluar
          </button>
        </div>
      </div>
      <p v-if="logoutError" role="alert" class="mx-auto max-w-6xl px-4 pb-3 text-sm text-danger md:px-8">
        {{ logoutError }}
      </p>
    </header>

    <!-- Padding bawah di mobile memberi ruang untuk bottom nav dan area aman layar. -->
    <main
      id="konten"
      tabindex="-1"
      class="mx-auto max-w-6xl px-4 pt-6 pb-[calc(5rem+env(safe-area-inset-bottom))] outline-none md:px-8 md:pb-10"
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
