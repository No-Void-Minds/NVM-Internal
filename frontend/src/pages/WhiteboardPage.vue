<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, useId, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseDialog from '../components/BaseDialog.vue'
import WbCard from '../components/whiteboard/WbCard.vue'
import WbTrashPanel from '../components/whiteboard/WbTrashPanel.vue'
import { ApiError } from '../api/client'
import {
  boardActivity,
  checklistOp,
  createCard,
  deleteCards,
  getBoard,
  patchCards,
  renameBoard,
  restoreBatch,
  searchBoards,
  type BoardView,
  type CardPatch,
  type ChecklistOp,
  type NewCard,
  type SearchResult,
} from '../api/whiteboard'
import { formatDateTime } from '../utils/format'
import {
  activityText,
  clamp,
  colors,
  COORD_LIMIT,
  defaultSize,
  emptyContent,
  fitView,
  GRID,
  intersects,
  kindLabel,
  MAX_SIZE,
  MAX_ZOOM,
  MIN_SIZE,
  MIN_ZOOM,
  nearestInDirection,
  newId,
  readingOrder,
  snap,
  toWorld,
  visibleWorld,
  zoomAt,
  type Activity,
  type Card,
  type CardColor,
  type CardKind,
  type Direction,
  type HeadingContent,
  type PMNode,
  type Viewport,
} from '../utils/whiteboard'

// Whiteboard seperti Milanote (docs/whiteboard.md): semua adalah papan. /whiteboard membuka papan akar bersama,
// /whiteboard/:id membuka papan lain di kanvas yang sama. Papan di dalam papan tampil sebagai kartu yang dibuka
// dengan klik atau Enter. Data dari server adalah sumber kebenaran: perubahan dikirim per field, isi kartu dijaga
// content_version. Perubahan orang lain baru terlihat setelah muat ulang sampai W2 menambah koneksi langsung.
const route = useRoute()
const router = useRouter()
const uid = useId()
const boardKey = computed(() =>
  typeof route.params.id === 'string' && route.params.id ? route.params.id : 'root',
)

const board = shallowRef<BoardView | null>(null)
const cards = shallowRef<Card[]>([])
const canWrite = ref(false)
const state = ref<'loading' | 'error' | 'ok'>('loading')
const loadError = ref('')
let abort: AbortController | null = null

// Id papan yang sedang terbuka untuk API tulis; "root" sampai jawaban pertama datang.
const bid = computed(() => board.value?.id ?? boardKey.value)
const byId = computed(() => new Map(cards.value.map((c) => [c.id, c])))

function upsert(updated: Card[]) {
  const map = new Map(cards.value.map((c) => [c.id, c]))
  for (const c of updated) map.set(c.id, c)
  cards.value = [...map.values()]
}
function patchLocal(id: string, fields: Partial<Card>) {
  cards.value = cards.value.map((c) => (c.id === id ? { ...c, ...fields } : c))
}

async function load(keepView = false) {
  abort?.abort()
  const ctrl = new AbortController()
  abort = ctrl
  if (!keepView) state.value = 'loading'
  try {
    const res = await getBoard(boardKey.value, ctrl.signal)
    board.value = res.board
    canWrite.value = res.can_write
    cards.value = res.board.cards
    state.value = 'ok'
    if (!keepView) {
      await nextTick()
      measure()
      fit()
    }
  } catch (e) {
    if (ctrl.signal.aborted) return
    loadError.value =
      e instanceof ApiError ? e.message : 'Papan gagal dimuat. Periksa koneksi, lalu coba lagi.'
    state.value = 'error'
  }
}

// ---------------------------------------------------------------------------------------------------------------
// Bar atas: breadcrumb, judul, cari

const crumbs = computed(() => board.value?.crumbs ?? [])
const parentCrumb = computed(() => crumbs.value.at(-2) ?? null)
function crumbLink(i: number) {
  return i === 0 ? { name: 'whiteboard' } : { name: 'whiteboard-board', params: { id: crumbs.value[i]!.id } }
}

const titleEditing = ref(false)
const titleDraft = ref('')
const titleError = ref('')
const titleInput = ref<HTMLInputElement | null>(null)
const titleButton = ref<HTMLButtonElement | null>(null)

async function editTitle() {
  if (!board.value || !canWrite.value) return
  titleDraft.value = board.value.title
  titleError.value = ''
  titleEditing.value = true
  await nextTick()
  titleInput.value?.select()
}

async function saveTitle() {
  const b = board.value
  if (!b) return
  const t = titleDraft.value.trim()
  if (!t || t === b.title) {
    titleEditing.value = false
    return
  }
  try {
    const res = await saving(() => renameBoard(b.id, t, b.version))
    if (!res) return
    board.value = {
      ...b,
      title: res.board.title,
      version: res.board.version,
      crumbs: b.crumbs.map((c) => (c.id === b.id ? { ...c, title: res.board.title } : c)),
    }
    titleEditing.value = false
    await nextTick()
    titleButton.value?.focus()
  } catch (e) {
    titleError.value = e instanceof ApiError ? e.message : 'Judul gagal disimpan.'
  }
}

function onTitleKey(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    saveTitle()
  } else if (e.key === 'Escape') {
    e.preventDefault()
    titleEditing.value = false
    nextTick(() => titleButton.value?.focus())
  }
}

const searchOpen = ref(false)
const query = ref('')
const results = shallowRef<SearchResult[] | null>(null)
const searchError = ref('')
const searchBox = ref<HTMLElement | null>(null)

async function search() {
  const q = query.value.trim()
  searchError.value = ''
  results.value = null
  if ([...q].length < 2) {
    searchError.value = 'Ketik minimal 2 huruf untuk mencari.'
    return
  }
  try {
    results.value = (await searchBoards(q)).results
  } catch (e) {
    searchError.value = e instanceof ApiError ? e.message : 'Pencarian gagal. Coba lagi.'
  }
}

function closeSearch() {
  results.value = null
  searchError.value = ''
}

function onSearchKey(e: KeyboardEvent) {
  if (e.key === 'Escape' && (results.value || searchError.value)) {
    e.preventDefault()
    closeSearch()
  }
}

function onDocPointer(e: PointerEvent) {
  if (searchBox.value && !searchBox.value.contains(e.target as Node)) closeSearch()
}
onMounted(() => document.addEventListener('pointerdown', onDocPointer))

// ---------------------------------------------------------------------------------------------------------------
// Viewport

const canvas = ref<HTMLElement | null>(null)
const view = ref<Viewport>({ x: 48, y: 48, zoom: 1 })
const size = ref({ w: 0, h: 0 })
let ro: ResizeObserver | null = null

function measure() {
  const el = canvas.value
  if (el) size.value = { w: el.clientWidth, h: el.clientHeight }
}
onMounted(() => {
  ro = new ResizeObserver(measure)
  if (canvas.value) ro.observe(canvas.value)
})
watch(canvas, (el, old) => {
  if (old) ro?.unobserve(old)
  if (el) {
    ro?.observe(el)
    measure()
  }
})

