<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PageHeader from '../components/PageHeader.vue'
import TimelineCalendar from '../components/TimelineCalendar.vue'
import TimelineAgenda from '../components/TimelineAgenda.vue'
import TimelineDayPopover from '../components/TimelineDayPopover.vue'
import TimelineEventPanel from '../components/TimelineEventPanel.vue'
import TimelineNotes from '../components/TimelineNotes.vue'
import TimelineMarker from '../components/TimelineMarker.vue'
import { ApiError } from '../api/client'
import { listPeople, type Person } from '../api/board'
import { listTimelineEvents } from '../api/timeline'
import { officeToday } from '../utils/recurrence'
import {
  categories,
  categoryLabel,
  isMonth,
  monthLabel,
  shiftMonth,
  type TimelineCategory,
  type TimelineEvent,
} from '../utils/timeline'

// Timeline (revisi 30): kalender bersama launching, produksi, dan kehadiran, plus Important Notes. Semua peran
// membaca; tombol tulis hanya muncul bila server mengirim can_write (Manajer).
const route = useRoute()
const router = useRouter()
const today = officeToday()
const thisMonth = today.slice(0, 7)

const month = computed(() => (isMonth(route.query.bulan) ? route.query.bulan : thisMonth))
function goMonth(m: string) {
  router.replace({ query: { ...route.query, bulan: m === thisMonth ? undefined : m } })
}

const events = shallowRef<TimelineEvent[]>([])
const canWrite = ref(false)
const state = ref<'loading' | 'error' | 'ok'>('loading')
const loadError = ref('')
let abort: AbortController | null = null

async function load() {
  abort?.abort()
  const ctrl = new AbortController()
  abort = ctrl
  state.value = 'loading'
  try {
    const res = await listTimelineEvents(month.value, ctrl.signal)
    events.value = res.events
    canWrite.value = res.can_write
    state.value = 'ok'
  } catch (e) {
    if (ctrl.signal.aborted) return
    loadError.value =
      e instanceof ApiError ? e.message : 'Jadwal gagal dimuat. Periksa koneksi, lalu coba lagi.'
    state.value = 'error'
  }
}
watch(month, load, { immediate: true })
onBeforeUnmount(() => abort?.abort())

// Filter kategori: tiga toggle, bawaan semua menyala. Hanya menyaring tampilan grid dan agenda; popover tanggal
// tetap menampilkan semua entri hari itu.
const shown = ref<Record<TimelineCategory, boolean>>({ launching: true, produksi: true, kehadiran: true })
const visible = computed(() => events.value.filter((e) => shown.value[e.category]))
const allHidden = computed(() => categories.every((c) => !shown.value[c]))

// Grid mulai 768 px, agenda di bawahnya. Dipilih lewat matchMedia, bukan CSS saja, supaya hanya satu yang ada di
// DOM: dua daftar tanggal yang sama-sama bisa difokus akan membingungkan pembaca layar.
const wide = ref(true)
let mq: MediaQueryList | null = null
function syncWide() {
  wide.value = !!mq?.matches
}
onMounted(() => {
  mq = window.matchMedia('(min-width: 768px)')
  syncWide()
  mq.addEventListener('change', syncWide)
})
onBeforeUnmount(() => mq?.removeEventListener('change', syncWide))

const calendar = ref<InstanceType<typeof TimelineCalendar> | null>(null)
const agenda = ref<HTMLElement | null>(null)

const popDate = ref<string | null>(null)
const popTrigger = ref<HTMLElement | null>(null)
function openDay(date: string, trigger: HTMLElement) {
  popDate.value = date
  popTrigger.value = trigger
}
function closeDay(returnFocus: boolean) {
  const date = popDate.value
  popDate.value = null
  if (returnFocus && date) focusDate(date)
}

function focusDate(date: string) {
  if (wide.value) return calendar.value?.focusDate(date)
  nextTick(() => agenda.value?.querySelector<HTMLElement>(`[data-date="${date}"]`)?.focus())
}

// Panel form. Sumber fokus diingat supaya Batal atau simpan mengembalikan fokus ke tanggal atau tombol asalnya.
const panelOpen = ref(false)
const panelEvent = ref<TimelineEvent | null>(null)
const panelDate = ref(today)
const panelReturn = ref<string | null>(null)
const createBtn = ref<HTMLButtonElement | null>(null)
const notice = ref('')

const people = ref<Person[] | null>(null)
const peopleError = ref('')
async function loadPeople() {
  if (people.value) return
  peopleError.value = ''
  try {
    people.value = (await listPeople()).slice().sort((a, b) => a.name.localeCompare(b.name, 'id'))
  } catch {
    peopleError.value = 'Daftar akun gagal dimuat. Tutup form, lalu buka lagi.'
  }
}

function openPanel(ev: TimelineEvent | null, date: string, returnTo: string | null) {
  popDate.value = null
  panelEvent.value = ev
  panelDate.value = date
  panelReturn.value = returnTo
  notice.value = ''
  panelOpen.value = true
  loadPeople()
}

