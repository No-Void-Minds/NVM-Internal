<script setup lang="ts">
import { ref, shallowRef, useId, watch } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import { ApiError } from '../../api/client'
import { addComment, deleteComment, editComment, listComments, type Comment } from '../../api/whiteboard'
import { formatDateTime } from '../../utils/format'
import { cardLabel, type Card } from '../../utils/whiteboard'

// Komentar per kartu. Semua yang login membaca; Staf dan Manajer menulis. Mengubah hanya penulisnya, menghapus
// penulisnya atau Manajer (aturan di server, can_moderate dari server). reload naik setiap kartu ini disiarkan
// berubah, supaya komentar orang lain langsung terlihat.
const props = defineProps<{
  open: boolean
  board: string
  card: Card | null
  canWrite: boolean
  reload: number
}>()
const emit = defineEmits<{ close: []; card: [card: Card] }>()

const uid = useId()
const list = shallowRef<Comment[]>([])
const me = ref(0)
const canModerate = ref(false)
const state = ref<'loading' | 'error' | 'ok'>('loading')
const error = ref('')
const draft = ref('')
const formError = ref('')
const busy = ref(false)
const editing = ref<string | null>(null)
const editDraft = ref('')
let abort: AbortController | null = null

async function load(quiet = false) {
  const c = props.card
  if (!c) return
  abort?.abort()
  const ctrl = new AbortController()
  abort = ctrl
  if (!quiet) state.value = 'loading'
  try {
    const res = await listComments(props.board, c.id, ctrl.signal)
    list.value = res.comments
    me.value = res.me
    canModerate.value = res.can_moderate
    state.value = 'ok'
  } catch (e) {
    if (ctrl.signal.aborted) return
    error.value = e instanceof ApiError ? e.message : 'Komentar gagal dimuat. Coba lagi.'
    state.value = 'error'
  }
}

watch(
  () => [props.open, props.card?.id] as const,
  ([open]) => {
    if (!open) return
    draft.value = ''
    formError.value = ''
    editing.value = null
    load()
  },
)
watch(
  () => props.reload,
  () => props.open && load(true),
)

async function run(work: () => Promise<{ card: Card }>) {
  busy.value = true
  formError.value = ''
  try {
    const res = await work()
    emit('card', res.card)
    await load(true)
    return true
  } catch (e) {
    formError.value = e instanceof ApiError ? e.message : 'Komentar gagal disimpan. Periksa koneksi.'
    return false
  } finally {
    busy.value = false
  }
}

async function submit() {
  const c = props.card
  const body = draft.value.trim()
  if (!c || busy.value) return
  if (!body) {
    formError.value = 'Tulis komentarnya dulu.'
    return
  }
  if (await run(() => addComment(props.board, c.id, body))) draft.value = ''
}

async function saveEdit(id: string) {
  if (await run(() => editComment(id, editDraft.value.trim()))) editing.value = null
}

function onDraftKey(e: KeyboardEvent) {
  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
    e.preventDefault()
    submit()
  }
}

const btn =
  'inline-flex min-h-11 items-center justify-center rounded-card border border-line bg-surface px-3 font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-60'
</script>

<template>
  <BaseDialog :open="open" title="Komentar" variant="panel" @close="emit('close')">
    <div class="flex min-h-0 flex-1 flex-col">
      <p v-if="card" class="border-b border-line px-5 py-2 text-sm break-words text-ink-muted">
        {{ cardLabel(card) }}
      </p>
      <div class="min-h-0 flex-1 overflow-y-auto px-5 py-4">
        <p v-if="state === 'loading'" role="status" class="text-ink-muted">Memuat komentar...</p>
        <div v-else-if="state === 'error'" role="alert" class="flex flex-col items-start gap-2">
          <p class="text-danger">{{ error }}</p>
          <button type="button" :class="btn" @click="load()">Coba lagi</button>
        </div>
        <p v-else-if="list.length === 0" class="text-ink-muted">Belum ada komentar.</p>
        <ol v-else class="flex flex-col gap-3">
          <li v-for="m in list" :key="m.id" class="border-b border-line pb-3 last:border-b-0">
            <p class="text-sm">
              <span class="font-semibold">{{ m.user_name }}</span>
              <span class="text-ink-muted">
                , <time :datetime="m.created_at">{{ formatDateTime(m.created_at) }}</time>
                <template v-if="m.edited_at">, diubah</template>
              </span>
            </p>
            <template v-if="editing === m.id">
              <label :for="`${uid}-e-${m.id}`" class="sr-only">Ubah komentar</label>
              <textarea
                :id="`${uid}-e-${m.id}`"
                v-model="editDraft"
                maxlength="2000"
                rows="3"
                class="mt-1 w-full rounded-card border border-line bg-surface p-2"
              />
              <div class="mt-1 flex flex-wrap gap-2">
                <button type="button" :class="btn" :disabled="busy" @click="saveEdit(m.id)">Simpan</button>
                <button type="button" :class="btn" @click="editing = null">Batal</button>
              </div>
            </template>
            <p v-else class="mt-1 break-words whitespace-pre-line">{{ m.body }}</p>
            <div v-if="canWrite && editing !== m.id" class="mt-1 flex flex-wrap gap-2">
              <button
                v-if="m.user_id === me"
                type="button"
                :class="btn"
                @click="((editing = m.id), (editDraft = m.body))"
              >
                Ubah<span class="sr-only"> komentar {{ m.user_name }}</span>
              </button>
              <button
                v-if="m.user_id === me || canModerate"
                type="button"
                :class="btn"
                class="text-urgent"
                :disabled="busy"
                @click="run(() => deleteComment(m.id))"
              >
                Hapus<span class="sr-only"> komentar {{ m.user_name }}</span>
              </button>
            </div>
          </li>
        </ol>
      </div>
      <form
        v-if="canWrite && card"
        class="border-t border-line px-5 py-3"
        novalidate
        @submit.prevent="submit"
      >
        <label :for="`${uid}-new`" class="text-sm font-medium">Komentar baru</label>
        <textarea
          :id="`${uid}-new`"
          v-model="draft"
          maxlength="2000"
          rows="3"
          :aria-invalid="formError ? 'true' : undefined"
          :aria-describedby="formError ? `${uid}-err` : undefined"
          class="mt-1 w-full rounded-card border border-line bg-surface p-2"
          @keydown="onDraftKey"
        />
        <p v-if="formError" :id="`${uid}-err`" role="alert" class="text-sm text-danger">{{ formError }}</p>
        <button type="submit" :class="btn" class="mt-2" :disabled="busy">Kirim komentar</button>
      </form>
    </div>
  </BaseDialog>
</template>
