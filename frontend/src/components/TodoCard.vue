<script setup lang="ts">
import { computed, nextTick, onMounted, ref, useId } from 'vue'
import { useTodos } from '../composables/useTodos'
import type { TodoItem } from '../api/todos'
import { caretHint } from './whiteboard/caret'

// Kartu To Do (revisi 28). Satu lembar catatan: permukaan krem dengan teks tinta, token yang sama dengan
// label ID task, dan baris-baris dipisah garis tipis seperti buku bergaris. Dipakai kartu di kolom kiri
// papan, panel di bar, dan lembar di ponsel, jadi tampilannya sama di ketiganya.
//
// Warna di kartu ini sengaja hanya memakai token tag (tag dan tag-ink), yang sama di mode terang dan gelap.
// Token mode-peka seperti ink-muted, danger, dan mark dirancang untuk latar yang ikut berganti, jadi di atas
// krem yang tetap terang kontrasnya jatuh di mode gelap.
const uid = useId()
const {
  items,
  state,
  error,
  undoable,
  openCount,
  doneCount,
  full,
  max,
  load,
  add,
  rename,
  toggle,
  remove,
  undo,
  clearUndo,
  clearDone,
  move,
  reorder,
} = useTodos()

const draft = ref('')
const input = ref<HTMLInputElement | null>(null)
const editingId = ref<number | null>(null)
const editDraft = ref('')
// Function ref, bukan ref bernama: isian ini ada di dalam v-for, dan ref bernama di dalam v-for
// dikumpulkan Vue menjadi array, jadi editInput.value.focus bukan fungsi.
const editInput = ref<HTMLTextAreaElement | null>(null)

function setEditInput(el: unknown) {
  editInput.value = el instanceof HTMLTextAreaElement ? el : null
}