function fit() {
  view.value = fitView(cards.value, size.value.w, size.value.h)
}
function zoomBy(factor: number) {
  view.value = zoomAt(view.value, size.value.w / 2, size.value.h / 2, view.value.zoom * factor)
}
const zoomPercent = computed(() => Math.round(view.value.zoom * 100))

const worldStyle = computed(() => ({
  transform: `translate(${view.value.x}px, ${view.value.y}px) scale(${view.value.zoom})`,
}))
const gridOn = ref(true)
const gridStyle = computed(() => {
  const step = GRID * 4 * view.value.zoom
  return gridOn.value
    ? { backgroundSize: `${step}px ${step}px`, backgroundPosition: `${view.value.x}px ${view.value.y}px` }
    : {}
})

function centerWorld() {
  return toWorld(view.value, size.value.w / 2, size.value.h / 2)
}

function reveal(card: Card) {
  const tl = toWorld(view.value, 0, 0)
  const br = toWorld(view.value, size.value.w, size.value.h)
  if (card.x >= tl.x && card.y >= tl.y && card.x + card.w <= br.x && card.y + card.h <= br.y) return
  const z = view.value.zoom
  view.value = {
    zoom: z,
    x: size.value.w / 2 - (card.x + card.w / 2) * z,
    y: size.value.h / 2 - (card.y + card.h / 2) * z,
  }
}

// ---------------------------------------------------------------------------------------------------------------
// Pilihan, fokus, dan kartu yang dirender

const selected = ref<Set<string>>(new Set())
const focusedId = ref<string | null>(null)
const editingId = ref<string | null>(null)

function select(ids: string[]) {
  selected.value = new Set(ids)
}

// Culling: hanya kartu di area terlihat plus satu layar margin yang di-mount. Kartu terpilih, terfokus, dan yang
// sedang diedit selalu ikut, supaya fokus keyboard dan editor tidak hilang saat kanvas digeser.
const rendered = computed(() => {
  const area = visibleWorld(view.value, size.value.w || 1, size.value.h || 1)
  const keep = new Set([...selected.value, focusedId.value, editingId.value])
  return readingOrder(cards.value.filter((c) => keep.has(c.id) || intersects(area, c)))
})
const ordered = computed(() => readingOrder(cards.value))
const rovingId = computed(() => focusedId.value ?? ordered.value[0]?.id ?? null)

const cardRefs = new Map<string, HTMLElement>()
function setCardRef(id: string, inst: unknown) {
  const el = (inst as { el?: HTMLElement } | null)?.el
  if (el) cardRefs.set(id, el)
  else cardRefs.delete(id)
}

async function focusCard(id: string) {
  focusedId.value = id
  const card = byId.value.get(id)
  if (card) reveal(card)
  await nextTick()
  cardRefs.get(id)?.focus()
}

function openBoard(card: Card) {
  if (card.kind !== 'board' || !card.ref) return
  router.push({ name: 'whiteboard-board', params: { id: card.ref.id } })
}

// ---------------------------------------------------------------------------------------------------------------
// Status simpan

const pending = ref(0)
const saveError = ref('')
const saveLabel = computed(() =>
  saveError.value ? 'Gagal disimpan' : pending.value > 0 ? 'Menyimpan...' : 'Tersimpan',
)
const announce = ref('')

async function saving<T>(work: () => Promise<T>): Promise<T | undefined> {
  pending.value++
  try {
    const out = await work()
    saveError.value = ''
    return out
  } catch (e) {
    saveError.value = e instanceof ApiError ? e.message : 'Perubahan gagal disimpan. Periksa koneksi.'
    if (!(e instanceof ApiError) || e.status !== 409) load(true)
    throw e
  } finally {
    pending.value--
  }
}

function onBeforeUnload(e: BeforeUnloadEvent) {
  if (pending.value > 0 || dirtyNotes.size > 0) e.preventDefault()
}
onMounted(() => window.addEventListener('beforeunload', onBeforeUnload))

// ---------------------------------------------------------------------------------------------------------------
// Buat, ubah isi, hapus

const toast = ref<{ text: string; batch?: string } | null>(null)
let toastTimer: ReturnType<typeof setTimeout> | undefined
function showToast(text: string, batch?: string) {
  toast.value = { text, batch }
  announce.value = text
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => (toast.value = null), 8000)
}

let placed = 0
// Kartu baru di titik dunia at (dari seret alat), atau di tengah layar dengan sedikit geser supaya kartu yang
// dibuat beruntun tidak menumpuk persis.
async function addCard(kind: CardKind, at?: { x: number; y: number }) {
  if (!canWrite.value) return
  const dim = defaultSize[kind]
  const c = at ?? centerWorld()
  const offset = at ? 0 : (placed++ % 5) * 24
  const input: NewCard = {
    id: newId(),
    kind,
    x: clamp(snap(c.x - dim.w / 2 + offset, gridOn.value), -COORD_LIMIT, COORD_LIMIT),
    y: clamp(snap(c.y - dim.h / 2 + offset, gridOn.value), -COORD_LIMIT, COORD_LIMIT),
    w: dim.w,
    h: dim.h,
    color: '',
  }
  if (kind === 'board') input.title = 'Papan tanpa judul'
  else input.content = emptyContent(kind)
  try {
    const res = await saving(() => createCard(bid.value, input))
    if (!res) return
    upsert([res.card])
    select([res.card.id])
    focusedId.value = res.card.id
    editingId.value = res.card.id
    announce.value = `${kindLabel[kind]} baru ditambahkan.`
  } catch {
    // Pesan galat sudah tampil di penanda simpan.
  }
}

// Simpan isi per kartu berurutan: simpan berikutnya memakai content_version dari jawaban sebelumnya, jadi ketikan
// sendiri tidak pernah dianggap bentrok.
const chains = new Map<string, Promise<unknown>>()
const dirtyNotes = new Map<string, PMNode>()
let noteTimer: ReturnType<typeof setTimeout> | undefined

function queueContent(id: string, content: Card['content']) {
  const prev = chains.get(id) ?? Promise.resolve()
  const next = prev.catch(() => undefined).then(() => saveContent(id, content))
  chains.set(id, next)
  return next
}

async function saveContent(id: string, content: Card['content']) {
  const card = byId.value.get(id)
  if (!card) return
  try {
    const res = await saving(() =>
      patchCards(bid.value, [{ id, content, base_content_version: card.content_version }]),
    )
    if (res) upsert(res.cards)
  } catch (e) {
    if (e instanceof ApiError && e.status === 409) await keepAsCopy(card, content)
  }
}

