<script setup lang="ts">
import { computed, onMounted, ref, useId } from 'vue'
import { RouterLink } from 'vue-router'
import LoadingSkeleton from './LoadingSkeleton.vue'
import ActivityCellFields from './ActivityCellFields.vue'
import ActivityBox from './ActivityBox.vue'
import ActivityFiles from './ActivityFiles.vue'
import ActivityChainButton from './ActivityChainButton.vue'
import ActivitySlotStatus from './ActivitySlotStatus.vue'
import { getMyActivities, linkActivity, type ActivityCell } from '../api/activities'
import { ApiError } from '../api/client'
import { useActivitySave } from '../composables/useActivitySave'
import { filledHours } from '../utils/activity'
import { pasteName } from '../utils/activityFile'
import { formatLong } from '../utils/recurrence'
import { dayLabel, shiftDay } from '../utils/activityWeek'

// Kartu Aktivitas di papan task (revisi 21), hanya untuk Staf. Revisi 37: bisa mundur sampai 2 hari kerja; daftar
// tanggal yang boleh diisi datang dari server (editable_dates), frontend tidak menghitungnya sendiri.
const id = useId()
const lipatKey = 'nvm-aktivitas-lipat'

const open = ref(true)
const date = ref('')
const today = ref('')
const editable = ref<string[]>([])
const cells = ref<ActivityCell[]>([])
// Nilai terakhir dari server, jadi sel yang tidak berubah tidak ikut dikirim saat kehilangan fokus.
const saved = ref<Record<number, { activity: string; output: string }>>({})
const loadState = ref<'loading' | 'ready' | 'error'>('loading')
const loadError = ref('')

const terisi = computed(() => filledHours(cells.value))
const { status, message, save, retry } = useActivitySave((_userId, slot, cell, day, savedDate) => {
  // Simpan saat kotak kehilangan fokus bisa selesai setelah tanggal diganti; hasilnya milik tanggal lama.
  if (savedDate !== date.value) return
  const i = cells.value.findIndex((c) => c.slot === slot)
  if (i === -1) return
  cells.value[i] = cell
  saved.value[slot] = { activity: cell.activity, output: cell.output }
  applyDay(day)
})

// Jawaban server untuk satu hari (revisi 33). Jam yang tersambung atau baru dilepas diganti utuh, karena isinya
// milik server. Jam lain hanya diperbarui data rantainya, supaya teks yang sedang diketik di sana tidak hilang.
function applyDay(day: ActivityCell[]) {
  day.forEach((fresh, i) => {
    const c = cells.value[i]
    if (!c || c.slot !== fresh.slot) return
    if (fresh.linked || c.linked) {
      cells.value[i] = fresh
      saved.value[fresh.slot] = { activity: fresh.activity, output: fresh.output }
      return
    }
    Object.assign(c, {
      head: fresh.head,
      span: fresh.span,
      joinable: fresh.joinable,
      link_hint: fresh.link_hint,
    })
  })
}

// Rantai sambung jam (revisi 33).
const linking = ref(0)
const linkError = ref('')

async function toggleLink(c: ActivityCell) {
  linking.value = c.slot
  linkError.value = ''
  try {
    applyDay((await linkActivity({ date: date.value, slot: c.slot, linked: !c.linked })).cells)
  } catch (err) {
    linkError.value = err instanceof ApiError ? err.message : 'Tidak bisa mengubah sambungan. Coba lagi.'
  } finally {
    linking.value = 0
  }
}

const headTime = (c: ActivityCell) => cells.value.find((x) => x.slot === c.head)?.time ?? ''

// Tombol rantai hanya untuk jam yang benar-benar bisa disambung sekarang, atau yang sudah tersambung. Jam
// yang akan ditolak server (link_hint berisi alasannya) tidak lagi memunculkan tombol mati (revisi 33b).
const chainable = (c: ActivityCell) => c.linked || (c.joinable && c.link_hint === '')

function remember(list: ActivityCell[]) {
  cells.value = list
  saved.value = Object.fromEntries(list.map((c) => [c.slot, { activity: c.activity, output: c.output }]))
}

