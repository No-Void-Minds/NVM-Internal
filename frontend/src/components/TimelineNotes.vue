<script setup lang="ts">
import { nextTick, onMounted, ref, useId } from 'vue'
import { ApiError } from '../api/client'
import {
  createTimelineNote,
  deleteTimelineNote,
  listTimelineNotes,
  reorderTimelineNotes,
  updateTimelineNote,
  type TimelineNote,
} from '../api/timeline'

// Important Notes (revisi 30). Sengaja dibedakan dari kartu krem To Do: garis tebal urgent, latar danger-bg, dan
// tanda Penting, supaya terbaca sebagai pengumuman wajib dibaca, bukan catatan pribadi. Hanya Manajer (can_write
// dari server) yang menambah, mengubah, menghapus, dan mengurutkan.
const uid = useId()
const notes = ref<TimelineNote[]>([])
const canWrite = ref(false)
const state = ref<'loading' | 'error' | 'ok'>('loading')
const error = ref('')
const busy = ref(false)

const editing = ref<number | null>(null)
const editTitle = ref('')
const editBody = ref('')
const confirming = ref<number | null>(null)
const newTitle = ref('')
const newBody = ref('')
const listEl = ref<HTMLElement | null>(null)

async function load() {
  state.value = 'loading'
  try {
    const res = await listTimelineNotes()
    notes.value = res.notes
    canWrite.value = res.can_write
    state.value = 'ok'
  } catch {
    state.value = 'error'
  }
}
onMounted(load)

function fail(e: unknown, fallback: string) {
  error.value = e instanceof ApiError ? e.message : fallback
}

async function run(fn: () => Promise<void>, fallback: string) {
  busy.value = true
  error.value = ''
  try {
    await fn()
  } catch (e) {
    fail(e, fallback)
  } finally {
    busy.value = false
  }
}

// Pilihan pertama yang ada yang difokus, mis. tombol arah yang sama, atau arah sebaliknya bila sudah mentok.
function focus(...sels: string[]) {
  nextTick(() => {
    for (const sel of sels) {
      const el = listEl.value?.querySelector<HTMLElement>(sel)
      if (el) return el.focus()
    }
  })
}

function startEdit(n: TimelineNote) {
  editing.value = n.id
  editTitle.value = n.title
  editBody.value = n.body
  confirming.value = null
  focus(`#${CSS.escape(`${uid}-edit-title`)}`)
}

function cancelEdit(id: number) {
  editing.value = null
  focus(`[data-note-edit="${id}"]`)
}

function saveEdit(n: TimelineNote) {
  return run(async () => {
    await updateTimelineNote(n.id, editTitle.value, editBody.value)
    await load()
    cancelEdit(n.id)
  }, 'Catatan gagal disimpan. Periksa koneksi, lalu coba lagi.')
}

function remove(n: TimelineNote) {
  return run(async () => {
    await deleteTimelineNote(n.id)
    confirming.value = null
    await load()
    focus(`#${CSS.escape(`${uid}-new-title`)}`)
  }, 'Catatan gagal dihapus. Periksa koneksi, lalu coba lagi.')
}

function move(i: number, dir: -1 | 1) {
  const next = [...notes.value]
  const [item] = next.splice(i, 1)
  next.splice(i + dir, 0, item!)
  const before = notes.value
  notes.value = next
  return run(async () => {
    try {
      await reorderTimelineNotes(next.map((n) => n.id))
    } catch (e) {
      notes.value = before
      throw e
    }
    const [same, other] = dir === -1 ? ['up', 'down'] : ['down', 'up']
    focus(`[data-note-move="${item!.id}-${same}"]`, `[data-note-move="${item!.id}-${other}"]`)
  }, 'Urutan gagal disimpan. Periksa koneksi, lalu coba lagi.')
}

function add() {
  return run(async () => {
    await createTimelineNote(newTitle.value, newBody.value)
    newTitle.value = ''
    newBody.value = ''
    await load()
    focus(`#${CSS.escape(`${uid}-new-title`)}`)
  }, 'Catatan gagal ditambahkan. Periksa koneksi, lalu coba lagi.')
}

const muted = 'text-ink-muted'
const field =
  'block min-h-11 w-full min-w-0 rounded-card border border-line bg-surface px-2 text-ink placeholder:text-ink-muted focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-mark'
const btn =
  'inline-flex min-h-11 min-w-11 items-center justify-center rounded-card px-2 text-sm font-medium hover:bg-ink/10 focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-mark disabled:opacity-60'
const icon = 'size-5'
</script>