// Isi kartu sudah diubah orang lain: ketikan disimpan sebagai kartu baru di sebelahnya, lalu papan dimuat ulang.
async function keepAsCopy(card: Card, content: Card['content']) {
  if (editingId.value === card.id) editingId.value = null
  dirtyNotes.delete(card.id)
  try {
    const res = await saving(() =>
      createCard(bid.value, {
        id: newId(),
        kind: card.kind,
        x: clamp(card.x + card.w + 16, -COORD_LIMIT, COORD_LIMIT),
        y: card.y,
        w: card.w,
        h: card.h,
        color: card.color ?? '',
        content,
      }),
    )
    saveError.value = ''
    await load(true)
    if (res) select([res.card.id])
    showToast('Kartu ini sudah diubah orang lain. Ketikanmu disimpan di kartu baru di sebelahnya.')
  } catch {
    // Pesan galat sudah tampil di penanda simpan.
  }
}

function onNoteInput(id: string, doc: PMNode) {
  dirtyNotes.set(id, doc)
  clearTimeout(noteTimer)
  noteTimer = setTimeout(flushNotes, 1500)
}

function flushNotes() {
  clearTimeout(noteTimer)
  for (const [id, doc] of dirtyNotes) {
    dirtyNotes.delete(id)
    queueContent(id, { doc })
  }
}

function onHeading(id: string, text: string) {
  const card = byId.value.get(id)
  if (!card) return
  queueContent(id, { text, level: (card.content as HeadingContent).level ?? 1 })
}

async function onRenameRef(id: string, title: string) {
  const card = byId.value.get(id)
  if (!card?.ref) return
  const ref = card.ref
  try {
    const res = await saving(() => renameBoard(ref.id, title, ref.version))
    if (res) patchLocal(id, { ref: { ...ref, title: res.board.title, version: res.board.version } })
  } catch {
    // Pesan galat sudah tampil di penanda simpan; 409 berarti papan baru diganti judulnya oleh orang lain.
    load(true)
  }
}

async function onChecklist(id: string, op: ChecklistOp) {
  try {
    const res = await saving(() => checklistOp(bid.value, id, op))
    if (res) upsert([res.card])
  } catch {
    // Papan sudah dimuat ulang oleh saving().
  }
}

function startEdit(id: string) {
  if (!canWrite.value) return
  select([id])
  focusedId.value = id
  editingId.value = id
}

async function stopEdit() {
  const id = editingId.value
  flushNotes()
  editingId.value = null
  if (id) await focusCard(id)
}

async function removeSelected() {
  if (!canWrite.value || selected.value.size === 0) return
  const ids = [...selected.value]
  if (editingId.value && ids.includes(editingId.value)) editingId.value = null
  const boards = ids.filter((id) => byId.value.get(id)?.kind === 'board').length
  try {
    const res = await saving(() => deleteCards(bid.value, ids))
    if (!res) return
    const gone = new Set(ids)
    cards.value = cards.value.filter((c) => !gone.has(c.id))
    select([])
    focusedId.value = null
    showToast(
      `${ids.length} kartu dipindah ke Tempat sampah${boards ? `, termasuk ${boards} papan beserta isinya` : ''}.`,
      res.batch,
    )
    canvas.value?.focus()
  } catch {
    // Pesan galat sudah tampil di penanda simpan.
  }
}

async function undoDelete() {
  const batch = toast.value?.batch
  if (!batch) return
  toast.value = null
  try {
    await saving(() => restoreBatch(batch))
    await load(true)
    announce.value = 'Kartu dipulihkan.'
  } catch {
    // Pesan galat sudah tampil di penanda simpan.
  }
}

const selectionColor = computed<string>(() => {
  const list = [...selected.value].map((id) => byId.value.get(id)?.color ?? '')
  return list.length && list.every((c) => c === list[0]) ? (list[0] ?? '') : 'campuran'
})

async function setColor(value: string) {
  if (!canWrite.value || value === 'campuran') return
  const ids = [...selected.value]
  const color = (value || null) as CardColor | null
  for (const id of ids) patchLocal(id, { color })
  try {
    const res = await saving(() =>
      patchCards(
        bid.value,
        ids.map((id) => ({ id, color: (value || '') as CardColor | '' })),
      ),
    )
    if (res) upsert(res.cards)
  } catch {
    // Papan sudah dimuat ulang oleh saving().
  }
}

async function sendGeometry(patches: CardPatch[]) {
  if (patches.length === 0) return
  try {
    const res = await saving(() => patchCards(bid.value, patches))
    if (res) upsert(res.cards)
  } catch {
    // Papan sudah dimuat ulang oleh saving().
  }
}

// ---------------------------------------------------------------------------------------------------------------
// Alat: klik menaruh kartu di tengah layar, seret menaruhnya di titik lepas.

type ToolKind = CardKind
const tools: { kind: ToolKind; label: string; path: string }[] = [
  { kind: 'note', label: 'Catatan', path: 'M5 4h10l4 4v12H5zM15 4v4h4M8 12h8M8 16h5' },
  { kind: 'heading', label: 'Judul', path: 'M6 5v14M18 5v14M6 12h12' },
  { kind: 'checklist', label: 'Checklist', path: 'M4 7l2 2 3-3M4 15l2 2 3-3M12 8h8M12 16h8' },
  { kind: 'board', label: 'Papan', path: 'M4 4h16v16H4zM8 8h3v3H8zM13 8h3M13 11h3M8 14h8' },
]
const toolDrag = ref<{
  kind: ToolKind
  x: number
  y: number
  moved: boolean
  startX: number
  startY: number
} | null>(null)
let suppressClick = false

function onToolDown(kind: ToolKind, e: PointerEvent) {
  if (!canWrite.value || (e.pointerType === 'mouse' && e.button !== 0)) return
  toolDrag.value = { kind, x: e.clientX, y: e.clientY, moved: false, startX: e.clientX, startY: e.clientY }
  window.addEventListener('pointermove', onToolMove)
  window.addEventListener('pointerup', onToolUp)
  window.addEventListener('pointercancel', onToolCancel)
}
function onToolMove(e: PointerEvent) {
  const d = toolDrag.value
  if (!d) return
  d.x = e.clientX
  d.y = e.clientY
  if (!d.moved && Math.hypot(e.clientX - d.startX, e.clientY - d.startY) > 6) d.moved = true
  toolDrag.value = { ...d }
}
function endToolDrag() {
  window.removeEventListener('pointermove', onToolMove)
  window.removeEventListener('pointerup', onToolUp)
  window.removeEventListener('pointercancel', onToolCancel)
}
function onToolCancel() {
  endToolDrag()
  toolDrag.value = null
}
function onToolUp(e: PointerEvent) {
  endToolDrag()
  const d = toolDrag.value
  toolDrag.value = null
  if (!d?.moved || !canvas.value) return
  suppressClick = true
  const r = canvas.value.getBoundingClientRect()
  if (e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom) return
  addCard(d.kind, toWorld(view.value, e.clientX - r.left, e.clientY - r.top))
}
function onToolClick(kind: ToolKind) {
  if (suppressClick) {
    suppressClick = false
    return
  }
  sheetOpen.value = false
  addCard(kind)
}
const toolGhost = computed(() => {
  const d = toolDrag.value
  if (!d?.moved) return null
  return {
    label: tools.find((t) => t.kind === d.kind)?.label ?? '',
    style: { left: `${d.x + 8}px`, top: `${d.y + 8}px` },
  }
})