async function load(day = date.value) {
  loadState.value = 'loading'
  try {
    const res = await getMyActivities(day)
    date.value = res.date
    today.value = res.today
    editable.value = res.editable_dates
    status.value = {}
    message.value = {}
    remember(res.cells)
    loadState.value = 'ready'
  } catch (err) {
    loadError.value = err instanceof ApiError ? err.message : 'Tidak bisa memuat aktivitas.'
    loadState.value = 'error'
  }
}

// editable_dates urut dari hari ini mundur, jadi "sebelumnya" adalah elemen berikutnya di daftar.
const at = computed(() => editable.value.indexOf(date.value))
const prevDate = computed(() => (at.value >= 0 ? (editable.value[at.value + 1] ?? '') : ''))
const nextDate = computed(() => (at.value > 0 ? editable.value[at.value - 1] : ''))
const isToday = computed(() => date.value !== '' && date.value === today.value)
const title = computed(() => {
  if (!date.value || isToday.value) return 'Aktivitas hari ini'
  return date.value === shiftDay(today.value, -1) ? 'Aktivitas kemarin' : 'Aktivitas'
})

function go(day: string) {
  if (!day || loadState.value === 'loading') return
  void load(day)
}

// Sel disimpan saat kehilangan fokus, hanya kalau isinya memang berubah.
function blur(cell: ActivityCell) {
  // Jam tersambung tidak punya isi sendiri; server menolaknya 409, jadi tidak perlu dikirim sama sekali.
  if (cell.linked) return
  const before = saved.value[cell.slot]
  if (before && before.activity === cell.activity && before.output === cell.output) return
  void save(0, date.value, cell.slot, cell.activity, cell.output)
}

// Urutan Tab di tabel yang diputar: per jam, Aktivitas lalu Output, bukan seluruh baris Aktivitas dulu.
// Tabel selalu row-major di DOM, jadi urutannya diatur di sini. Tanpa tabindex positif: Tab dari sel
// terakhir dibiarkan keluar dari tabel seperti biasa.
const tableEl = ref<HTMLElement | null>(null)

function onTab(e: KeyboardEvent) {
  const el = e.target
  if (!(el instanceof HTMLTextAreaElement) || !tableEl.value) return
  const order = [...tableEl.value.querySelectorAll<HTMLTextAreaElement>('textarea')].sort(
    (a, b) => tabRank(a) - tabRank(b),
  )
  const next = order[order.indexOf(el) + (e.shiftKey ? -1 : 1)]
  if (!next) return
  e.preventDefault()
  next.focus()
}

function tabRank(el: HTMLTextAreaElement): number {
  return Number(el.dataset.slot) * 2 + (el.dataset.kind === 'output' ? 1 : 0)
}

// Lampiran per jam (revisi 24). Acuan ke tiap komponen ActivityFiles supaya drag and drop dan paste bisa
// menyerahkan berkasnya ke jam yang benar, tanpa menyalin logika unggah ke sini.
type FilesHandle = { add: (files: File[]) => void }
const fileBoxes = ref<Record<number, FilesHandle | null>>({})
const dragSlot = ref(0)

function setFileBox(slot: number, el: unknown) {
  fileBoxes.value[slot] = (el as FilesHandle | null) ?? null
}

function dropFiles(e: DragEvent, slot: number) {
  dragSlot.value = 0
  const files = Array.from(e.dataTransfer?.files ?? [])
  if (files.length) fileBoxes.value[slot]?.add(files)
}

function dragLeave(e: DragEvent, slot: number) {
  // dragleave juga terpicu saat kursor pindah ke elemen di dalam sel; hanya keluar sel yang dihitung.
  if ((e.currentTarget as HTMLElement).contains(e.relatedTarget as Node | null)) return
  if (dragSlot.value === slot) dragSlot.value = 0
}

