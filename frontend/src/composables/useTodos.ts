import { computed, ref } from 'vue'
import { ApiError } from '../api/client'
import {
  addTodo,
  clearDoneTodos,
  deleteTodo,
  listTodos,
  renameTodo,
  reorderTodos,
  setTodoDone,
  type TodoItem,
} from '../api/todos'

// Satu daftar To Do untuk seluruh halaman (revisi 28). Kartu di kolom kiri papan dan panel di bar memakai
// state yang sama, jadi daftarnya tidak pernah berbeda antara keduanya dan hanya satu permintaan yang jalan.
// Semua perubahan optimistis: tampil dulu, dikembalikan kalau server menolak, dan galatnya ditampilkan.
const items = ref<TodoItem[]>([])
const max = ref(100)
const state = ref<'loading' | 'ready' | 'error'>('loading')
const error = ref('')
// Item yang baru dihapus, untuk tombol Urungkan di toast.
const undoable = ref<{ item: TodoItem; index: number } | null>(null)
let undoTimer: ReturnType<typeof setTimeout> | null = null
let loading: Promise<void> | null = null
// Tambah yang masih jalan, dan peta id sementara ke id dari server. Baris hasil tambah sudah tampil
// sebelum jawabannya datang, jadi tanpa keduanya perubahan berikutnya bisa mengirim id sementara yang
// negatif ke server dan dijawab 404 (lihat Keputusan Sprint 11y).
const adding = new Set<Promise<unknown>>()
const swapped = new Map<number, number>()

async function settled() {
  if (adding.size) await Promise.allSettled([...adding])
}

function real(id: number): number {
  return swapped.get(id) ?? id
}

function messageOf(err: unknown): string {
  return err instanceof ApiError ? err.message : 'Terjadi kesalahan yang tidak terduga. Coba lagi.'
}

// Urutan tampilan: belum selesai dulu, lalu yang selesai, sama dengan urutan dari server.
function sortLocal() {
  items.value = [...items.value.filter((i) => !i.done), ...items.value.filter((i) => i.done)]
}

async function load(force = false) {
  if (loading && !force) return loading
  state.value = items.value.length ? 'ready' : 'loading'
  loading = (async () => {
    try {
      const res = await listTodos()
      items.value = res.todos
      max.value = res.max
      state.value = 'ready'
      error.value = ''
    } catch (err) {
      error.value = messageOf(err)
      state.value = 'error'
    }
  })()
  return loading
}

// Perubahan optimistis: snapshot dulu, kembalikan kalau server menolak.
async function optimistic(apply: () => void, call: () => Promise<unknown>) {
  const before = items.value.map((i) => ({ ...i }))
  error.value = ''
  apply()
  try {
    await call()
  } catch (err) {
    items.value = before
    error.value = messageOf(err)
  }
}

export function useTodos() {
  const openCount = computed(() => items.value.filter((i) => i.done === false).length)
  const doneCount = computed(() => items.value.filter((i) => i.done).length)
  const full = computed(() => items.value.length >= max.value)

  function add(title: string) {
    const clean = title.trim()
    if (!clean || full.value) return Promise.resolve()
    // Id sementara negatif: diganti id dari server begitu jawabannya datang.
    const temp: TodoItem = { id: -Date.now(), title: clean, done: false }
    const before = items.value.map((i) => ({ ...i }))
    error.value = ''
    items.value = [...items.value, temp]
    sortLocal()
    const p = (async () => {
      try {
        const res = await addTodo(clean)
        swapped.set(temp.id, res.todo.id)
        items.value = items.value.map((i) => (i.id === temp.id ? res.todo : i))
      } catch (err) {
        items.value = before
        error.value = messageOf(err)
      }
    })()
    adding.add(p)
    return p.finally(() => adding.delete(p))
  }

  async function rename(id: number, title: string) {
    const clean = title.trim()
    if (!clean) return
    await settled()
    const cur = items.value.find((i) => i.id === real(id))
    if (!cur || clean === cur.title) return
    return optimistic(
      () => {
        cur.title = clean
      },
      () => renameTodo(cur.id, clean),
    )
  }

  async function toggle(id: number) {
    await settled()
    const cur = items.value.find((i) => i.id === real(id))
    if (!cur) return
    const done = !cur.done
    return optimistic(
      () => {
        cur.done = done
        // Saat dicentang, pemindahan ke kelompok Selesai ditunda sedikit supaya goresan centang 150 ms
        // sempat terlihat di tempatnya. Saat batal centang tidak perlu ditunda.
        if (done) setTimeout(sortLocal, 180)
        else sortLocal()
      },
      () => setTodoDone(cur.id, done),
    )
  }

  function clearUndo() {
    if (undoTimer) clearTimeout(undoTimer)
    undoTimer = null
    undoable.value = null
  }

  async function remove(id: number) {
    await settled()
    const nyata = real(id)
    const index = items.value.findIndex((i) => i.id === nyata)
    if (index === -1) return
    const item = { ...items.value[index] }
    await optimistic(
      () => {
        items.value = items.value.filter((i) => i.id !== nyata)
      },
      () => deleteTodo(nyata),
    )
    if (error.value) return
    // Toast Urungkan hidup 6 detik. Id-nya berganti setelah diurungkan, karena barisnya dibuat ulang.
    if (undoTimer) clearTimeout(undoTimer)
    undoable.value = { item, index }
    undoTimer = setTimeout(clearUndo, 6000)
  }

  async function undo() {
    const saved = undoable.value
    clearUndo()
    if (!saved) return
    await settled()
    await settled()
    error.value = ''
    try {
      const res = await addTodo(saved.item.title)
      items.value = [...items.value, res.todo]
      if (saved.item.done) {
        await setTodoDone(res.todo.id, true)
        res.todo.done = true
      }
      // Dikembalikan ke posisi semula, bukan ditaruh di akhir.
      const order = items.value.filter((i) => i.id !== res.todo.id)
      order.splice(Math.min(saved.index, order.length), 0, res.todo)
      items.value = order
      sortLocal()
      await reorderTodos(items.value.map((i) => i.id))
    } catch (err) {
      error.value = messageOf(err)
      await load(true)
    }
  }

  function clearDone() {
    return optimistic(
      () => {
        items.value = items.value.filter((i) => !i.done)
      },
      () => clearDoneTodos(),
    )
  }

  // move dipakai tombol naik dan turun, supaya urutan bisa diubah tanpa drag (keyboard).
  async function move(id: number, delta: number) {
    await settled()
    const open = items.value.filter((i) => !i.done)
    const from = open.findIndex((i) => i.id === real(id))
    const to = from + delta
    if (from === -1 || to < 0 || to >= open.length) return
    const next = [...open]
    const [moved] = next.splice(from, 1)
    next.splice(to, 0, moved)
    return reorder([...next, ...items.value.filter((i) => i.done)].map((i) => i.id))
  }

  async function reorder(ids: number[]) {
    await settled()
    const nyata = ids.map(real)
    const byId = new Map(items.value.map((i) => [i.id, i]))
    const next = nyata.map((id) => byId.get(id)).filter((i): i is TodoItem => !!i)
    if (next.length !== items.value.length) return
    return optimistic(
      () => {
        items.value = next
      },
      () => reorderTodos(nyata),
    )
  }

  return {
    items,
    max,
    state,
    error,
    undoable,
    openCount,
    doneCount,
    full,
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
  }
}