const sheetOpen = ref(false)
const trashOpen = ref(false)

function openTrash() {
  sheetOpen.value = false
  trashOpen.value = true
}

function onRestored(boardId: string) {
  if (boardId === board.value?.id) load(true)
  else
    router.push(
      boardId === crumbs.value[0]?.id
        ? { name: 'whiteboard' }
        : { name: 'whiteboard-board', params: { id: boardId } },
    )
  showToast('Isi Tempat sampah dipulihkan.')
}

// ---------------------------------------------------------------------------------------------------------------
// Gestur: seret kartu, ubah ukuran, geser kanvas, kotak seleksi, pinch

type Gesture =
  | {
      kind: 'move'
      startX: number
      startY: number
      ids: string[]
      started: boolean
      anchor: string
      touchTap: boolean
    }
  | { kind: 'resize'; startX: number; startY: number; id: string }
  | { kind: 'pan'; startX: number; startY: number; view: Viewport; tapCard: string | null }
  | { kind: 'box'; startX: number; startY: number; additive: boolean; base: Set<string> }
  | { kind: 'pinch' }

let gesture: Gesture | null = null
const touches = new Map<number, { x: number; y: number }>()
let pinchStart: { dist: number; mid: { x: number; y: number }; view: Viewport } | null = null
const dragOffset = ref({ dx: 0, dy: 0 })
const resizeDelta = ref({ dw: 0, dh: 0 })
const box = ref<{ x: number; y: number; w: number; h: number } | null>(null)
const gestureResizeId = ref<string | null>(null)
const spaceHeld = ref(false)
const panMode = ref(false)

function local(e: PointerEvent) {
  const r = canvas.value!.getBoundingClientRect()
  return { x: e.clientX - r.left, y: e.clientY - r.top }
}

function trackTouch(e: PointerEvent) {
  if (e.pointerType !== 'touch') return false
  touches.set(e.pointerId, local(e))
  if (touches.size === 2) {
    const [a, b] = [...touches.values()]
    pinchStart = {
      dist: Math.hypot(a!.x - b!.x, a!.y - b!.y) || 1,
      mid: { x: (a!.x + b!.x) / 2, y: (a!.y + b!.y) / 2 },
      view: { ...view.value },
    }
    gesture = { kind: 'pinch' }
    dragOffset.value = { dx: 0, dy: 0 }
    return true
  }
  return false
}

function begin(e: PointerEvent) {
  // Capture menjaga gerakan tetap terbaca saat penunjuk keluar kanvas. Pointer yang sudah lepas (atau sintetis)
  // membuat setPointerCapture melempar; listener di window tetap jalan tanpanya.
  try {
    canvas.value?.setPointerCapture(e.pointerId)
  } catch {
    // Lihat komentar di atas.
  }
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
  window.addEventListener('pointercancel', onUp)
}

function onCanvasDown(e: PointerEvent) {
  if (e.target !== canvas.value && !(e.target as HTMLElement).classList.contains('wb-world')) return
  if (editingId.value) stopEdit()
  // Fokus ke kanvas supaya pintasan keyboard langsung berlaku setelah klik di area kosong.
  canvas.value?.focus({ preventScroll: true })
  if (trackTouch(e)) return
  const p = local(e)
  if (e.pointerType === 'touch' || e.button === 1 || spaceHeld.value || panMode.value) {
    gesture = { kind: 'pan', startX: p.x, startY: p.y, view: { ...view.value }, tapCard: null }
  } else if (e.button === 0) {
    const additive = e.shiftKey
    gesture = {
      kind: 'box',
      startX: p.x,
      startY: p.y,
      additive,
      base: additive ? new Set(selected.value) : new Set(),
    }
    if (!additive) select([])
  } else {
    return
  }
  begin(e)
}

function onCardGrab(card: Card, e: PointerEvent) {
  if (e.button !== 0 && e.pointerType === 'mouse') return
  if (editingId.value && editingId.value !== card.id) stopEdit()
  focusedId.value = card.id
  if (trackTouch(e)) return
  const p = local(e)
  // Sentuhan pada kartu yang belum terpilih menggeser kanvas, supaya papan penuh kartu tetap bisa digeser di
  // ponsel. Ketukan tanpa gerak memilih kartu (kartu papan langsung dibuka); ketukan pada kartu terpilih membukanya.
  if (e.pointerType === 'touch' && !selected.value.has(card.id)) {
    gesture = { kind: 'pan', startX: p.x, startY: p.y, view: { ...view.value }, tapCard: card.id }
    begin(e)
    return
  }
  if (e.shiftKey) {
    const next = new Set(selected.value)
    if (next.has(card.id)) next.delete(card.id)
    else next.add(card.id)
    selected.value = next
    return
  }
  if (!selected.value.has(card.id)) select([card.id])
  gesture = {
    kind: 'move',
    startX: p.x,
    startY: p.y,
    ids: canWrite.value ? [...selected.value] : [],
    started: false,
    anchor: card.id,
    touchTap: e.pointerType === 'touch',
  }
  begin(e)
}

function onResizeStart(card: Card, e: PointerEvent) {
  if (!canWrite.value) return
  const p = local(e)
  gesture = { kind: 'resize', startX: p.x, startY: p.y, id: card.id }
  gestureResizeId.value = card.id
  begin(e)
}

function onMove(e: PointerEvent) {
  if (!gesture || !canvas.value) return
  const p = local(e)
  if (e.pointerType === 'touch' && touches.has(e.pointerId)) touches.set(e.pointerId, p)
  const z = view.value.zoom
  switch (gesture.kind) {
    case 'pinch': {
      if (!pinchStart || touches.size < 2) return
      const [a, b] = [...touches.values()]
      const dist = Math.hypot(a!.x - b!.x, a!.y - b!.y)
      const mid = { x: (a!.x + b!.x) / 2, y: (a!.y + b!.y) / 2 }
      const zoom = clamp((pinchStart.view.zoom * dist) / pinchStart.dist, MIN_ZOOM, MAX_ZOOM)
      const w = toWorld(pinchStart.view, pinchStart.mid.x, pinchStart.mid.y)
      view.value = { zoom, x: mid.x - w.x * zoom, y: mid.y - w.y * zoom }
      return
    }
    case 'pan':
      view.value = {
        ...gesture.view,
        x: gesture.view.x + p.x - gesture.startX,
        y: gesture.view.y + p.y - gesture.startY,
      }
      return
    case 'box': {
      const x = Math.min(p.x, gesture.startX)
      const y = Math.min(p.y, gesture.startY)
      box.value = { x, y, w: Math.abs(p.x - gesture.startX), h: Math.abs(p.y - gesture.startY) }
      const a = toWorld(view.value, x, y)
      const rect = { x: a.x, y: a.y, w: box.value.w / z, h: box.value.h / z }
      const hit = cards.value.filter((c) => intersects(rect, c)).map((c) => c.id)
      selected.value = new Set([...gesture.base, ...hit])
      return
    }
    case 'move': {
      if (!gesture.started && Math.hypot(p.x - gesture.startX, p.y - gesture.startY) < 4) return
      gesture.started = true
      if (gesture.ids.length === 0) return
      const dx = (p.x - gesture.startX) / z
      const dy = (p.y - gesture.startY) / z
      // Snap memakai kartu yang dipegang sebagai jangkar; kartu lain ikut dengan jarak yang sama.
      const anchor = byId.value.get(gesture.anchor)
      if (!anchor) return
      dragOffset.value = {
        dx: snap(anchor.x + dx, gridOn.value) - anchor.x,
        dy: snap(anchor.y + dy, gridOn.value) - anchor.y,
      }
      return
    }
    case 'resize': {
      const card = byId.value.get(gesture.id)
      if (!card) return
      const w = clamp(snap(card.w + (p.x - gesture.startX) / z, gridOn.value), MIN_SIZE, MAX_SIZE)
      const h = clamp(snap(card.h + (p.y - gesture.startY) / z, gridOn.value), MIN_SIZE, MAX_SIZE)
      resizeDelta.value = { dw: w - card.w, dh: h - card.h }
    }
  }
}