function closePanel() {
  panelOpen.value = false
  if (panelReturn.value) focusDate(panelReturn.value)
  else nextTick(() => createBtn.value?.focus())
}

async function onSaved(date: string, message: string) {
  notice.value = message
  panelOpen.value = false
  if (date.slice(0, 7) !== month.value) goMonth(date.slice(0, 7))
  else await load()
  focusDate(panelReturn.value ?? date)
}

async function onDeleted() {
  notice.value = 'Entri dihapus.'
  await load()
}

const navBtn =
  'inline-flex min-h-11 items-center justify-center rounded-card border border-line px-3 font-medium hover:bg-canvas active:translate-y-px'
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Timeline" description="Jadwal launching, produksi, dan kehadiran tim." />

    <p v-if="notice" role="status" class="rounded-card border border-line bg-surface px-3 py-2 text-sm">
      {{ notice }}
    </p>

    <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_20rem] lg:items-start">
      <section aria-labelledby="timeline-month" class="min-w-0 space-y-4">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h2 id="timeline-month" class="text-2xl font-bold first-letter:uppercase" aria-live="polite">
            {{ monthLabel(month) }}
          </h2>
          <div class="flex flex-wrap gap-2">
            <button
              type="button"
              :class="[navBtn, 'w-11 px-0']"
              aria-label="Bulan sebelumnya"
              @click="goMonth(shiftMonth(month, -1))"
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
                <path d="M15 6l-6 6 6 6" />
              </svg>
            </button>
            <button type="button" :class="navBtn" :disabled="month === thisMonth" @click="goMonth(thisMonth)">
              Bulan ini
            </button>
            <button
              type="button"
              :class="[navBtn, 'w-11 px-0']"
              aria-label="Bulan berikutnya"
              @click="goMonth(shiftMonth(month, 1))"
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
                <path d="M9 6l6 6-6 6" />
              </svg>
            </button>
          </div>
        </div>

        <div role="group" aria-label="Tampilkan kategori" class="flex flex-wrap gap-2">
          <button
            v-for="c in categories"
            :key="c"
            type="button"
            :aria-pressed="shown[c]"
            class="inline-flex min-h-11 items-center gap-2 rounded-card border border-line px-3 text-sm font-medium text-ink-muted hover:bg-canvas aria-pressed:border-mark aria-pressed:text-ink"
            @click="shown[c] = !shown[c]"
          >
            <!-- Tanda centang ikut menunjukkan status toggle, supaya tidak hanya bergantung pada warna garis. -->
            <svg
              aria-hidden="true"
              viewBox="0 0 24 24"
              class="size-4"
              :class="shown[c] ? '' : 'invisible'"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M5 12l5 5 9-10" />
            </svg>
            <TimelineMarker :event="{ category: c, user: { id: 0, name: categoryLabel[c] } }" />
            {{ categoryLabel[c] }}
          </button>
        </div>

        <p v-if="state === 'loading'" role="status" class="text-sm text-ink-muted">Memuat jadwal...</p>
        <div v-else-if="state === 'error'" class="rounded-card bg-danger-bg p-3 text-sm text-danger">
          <p role="alert">{{ loadError }}</p>
          <button
            type="button"
            class="mt-2 inline-flex min-h-11 items-center rounded-card border border-line bg-surface px-4 font-medium text-ink hover:bg-canvas"
            @click="load"
          >
            Muat ulang
          </button>
        </div>
        <p v-else-if="allHidden" class="text-sm text-ink-muted">
          Semua kategori disembunyikan. Nyalakan salah satu di atas.
        </p>
        <p v-else-if="!visible.length" class="text-sm text-ink-muted">Belum ada jadwal di bulan ini.</p>

        <TimelineCalendar
          v-if="wide"
          ref="calendar"
          :month="month"
          :today="today"
          :events="state === 'ok' ? visible : null"
          @open="openDay"
          @month="goMonth"
        />
        <div v-else-if="state === 'ok'" ref="agenda">
          <TimelineAgenda :month="month" :today="today" :events="visible" @open="openDay" />
        </div>
      </section>

      <aside class="min-w-0 space-y-4" aria-label="Kartu dan catatan">
        <button
          v-if="canWrite"
          ref="createBtn"
          type="button"
          class="inline-flex min-h-11 w-full items-center justify-center rounded-card bg-accent px-4 font-medium text-on-accent hover:bg-accent-hover active:translate-y-px"
          @click="openPanel(null, today.startsWith(month) ? today : `${month}-01`, null)"
        >
          Buat kartu
        </button>
        <TimelineNotes />
      </aside>
    </div>

    <TimelineDayPopover
      :date="popDate"
      :trigger="popTrigger"
      :events="events"
      :can-write="canWrite"
      @close="closeDay"
      @add="(d) => openPanel(null, d, d)"
      @edit="(ev) => openPanel(ev, ev.start_date, popDate)"
      @deleted="onDeleted"
    />
    <TimelineEventPanel
      :open="panelOpen"
      :event="panelEvent"
      :date="panelDate"
      :people="people"
      :people-error="peopleError"
      @close="closePanel"
      @saved="onSaved"
    />
  </div>
</template>
