<script setup lang="ts">
import { computed, nextTick, ref, useId, watch } from 'vue'
import TimelineMarker from './TimelineMarker.vue'
import { ApiError } from '../api/client'
import { deleteTimelineEvent } from '../api/timeline'
import { dayLabel, rangeLabel } from '../utils/activityWeek'
import { categoryLabel, eventCovers, eventSummary, type TimelineEvent } from '../utils/timeline'

// Popover satu tanggal, pola PersonPopover (Sprint 11l): popover="manual" di top layer, menempel di sel mulai
// 640 px dan menjadi lembar dari bawah di bawahnya, klik luar dan Escape menutup lalu fokus kembali ke sel.
const props = defineProps<{
  date: string | null
  trigger: HTMLElement | null
  events: TimelineEvent[]
  canWrite: boolean
}>()
const emit = defineEmits<{
  close: [returnFocus: boolean]
  add: [date: string]
  edit: [event: TimelineEvent]
  deleted: []
}>()

const root = ref<HTMLElement | null>(null)
const card = ref<HTMLElement | null>(null)
const closeBtn = ref<HTMLButtonElement | null>(null)
const titleId = useId()
const sheet = ref(false)
const pos = ref<{ top: number; left: number } | null>(null)
const confirming = ref<number | null>(null)
const deleting = ref(false)
const error = ref('')

const list = computed(() => (props.date ? props.events.filter((e) => eventCovers(e, props.date!)) : []))

const GAP = 8
function place() {
  if (!props.trigger || !card.value) return
  sheet.value = !window.matchMedia('(min-width: 640px)').matches
  if (sheet.value) return
  const r = props.trigger.getBoundingClientRect()
  const w = card.value.offsetWidth
  const h = card.value.offsetHeight
  const vw = document.documentElement.clientWidth
  const vh = window.innerHeight
  let left = r.left + w > vw - GAP ? r.right - w : r.left
  left = Math.min(Math.max(GAP, left), vw - w - GAP)
  let top = r.bottom + GAP
  if (top + h > vh - GAP) top = r.top - h - GAP
  top = Math.max(GAP, Math.min(top, vh - h - GAP))
  pos.value = { top, left }
}

function onKeydown(e: KeyboardEvent) {
  if (e.key !== 'Escape') return
  e.preventDefault()
  e.stopPropagation()
  // Escape bertahap: konfirmasi hapus dulu, baru popover-nya.
  if (confirming.value !== null) {
    confirming.value = null
    return
  }
  emit('close', true)
}

function onPointerdown(e: PointerEvent) {
  if (e.target instanceof Node && card.value?.contains(e.target)) return
  // Klik di sel lain membuka tanggal itu lewat penangan sel sendiri; popover ini cukup ditutup tanpa menarik fokus.
  emit('close', !(e.target instanceof Element && e.target.closest('[data-date]')))
}

function onFocusin(e: FocusEvent) {
  const t = e.target
  if (t instanceof Node && (card.value?.contains(t) || t === props.trigger)) return
  emit('close', false)
}

function onReflow() {
  place()
}

function listen(on: boolean) {
  const m = on ? 'addEventListener' : 'removeEventListener'
  document[m]('keydown', onKeydown as EventListener, true)
  document[m]('pointerdown', onPointerdown as EventListener, true)
  document[m]('focusin', onFocusin as EventListener, true)
  window[m]('resize', onReflow)
  window[m]('scroll', onReflow, true)
}

watch(
  () => props.date,
  async (d, old) => {
    if (!old && d) listen(true)
    if (old && !d) listen(false)
    confirming.value = null
    error.value = ''
    if (!d) return
    pos.value = null
    await nextTick()
    root.value?.showPopover()
    place()
    // Satu render lagi: sebelum posisi tertulis, popover masih tersembunyi dan focus() ditolak (Sprint 11o).
    await nextTick()
    closeBtn.value?.focus()
  },
)

// Isi berubah (mis. setelah hapus) mengubah tinggi kartu, jadi posisinya dihitung ulang.
watch(list, async () => {
  await nextTick()
  place()
})