function onUp(e: PointerEvent) {
  touches.delete(e.pointerId)
  const g = gesture
  if (g?.kind === 'pinch' && touches.size > 0) return
  gesture = null
  window.removeEventListener('pointermove', onMove)
  window.removeEventListener('pointerup', onUp)
  window.removeEventListener('pointercancel', onUp)
  box.value = null
  if (!g) return
  if (g.kind === 'pan' && g.tapCard) {
    const p = local(e)
    if (Math.hypot(p.x - g.startX, p.y - g.startY) >= 8) return
    const card = byId.value.get(g.tapCard)
    if (card?.kind === 'board') openBoard(card)
    else select([g.tapCard])
    return
  }
  if (g.kind === 'move') {
    const { dx, dy } = dragOffset.value
    dragOffset.value = { dx: 0, dy: 0 }
    if (!g.started) {
      // Klik pada kartu papan membukanya; ketukan pada kartu lain yang sudah terpilih membuka editornya.
      const card = byId.value.get(g.anchor)
      if (card?.kind === 'board' && !e.shiftKey) openBoard(card)
      else if (g.touchTap && canWrite.value) startEdit(g.anchor)
      return
    }
    if (dx === 0 && dy === 0) return
    const patches: CardPatch[] = []
    for (const id of g.ids) {
      const c = byId.value.get(id)
      if (!c) continue
      const x = clamp(c.x + dx, -COORD_LIMIT, COORD_LIMIT)
      const y = clamp(c.y + dy, -COORD_LIMIT, COORD_LIMIT)
      patchLocal(id, { x, y })
      patches.push({ id, x, y })
    }
    announce.value = `${patches.length} kartu dipindah.`
    sendGeometry(patches)
  }
  if (g.kind === 'resize') {
    const { dw, dh } = resizeDelta.value
    resizeDelta.value = { dw: 0, dh: 0 }
    gestureResizeId.value = null
    const c = byId.value.get(g.id)
    if (!c || (dw === 0 && dh === 0)) return
    patchLocal(g.id, { w: c.w + dw, h: c.h + dh })
    sendGeometry([{ id: g.id, w: c.w + dw, h: c.h + dh }])
  }
}

function onWheel(e: WheelEvent) {
  const r = canvas.value!.getBoundingClientRect()
  if (e.ctrlKey || e.metaKey) {
    view.value = zoomAt(
      view.value,
      e.clientX - r.left,
      e.clientY - r.top,
      view.value.zoom * Math.exp(-e.deltaY * 0.01),
    )
  } else {
    view.value = { ...view.value, x: view.value.x - e.deltaX, y: view.value.y - e.deltaY }
  }
}

// ---------------------------------------------------------------------------------------------------------------
// Keyboard

let nudgeTimer: ReturnType<typeof setTimeout> | undefined
const nudged = new Set<string>()

function nudge(dx: number, dy: number) {
  for (const id of selected.value) {
    const c = byId.value.get(id)
    if (!c) continue
    patchLocal(id, {
      x: clamp(c.x + dx, -COORD_LIMIT, COORD_LIMIT),
      y: clamp(c.y + dy, -COORD_LIMIT, COORD_LIMIT),
    })
    nudged.add(id)
  }
  announce.value = `${selected.value.size} kartu digeser.`
  clearTimeout(nudgeTimer)
  nudgeTimer = setTimeout(() => {
    const patches = [...nudged].flatMap((id) => {
      const c = byId.value.get(id)
      return c ? [{ id, x: c.x, y: c.y }] : []
    })
    nudged.clear()
    sendGeometry(patches)
  }, 400)
}

const arrows: Record<string, Direction> = {
  ArrowUp: 'up',
  ArrowDown: 'down',
  ArrowLeft: 'left',
  ArrowRight: 'right',
}

function onKeydown(e: KeyboardEvent) {
  const target = e.target as HTMLElement
  // Saat kartu diedit, Escape yang tidak ditangkap editornya (mis. fokus masih di kartu, belum pindah ke isian)
  // tetap menutup editor; tombol lain dibiarkan untuk editor.
  if (editingId.value) {
    if (e.key === 'Escape') {
      e.preventDefault()
      stopEdit()
    }
    return
  }
  if (target.closest('input, textarea, select, [contenteditable="true"]')) return
  const mod = e.ctrlKey || e.metaKey
  if (e.key === ' ' && !mod) {
    spaceHeld.value = true
    e.preventDefault()
    return
  }
  const dir = arrows[e.key]
  if (dir) {
    e.preventDefault()
    if (e.shiftKey && canWrite.value && selected.value.size > 0) {
      const step = mod ? GRID * 8 : GRID
      nudge(
        dir === 'left' ? -step : dir === 'right' ? step : 0,
        dir === 'up' ? -step : dir === 'down' ? step : 0,
      )
      return
    }
    const from = focusedId.value ? byId.value.get(focusedId.value) : undefined
    const next = from ? nearestInDirection(from, cards.value, dir) : ordered.value[0]
    if (next) {
      select([next.id])
      focusCard(next.id)
    }
    return
  }
  if (mod && (e.key === 'a' || e.key === 'A')) {
    e.preventDefault()
    select(cards.value.map((c) => c.id))
    announce.value = `${cards.value.length} kartu terpilih.`
    return
  }
  if (mod && e.key === '0') {
    e.preventDefault()
    fit()
    return
  }
  if (mod && (e.key === '=' || e.key === '+')) {
    e.preventDefault()
    zoomBy(1.25)
    return
  }
  if (mod && e.key === '-') {
    e.preventDefault()
    zoomBy(0.8)
    return
  }
  if (mod || e.altKey) return
  switch (e.key) {
    case 'Enter': {
      const card = focusedId.value ? byId.value.get(focusedId.value) : undefined
      if (!card) return
      e.preventDefault()
      if (card.kind === 'board') openBoard(card)
      else if (canWrite.value) startEdit(card.id)
      return
    }
    case 'Escape':
      if (selected.value.size > 0) {
        e.preventDefault()
        select([])
        announce.value = 'Pilihan dibatalkan.'
      }
      return
    case 'Delete':
    case 'Backspace':
      if (selected.value.size > 0) {
        e.preventDefault()
        removeSelected()
      }
      return
    case 'Home':
    case 'End': {
      e.preventDefault()
      const c = e.key === 'Home' ? ordered.value[0] : ordered.value.at(-1)
      if (c) {
        select([c.id])
        focusCard(c.id)
      }
      return
    }
    case 'n':
    case 'N':
      e.preventDefault()
      addCard('note')
      return
    case 'g':
    case 'G':
      e.preventDefault()
      gridOn.value = !gridOn.value
      announce.value = gridOn.value ? 'Grid dan snap menyala.' : 'Grid dan snap mati.'
  }
}