// Tangkapan layar yang ditempel (Ctrl+V) langsung jadi lampiran jam itu. Teks biasa tidak disentuh, jadi
// paste teks ke textarea tetap bekerja seperti biasa.
function onPaste(e: ClipboardEvent, c: ActivityCell) {
  const images = Array.from(e.clipboardData?.files ?? []).filter((f) => f.type.startsWith('image/'))
  if (images.length === 0) return
  e.preventDefault()
  const named = images.map((f, i) => {
    const ext = f.type === 'image/jpeg' ? 'jpg' : f.type === 'image/webp' ? 'webp' : 'png'
    return new File([f], pasteName(c.time, c.files.length + i + 1, ext), { type: f.type })
  })
  fileBoxes.value[c.slot]?.add(named)
}

const cellDrop = 'rounded-card transition-colors motion-reduce:transition-none'
const navButton =
  'inline-flex size-11 shrink-0 items-center justify-center rounded-card text-ink hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:cursor-not-allowed disabled:text-ink-muted disabled:hover:bg-transparent'

function toggle() {
  open.value = !open.value
  try {
    localStorage.setItem(lipatKey, open.value ? 'buka' : 'lipat')
  } catch {
    // Mode privat atau penyimpanan diblokir: status lipat cukup berlaku untuk kunjungan ini.
  }
}

onMounted(() => {
  try {
    open.value = localStorage.getItem(lipatKey) !== 'lipat'
  } catch {
    open.value = true
  }
  void load()
})
</script>

