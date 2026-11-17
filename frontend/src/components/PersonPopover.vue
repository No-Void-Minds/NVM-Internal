<script setup lang="ts">
import { computed, nextTick, ref, shallowRef, useId, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { ApiError } from '../api/client'
import { boardUsername, getPersonCard, type PersonCard } from '../api/people'
import { usePersonPopover } from '../composables/usePersonPopover'
import { roleLabel } from '../utils/roles'
import UserAvatar from './UserAvatar.vue'

// Satu instance di App.vue. Isinya di-teleport ke <dialog> terdekat dari pemicu: modal dialog membuat semua di
// luarnya inert, jadi popover yang dibuka dari panel detail harus berada di dalam panel itu. popover="manual"
// menaruhnya di top layer di atas panel, dan tutupnya diatur sendiri supaya klik luar tidak ikut menutup panel.
const { current, close } = usePersonPopover()
const root = ref<HTMLElement | null>(null)
const card = ref<HTMLElement | null>(null)
const closeBtn = ref<HTMLButtonElement | null>(null)
const titleId = useId()

type State = { kind: 'loading' } | { kind: 'error'; message: string } | { kind: 'ok'; person: PersonCard }
const state = shallowRef<State>({ kind: 'loading' })
const username = ref<string | null>(null)
// Di bawah 640 px kartu menjadi lembar dari bawah; di atasnya menempel di pemicu.
const sheet = ref(false)
const pos = ref<{ top: number; left: number } | null>(null)

const target = computed<HTMLElement | string>(() => current.value?.trigger.closest('dialog') ?? 'body')

async function load(id: number) {
  state.value = { kind: 'loading' }
  username.value = null
  try {
    const person = await getPersonCard(id)
    if (current.value?.id !== id) return
    state.value = { kind: 'ok', person }
    if (person.is_active) {
      const u = await boardUsername(id).catch(() => null)
      if (current.value?.id === id) username.value = u
    }
  } catch (e) {
    if (current.value?.id !== id) return
    state.value = {
      kind: 'error',
      message: e instanceof ApiError ? e.message : 'Profil gagal dimuat. Periksa koneksi, lalu coba lagi.',
    }
  }
  await nextTick()
  place()
}

const GAP = 8
function place() {
  const c = current.value
  if (!c || !card.value) return
  sheet.value = !window.matchMedia('(min-width: 640px)').matches
  if (sheet.value) return
  const r = c.trigger.getBoundingClientRect()
  const w = card.value.offsetWidth
  const h = card.value.offsetHeight
  const vw = document.documentElement.clientWidth
  const vh = window.innerHeight
  // Mepet kanan: rata kanan dengan pemicu. Mepet bawah: dibuka ke atas. Lalu dijepit ke layar.
  let left = r.left + w > vw - GAP ? r.right - w : r.left
  left = Math.min(Math.max(GAP, left), vw - w - GAP)
  let top = r.bottom + GAP
  if (top + h > vh - GAP) top = r.top - h - GAP
  top = Math.max(GAP, top)
  pos.value = { top, left }
}

function onKeydown(e: KeyboardEvent) {
  if (e.key !== 'Escape') return
  // Dibatalkan supaya Escape yang sama tidak ikut menutup panel detail di belakangnya.
  e.preventDefault()
  e.stopPropagation()
  close(true)
}

function inside(t: EventTarget | null) {
  return t instanceof Node && !!card.value?.contains(t)
}

function swallowNextClick() {
  const eat = (e: Event) => {
    e.preventDefault()
    e.stopPropagation()
  }
  document.addEventListener('click', eat, { capture: true, once: true })
  document.addEventListener(
    'pointerup',
    () => setTimeout(() => document.removeEventListener('click', eat, { capture: true })),
    { capture: true, once: true },
  )
}

function onPointerdown(e: PointerEvent) {
  if (inside(e.target)) return
  // Pemicu lain mengurus dirinya sendiri: popover langsung pindah ke sana.
  if (e.target instanceof Element && e.target.closest('[data-person-trigger]')) return
  // Klik luar hanya menutup popover. Klik itu ditelan supaya tidak ikut menutup panel atau membuka kartu.
  swallowNextClick()
  close(true)
}

function onFocusin(e: FocusEvent) {
  const t = e.target
  if (inside(t) || t === current.value?.trigger) return
  if (t instanceof Element && t.closest('[data-person-trigger]')) return
  close(false)
}

function onReflow() {
  if (current.value) place()
}

function listen(on: boolean) {
  const m = on ? 'addEventListener' : 'removeEventListener'
  document[m]('keydown', onKeydown as EventListener, true)
  document[m]('pointerdown', onPointerdown as EventListener, true)
  document[m]('focusin', onFocusin as EventListener, true)
  window[m]('resize', onReflow)
  window[m]('scroll', onReflow, true)
}

watch(current, async (c, old) => {
  if (!old && c) listen(true)
  if (old && !c) listen(false)
  if (!c) return
  pos.value = null
  load(c.id)
  await nextTick()
  root.value?.showPopover()
  place()
  closeBtn.value?.focus()
})
</script>

<template>
  <Teleport v-if="current" :key="current.token" :to="target">
    <div
      ref="root"
      popover="manual"
      class="m-0 overflow-visible border-0 bg-transparent p-0 text-ink"
      :class="sheet ? 'fixed inset-0 size-full max-h-none max-w-none' : 'fixed inset-auto'"
      :style="
        sheet
          ? undefined
          : { top: `${pos?.top ?? 0}px`, left: `${pos?.left ?? 0}px`, visibility: pos ? undefined : 'hidden' }
      "
    >
      <div v-if="sheet" class="absolute inset-0 bg-scrim" aria-hidden="true" />
      <Transition
        appear
        enter-from-class="opacity-0 translate-y-1"
        enter-active-class="transition duration-150 ease-out motion-reduce:transition-none"
      >
        <div
          ref="card"
          role="dialog"
          :aria-labelledby="titleId"
          class="border-line bg-surface shadow-dialog"
          :class="
            sheet
              ? 'absolute inset-x-0 bottom-0 rounded-t-card border-t px-5 pt-3 pb-[calc(1.25rem+env(safe-area-inset-bottom))]'
              : 'w-72 rounded-card border p-4'
          "
        >
          <div class="flex items-start gap-3">
            <UserAvatar
              :user-id="current.id"
              :name="current.name"
              :src="state.kind === 'ok' ? state.person.avatar_url : null"
              size="lg"
              decorative
            />
            <div class="min-w-0 flex-1 pt-1">
              <h2 :id="titleId" class="font-semibold break-words">
                {{ state.kind === 'ok' ? state.person.name : current.name }}
              </h2>
              <p v-if="state.kind === 'ok'" class="text-sm text-ink-muted">
                {{ roleLabel(state.person.role) }}
              </p>
            </div>
            <button
              ref="closeBtn"
              type="button"
              aria-label="Tutup profil"
              class="-mt-2 -mr-2 inline-flex size-11 shrink-0 items-center justify-center rounded-card text-ink-muted hover:bg-canvas hover:text-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
              @click="close(true)"
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
                <path d="M6 6l12 12M18 6L6 18" />
              </svg>
            </button>
          </div>

          <p v-if="state.kind === 'loading'" role="status" class="mt-3 text-sm text-ink-muted">
            Memuat profil...
          </p>

          <div v-else-if="state.kind === 'error'" class="mt-3">
            <p role="alert" class="text-sm text-danger">{{ state.message }}</p>
            <button
              type="button"
              class="mt-2 inline-flex min-h-11 items-center rounded-card border border-line px-4 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
              @click="load(current.id)"
            >
              Coba lagi
            </button>
          </div>

          <template v-else>
            <dl class="mt-3 grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 text-sm">
              <dt class="text-ink-muted">Divisi</dt>
              <dd class="break-words">{{ state.person.division_name }}</dd>
              <dt class="text-ink-muted">Jabatan</dt>
              <dd class="break-words" :class="state.person.job_title ? '' : 'text-ink-muted'">
                {{ state.person.job_title ?? 'Jabatan belum diisi' }}
              </dd>
            </dl>
            <p
              v-if="!state.person.is_active"
              class="mt-3 inline-flex items-center gap-1.5 rounded-chip border border-line px-2 py-0.5 text-sm text-ink-muted"
            >
              <svg
                aria-hidden="true"
                viewBox="0 0 24 24"
                class="size-4"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
              >
                <circle cx="12" cy="12" r="9" />
                <path d="M5.6 5.6l12.8 12.8" />
              </svg>
              Akun nonaktif
            </p>
            <RouterLink
              v-else-if="username"
              :to="{ name: 'tasks', query: { pic: username } }"
              class="mt-4 inline-flex min-h-11 w-full items-center justify-center rounded-card border border-line text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
              @click="close(false)"
            >
              Lihat task-nya di papan
            </RouterLink>
          </template>
        </div>
      </Transition>
    </div>
  </Teleport>
</template>