function onKeyup(e: KeyboardEvent) {
  if (e.key === ' ') spaceHeld.value = false
}

// ---------------------------------------------------------------------------------------------------------------
// Riwayat

const historyOpen = ref(false)
const history = shallowRef<Activity[]>([])
const historyState = ref<'loading' | 'error' | 'ok'>('loading')
const historyError = ref('')

async function openHistory() {
  historyOpen.value = true
  historyState.value = 'loading'
  try {
    history.value = (await boardActivity(bid.value)).activity
    historyState.value = 'ok'
  } catch (e) {
    historyError.value = e instanceof ApiError ? e.message : 'Riwayat gagal dimuat.'
    historyState.value = 'error'
  }
}

watch(
  boardKey,
  () => {
    flushNotes()
    selected.value = new Set()
    editingId.value = null
    focusedId.value = null
    titleEditing.value = false
    searchOpen.value = false
    closeSearch()
    load()
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  flushNotes()
  abort?.abort()
  ro?.disconnect()
  clearTimeout(toastTimer)
  endToolDrag()
  document.removeEventListener('pointerdown', onDocPointer)
  window.removeEventListener('beforeunload', onBeforeUnload)
  window.removeEventListener('pointermove', onMove)
  window.removeEventListener('pointerup', onUp)
  window.removeEventListener('pointercancel', onUp)
})

const btn =
  'inline-flex min-h-11 items-center justify-center gap-1.5 rounded-card border border-line bg-surface px-3 font-medium hover:bg-canvas active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
const iconBtn =
  'inline-flex size-11 shrink-0 items-center justify-center rounded-card border border-line bg-surface hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark aria-pressed:bg-accent aria-pressed:text-on-accent'