<template>
  <section class="mt-4 rounded-card border border-line bg-surface">
    <!-- Tombol tanggal berdiri di samping tombol lipat, bukan di dalamnya: tombol tidak boleh bersarang. -->
    <div class="flex items-center gap-1 pr-2">
      <h2 class="min-w-0 flex-1">
        <button
          type="button"
          class="flex min-h-11 w-full flex-wrap items-center justify-between gap-x-3 gap-y-1 px-4 py-2 text-left font-medium focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
          :aria-expanded="open"
          :aria-controls="id"
          @click="toggle"
        >
          <span class="flex items-center gap-2">
            <svg
              aria-hidden="true"
              viewBox="0 0 24 24"
              class="size-5 shrink-0 transition-transform motion-reduce:transition-none"
              :class="open ? 'rotate-90' : ''"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M9 6l6 6-6 6" />
            </svg>
            {{ title }}<template v-if="date">,</template>
            <span v-if="date" class="font-normal text-ink-muted">{{ formatLong(date) }}</span>
          </span>
          <span class="text-sm font-normal text-ink-muted"
            >{{ terisi }} dari {{ cells.length || 8 }} jam terisi</span
          >
        </button>
      </h2>
      <button
        type="button"
        :class="navButton"
        :disabled="!prevDate || loadState === 'loading'"
        :aria-label="
          prevDate
            ? `Aktivitas sebelumnya, ${dayLabel(prevDate)}`
            : 'Aktivitas sebelumnya, sudah batas paling awal'
        "
        @click="go(prevDate)"
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
      <button
        type="button"
        :class="navButton"
        :disabled="!nextDate || loadState === 'loading'"
        :aria-label="
          nextDate ? `Aktivitas berikutnya, ${dayLabel(nextDate)}` : 'Aktivitas berikutnya, sudah hari ini'
        "
        @click="go(nextDate)"
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

    <div v-show="open" :id="id" class="border-t border-line p-4">
      <!-- Tanggal lalu ditandai jelas, supaya tidak ada yang mengira sedang mengisi hari ini. -->
      <p
        v-if="date && !isToday"
        role="status"
        class="mb-3 rounded-card border border-line bg-canvas px-3 py-2 text-sm font-medium"
      >
        Kamu sedang mengisi aktivitas {{ dayLabel(date) }}.
      </p>
      <LoadingSkeleton v-if="loadState === 'loading'" label="Memuat aktivitas..." />

      <div v-else-if="loadState === 'error'" role="alert" class="rounded-card bg-danger-bg p-4 text-danger">
        <p class="font-medium">{{ loadError }}</p>
        <button
          type="button"
          class="mt-3 inline-flex min-h-11 items-center rounded-card border border-danger px-4 font-medium"
          @click="load()"
        >
          Muat ulang
        </button>
      </div>

      <template v-else>
        <p v-if="terisi === 0" class="mb-3 text-sm text-ink-muted">
          Belum ada jam yang terisi. Tulis apa yang kamu kerjakan dan hasilnya, lalu klik di luar kotak untuk
          menyimpan.
        </p>
        <p v-if="linkError" role="alert" class="mb-3 rounded-card bg-danger-bg p-3 text-danger">
          {{ linkError }}
        </p>
        <!-- Mulai 768 px tabelnya diputar: jam jadi kepala kolom, jadi kartu ini hanya tiga baris tinggi
             (revisi 21). Wadahnya yang bergulir ke samping bila delapan kolom tidak muat, bukan halamannya;
             kolom label tetap terlihat karena lengket di kiri. -->
        <div ref="tableEl" class="hidden overflow-x-auto md:block" @keydown.tab="onTab">
          <table class="w-full table-fixed border-collapse text-left min-w-[87rem]">
            <thead>
              <tr>
                <th scope="col" class="sticky left-0 z-10 w-28 bg-surface px-2 pb-2">
                  <span class="sr-only">Isian</span>
                </th>
                <!-- pl-6 pr-4 berlaku di semua kolom jam: celah tetap tempat tombol rantai duduk, jadi
                     kolom tanpa tombol pun tidak bergeser (revisi 33b). -->
                <th
                  v-for="c in cells"
                  :key="c.slot"
                  scope="col"
                  class="pb-2 pl-6 pr-4 text-sm font-medium whitespace-nowrap"
                >
                  {{ c.time }}
                  <!-- Label Lanjutan dan status simpan ada di kepala kolom, bukan di dalam selnya: di dalam
                       sel keduanya memakan baris dan membuat tinggi kolom melompat. Sebagai kepala kolom
                       (scope="col") baris ini juga ikut dibacakan untuk kedua sel di bawahnya, dan id-nya
                       yang dipakai aria-describedby kedua isian. -->
                  <ActivitySlotStatus
                    :id="`${id}-status-${c.slot}`"
                    :time="c.time"
                    :linked="c.linked"
                    :head-time="headTime(c)"
                    :status="status[`0:${c.slot}`]"
                    :message="message[`0:${c.slot}`]"
                    :updated-by="c.updated_by"
                    @retry="retry(0, c.slot)"
                  />
                </th>
              </tr>
            </thead>
            <tbody>
              <tr class="align-top">
                <th scope="row" class="sticky left-0 z-10 bg-surface px-2 py-1 text-left font-medium">
                  Aktivitas
                </th>
                <td
                  v-for="c in cells"
                  :key="c.slot"
                  class="py-1 pl-6 pr-4"
                  :class="[cellDrop, dragSlot === c.slot ? 'bg-canvas outline-2 outline-mark' : '']"
                  @dragover.prevent="dragSlot = c.linked ? 0 : c.slot"
                  @dragleave="dragLeave($event, c.slot)"
                  @drop.prevent="dropFiles($event, c.slot)"
                  @paste="onPaste($event, c)"
                >
                  <div class="relative">
                    <!-- Tombol rantai duduk di celah antara isian ini dan isian jam sebelumnya, di tengah
                         tinggi isian itu, bukan di baris judul (revisi 33b). -->
                    <ActivityChainButton
                      v-if="chainable(c)"
                      :cell="c"
                      :busy="linking === c.slot"
                      class="absolute top-1/2 -left-6 z-10 -translate-x-1/2 -translate-y-1/2"
                      @toggle="toggleLink(c)"
                    />
                    <!-- Jam tersambung memakai isian yang sama dalam keadaan hanya dibaca, jadi tinggi,
                         lebar, dan patahan teksnya sama dengan jam kepalanya. -->
                    <ActivityBox
                      v-model="c.activity"
                      :readonly="c.linked"
                      :num="c.slot"
                      :time="c.time"
                      kind="activity"
                      label="Aktivitas"
                      placeholder="Apa yang dikerjakan"
                      :status-id="`${id}-status-${c.slot}`"
                      @blur="blur(c)"
                    />
                  </div>
                </td>
              </tr>
              <tr class="align-top">
                <th scope="row" class="sticky left-0 z-10 bg-surface px-2 py-1 text-left font-medium">
                  Output
                </th>
                <td
                  v-for="c in cells"
                  :key="c.slot"
                  class="py-1 pl-6 pr-4"
                  :class="[cellDrop, dragSlot === c.slot ? 'bg-canvas outline-2 outline-mark' : '']"
                  @dragover.prevent="dragSlot = c.linked ? 0 : c.slot"
                  @dragleave="dragLeave($event, c.slot)"
                  @drop.prevent="dropFiles($event, c.slot)"
                  @paste="onPaste($event, c)"
                >
                  <ActivityBox
                    v-model="c.output"
                    :readonly="c.linked"
                    :num="c.slot"
                    :time="c.time"
                    kind="output"
                    label="Output"
                    placeholder="Hasil atau tautan"
                    :status-id="`${id}-status-${c.slot}`"
                    @blur="blur(c)"
                  />
                  <ActivityFiles
                    :ref="(el) => setFileBox(c.slot, el)"
                    :user-id="0"
                    :date="date"
                    :num="c.slot"
                    :time="c.time"
                    :files="c.files"
                    :can-manage="!c.linked"
                    compact
                    class="mt-1"
                    @changed="load()"
                  />
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Di bawah 768 px tiap jam tetap jadi kartu bertumpuk; mulai 640 px barisnya sejajar jadi tiga kolom. -->
        <div
          class="mb-2 hidden gap-3 px-1 text-sm font-medium text-ink-muted sm:grid sm:grid-cols-[7rem_minmax(0,1fr)_minmax(0,1fr)] md:hidden"
          aria-hidden="true"
        >
          <span>Jam</span>
          <span>Aktivitas</span>
          <span>Output</span>
        </div>
        <ul class="grid gap-3 md:hidden">
          <li
            v-for="c in cells"
            :key="c.slot"
            class="grid gap-2 rounded-card border border-line p-3 sm:grid-cols-[7rem_minmax(0,1fr)_minmax(0,1fr)] sm:items-start sm:gap-3"
            :class="dragSlot === c.slot ? 'bg-canvas outline-2 outline-mark' : ''"
            @dragover.prevent="dragSlot = c.linked ? 0 : c.slot"
            @dragleave="dragLeave($event, c.slot)"
            @drop.prevent="dropFiles($event, c.slot)"
            @paste="onPaste($event, c)"
          >
            <div class="flex flex-wrap items-center gap-x-2">
              <ActivityChainButton
                v-if="chainable(c)"
                :cell="c"
                :busy="linking === c.slot"
                @toggle="toggleLink(c)"
              />
              <!-- Ruang tombol tetap disisakan, jadi label jam semua kartu tetap sejajar (revisi 33b). -->
              <span v-else class="size-11 shrink-0" aria-hidden="true"></span>
              <p class="font-medium">{{ c.time }}</p>
              <ActivitySlotStatus
                :id="`${id}-status-${c.slot}`"
                :time="c.time"
                :linked="c.linked"
                :head-time="headTime(c)"
                :status="status[`0:${c.slot}`]"
                :message="message[`0:${c.slot}`]"
                :updated-by="c.updated_by"
                class="w-full"
                @retry="retry(0, c.slot)"
              />
            </div>
            <ActivityCellFields
              v-model:activity="c.activity"
              v-model:output="c.output"
              :time="c.time"
              :updated-by="c.updated_by"
              :readonly="c.linked"
              hide-status
              @blur="blur(c)"
            />
            <ActivityFiles
              :ref="(el) => setFileBox(c.slot, el)"
              :user-id="0"
              :date="date"
              :num="c.slot"
              :time="c.time"
              :files="c.files"
              :can-manage="!c.linked"
              class="sm:col-span-3"
              @changed="load()"
            />
          </li>
        </ul>
        <p class="mt-3 text-sm text-ink-muted">
          Hari lain dan isian tim ada di
          <RouterLink
            class="underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
            to="/aktivitas"
          >
            Aktivitas Harian
          </RouterLink>
          .
        </p>
      </template>
    </div>
  </section>
</template>