<template>
  <section
    ref="listEl"
    :aria-labelledby="`${uid}-title`"
    class="relative rounded-card border-2 border-l-8 border-urgent bg-danger-bg text-ink"
  >
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      class="absolute top-2 right-2 size-6 text-urgent"
      fill="none"
      stroke="currentColor"
      stroke-width="1.75"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path
        d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l8.57-8.57A4 4 0 1 1 18 8.84l-8.59 8.57a2 2 0 0 1-2.83-2.83l8.49-8.48"
      />
    </svg>
    <div class="flex flex-wrap items-center gap-2 px-3 pt-3 pr-10">
      <svg
        aria-hidden="true"
        viewBox="0 0 24 24"
        class="size-5 shrink-0 text-urgent"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path
          d="M10.3 3.9L1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0zM12 9v4M12 17h.01"
        />
      </svg>
      <h2 :id="`${uid}-title`" class="text-lg font-bold text-danger">Important Notes</h2>
      <span class="rounded-chip bg-urgent px-2 text-xs font-semibold uppercase text-on-urgent">Penting</span>
    </div>

    <p v-if="error" role="alert" class="mx-3 mt-2 border-l-2 border-urgent pl-2 text-sm">
      Gagal: {{ error }}
    </p>

    <p v-if="state === 'loading'" role="status" class="px-3 py-3 text-sm" :class="muted">Memuat catatan...</p>
    <div v-else-if="state === 'error'" class="px-3 py-3 text-sm">
      <p>Catatan gagal dimuat.</p>
      <button type="button" :class="[btn, 'mt-1 border border-line']" @click="load">Muat ulang</button>
    </div>

    <template v-else>
      <p v-if="!notes.length" class="px-3 py-3 text-sm" :class="muted">Belum ada catatan penting.</p>
      <ul v-else class="mt-2 divide-y divide-line border-t border-line">
        <li v-for="(n, i) in notes" :key="n.id" class="px-3 py-2">
          <form
            v-if="editing === n.id"
            class="space-y-2"
            @submit.prevent="saveEdit(n)"
            @keydown.esc.stop.prevent="cancelEdit(n.id)"
          >
            <label :for="`${uid}-edit-title`" class="block text-sm font-medium">Judul catatan</label>
            <input :id="`${uid}-edit-title`" v-model="editTitle" maxlength="120" :class="field" />
            <label :for="`${uid}-edit-body`" class="block text-sm font-medium">Isi</label>
            <textarea
              :id="`${uid}-edit-body`"
              v-model="editBody"
              rows="3"
              maxlength="2000"
              :class="[field, 'py-2']"
            />
            <div class="flex gap-2">
              <button
                type="submit"
                :disabled="busy"
                :class="[btn, 'bg-urgent text-on-urgent hover:bg-urgent/90']"
              >
                Simpan
              </button>
              <button type="button" :class="btn" @click="cancelEdit(n.id)">Batal</button>
            </div>
          </form>
          <template v-else>
            <p class="font-semibold [overflow-wrap:anywhere]">{{ n.title }}</p>
            <p v-if="n.body" class="mt-0.5 text-sm whitespace-pre-line [overflow-wrap:anywhere]">
              {{ n.body }}
            </p>
            <div v-if="canWrite" class="mt-1 flex flex-wrap items-center gap-1">
              <template v-if="confirming === n.id">
                <p class="w-full text-sm">Hapus catatan ini?</p>
                <button
                  type="button"
                  :disabled="busy"
                  :class="[btn, 'bg-urgent text-on-urgent hover:bg-urgent/90']"
                  @click="remove(n)"
                >
                  Ya, hapus
                </button>
                <button type="button" :class="btn" @click="confirming = null">Batal</button>
              </template>
              <template v-else>
                <button
                  type="button"
                  :data-note-edit="n.id"
                  :class="btn"
                  :aria-label="`Ubah catatan ${n.title}`"
                  @click="startEdit(n)"
                >
                  Ubah
                </button>
                <button
                  type="button"
                  :class="btn"
                  :aria-label="`Hapus catatan ${n.title}`"
                  @click="confirming = n.id"
                >
                  Hapus
                </button>
                <button
                  v-if="i > 0"
                  type="button"
                  :data-note-move="`${n.id}-up`"
                  :disabled="busy"
                  :class="btn"
                  :aria-label="`Naikkan catatan ${n.title}`"
                  @click="move(i, -1)"
                >
                  <svg
                    aria-hidden="true"
                    viewBox="0 0 24 24"
                    :class="icon"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.75"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <path d="M12 19V5M5 12l7-7 7 7" />
                  </svg>
                </button>
                <button
                  v-if="i < notes.length - 1"
                  type="button"
                  :data-note-move="`${n.id}-down`"
                  :disabled="busy"
                  :class="btn"
                  :aria-label="`Turunkan catatan ${n.title}`"
                  @click="move(i, 1)"
                >
                  <svg
                    aria-hidden="true"
                    viewBox="0 0 24 24"
                    :class="icon"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.75"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <path d="M12 5v14M19 12l-7 7-7-7" />
                  </svg>
                </button>
              </template>
            </div>
          </template>
        </li>
      </ul>

      <form v-if="canWrite" class="space-y-2 border-t border-line px-3 py-3" @submit.prevent="add">
        <label :for="`${uid}-new-title`" class="block text-sm font-medium">Judul catatan baru</label>
        <input
          :id="`${uid}-new-title`"
          v-model="newTitle"
          maxlength="120"
          autocomplete="off"
          :class="field"
        />
        <label :for="`${uid}-new-body`" class="block text-sm font-medium">Isi</label>
        <textarea
          :id="`${uid}-new-body`"
          v-model="newBody"
          rows="2"
          maxlength="2000"
          :class="[field, 'py-2']"
        />
        <button type="submit" :disabled="busy" :class="[btn, 'border border-line']">Tambah catatan</button>
      </form>
    </template>
  </section>
</template>