const toolBtn =
  'flex min-h-14 w-full flex-col items-center justify-center gap-0.5 rounded-card px-1 text-xs text-ink hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-mark touch-none'
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <!-- Bar atas: lokasi, judul, cari, status simpan, riwayat. -->
    <div class="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-line bg-surface px-3 py-1.5">
      <nav aria-label="Lokasi papan" class="hidden min-w-0 md:block">
        <ol class="flex flex-wrap items-center gap-x-1 text-sm text-ink-muted">
          <template v-for="(c, i) in crumbs.slice(0, -1)" :key="c.id">
            <li class="min-w-0">
              <RouterLink
                :to="crumbLink(i)"
                class="inline-flex min-h-11 items-center break-all underline underline-offset-2 hover:text-ink focus-visible:outline-2 focus-visible:outline-mark"
              >
                {{ c.title }}
              </RouterLink>
            </li>
            <li aria-hidden="true">
              <svg
                viewBox="0 0 24 24"
                class="size-4"
                fill="none"
                stroke="currentColor"
                stroke-width="1.75"
                stroke-linecap="round"
              >
                <path d="M9 6l6 6-6 6" />
              </svg>
            </li>
          </template>
        </ol>
      </nav>
      <RouterLink
        v-if="parentCrumb"
        :to="crumbLink(crumbs.length - 2)"
        :class="iconBtn"
        class="md:hidden"
        :aria-label="`Kembali ke ${parentCrumb.title}`"
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
          <path d="M15 6l-6 6 6 6" />
        </svg>
      </RouterLink>

      <div class="flex min-w-0 flex-1 items-center gap-1">
        <template v-if="titleEditing">
          <label :for="`${uid}-title`" class="sr-only">Judul papan</label>
          <input
            :id="`${uid}-title`"
            ref="titleInput"
            v-model="titleDraft"
            maxlength="120"
            :aria-invalid="titleError ? 'true' : undefined"
            :aria-describedby="titleError ? `${uid}-title-err` : undefined"
            class="min-h-11 min-w-0 flex-1 rounded-card border border-line bg-surface px-2 text-lg font-bold"
            @keydown="onTitleKey"
            @blur="saveTitle"
          />
          <p v-if="titleError" :id="`${uid}-title-err`" role="alert" class="text-sm text-danger">
            {{ titleError }}
          </p>
        </template>
        <template v-else>
          <h1 class="min-w-0 truncate text-lg font-bold">{{ board?.title ?? 'Whiteboard' }}</h1>
          <button
            v-if="canWrite && board"
            ref="titleButton"
            type="button"
            :class="iconBtn"
            class="border-transparent"
            aria-label="Ubah judul papan"
            title="Ubah judul papan"
            @click="editTitle"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 24 24"
              class="size-4"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linejoin="round"
            >
              <path d="M4 20h4L19 9l-4-4L4 16zM13 7l4 4" />
            </svg>
          </button>
        </template>
      </div>

      <div ref="searchBox" class="relative flex items-center gap-2" @keydown="onSearchKey">
        <button
          type="button"
          :class="iconBtn"
          class="md:hidden"
          :aria-expanded="searchOpen"
          :aria-controls="`${uid}-search`"
          aria-label="Cari papan"
          @click="searchOpen = !searchOpen"
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
            <path d="M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14M20 20l-4-4" />
          </svg>
        </button>
        <form
          :id="`${uid}-search`"
          role="search"
          class="max-md:absolute max-md:top-12 max-md:right-0 max-md:z-30 max-md:w-[min(20rem,calc(100vw-1.5rem))] max-md:rounded-card max-md:border max-md:border-line max-md:bg-surface max-md:p-2 max-md:shadow-menu"
          :class="searchOpen ? '' : 'max-md:hidden'"
          novalidate
          @submit.prevent="search"
        >
          <label :for="`${uid}-q`" class="sr-only">Cari judul papan</label>
          <input
            :id="`${uid}-q`"
            v-model="query"
            type="search"
            enterkeyhint="search"
            maxlength="100"
            placeholder="Cari papan"
            :aria-invalid="searchError ? 'true' : undefined"
            :aria-describedby="searchError ? `${uid}-qerr` : undefined"
            class="min-h-11 w-full rounded-card border border-line bg-surface px-3 md:w-56"
          />
          <div
            v-if="results || searchError"
            class="z-30 mt-1 max-h-72 overflow-y-auto rounded-card border border-line bg-surface shadow-menu md:absolute md:right-0 md:w-80"
          >
            <p v-if="searchError" :id="`${uid}-qerr`" role="alert" class="p-3 text-sm text-danger">
              {{ searchError }}
            </p>
            <p v-else-if="results && results.length === 0" role="status" class="p-3 text-sm text-ink-muted">
              Tidak ada papan dengan judul itu.
            </p>
            <ul v-else-if="results" aria-label="Hasil pencarian papan">
              <li v-for="r in results" :key="r.id">
                <RouterLink
                  :to="{ name: 'whiteboard-board', params: { id: r.id } }"
                  class="flex min-h-11 flex-col justify-center px-3 py-2 hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-mark"
                >
                  <span class="font-semibold break-words">{{ r.title }}</span>
                  <span class="text-sm text-ink-muted">{{ r.path }}</span>
                </RouterLink>
              </li>
            </ul>
          </div>
        </form>
      </div>

      <p
        class="inline-flex min-h-11 items-center gap-1.5 text-sm"
        :class="saveError ? 'text-danger' : 'text-ink-muted'"
        role="status"
      >
        <svg
          v-if="saveError"
          aria-hidden="true"
          viewBox="0 0 24 24"
          class="size-4"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
        >
          <path d="M12 8v5M12 16h.01M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18" />
        </svg>
        <span data-testid="wb-save">{{ saveLabel }}</span>
        <span v-if="saveError" class="sr-only">: {{ saveError }}</span>
      </p>
      <button type="button" :class="btn" @click="openHistory">Riwayat</button>
      <p v-if="saveError" class="w-full text-sm text-danger">{{ saveError }}</p>
    </div>

    <div class="relative flex min-h-0 flex-1">
      <!-- Sidebar alat (mulai 768 px). Klik menaruh kartu di tengah layar, seret menaruhnya di titik lepas. -->
      <aside
        aria-label="Alat papan"
        class="hidden w-20 shrink-0 flex-col justify-between border-r border-line bg-surface p-1.5 md:flex"
      >
        <ul v-if="canWrite" class="flex flex-col gap-1">
          <li v-for="t in tools" :key="t.kind">
            <button
              type="button"
              :class="toolBtn"
              :title="`${t.label}: klik untuk menaruh di tengah, atau seret ke kanvas`"
              @pointerdown="onToolDown(t.kind, $event)"
              @click="onToolClick(t.kind)"
            >
              <svg
                aria-hidden="true"
                viewBox="0 0 24 24"
                class="size-6"
                fill="none"
                stroke="currentColor"
                stroke-width="1.75"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <path :d="t.path" />
              </svg>
              {{ t.label }}
            </button>
          </li>
        </ul>
        <span v-if="!canWrite" class="px-1 py-2 text-center text-xs text-ink-muted">Hanya baca</span>
        <button type="button" :class="toolBtn" @click="openTrash">
          <svg
            aria-hidden="true"
            viewBox="0 0 24 24"
            class="size-6"
            fill="none"
            stroke="currentColor"
            stroke-width="1.75"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3" />
          </svg>
          Tempat sampah
        </button>
      </aside>

      <div class="relative min-w-0 flex-1">
        <div
          v-if="state === 'loading'"
          class="flex h-full items-center justify-center text-ink-muted"
          role="status"
        >
          Memuat papan...
        </div>
        <div
          v-else-if="state === 'error'"
          class="flex h-full flex-col items-center justify-center gap-3 p-6 text-center"
          role="alert"
        >
          <p>{{ loadError }}</p>
          <div class="flex flex-wrap justify-center gap-2">
            <button type="button" :class="btn" @click="load()">Coba lagi</button>
            <RouterLink :to="{ name: 'whiteboard' }" :class="btn">Ke papan Whiteboard</RouterLink>
          </div>
        </div>

        <div
          v-show="state === 'ok'"
          :id="`${uid}-canvas`"
          ref="canvas"
          role="application"
          :aria-label="`Kanvas papan ${board?.title ?? ''}`"
          :aria-describedby="`${uid}-help`"
          tabindex="-1"
          class="wb-canvas absolute inset-0 touch-none overflow-hidden bg-canvas outline-none focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-mark"
          :class="[gridOn ? 'wb-grid' : '', spaceHeld || panMode ? 'cursor-grab' : '']"
          :style="gridStyle"
          @pointerdown="onCanvasDown"
          @wheel.prevent="onWheel"
          @keydown="onKeydown"
          @keyup="onKeyup"
        >
          <div class="wb-world absolute top-0 left-0 origin-top-left" :style="worldStyle">
            <WbCard
              v-for="c in rendered"
              :key="c.id"
              :ref="(inst) => setCardRef(c.id, inst)"
              :card="c"
              :selected="selected.has(c.id)"
              :editing="editingId === c.id"
              :can-write="canWrite"
              :focusable="rovingId === c.id"
              :zoom="view.zoom"
              :dx="selected.has(c.id) ? dragOffset.dx : 0"
              :dy="selected.has(c.id) ? dragOffset.dy : 0"
              :dw="gestureResizeId === c.id ? resizeDelta.dw : 0"
              :dh="gestureResizeId === c.id ? resizeDelta.dh : 0"
              @focus="focusedId = c.id"
              @grab="onCardGrab(c, $event)"
              @resize="onResizeStart(c, $event)"
              @edit="c.kind === 'board' ? openBoard(c) : startEdit(c.id)"
              @close="stopEdit"
              @note="onNoteInput(c.id, $event)"
              @heading="onHeading(c.id, $event)"
              @rename="onRenameRef(c.id, $event)"
              @checklist="onChecklist(c.id, $event)"
            />
          </div>
          <div
            v-if="box"
            aria-hidden="true"
            class="pointer-events-none absolute border border-dashed border-mark bg-mark/10"
            :style="{ left: `${box.x}px`, top: `${box.y}px`, width: `${box.w}px`, height: `${box.h}px` }"
          />
          <div
            v-if="cards.length === 0"
            class="pointer-events-none absolute inset-0 flex items-center justify-center p-6 text-center text-ink-muted"
          >
            <p class="max-w-sm">
              {{
                canWrite
                  ? 'Papan masih kosong. Tambah kartu dari alat di kiri (di ponsel: tombol Tambah), atau tekan N.'
                  : 'Papan ini masih kosong.'
              }}
            </p>
          </div>
        </div>

        <!-- Pilihan: warna, ubah isi, hapus. -->
        <div
          v-if="state === 'ok' && canWrite && selected.size > 0"
          class="absolute top-2 left-2 z-20 flex max-w-[calc(100%-1rem)] flex-wrap items-center gap-2 rounded-card border border-line bg-surface p-1.5 shadow-menu"
        >
          <label :for="`${uid}-color`" class="sr-only">Warna kartu terpilih</label>
          <select
            :id="`${uid}-color`"
            class="min-h-11 rounded-card border border-line bg-surface px-2"
            :value="selectionColor"
            @change="setColor(($event.target as HTMLSelectElement).value)"
          >
            <option v-if="selectionColor === 'campuran'" value="campuran" disabled>Warna campuran</option>
            <option v-for="c in colors" :key="c.label" :value="c.value ?? ''">Warna: {{ c.label }}</option>
          </select>
          <button v-if="selected.size === 1" type="button" :class="btn" @click="startEdit([...selected][0]!)">
            {{ byId.get([...selected][0]!)?.kind === 'board' ? 'Ubah judul' : 'Ubah isi' }}
          </button>
          <button type="button" :class="btn" class="text-urgent" @click="removeSelected">
            Hapus {{ selected.size }} kartu
          </button>
        </div>

        <!-- Kontrol tampilan di kanan bawah; tombol Tambah di kiri bawah untuk ponsel. -->
        <div
          v-if="state === 'ok'"
          class="absolute right-2 bottom-2 z-20 flex flex-wrap items-center justify-end gap-1 rounded-card border border-line bg-surface p-1 shadow-menu"
        >
          <button
            type="button"
            :class="iconBtn"
            class="border-transparent"
            aria-label="Perkecil"
            title="Perkecil (Ctrl -)"
            @click="zoomBy(0.8)"
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
              <path d="M5 12h14" />
            </svg>
          </button>
          <span class="min-w-11 text-center text-sm tabular-nums">{{ zoomPercent }}%</span>
          <button
            type="button"
            :class="iconBtn"
            class="border-transparent"
            aria-label="Perbesar"
            title="Perbesar (Ctrl +)"
            @click="zoomBy(1.25)"
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
              <path d="M12 5v14M5 12h14" />
            </svg>
          </button>
          <button
            type="button"
            :class="iconBtn"
            class="border-transparent"
            aria-label="Pas ke layar"
            title="Pas ke layar (Ctrl 0)"
            @click="fit"
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
              <path d="M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5" />
            </svg>
          </button>
          <button
            type="button"
            :class="iconBtn"
            class="border-transparent"
            :aria-pressed="gridOn ? 'true' : 'false'"
            aria-label="Grid dan snap"
            title="Grid dan snap (G)"
            @click="gridOn = !gridOn"
          >
            <svg
              aria-hidden="true"
              viewBox="0 0 24 24"
              class="size-5"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
            >
              <path d="M4 4h16v16H4zM4 12h16M12 4v16" />
            </svg>
          </button>
          <button
            type="button"
            :class="iconBtn"
            class="border-transparent max-md:hidden"
            :aria-pressed="panMode ? 'true' : 'false'"
            aria-label="Mode geser kanvas"
            title="Mode geser kanvas (tahan Spasi)"
            @click="panMode = !panMode"
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
                d="M12 3v18M3 12h18M12 3l-3 3M12 3l3 3M12 21l-3-3M12 21l3-3M3 12l3-3M3 12l3 3M21 12l-3-3M21 12l-3 3"
              />
            </svg>
          </button>
        </div>
        <button
          v-if="state === 'ok'"
          type="button"
          class="absolute bottom-2 left-2 z-20 inline-flex min-h-11 items-center gap-1.5 rounded-card bg-accent px-4 font-medium text-on-accent shadow-menu hover:bg-accent-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark md:hidden"
          @click="sheetOpen = true"
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
            <path d="M12 5v14M5 12h14" />
          </svg>
          {{ canWrite ? 'Tambah' : 'Menu papan' }}
        </button>

        <div
          v-if="toast"
          class="absolute right-2 bottom-16 left-2 z-30 mx-auto flex max-w-md flex-wrap items-center justify-between gap-2 rounded-card border border-line bg-surface p-3 shadow-dialog"
          role="status"
        >
          <span class="min-w-0">{{ toast.text }}</span>
          <div class="flex gap-2">
            <button v-if="toast.batch && canWrite" type="button" :class="btn" @click="undoDelete">
              Urungkan
            </button>
            <button type="button" :class="btn" @click="toast = null">Tutup</button>
          </div>
        </div>
      </div>
    </div>

    <p :id="`${uid}-help`" class="sr-only">
      Panah: pindah antar kartu. Shift dan panah: geser kartu terpilih. Enter: ubah isi atau buka papan.
      Delete: hapus. Ctrl A: pilih semua. N: catatan baru. Ctrl 0: pas ke layar. Tahan Spasi untuk menggeser
      kanvas.
    </p>
    <p class="sr-only" aria-live="polite">{{ announce }}</p>

    <div
      v-if="toolGhost"
      aria-hidden="true"
      class="pointer-events-none fixed z-50 rounded-card border border-mark bg-surface px-3 py-2 text-sm font-medium shadow-drag"
      :style="toolGhost.style"
    >
      {{ toolGhost.label }}
    </div>

    <!-- Ponsel: lembar pilihan alat menggantikan sidebar. -->
    <BaseDialog
      :open="sheetOpen"
      :title="canWrite ? 'Tambah ke papan' : 'Menu papan'"
      @close="sheetOpen = false"
    >
      <div class="grid grid-cols-2 gap-2 p-4">
        <template v-if="canWrite">
          <button
            v-for="t in tools"
            :key="t.kind"
            type="button"
            :class="btn"
            class="min-h-14"
            @click="onToolClick(t.kind)"
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
              <path :d="t.path" />
            </svg>
            {{ t.label }}
          </button>
        </template>
        <button type="button" :class="btn" class="col-span-2 min-h-14" @click="openTrash">
          Tempat sampah
        </button>
      </div>
    </BaseDialog>

    <WbTrashPanel :open="trashOpen" @close="trashOpen = false" @restored="onRestored" />

    <BaseDialog :open="historyOpen" title="Riwayat papan" variant="panel" @close="historyOpen = false">
      <div class="min-h-0 flex-1 overflow-y-auto px-5 py-4">
        <p v-if="historyState === 'loading'" role="status" class="text-ink-muted">Memuat riwayat...</p>
        <p v-else-if="historyState === 'error'" role="alert" class="text-danger">{{ historyError }}</p>
        <p v-else-if="history.length === 0" class="text-ink-muted">Belum ada riwayat.</p>
        <ol v-else class="flex flex-col gap-3">
          <li v-for="a in history" :key="a.id" class="border-b border-line pb-3 last:border-b-0">
            <p>
              <span class="font-semibold">{{ a.user_name }}</span> {{ activityText(a) }}
            </p>
            <p class="text-sm text-ink-muted">
              <time :datetime="a.updated_at">{{ formatDateTime(a.updated_at) }}</time>
            </p>
          </li>
        </ol>
      </div>
    </BaseDialog>
  </div>
</template>