// Isian ubah setinggi isinya, supaya judul panjang terbaca utuh dan terbungkus saat diubah (revisi 46).
function grow() {
  const el = editInput.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${el.scrollHeight}px`
}
const dragId = ref<number | null>(null)
// Mode Atur: satu-satunya tempat kontrol urut dan hapus, di semua perangkat (revisi 47).
const manage = ref(false)
// Kelompok Selesai: tertutup sendiri kalau isinya lebih dari tiga, sampai pemakai memutuskan sendiri.
const doneChoice = ref<boolean | null>(null)
const doneOpen = computed(() => doneChoice.value ?? doneCount.value <= 3)

const open = computed(() => items.value.filter((i) => !i.done))
const done = computed(() => items.value.filter((i) => i.done))

onMounted(() => void load())

async function submit() {
  const title = draft.value
  if (!title.trim()) return
  draft.value = ''
  // Fokus tetap di isian supaya beberapa to do bisa ditulis berurutan.
  input.value?.focus()
  await add(title)
}

// Revisi 46: teks tidak diseleksi semua, supaya kata yang mau diganti tetap terlihat. Kursor di posisi dobel
// klik, atau di akhir teks bila dibuka dengan keyboard.
async function startEdit(item: TodoItem, offset?: number) {
  if (editingId.value === item.id) return
  editingId.value = item.id
  editDraft.value = item.title
  await nextTick()
  const el = editInput.value
  if (!el) return
  grow()
  const at = offset === undefined ? el.value.length : Math.max(0, Math.min(el.value.length, offset))
  el.focus()
  el.setSelectionRange(at, at)
}

function editAt(item: TodoItem, e: MouseEvent) {
  void startEdit(item, caretHint(e.clientX, e.clientY, e.target).offset)
}

// Klik mouse sekali tidak membuka isian: klik kedua dari dobel klik akan jatuh di isian dan menyeleksi satu
// kata. Klik dari Enter atau Spasi (detail 0) tetap membuka isian.
function onTitleClick(item: TodoItem, e: MouseEvent) {
  if (e.detail === 0) void startEdit(item)
}

// Dobel ketuk di layar sentuh: dua ketukan di judul yang sama dalam 350 ms. dblclick tidak selalu dikirim
// peramban sentuh; bila tetap dikirim, startEdit mengabaikan panggilan kedua.
let lastTap = { id: 0, at: 0 }
function onTitleTap(item: TodoItem, e: PointerEvent) {
  if (e.pointerType !== 'touch') return
  if (lastTap.id === item.id && e.timeStamp - lastTap.at < 350) {
    lastTap = { id: 0, at: 0 }
    void startEdit(item, caretHint(e.clientX, e.clientY, e.target).offset).then(keepCaret)
  } else lastTap = { id: item.id, at: e.timeStamp }
}

// Gestur dobel ketuk bawaan peramban ikut sampai ke isian yang baru muncul di bawah jari dan menyeleksi satu
// kata. Selama 600 ms, seleksi dari gestur itu dikembalikan ke posisi kursor yang dipilih.
function keepCaret() {
  const el = editInput.value
  if (!el) return
  const at = el.selectionStart
  const restore = () => el.setSelectionRange(at, at)
  for (const t of ['dblclick', 'select']) el.addEventListener(t, restore)
  setTimeout(() => {
    for (const t of ['dblclick', 'select']) el.removeEventListener(t, restore)
  }, 600)
}

// Enter menyimpan (judul to do satu baris), bukan membuat baris baru.
function onEditKey(e: KeyboardEvent, item: TodoItem) {
  if (e.key === 'Enter' && !e.isComposing) {
    e.preventDefault()
    void commitEdit(item)
  }
}

async function commitEdit(item: TodoItem) {
  // Dipanggil juga dari blur. Setelah Escape mode ubah sudah tertutup, jadi di situ tidak ada yang disimpan.
  if (editingId.value !== item.id) return
  // Baris baru dari teks yang ditempel menjadi spasi.
  const title = editDraft.value.replace(/\s*\n\s*/g, ' ')
  editingId.value = null
  await rename(item.id, title)
}

// Escape: judul semula kembali (editDraft tidak pernah dikirim), keluar dari mode ubah, dan fokus kembali
// ke tombol judul. Perambatannya dihentikan di template, supaya Escape ini tidak sampai ke pendengar
// keydown di document milik TodoPanel dan menutup panelnya. Escape berikutnya baru menutup panel.
async function cancelEdit(item: TodoItem) {
  editingId.value = null
  await nextTick()
  document.getElementById(`${uid}-title-${item.id}`)?.focus()
}

function onDrop(target: TodoItem) {
  const id = dragId.value
  dragId.value = null
  if (id === null || id === target.id) return
  const order = items.value.map((i) => i.id).filter((i) => i !== id)
  const at = order.indexOf(target.id)
  order.splice(at === -1 ? order.length : at, 0, id)
  void reorder(order)
}

// Garis tipis dan teks redup di atas krem: tag-ink dengan transparansi, bukan ink-muted atau line, supaya
// angkanya tetap sama di mode gelap. Kontras dihitung di Keputusan Sprint 11y.
const rule = 'border-tag-ink/60'
const muted = 'text-tag-ink/70'
const focusRing = 'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-tag-ink'
const iconButton = `inline-flex size-11 shrink-0 items-center justify-center rounded-card hover:bg-tag-ink/10 ${focusRing} disabled:opacity-40`
// min-w-11: "Atur" dengan px-2 hanya 41 px lebarnya, di bawah tap target 44 px.
const headButton = `inline-flex min-h-11 min-w-11 items-center justify-center rounded-card px-2 text-sm font-medium hover:bg-tag-ink/10 ${focusRing}`
// Revisi 47: kontrol urut dan hapus hanya tampil di mode Atur, di barisnya sendiri di bawah teks. Lapisan hover
// lama menutupi ujung teks panjang; ruang tetap di kanan tidak muat di kartu 17rem (lima tombol 44 px).
const rowControls = 'flex items-center justify-end gap-1 pb-1'

// Judul yang memuat URL http atau https mendapat ikon buka tautan. Judulnya sendiri tetap teks biasa.
function titleUrl(title: string): string | null {
  const m = title.match(/https?:\/\/\S+/i)
  if (!m) return null
  try {
    const u = new URL(m[0])
    return u.protocol === 'http:' || u.protocol === 'https:' ? u.href : null
  } catch {
    return null
  }
}
</script>

<template>
  <section class="overflow-hidden rounded-card bg-tag text-tag-ink">
    <header class="flex flex-wrap items-center justify-between gap-x-2 px-3 pt-2">
      <h2 class="font-semibold">To Do</h2>
      <!-- Kartu kosong tidak punya yang bisa diringkas atau diatur, jadi keduanya tidak ditampilkan. -->
      <div v-if="items.length" class="flex items-center gap-1">
        <p class="text-sm" :class="muted">{{ openCount > 0 ? `${openCount} tersisa` : 'Semua selesai' }}</p>
        <button
          type="button"
          :class="headButton"
          :aria-pressed="manage"
          class="aria-pressed:bg-tag-ink aria-pressed:text-tag"
          @click="manage = !manage"
        >
          Atur
        </button>
      </div>
    </header>

    <p v-if="error" role="alert" class="mx-3 mt-2 border-l-2 border-tag-ink pl-2 text-sm">
      Gagal: {{ error }}
    </p>

    <p v-if="state === 'loading'" class="px-3 py-3 text-sm" :class="muted">Memuat to do...</p>

    <div v-else-if="state === 'error'" class="px-3 py-3 text-sm">
      <p>Daftar to do gagal dimuat.</p>
      <button type="button" :class="[headButton, 'mt-1 border', rule]" @click="load(true)">Muat ulang</button>
    </div>

    <template v-else>
      <!-- Daftar panjang bergulir di dalam kartunya sendiri, jadi kolom papan tidak ikut memanjang. -->
      <ul class="max-h-[22rem] divide-y overflow-y-auto border-t" :class="[rule, 'divide-tag-ink/60']">
        <!-- Kosong: satu kalimat, dan mengkliknya memfokuskan isian di baris bawahnya. Tanpa garis kosong,
             karena garis yang tidak bisa diklik terbaca seperti baris isian yang rusak. aria-label di isian
             tetap menang sebagai nama aksesibelnya, jadi label ini hanya menambah area klik. -->
        <li v-if="items.length === 0" class="px-1">
          <label :for="`${uid}-add`" class="block min-h-11 cursor-text px-2 py-3 text-sm" :class="muted">
            Belum ada to do.
          </label>
        </li>

        <li
          v-for="(item, i) in open"
          :key="item.id"
          draggable="true"
          class="relative px-1"
          :class="dragId === item.id ? 'bg-tag-ink/10' : ''"
          @dragstart="dragId = item.id"
          @dragend="dragId = null"
          @dragover.prevent
          @drop.prevent="onDrop(item)"
        >
          <div class="flex min-h-11 items-start gap-1">
            <label class="relative flex size-11 shrink-0 cursor-pointer items-center justify-center">
              <span class="sr-only">Tandai selesai {{ item.title }}</span>
              <input type="checkbox" class="peer sr-only" :checked="item.done" @change="toggle(item.id)" />
              <span
                class="flex size-5 items-center justify-center rounded-[2px] border peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-tag-ink"
                :class="rule"
              >
                <!-- Centang digambar 150 ms lewat stroke-dashoffset; diam saat prefers-reduced-motion. -->
                <svg
                  aria-hidden="true"
                  viewBox="0 0 24 24"
                  class="size-4"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="3"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                >
                  <path
                    d="M5 13l4 4L19 7"
                    class="transition-[stroke-dashoffset] duration-150 ease-out motion-reduce:transition-none"
                    :style="{ strokeDasharray: 24, strokeDashoffset: item.done ? 0 : 24 }"
                  />
                </svg>
              </span>
            </label>

            <form v-if="editingId === item.id" class="min-w-0 flex-1 py-1" @submit.prevent="commitEdit(item)">
              <label :for="`${uid}-edit-${item.id}`" class="sr-only">Ubah to do</label>
              <!-- Selebar baris dan tumbuh mengikuti isinya; kontrol baris disembunyikan selama mengetik. Garis
                   bawah tag-ink 2 px menjadi penanda fokus yang selalu terlihat. -->
              <textarea
                :id="`${uid}-edit-${item.id}`"
                :ref="setEditInput"
                v-model="editDraft"
                rows="1"
                maxlength="200"
                class="block min-h-11 w-full min-w-0 resize-none overflow-hidden border-b-2 border-tag-ink bg-transparent px-1 py-2.5 leading-6 outline-none [overflow-wrap:anywhere]"
                @input="grow"
                @keydown="onEditKey($event, item)"
                @keydown.esc.stop.prevent="cancelEdit(item)"
                @blur="commitEdit(item)"
              />
            </form>
            <!-- overflow-wrap anywhere: judul berupa URL panjang tanpa spasi tetap patah di lebar baris,
               bukan memaksa barisnya melebar. -->
            <button
              v-else
              :id="`${uid}-title-${item.id}`"
              type="button"
              data-caret
              class="min-w-0 flex-1 touch-manipulation py-3 text-left [overflow-wrap:anywhere]"
              :class="focusRing"
              :aria-label="`Ubah ${item.title}`"
              title="Dobel klik atau Enter untuk mengubah"
              @click="onTitleClick(item, $event)"
              @dblclick="editAt(item, $event)"
              @pointerup="onTitleTap(item, $event)"
            >
              {{ item.title }}
            </button>
            <!-- Judul yang memuat URL tetap teks biasa, supaya mengkliknya membuka mode ubah. Tautannya dibuka
                 lewat ikon ini, di slot 44 px tetap di samping teks: selalu tampil, jadi baris tidak bergeser. -->
            <a
              v-if="editingId !== item.id && titleUrl(item.title)"
              :href="titleUrl(item.title) ?? undefined"
              target="_blank"
              rel="noopener noreferrer"
              :class="iconButton"
              :aria-label="`Buka tautan ${item.title}`"
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
                <path
                  d="M10 13a5 5 0 0 0 7 0l3-3a5 5 0 0 0-7-7l-1 1M14 11a5 5 0 0 0-7 0l-3 3a5 5 0 0 0 7 7l1-1"
                />
              </svg>
            </a>
          </div>

          <div v-if="manage && editingId !== item.id" :class="rowControls">
            <span class="hidden size-11 cursor-grab items-center justify-center sm:flex" aria-hidden="true">
              <svg
                viewBox="0 0 24 24"
                class="size-4"
                fill="none"
                stroke="currentColor"
                stroke-width="1.75"
                stroke-linecap="round"
              >
                <path d="M9 6h.01M9 12h.01M9 18h.01M15 6h.01M15 12h.01M15 18h.01" />
              </svg>
            </span>
            <button
              type="button"
              :class="iconButton"
              :disabled="i === 0"
              :aria-label="`Naikkan ${item.title}`"
              @click="move(item.id, -1)"
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
                <path d="M12 19V5M6 11l6-6 6 6" />
              </svg>
            </button>
            <button
              type="button"
              :class="iconButton"
              :disabled="i >= open.length - 1"
              :aria-label="`Turunkan ${item.title}`"
              @click="move(item.id, 1)"
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
                <path d="M12 5v14M6 13l6 6 6-6" />
              </svg>
            </button>
            <button
              type="button"
              :class="iconButton"
              :aria-label="`Hapus ${item.title}`"
              @click="remove(item.id)"
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
                <path d="M5 7h14M10 7V5h4v2M7 7l1 12h8l1-12" />
              </svg>
            </button>
          </div>
        </li>

        <!-- Baris tambah: baris terakhir daftar, gaya sama, isian tanpa kotak. -->
        <li class="flex min-h-11 items-center gap-1 px-1">
          <span class="flex size-11 shrink-0 items-center justify-center" aria-hidden="true">
            <svg
              viewBox="0 0 24 24"
              class="size-5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
            >
              <path d="M12 5v14M5 12h14" />
            </svg>
          </span>
          <form class="flex min-w-0 flex-1 items-center gap-1" @submit.prevent="submit">
            <input
              :id="`${uid}-add`"
              ref="input"
              v-model="draft"
              type="text"
              maxlength="200"
              :disabled="full"
              aria-label="Tulis to do"
              :aria-describedby="full ? `${uid}-full` : undefined"
              placeholder="Tulis to do..."
              class="min-h-11 w-full min-w-0 bg-transparent outline-none placeholder:text-tag-ink/70 disabled:cursor-not-allowed"
              :class="focusRing"
            />
            <button
              type="submit"
              :class="iconButton"
              :disabled="full || !draft.trim()"
              aria-label="Tambah to do"
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
                <path d="M5 12h13M12 5l7 7-7 7" />
              </svg>
            </button>
          </form>
        </li>
      </ul>

      <p v-if="full" :id="`${uid}-full`" class="px-3 py-2 text-sm" :class="muted">
        Daftar sudah penuh ({{ max }} item). Hapus atau bersihkan yang selesai dulu.
      </p>

      <!-- Yang selesai pindah ke kelompoknya sendiri, dengan tombol Bersihkan di kepalanya. -->
      <section v-if="doneCount > 0" class="border-t" :class="rule">
        <div class="flex items-center justify-between gap-2 px-1">
          <button
            type="button"
            :class="headButton"
            :aria-expanded="doneOpen"
            :aria-controls="`${uid}-done`"
            @click="doneChoice = !doneOpen"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 24 24"
              class="mr-1 size-4 transition-transform motion-reduce:transition-none"
              :class="doneOpen ? 'rotate-90' : ''"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M9 6l6 6-6 6" />
            </svg>
            Selesai ({{ doneCount }})
          </button>
          <button type="button" :class="headButton" @click="clearDone">Bersihkan</button>
        </div>
        <ul
          v-show="doneOpen"
          :id="`${uid}-done`"
          class="divide-y border-t"
          :class="[rule, 'divide-tag-ink/60']"
        >
          <li v-for="item in done" :key="item.id" class="relative px-1">
            <div class="flex min-h-11 items-start gap-1">
              <label class="relative flex size-11 shrink-0 cursor-pointer items-center justify-center">
                <span class="sr-only">Batalkan centang {{ item.title }}</span>
                <input type="checkbox" class="peer sr-only" :checked="true" @change="toggle(item.id)" />
                <span
                  class="flex size-5 items-center justify-center rounded-[2px] border border-tag-ink peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-tag-ink"
                >
                  <svg
                    aria-hidden="true"
                    viewBox="0 0 24 24"
                    class="size-4"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="3"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <path d="M5 13l4 4L19 7" :style="{ strokeDasharray: 24, strokeDashoffset: 0 }" />
                  </svg>
                </span>
              </label>
              <p
                class="min-w-0 flex-1 py-3 line-through decoration-1 [overflow-wrap:anywhere]"
                :class="muted"
              >
                {{ item.title }}
              </p>
            </div>
            <div v-if="manage" :class="rowControls">
              <button
                type="button"
                :class="iconButton"
                :aria-label="`Hapus ${item.title}`"
                @click="remove(item.id)"
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
                  <path d="M5 7h14M10 7V5h4v2M7 7l1 12h8l1-12" />
                </svg>
              </button>
            </div>
          </li>
        </ul>
      </section>

      <!-- Hapus tidak pakai konfirmasi; yang terhapus bisa diurungkan selama enam detik. -->
      <div
        v-if="undoable"
        role="status"
        class="flex flex-wrap items-center justify-between gap-2 border-t px-1 py-1 text-sm"
        :class="rule"
      >
        <span class="px-2">To do dihapus.</span>
        <span class="flex gap-1">
          <button type="button" :class="headButton" @click="undo">Urungkan</button>
          <button type="button" :class="headButton" aria-label="Tutup pesan" @click="clearUndo">Tutup</button>
        </span>
      </div>
    </template>
  </section>
</template>
