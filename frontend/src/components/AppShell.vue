<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import UserAvatar from './UserAvatar.vue'
import ThemeToggle from './ThemeToggle.vue'
import BrandLogo from './BrandLogo.vue'
import { useSessionStore } from '../stores/session'
import { useLogout } from '../composables/useLogout'
import { roleLabel, type Role } from '../utils/roles'

type NavItem = { to: string; label: string; iconPath: string; roles?: Role[]; primary?: boolean }

// Hanya halaman yang sudah ada yang boleh masuk sini. primary: tampil di bar bawah ponsel, sisanya di menu Lainnya.
const navItems: NavItem[] = [
  {
    to: '/task',
    label: 'Task',
    primary: true,
    iconPath: 'M9 11l3 3 8-8M20 12v7a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11',
  },
  {
    to: '/arsip',
    label: 'Arsip',
    primary: true,
    iconPath:
      'M3 3h18a1 1 0 0 1 1 1v3a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1M4 8v11a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8M10 12h4',
  },
  { to: '/aktivitas', label: 'Aktivitas', iconPath: 'M12 8v4l3 2M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18' },
  {
    to: '/pengguna',
    label: 'Pengguna',
    roles: ['admin', 'manager'],
    iconPath:
      'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75',
  },
  {
    to: '/pengaturan',
    label: 'Pengaturan',
    roles: ['admin', 'manager'],
    iconPath: 'M4 21v-7M4 10V3M12 21v-9M12 8V3M20 21v-5M20 12V3M1 14h6M9 8h6M17 16h6',
  },
  {
    to: '/audit-log',
    label: 'Audit Log',
    roles: ['admin', 'manager'],
    iconPath:
      'M9 5H7a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2h-2M9 5a2 2 0 0 0 2 2h2a2 2 0 0 0 2-2M9 5a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2M9 12h6M9 16h6',
  },
  // Revisi 23: Status hanya untuk Admin, di top bar desktop maupun menu Lainnya (keduanya dari visibleNav).
  { to: '/status', label: 'Status', roles: ['admin'], iconPath: 'M3 12h4l3-8 4 16 3-8h4' },
  {
    to: '/profil',
    label: 'Profil',
    primary: true,
    iconPath: 'M20 21a8 8 0 0 0-16 0M12 13a4 4 0 1 0 0-8 4 4 0 0 0 0 8',
  },
]

const session = useSessionStore()
const route = useRoute()
const { logout, loggingOut, logoutError } = useLogout()

const visibleNav = computed(() =>
  navItems.filter((item) => !item.roles || (session.user && item.roles.includes(session.user.role))),
)

// RouterLink hanya menandai rute yang persis sama, padahal /task/:id rute terpisah dan tetap bagian dari Task.
function isCurrent(to: string): 'page' | undefined {
  return route.path === to || route.path.startsWith(`${to}/`) ? 'page' : undefined
}

// NVM Launcher di PC tim (revisi 5). Kosong di dev: tombol tidak tampil. Aksesnya dijaga Caddy forward_auth ke /api/auth/verify.
const dashboardUrl = (import.meta.env.VITE_DASHBOARD_URL ?? '').trim()
const dashboardIcon = 'M3 3h7v7H3zM14 3h7v7h-7zM14 14h7v7h-7zM3 14h7v7H3z'

const primaryNav = computed(() => visibleNav.value.filter((i) => i.primary))
const moreNav = computed(() => visibleNav.value.filter((i) => !i.primary))
const moreCurrent = computed(() => moreNav.value.some((i) => isCurrent(i.to)))

// Menu Lainnya di ponsel: pola disclosure (tombol dan daftar tautan biasa), jadi cukup Tab, Enter, dan Escape.
const moreOpen = ref(false)
const moreToggle = ref<HTMLButtonElement | null>(null)
const moreMenu = ref<HTMLElement | null>(null)

async function toggleMore() {
  moreOpen.value = !moreOpen.value
  if (moreOpen.value) {
    await nextTick()
    moreMenu.value?.querySelector<HTMLElement>('a, button')?.focus()
  }
}

function closeMore(returnFocus: boolean) {
  if (!moreOpen.value) return
  moreOpen.value = false
  if (returnFocus) moreToggle.value?.focus()
}