async function remove(ev: TimelineEvent) {
  deleting.value = true
  error.value = ''
  try {
    await deleteTimelineEvent(ev.id)
    // Konfirmasi entri lain yang dibuka selama permintaan ini berjalan jangan ikut tertutup.
    if (confirming.value === ev.id) confirming.value = null
    emit('deleted')
    await nextTick()
    closeBtn.value?.focus()
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : 'Entri gagal dihapus. Periksa koneksi, lalu coba lagi.'
  } finally {
    deleting.value = false
  }
}

function rangeText(ev: TimelineEvent) {
  return ev.end_date ? rangeLabel(ev.start_date, ev.end_date) : null
}

const btn =
  'inline-flex min-h-11 items-center rounded-card border border-line px-3 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-60'
</script>

<template>
  <Teleport v-if="date" to="body">
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
          class="flex flex-col border-line bg-surface shadow-dialog"
          :class="
            sheet
              ? 'absolute inset-x-0 bottom-0 max-h-[85dvh] rounded-t-card border-t px-5 pt-3 pb-[calc(1.25rem+env(safe-area-inset-bottom))]'
              : 'max-h-[min(32rem,calc(100dvh-1rem))] w-80 rounded-card border p-4'
          "
        >
          <div class="flex items-start gap-2">
            <h2 :id="titleId" class="min-w-0 flex-1 pt-2 font-semibold">{{ dayLabel(date) }}</h2>
            <button
              ref="closeBtn"
              type="button"
              aria-label="Tutup"
              class="-mr-2 inline-flex size-11 shrink-0 items-center justify-center rounded-card text-ink-muted hover:bg-canvas hover:text-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
              @click="emit('close', true)"
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

          <p v-if="error" role="alert" class="mt-2 rounded-card bg-danger-bg p-2 text-sm text-danger">
            {{ error }}
          </p>

          <div class="-mx-1 mt-2 min-h-0 flex-1 overflow-y-auto px-1">
            <p v-if="!list.length" class="text-sm text-ink-muted">Belum ada entri di tanggal ini.</p>
            <ul v-else class="divide-y divide-line/40">
              <li v-for="ev in list" :key="ev.id" class="py-2">
                <p class="flex items-center gap-1.5 text-sm text-ink-muted">
                  <TimelineMarker :event="ev" />
                  {{ categoryLabel[ev.category] }}
                </p>
                <p class="mt-1 font-semibold [overflow-wrap:anywhere]">{{ eventSummary(ev) }}</p>
                <p v-if="rangeText(ev)" class="text-sm text-ink-muted">{{ rangeText(ev) }}</p>
                <p v-if="ev.description" class="mt-1 text-sm whitespace-pre-line [overflow-wrap:anywhere]">
                  {{ ev.description }}
                </p>
                <div v-if="canWrite" class="mt-2 flex flex-wrap gap-2">
                  <template v-if="confirming === ev.id">
                    <p class="w-full text-sm">Hapus entri ini? Tindakan ini tidak bisa diurungkan.</p>
                    <button
                      type="button"
                      :disabled="deleting"
                      class="inline-flex min-h-11 items-center rounded-card bg-urgent px-3 text-sm font-medium text-on-urgent hover:opacity-90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-60"
                      @click="remove(ev)"
                    >
                      {{ deleting ? 'Menghapus...' : 'Ya, hapus' }}
                    </button>
                    <button type="button" :class="btn" @click="confirming = null">Batal</button>
                  </template>
                  <template v-else>
                    <button
                      type="button"
                      :class="btn"
                      :aria-label="`Ubah ${eventSummary(ev)}`"
                      @click="emit('edit', ev)"
                    >
                      Ubah
                    </button>
                    <button
                      type="button"
                      :class="btn"
                      :aria-label="`Hapus ${eventSummary(ev)}`"
                      @click="confirming = ev.id"
                    >
                      Hapus
                    </button>
                  </template>
                </div>
              </li>
            </ul>
          </div>

          <button
            v-if="canWrite"
            type="button"
            class="mt-3 inline-flex min-h-11 w-full items-center justify-center rounded-card bg-accent px-4 text-sm font-medium text-on-accent hover:bg-accent-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
            @click="emit('add', date)"
          >
            Tambah di tanggal ini
          </button>
        </div>
      </Transition>
    </div>
  </Teleport>
</template>