// Fokus atau klik di luar tombol dan menu menutup menu, supaya tidak tertinggal terbuka di atas konten.
function onMoreFocusOut(e: FocusEvent) {
  const wrapper = e.currentTarget as HTMLElement
  if (!wrapper.contains(e.relatedTarget as Node | null)) closeMore(false)
}

watch(
  () => route.fullPath,
  () => closeMore(false),
)

// Satu sumber padding kiri dan kanan untuk top bar dan wadah konten (Sprint 11c): selebar layar dan tanpa
// mx-auto, jadi tepi kiri judul halaman sejajar dengan logo dan posisi top bar tidak berubah antar halaman.
const gutter = 'px-4 md:px-8'

// Papan butuh seluruh lebar layar untuk empat kolom. Halaman lain ditengahkan dan dibatasi lebarnya,
// karena tabel dan form yang direntang sampai 1920 px menyisakan ruang kosong besar di kanan.
// Top bar tetap selebar layar di semua rute, jadi logo dan menu tidak pernah berpindah (Sprint 11c).
// Lebar penuh hanya untuk halaman yang kolomnya memang butuh seluruh layar: papan (empat kolom status) dan
// Aktivitas Harian (satu kolom per Staf, revisi 21). Sisanya ditengahkan, lihat Keputusan Sprint 11c.
const fullWidth = ['tasks', 'activity']
const contentWidth = computed(() =>
  fullWidth.includes(String(route.name)) ? 'w-full' : 'mx-auto w-full max-w-6xl',
)
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
      <div class="flex h-14 items-center gap-4 lg:gap-6" :class="gutter">
        <!-- Logo resmi NVM diikuti "Internal". Di lebar md kata itu hanya untuk pembaca layar, karena tujuh item navigasi
             Manajer dan Admin tidak muat bersama teksnya. -->
        <span class="flex shrink-0 items-center gap-2">
          <BrandLogo class="h-5" />
          <span class="sr-only">NVM </span>
          <span class="font-bold whitespace-nowrap max-lg:sr-only max-md:not-sr-only">Internal</span>
        </span>
        <nav aria-label="Navigasi utama" class="hidden md:flex md:gap-1">
          <RouterLink
            v-for="item in visibleNav"
            :key="item.to"
            :to="item.to"
            :aria-current="isCurrent(item.to)"
            class="inline-flex min-h-11 items-center rounded-card px-2 text-sm whitespace-nowrap text-ink-muted lg:text-base hover:bg-canvas lg:px-3 hover:text-ink active:bg-canvas aria-[current=page]:font-medium aria-[current=page]:text-ink"
          >
            {{ item.label }}
          </RouterLink>
        </nav>

        <div v-if="session.user" class="ml-auto hidden items-center gap-3 md:flex">
          <a
            v-if="dashboardUrl"
            :href="dashboardUrl"
            target="_blank"
            rel="noopener"
            aria-label="Dashboard (buka di tab baru)"
            class="inline-flex min-h-11 min-w-11 items-center justify-center gap-2 rounded-card border border-line px-2 text-sm font-medium hover:bg-canvas active:translate-y-px lg:px-3"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 24 24"
              class="size-5 shrink-0"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linejoin="round"
            >
              <path :d="dashboardIcon" />
            </svg>
            <!-- Di lebar md hanya ikon, supaya top bar tujuh item tetap muat; nama aksesibel dari aria-label. -->
            <span class="hidden lg:inline">Dashboard</span>
          </a>
          <ThemeToggle variant="icon" />
          <!-- Di lebar md foto disembunyikan: tujuh item navigasi, Dashboard, dan tombol mode tidak muat bersamanya di 768 px

               (terukur meluap 12 px, Sprint 13). Foto ini dekoratif; nama tetap ada untuk pembaca layar. -->
          <UserAvatar
            class="max-lg:hidden"
            :user-id="session.user.id"
            :name="session.user.name"
            :avatar-id="session.user.avatar_id"
            size="md"
            decorative
          />
          <!-- Di lebar md tujuh item navigasi Manajer dan Admin tidak muat bersama nama; nama tetap dibaca pembaca layar. -->
          <span class="sr-only text-right text-sm leading-tight lg:not-sr-only">
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
      <p v-if="logoutError" role="alert" class="pb-3 text-sm text-danger" :class="gutter">
        {{ logoutError }}
      </p>
    </header>

    <!-- Padding bawah di mobile memberi ruang untuk bottom nav dan area aman layar. -->
    <main
      id="konten"
      tabindex="-1"
      :class="[gutter, contentWidth]"
      class="pt-6 pb-[calc(5rem+env(safe-area-inset-bottom))] outline-none md:pb-10"
    >
      <slot />
    </main>

    <nav
      aria-label="Navigasi utama"
      class="fixed inset-x-0 bottom-0 border-t border-line bg-surface pb-[env(safe-area-inset-bottom)] md:hidden"
    >
      <!-- Maksimal empat item di bar bawah (keputusan Sprint 9 dan 11); halaman lain dan Keluar ada di menu Lainnya. -->
      <ul class="flex">
        <li v-for="item in primaryNav" :key="item.to" class="min-w-0 flex-1">
          <RouterLink
            :to="item.to"
            :aria-current="isCurrent(item.to)"
            class="flex min-h-14 flex-col items-center justify-center gap-1 text-xs text-ink-muted active:bg-canvas aria-[current=page]:font-medium aria-[current=page]:text-mark"
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
            <span class="max-w-full truncate px-0.5">{{ item.label }}</span>
          </RouterLink>
        </li>
        <li class="relative min-w-0 flex-1" @focusout="onMoreFocusOut" @keydown.esc="closeMore(true)">
          <button
            ref="moreToggle"
            type="button"
            class="flex min-h-14 w-full flex-col items-center justify-center gap-1 text-xs active:bg-canvas"
            :class="moreCurrent || moreOpen ? 'font-medium text-mark' : 'text-ink-muted'"
            :aria-expanded="moreOpen"
            aria-controls="menu-lainnya"
            @click="toggleMore"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 24 24"
              class="size-5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
            >
              <path d="M4 6h16M4 12h16M4 18h16" />
            </svg>
            <span class="max-w-full truncate px-0.5">Lainnya</span>
          </button>
          <Transition name="pop">
            <div
              v-if="moreOpen"
              id="menu-lainnya"
              ref="moreMenu"
              class="absolute right-2 bottom-full mb-2 w-60 rounded-card border border-line bg-surface py-1 shadow-menu"
            >
              <ul>
                <li v-for="item in moreNav" :key="item.to">
                  <RouterLink
                    :to="item.to"
                    :aria-current="isCurrent(item.to)"
                    class="flex min-h-11 items-center gap-3 px-4 text-ink hover:bg-canvas aria-[current=page]:font-medium aria-[current=page]:text-mark"
                  >
                    <svg
                      aria-hidden="true"
                      viewBox="0 0 24 24"
                      class="size-5 shrink-0 text-ink-muted"
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
              <a
                v-if="dashboardUrl"
                :href="dashboardUrl"
                target="_blank"
                rel="noopener"
                class="flex min-h-11 items-center gap-3 px-4 text-ink hover:bg-canvas"
              >
                <svg
                  aria-hidden="true"
                  viewBox="0 0 24 24"
                  class="size-5 shrink-0 text-ink-muted"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.75"
                  stroke-linejoin="round"
                >
                  <path :d="dashboardIcon" />
                </svg>
                Dashboard<span class="sr-only"> (buka di tab baru)</span>
              </a>
              <ThemeToggle variant="row" />
              <div v-if="session.user" class="mt-1 border-t border-line pt-1">
                <p class="truncate px-4 pt-1 text-sm text-ink-muted">
                  {{ session.user.name }}, {{ roleLabel(session.user.role) }}
                </p>
                <button
                  type="button"
                  :disabled="loggingOut"
                  class="flex min-h-11 w-full items-center gap-3 px-4 text-left text-ink hover:bg-canvas disabled:opacity-60"
                  @click="logout"
                >
                  <svg
                    aria-hidden="true"
                    viewBox="0 0 24 24"
                    class="size-5 shrink-0 text-ink-muted"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.75"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9" />
                  </svg>
                  {{ loggingOut ? 'Keluar...' : 'Keluar' }}
                </button>
              </div>
            </div>
          </Transition>
        </li>
      </ul>
    </nav>
  </div>
</template>
