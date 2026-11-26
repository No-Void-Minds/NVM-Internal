<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, useId, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import BaseDialog from '../components/BaseDialog.vue'
import UserAvatar from '../components/UserAvatar.vue'
import WbCard from '../components/whiteboard/WbCard.vue'
import WbTrashPanel from '../components/whiteboard/WbTrashPanel.vue'
import WbConnectorLayer from '../components/whiteboard/WbConnectorLayer.vue'
import WbConnectorDialog from '../components/whiteboard/WbConnectorDialog.vue'
import WbCommentsPanel from '../components/whiteboard/WbCommentsPanel.vue'
import WbMoveDialog from '../components/whiteboard/WbMoveDialog.vue'
import WbLinkDialog from '../components/whiteboard/WbLinkDialog.vue'
import WbInk from '../components/whiteboard/WbInk.vue'
import WbPenBar from '../components/whiteboard/WbPenBar.vue'
import WbTemplateDialog from '../components/whiteboard/WbTemplateDialog.vue'
import WbExportDialog from '../components/whiteboard/WbExportDialog.vue'
import WbListView from '../components/whiteboard/WbListView.vue'
import WbShapePalette from '../components/whiteboard/WbShapePalette.vue'
import WbShapeStyle from '../components/whiteboard/WbShapeStyle.vue'
import WbDiagramHandles from '../components/whiteboard/WbDiagramHandles.vue'
import WbContextMenu, { type MenuItem } from '../components/whiteboard/WbContextMenu.vue'
import WbStylePanel from '../components/whiteboard/WbStylePanel.vue'
import WbImageDialog from '../components/whiteboard/WbImageDialog.vue'
import WbTeamDialog from '../components/whiteboard/WbTeamDialog.vue'
import WbPicker from '../components/whiteboard/WbPicker.vue'
import type { DrawnLine } from '../components/whiteboard/WbConnectorLayer.vue'
import type { ConnectorUpdate } from '../components/whiteboard/WbConnectorDialog.vue'
import {
  SHAPES,
  angleFrom,
  center as boxCenter,
  newShapeContent,
  portPoint,
  portsFor,
  resizeBox,
  routeConnector,
  type Handle,
  type Port,
  type Pt,
  type ShapeContent,
  type ShapeType,
} from '../utils/wbShapes'
import { inkColor, penColors } from '../components/whiteboard/wbColors'
import { caretHint, type CaretHint } from '../components/whiteboard/caret'
import { ApiError, notifyUnauthorized } from '../api/client'
import {
  boardActivity,
  checklistOp,
  createCard,
  createConnector,
  createDrawing,
  deleteCards,
  deleteConnector,
  downloadUrl,
  drawingOp,
  fetchPreview,
  fromTemplate,
  getBoard,
  moveCards,
  patchCards,
  renameBoard,
  replaceFile,
  restoreBatch,
  searchBoards,
  setTemplate,
  tableOp,
  updateConnector,
  uploadFile,
  type BoardView,
  type CardHit,
  type CardPatch,
  type ChecklistOp,
  type NewCard,
  type PenStroke,
  type SearchResult,
  type TableOp,
  type TemplateInfo,
} from '../api/whiteboard'
import { formatBytes } from '../utils/attachment'
import { pastedFiles } from '../utils/pasteFiles'
import { useSessionStore } from '../stores/session'
import { formatDateTime } from '../utils/format'
import {
  activityText,
  bounds,
  cardLabel,
  clamp,
  colors,
  columnLayout,
  columnOrderAt,
  COORD_LIMIT,
  DRAWING_SPACE,
  defaultSize,
  drawingPoints,
  emptyContent,
  fitView,
  GRID,
  hitStroke,
  intersects,
  kindLabel,
  MAX_DRAWING_POINTS,
  MAX_SIZE,
  MAX_STROKE_POINTS,
  MAX_STROKES,
  MAX_ZOOM,
  MIN_SIZE,
  MIN_ZOOM,
  nearestInDirection,
  newId,
  readingOrder,
  simplify,
  snap,
  strokesBox,
  toWorld,
  visibleWorld,
  wheelPixels,
  wheelZoomFactor,
  zoomAt,
  type Activity,
  type CaptionContent,
  type Card,
  type CardColor,
  type CardStyle,
  type Crop,
  type CardKind,
  type ChecklistContent,
  type Connector,
  type Direction,
  type DrawingContent,
  type Rect,
  type HeadingContent,
  type PMNode,
  type Viewport,
} from '../utils/whiteboard'
import { geo, layoutPatches, push, type Geo, type Step } from '../utils/wbHistory'
import { WbSocket, reconnectDelay, type SocketState } from '../utils/wbSocket'
import {
  arrange,
  expand,
  group as groupPaths,
  hasGroup,
  remap,
  tidy,
  ungroup as ungroupPaths,
  unitOf,
  units,
  type AlignKind,
} from '../utils/wbGroups'
import { normalize, pasteStyle, styleKeys, type StyleKey } from '../utils/wbStyle'
import {
  DEFAULT_PERSON,
  PERSON_LAYOUTS,
  PERSON_SIZES,
  PERSON_VIEWS,
  gridAt,
  personSize,
  type PersonContent,
  type PersonInfo,
} from '../utils/wbPerson'
import { find as findShortcut, label as keyLabel, type ShortcutId } from '../utils/wbShortcuts'

// Whiteboard seperti Milanote (docs/whiteboard.md): semua adalah papan. /whiteboard membuka papan akar bersama,
// /whiteboard/:id membuka papan lain di kanvas yang sama. Data dari server adalah sumber kebenaran: perubahan
// dikirim per field lewat REST, isi kartu dijaga content_version. Sejak W2 perubahan orang lain, kehadiran, kursor,
// pilihan, gerakan seret, dan kunci edit datang lewat WebSocket (utils/wbSocket.ts).
const route = useRoute()
const router = useRouter()
const uid = useId()
const session = useSessionStore()
const myId = computed(() => session.user?.id ?? 0)
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
const connectors = shallowRef<Connector[]>([])

// Geometri tampil (W3): isi kolom ditumpuk di kolomnya dan kolom memanjang mengikuti isinya. Posisi tersimpan isi
// kolom diabaikan selama kartunya ada di kolom.
const layout = computed(() => columnLayout(cards.value))
function shown(c: Card): Card {
  const ch = layout.value.children.get(c.id)
  if (ch) return { ...c, x: ch.x, y: ch.y, w: ch.w, z: ch.z + 1 }
  const h = layout.value.heights.get(c.id)
  return h === undefined ? c : { ...c, h }
}
const shownCards = computed(() => cards.value.map(shown))
const shownById = computed(() => new Map(shownCards.value.map((c) => [c.id, c])))

function upsertConnectors(list: Connector[]) {
  const map = new Map(connectors.value.map((c) => [c.id, c]))
  for (const c of list) {
    const cur = map.get(c.id)
    if (cur && cur.version > c.version) continue
    map.set(c.id, c)
    lastSeq = Math.max(lastSeq, c.seq)
  }
  connectors.value = [...map.values()]
}
function removeConnectors(ids: string[]) {
  const gone = new Set(ids)
  connectors.value = connectors.value.filter((c) => !gone.has(c.id))
}
// seq terbesar yang sudah diterapkan; dikirim saat join supaya server menyusulkan perubahan yang terlewat.
let lastSeq = 0

function upsert(updated: Card[]) {
  const map = new Map(cards.value.map((c) => [c.id, c]))
  for (const c of updated) {
    const cur = map.get(c.id)
    // Versi yang tidak lebih baru dari yang dipegang diabaikan, jadi urutan sampai yang acak tidak mengembalikan
    // keadaan lama.
    if (cur && cur.version > c.version) continue
    map.set(c.id, c)
    lastSeq = Math.max(lastSeq, c.seq)
  }
  cards.value = [...map.values()]
}
function patchLocal(id: string, fields: Partial<Card>) {
  cards.value = cards.value.map((c) => (c.id === id ? { ...c, ...fields } : c))
}
function removeLocal(ids: string[]) {
  const gone = new Set(ids)
  cards.value = cards.value.filter((c) => !gone.has(c.id))
  if ([...selected.value].some((id) => gone.has(id)))
    select([...selected.value].filter((id) => !gone.has(id)))
  if (focusedId.value && gone.has(focusedId.value)) focusedId.value = null
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
    connectors.value = res.board.connectors ?? []
    lastSeq = res.board.seq
    state.value = 'ok'
    sendJoin()
    if (!keepView) {
      await nextTick()
      measure()
      fit()
    }
    void jumpToQueryCard()
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

function applyTitle(title: string, version: number) {
  const b = board.value
  if (!b) return
  board.value = {
    ...b,
    title,
    version,
    crumbs: b.crumbs.map((c) => (c.id === b.id ? { ...c, title } : c)),
  }
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
    applyTitle(res.board.title, res.board.version)
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
// Kartu yang isinya cocok (W4), di semua papan.
const cardHits = shallowRef<CardHit[]>([])
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
    const res = await searchBoards(q)
    cardHits.value = res.cards ?? []
    results.value = res.results
  } catch (e) {
    searchError.value = e instanceof ApiError ? e.message : 'Pencarian gagal. Coba lagi.'
  }
}

function closeSearch() {
  results.value = null
  cardHits.value = []
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

// Di bawah 768 px bar pilihan (dua baris) menempati sekitar 110 px teratas kanvas, dan kontrol zoom serta tombol
// Tambah sekitar 110 px terbawah.
const FIT_TOP_PHONE = 160
const FIT_BOTTOM_PHONE = 120
function fit() {
  const w = size.value.w
  const phone = w < 768
  view.value = fitView(
    shownCards.value,
    w,
    size.value.h,
    phone ? FIT_TOP_PHONE : 48,
    phone ? FIT_BOTTOM_PHONE : 48,
  )
}
function zoomBy(factor: number) {
  view.value = zoomAt(view.value, size.value.w / 2, size.value.h / 2, view.value.zoom * factor)
}
const zoomPercent = computed(() => Math.round(view.value.zoom * 100))

const worldStyle = computed(() => ({
  transform: `translate(${view.value.x}px, ${view.value.y}px) scale(${view.value.zoom})`,
}))
const gridOn = ref(true)
// Snap terpisah dari grid sejak revisi 50 (menu klik kanan punya keduanya).
const snapOn = ref(true)
const gridStyle = computed(() => {
  const step = GRID * 4 * view.value.zoom
  return gridOn.value
    ? { backgroundSize: `${step}px ${step}px`, backgroundPosition: `${view.value.x}px ${view.value.y}px` }
    : {}
})

function centerWorld() {
  return toWorld(view.value, size.value.w / 2, size.value.h / 2)
}

function centerOn(x: number, y: number) {
  const z = view.value.zoom
  view.value = { zoom: z, x: size.value.w / 2 - x * z, y: size.value.h / 2 - y * z }
}

function reveal(stored: Card) {
  const card = shownById.value.get(stored.id) ?? stored
  const tl = toWorld(view.value, 0, 0)
  const br = toWorld(view.value, size.value.w, size.value.h)
  if (card.x >= tl.x && card.y >= tl.y && card.x + card.w <= br.x && card.y + card.h <= br.y) return
  centerOn(card.x + card.w / 2, card.y + card.h / 2)
}

// ---------------------------------------------------------------------------------------------------------------
// Pilihan, fokus, dan kartu yang dirender

const selected = ref<Set<string>>(new Set())
const focusedId = ref<string | null>(null)
const editingId = ref<string | null>(null)

function select(ids: string[]) {
  selected.value = new Set(ids)
}

// Gerakan seret orang lain yang sedang berlangsung, menimpa posisi tersimpan sampai dilepas atau 3 detik diam.
const remoteMoves = shallowRef<Map<string, { x: number; y: number; w: number; h: number; at: number }>>(
  new Map(),
)

// Culling: hanya kartu di area terlihat plus satu layar margin yang di-mount. Kartu terpilih, terfokus, dan yang
// sedang diedit selalu ikut, supaya fokus keyboard dan editor tidak hilang saat kanvas digeser.
const rendered = computed(() => {
  const area = visibleWorld(view.value, size.value.w || 1, size.value.h || 1)
  const keep = new Set([...selected.value, focusedId.value, editingId.value])
  const moves = remoteMoves.value
  const list = shownCards.value.map((c) => {
    const m = moves.get(c.id)
    return m ? { ...c, x: m.x, y: m.y, w: m.w, h: m.h } : c
  })
  return readingOrder(list.filter((c) => keep.has(c.id) || intersects(area, c)))
})
const ordered = computed(() => readingOrder(shownCards.value))

// Kartu ikut bergeser saat diseret bila kartu itu sendiri atau kolomnya terpilih.
function dragged(c: Card) {
  return selected.value.has(c.id) || (!!c.column_id && selected.value.has(c.column_id))
}
// Geometri tampil plus geser seret, untuk ujung garis penghubung.
const geomMap = computed(() => {
  const { dx, dy } = dragOffset.value
  const out = new Map<string, Rect>()
  for (const c of shownCards.value) {
    const d = dragged(c)
    const r = gestureResizeId.value === c.id ? resizeDelta.value : { dx: 0, dy: 0, dw: 0, dh: 0 }
    out.set(c.id, {
      x: c.x + (d ? dx : 0) + r.dx,
      y: c.y + (d ? dy : 0) + r.dy,
      w: c.w + r.dw,
      h: c.h + r.dh,
    })
  }
  return out
})
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

// Lompat ke kartu dari hasil pencarian isi (W4): ?kartu=<id>. Hanya setelah papan tujuan selesai dimuat, dan
// parameternya dibuang sesudah dipakai supaya muat ulang halaman tidak melompat lagi.
async function jumpToQueryCard() {
  const id = typeof route.query.kartu === 'string' ? route.query.kartu : ''
  const b = board.value
  if (
    !id ||
    state.value !== 'ok' ||
    !b ||
    (b.id !== boardKey.value && !(boardKey.value === 'root' && b.is_root))
  )
    return
  void router.replace({ query: { ...route.query, kartu: undefined } })
  if (!byId.value.has(id)) {
    showToast('Kartu itu sudah tidak ada di papan ini.')
    return
  }
  listMode.value = false
  await nextTick()
  select([id])
  await focusCard(id)
}
watch(
  () => route.query.kartu,
  () => void jumpToQueryCard(),
)

// Tampilan Daftar (W4): kanvas tetap terpasang (tersembunyi), jadi kembali ke kanvas tidak memuat ulang papan.
const listMode = ref(false)
async function toCanvas(id: string) {
  listMode.value = false
  await nextTick()
  measure()
  select([id])
}
async function listShow(id: string) {
  await toCanvas(id)
  await focusCard(id)
}
async function listEdit(id: string) {
  await toCanvas(id)
  startEdit(id)
}
function listOpen(id: string) {
  const c = byId.value.get(id)
  if (c) openBoard(c)
}
function listMove(id: string) {
  select([id])
  moveError.value = ''
  moveOpen.value = true
}
function listRemove(id: string) {
  select([id])
  void removeSelected()
}

// Template (W4)
const templateOpen = ref(false)
const templateBusy = ref(false)
const templateError = ref('')
async function placeTemplate(t: TemplateInfo) {
  if (!canWrite.value || templateBusy.value) return
  const dim = defaultSize.board
  const c = toolAt ?? centerWorld()
  templateBusy.value = true
  templateError.value = ''
  try {
    const res = await fromTemplate(bid.value, {
      template_id: t.id,
      id: newId(),
      x: clamp(snap(c.x - dim.w / 2, snapOn.value), -COORD_LIMIT, COORD_LIMIT),
      y: clamp(snap(c.y - dim.h / 2, snapOn.value), -COORD_LIMIT, COORD_LIMIT),
      w: dim.w,
      h: dim.h,
    })
    upsert([res.card])
    record({ kind: 'create', ids: [res.card.id] })
    templateOpen.value = false
    select([res.card.id])
    await focusCard(res.card.id)
    announce.value = `Papan ${t.title} dibuat dari template.`
  } catch (e) {
    templateError.value = e instanceof ApiError ? e.message : 'Papan gagal dibuat dari template. Coba lagi.'
  } finally {
    templateBusy.value = false
  }
}
async function toggleTemplate() {
  const b = board.value
  if (!b || b.is_root || !canWrite.value) return
  sheetOpen.value = false
  try {
    const res = await setTemplate(b.id, !b.is_template)
    board.value = { ...b, is_template: res.board.is_template }
    announce.value = res.board.is_template ? 'Papan ini sekarang template.' : 'Papan ini bukan template lagi.'
  } catch (e) {
    showToast(e instanceof ApiError ? e.message : 'Status template gagal diubah. Coba lagi.')
  }
}

const exportOpen = ref(false)
function openExport() {
  sheetOpen.value = false
  exportOpen.value = true
}

function openBoard(card: Card) {
  if (card.kind !== 'board' || !card.ref) return
  router.push({ name: 'whiteboard-board', params: { id: card.ref.id } })
}

// ---------------------------------------------------------------------------------------------------------------
// Status simpan dan antrean saat terputus

const pending = ref(0)
const saveError = ref('')
const sockState = ref<SocketState>('connecting')
// netDown: ada perubahan yang gagal terkirim karena jaringan atau backend tidak menjawab; menunggu di antrean.
const netDown = ref(false)
const offlineQueue: (() => Promise<unknown>)[] = []
// Teks pendek supaya bar atas tidak pindah baris saat terputus; kalimat lengkapnya ada di penanda tengah kanvas.
const saveLabel = computed(() =>
  sockState.value !== 'open' || netDown.value
    ? 'Terputus'
    : saveError.value
      ? 'Gagal disimpan'
      : pending.value > 0
        ? 'Menyimpan...'
        : 'Tersimpan',
)
const announce = ref('')

// Galat jaringan: fetch gagal (status 0) atau proxy menjawab tanpa backend (backend sedang di-restart).
function isNetworkError(e: unknown) {
  return e instanceof ApiError && (e.status === 0 || e.code === 'backend_unreachable')
}

// saving membungkus setiap tulisan REST. Galat jaringan tidak membuang perubahan: pekerjaannya masuk antrean dan
// diulang saat tersambung lagi (hasilnya datang lewat siaran changed). Galat lain memuat ulang papan dari server.
async function saving<T>(work: () => Promise<T>): Promise<T | undefined> {
  pending.value++
  try {
    const out = await work()
    saveError.value = ''
    return out
  } catch (e) {
    if (isNetworkError(e)) {
      netDown.value = true
      offlineQueue.push(work)
      scheduleFlush()
      return undefined
    }
    saveError.value = e instanceof ApiError ? e.message : 'Perubahan gagal disimpan. Periksa koneksi.'
    if (!(e instanceof ApiError) || e.status !== 409) load(true)
    throw e
  } finally {
    pending.value--
  }
}

// Antrean dicoba ulang dengan jeda sendiri (1, 2, 4, sampai 30 detik). Event online dan WebSocket yang bergabung
// ulang tetap mengirimnya lebih cepat, tetapi tidak selalu terjadi: galat jaringan sesaat yang tidak memutus
// WebSocket, atau event online yang datang sebelum jaringan benar-benar pulih, membuat antrean menggantung.
let flushTimer: ReturnType<typeof setTimeout> | undefined
let flushAttempt = 0
function scheduleFlush() {
  if (flushTimer) return
  flushTimer = setTimeout(() => {
    flushTimer = undefined
    flushQueue()
  }, reconnectDelay(flushAttempt++))
}
onBeforeUnmount(() => clearTimeout(flushTimer))

let flushing = false
async function flushQueue() {
  if (flushing) return
  flushing = true
  try {
    while (offlineQueue.length > 0) {
      const work = offlineQueue[0]!
      try {
        await work()
      } catch (e) {
        if (isNetworkError(e)) {
          scheduleFlush()
          return
        }
        // Perubahan yang ditolak setelah tersambung (mis. isi sudah diubah orang lain) tidak diulang; papan
        // dimuat ulang supaya sama dengan server.
        saveError.value = e instanceof ApiError ? e.message : 'Sebagian perubahan gagal disimpan.'
        load(true)
      }
      offlineQueue.shift()
    }
    netDown.value = false
    flushAttempt = 0
  } finally {
    flushing = false
  }
}

function onBeforeUnload(e: BeforeUnloadEvent) {
  if (pending.value > 0 || dirtyNotes.size > 0 || offlineQueue.length > 0) e.preventDefault()
}
onMounted(() => window.addEventListener('beforeunload', onBeforeUnload))

// ---------------------------------------------------------------------------------------------------------------
// Kolaborasi langsung (W2)

type Who = { id: number; name: string; color: number; tab?: string }
type Person = {
  id: number
  name: string
  avatar_id: string | null
  color: number
  idle: boolean
  tabs: number
  editing?: string
}

// Satu id per tab: kunci edit dimiliki pasangan pengguna dan tab, jadi dua tab akun yang sama bisa dibedakan.
const tabId = newId()
const people = shallowRef<Person[]>([])
const others = computed(() => people.value.filter((p) => p.id !== myId.value))
const cursors = ref<
  Record<string, { x: number; y: number; name: string; color: number; at: number; user: number }>
>({})
const remoteSel = ref<Record<number, { ids: string[]; who: Who }>>({})
const remoteLocks = ref<Record<string, Who>>({})
const remoteGrabs = ref<Record<string, Who>>({})
// Label sementara "Dipindah Nala" saat perubahan tata letak sendiri ditimpa orang lain dalam 2 detik.
const overwritten = ref<Record<string, { name: string; color: number }>>({})
const recentGeom = new Map<string, number>()

let socket: WbSocket | null = null

// Origin yang sama dengan halaman; wss di produksi (https), ws di dev.
function socketURL(): string {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${location.host}/api/whiteboard/ws`
}

function send(m: Record<string, unknown>) {
  socket?.send(m)
}

function sendJoin() {
  if (board.value) send({ t: 'join', board: board.value.id, last_seq: lastSeq, tab_id: tabId })
}

function nameOf(userId: number) {
  return people.value.find((p) => p.id === userId)?.name ?? 'orang lain'
}

function remoteFor(id: string): { color: number; label: string; strong: boolean } | null {
  const lock = remoteLocks.value[id]
  if (lock) {
    const same = lock.id === myId.value
    return {
      color: lock.color,
      label: same ? 'Kamu mengedit di tab lain' : `${lock.name} sedang mengedit`,
      strong: true,
    }
  }
  const grab = remoteGrabs.value[id]
  if (grab) return { color: grab.color, label: `${grab.name} sedang memindahkan`, strong: true }
  const ow = overwritten.value[id]
  if (ow) return { color: ow.color, label: `Dipindah ${ow.name}`, strong: false }
  const by = Object.values(remoteSel.value).filter((s) => s.who.id !== myId.value && s.ids.includes(id))
  if (by.length)
    return { color: by.at(-1)!.who.color, label: by.map((s) => s.who.name).join(', '), strong: false }
  return null
}

// Dialog: kartu yang sedang diedit dihapus orang lain, papan yang sedang dibuka dihapus, dan kunci di tab lain.
const deletedWhileEditing = ref<{ by: string; batch: string } | null>(null)
const boardGone = ref<{ by: string; batch: string } | null>(null)
const tabPrompt = ref<string | null>(null)
let deletingIds = new Set<string>()

function applyChanged(list: Card[], by?: number) {
  const now = Date.now()
  for (const c of list) {
    const local = byId.value.get(c.id)
    if (
      by &&
      by !== myId.value &&
      local &&
      (recentGeom.get(c.id) ?? 0) > now - 2000 &&
      (local.x !== c.x || local.y !== c.y)
    ) {
      overwritten.value = { ...overwritten.value, [c.id]: { name: nameOf(by), color: colorOf(by) } }
      setTimeout(() => (overwritten.value = omit(overwritten.value, c.id)), 2000)
    }
  }
  if (remoteMoves.value.size) {
    const m = new Map(remoteMoves.value)
    for (const c of list) m.delete(c.id)
    remoteMoves.value = m
  }
  upsert(list)
}

function omit<T>(obj: Record<string, T>, key: string): Record<string, T> {
  const out = { ...obj }
  delete out[key]
  return out
}

function colorOf(userId: number) {
  return people.value.find((p) => p.id === userId)?.color ?? (((userId % 8) + 8) % 8) + 1
}

function onSocketMessage(m: Record<string, unknown>) {
  switch (m.t) {
    case 'joined': {
      if (m.resync) load(true)
      else {
        if (Array.isArray(m.cards)) applyChanged(m.cards as Card[])
        if (Array.isArray(m.deleted)) removeLocal(m.deleted as string[])
        if (Array.isArray(m.connectors)) upsertConnectors(m.connectors as Connector[])
        if (Array.isArray(m.connectors_gone)) removeConnectors(m.connectors_gone as string[])
      }
      const locks: Record<string, Who> = {}
      for (const l of (m.locks as { card: string; user: Who }[]) ?? []) locks[l.card] = l.user
      remoteLocks.value = locks
      remoteGrabs.value = (m.grabs as Record<string, Who>) ?? {}
      // Setelah sambung ulang: kunci kartu yang sedang diedit diminta lagi, lalu antrean perubahan dikirim.
      if (editingId.value) send({ t: 'lock', card: editingId.value })
      flushQueue()
      return
    }
    case 'presence':
      people.value = (m.people as Person[]) ?? []
      return
    case 'person': {
      // Profil berubah (revisi 51): kartu orang pengguna itu ikut berubah tanpa memuat ulang papan.
      const p = m.person as PersonInfo
      if (cards.value.some((c) => c.kind === 'person' && (c.content as PersonContent).user_id === p.id))
        cards.value = cards.value.map((c) =>
          c.kind === 'person' && (c.content as PersonContent).user_id === p.id ? { ...c, person: p } : c,
        )
      return
    }
    case 'cursor': {
      const u = m.user as Who
      cursors.value = {
        ...cursors.value,
        [`${u.id}:${u.tab ?? ''}`]: {
          x: Number(m.x),
          y: Number(m.y),
          name: u.name,
          color: u.color,
          at: Date.now(),
          user: u.id,
        },
      }
      return
    }
    case 'select': {
      const u = m.user as Who
      remoteSel.value = { ...remoteSel.value, [u.id]: { ids: (m.ids as string[]) ?? [], who: u } }
      return
    }
    case 'grab_result':
      onGrabResult(m.granted as string[], m.denied as { id: string; user: Who }[])
      return
    case 'grabbed': {
      const next = { ...remoteGrabs.value }
      for (const id of m.ids as string[]) next[id] = m.user as Who
      remoteGrabs.value = next
      return
    }
    case 'moving': {
      const next = new Map(remoteMoves.value)
      for (const mv of m.moves as { id: string; x: number; y: number; w: number; h: number }[])
        next.set(mv.id, { ...mv, at: Date.now() })
      remoteMoves.value = next
      return
    }
    case 'released': {
      const g = { ...remoteGrabs.value }
      const mv = new Map(remoteMoves.value)
      for (const id of m.ids as string[]) {
        delete g[id]
        mv.delete(id)
      }
      remoteGrabs.value = g
      remoteMoves.value = mv
      return
    }
    case 'lock_denied': {
      const card = String(m.card)
      if (editingId.value === card) editingId.value = null
      if (m.missing) showToast('Kartu ini sudah tidak ada di papan.')
      else if (m.same_user) tabPrompt.value = card
      else showToast(`${(m.user as Who).name} sedang mengedit kartu ini.`)
      return
    }
    case 'lock_granted': {
      const card = String(m.card)
      if (tabPrompt.value === card || pendingForce === card) {
        pendingForce = null
        tabPrompt.value = null
        beginEdit(card)
      }
      return
    }
    case 'locked':
      remoteLocks.value = { ...remoteLocks.value, [String(m.card)]: m.user as Who }
      return
    case 'unlocked':
      remoteLocks.value = omit(remoteLocks.value, String(m.card))
      if (readonlyFor.value?.id === m.card) readonlyFor.value = null
      return
    case 'lock_release':
      // Tab lain milik akun yang sama minta kunci kartu ini: simpan dulu, lalu lepas.
      releaseForOtherTab(String(m.card))
      return
    case 'lock_lost':
      if (editingId.value === m.card) {
        flushNotes()
        editingId.value = null
        showToast('Kartu ini sekarang diedit di tab lain.')
      }
      return
    case 'draft': {
      const id = String(m.card)
      if (editingId.value !== id && byId.value.has(id))
        patchLocal(id, { content: m.content as Card['content'] })
      return
    }
    case 'changed':
      applyChanged(m.cards as Card[], Number(m.by))
      if (commentsFor.value && (m.cards as Card[]).some((c) => c.id === commentsFor.value))
        commentsReload.value++
      return
    case 'connectors':
      upsertConnectors(m.items as Connector[])
      return
    case 'connectors_gone':
      removeConnectors(m.ids as string[])
      return
    case 'moved': {
      const ids = m.ids as string[]
      if (editingId.value && ids.includes(editingId.value)) {
        dirtyNotes.delete(editingId.value)
        editingId.value = null
        showToast(`Kartu yang sedang kamu edit dipindah ke papan lain oleh ${String(m.by)}.`)
      }
      removeLocal(ids)
      removeConnectors((m.connectors_gone as string[]) ?? [])
      return
    }
    case 'deleted': {
      const ids = m.ids as string[]
      const mine = ids.every((id) => deletingIds.has(id))
      if (editingId.value && ids.includes(editingId.value) && !mine) {
        dirtyNotes.delete(editingId.value)
        editingId.value = null
        deletedWhileEditing.value = { by: String(m.by), batch: String(m.batch) }
      }
      removeLocal(ids)
      return
    }
    case 'board_deleted':
      boardGone.value = { by: String(m.by), batch: String(m.batch) }
      return
    case 'renamed':
      if (board.value?.id === m.board) applyTitle(String(m.title), Number(m.version))
      return
    case 'ink': {
      const u = m.user as Who
      remoteInk.value = {
        ...remoteInk.value,
        [u.id]: {
          points: m.points as number[],
          color: String(m.color),
          width: Number(m.width),
          at: Date.now(),
        },
      }
      return
    }
    case 'ink_end': {
      const u = m.user as Who
      // Dibiarkan sebentar sampai goresan tersimpannya datang lewat changed, supaya tidak berkedip.
      setTimeout(() => {
        const next = { ...remoteInk.value }
        delete next[u.id]
        remoteInk.value = next
      }, 600)
      return
    }
    case 'resync':
      load(true)
      return
    case 'denied':
      showToast(String(m.message))
      return
    case 'board_missing':
      loadError.value = 'Papan ini sudah tidak ada. Mungkin sudah dipindah ke Tempat sampah.'
      state.value = 'error'
  }
}

onMounted(() => {
  socket = new WbSocket(socketURL(), {
    message: onSocketMessage,
    state: (s) => {
      sockState.value = s
      if (s !== 'open') {
        // Orang lain tidak terlihat selama terputus; daftar kembali saat join.
        cursors.value = {}
        people.value = people.value.filter((p) => p.id === myId.value)
      }
    },
    open: sendJoin,
    sessionEnded: () => notifyUnauthorized(),
    tooMany: () =>
      showToast('Terlalu banyak tab Whiteboard terbuka. Tutup sebagian, lalu muat ulang halaman ini.'),
  })
  socket.start()
  window.addEventListener('online', onOnline)
})

function onOnline() {
  socket?.retryNow()
  flushQueue()
}

// Kursor dikirim paling banyak 20 kali per detik, dalam koordinat dunia.
let lastCursor = 0
function onCanvasPointerMove(e: PointerEvent) {
  markActive()
  if (!canvas.value || e.pointerType === 'touch') return
  const now = performance.now()
  if (now - lastCursor < 50) return
  lastCursor = now
  const r = canvas.value.getBoundingClientRect()
  const w = toWorld(view.value, e.clientX - r.left, e.clientY - r.top)
  send({ t: 'cursor', x: Math.round(w.x), y: Math.round(w.y) })
}

const cursorList = computed(() => {
  const v = view.value
  const now = Date.now()
  return Object.entries(cursors.value)
    .filter(
      ([, c]) =>
        c.user !== myId.value && now - c.at < 10000 && !people.value.find((p) => p.id === c.user)?.idle,
    )
    .map(([key, c]) => {
      const sx = c.x * v.zoom + v.x
      const sy = c.y * v.zoom + v.y
      // Perkiraan lebar label (teks xs sekitar 7 px per huruf plus padding); cukup untuk memilih sisi.
      const labelW = c.name.length * 7.5 + 32
      return {
        key,
        name: c.name,
        color: c.color,
        flipX: sx + 20 + labelW > size.value.w,
        flipY: sy + 30 > size.value.h,
        style: { transform: `translate(${sx}px, ${sy}px)` },
      }
    })
})
// Kursor dan gerakan seret yang berhenti dibersihkan berkala; kartu orang yang putus di tengah seret kembali ke
// posisi tersimpan setelah 3 detik.
let sweepTimer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  sweepTimer = setInterval(() => {
    const now = Date.now()
    if ([...remoteMoves.value.values()].some((m) => now - m.at > 3000)) {
      remoteMoves.value = new Map([...remoteMoves.value].filter(([, m]) => now - m.at <= 3000))
    }
    cursors.value = Object.fromEntries(Object.entries(cursors.value).filter(([, c]) => now - c.at < 10000))
    if (Object.values(remoteInk.value).some((k) => now - k.at > 3000))
      remoteInk.value = Object.fromEntries(
        Object.entries(remoteInk.value).filter(([, k]) => now - k.at <= 3000),
      )
    checkIdle()
  }, 1000)
})

function follow(p: Person) {
  const c = Object.values(cursors.value)
    .filter((x) => x.user === p.id)
    .sort((a, b) => b.at - a.at)[0]
  if (!c) {
    showToast(`${p.name} belum menggerakkan kursor di papan ini.`)
    return
  }
  centerOn(c.x, c.y)
  announce.value = `Mengikuti ${p.name}.`
}

// Diam: tanpa gerak penunjuk, ketikan, atau tab tersembunyi selama 3 menit.
let lastInput = Date.now()
let idleSent = false
function markActive() {
  lastInput = Date.now()
  if (idleSent && !document.hidden) {
    idleSent = false
    send({ t: 'active' })
  }
}
function checkIdle() {
  const idle = document.hidden || Date.now() - lastInput > 180000
  if (idle && !idleSent) {
    idleSent = true
    send({ t: 'idle' })
  } else if (!idle && idleSent) {
    idleSent = false
    send({ t: 'active' })
  }
}

// Pilihan sendiri disiarkan supaya orang lain melihat bingkai warna kita.
let selectTimer: ReturnType<typeof setTimeout> | undefined
watch(selected, (s) => {
  if (!canWrite.value) return
  clearTimeout(selectTimer)
  selectTimer = setTimeout(() => send({ t: 'select', ids: [...s].slice(0, 500) }), 80)
})

// ---------------------------------------------------------------------------------------------------------------
// Buat, ubah isi, hapus

const toast = ref<{ text: string; batch?: string; board?: string } | null>(null)
let toastTimer: ReturnType<typeof setTimeout> | undefined
function showToast(text: string, batch?: string, board?: string) {
  toast.value = { text, batch, board }
  announce.value = text
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => (toast.value = null), 8000)
}

const history: { undo: Step[]; redo: Step[] } = { undo: [], redo: [] }
const historyVersion = ref(0)
function record(step: Step) {
  push(history, step)
  historyVersion.value++
}
const canUndo = computed(() => historyVersion.value >= 0 && history.undo.length > 0)
const canRedo = computed(() => historyVersion.value >= 0 && history.redo.length > 0)

let placed = 0
// Kartu baru di titik dunia at (dari seret alat), atau di tengah layar dengan sedikit geser supaya kartu yang
// dibuat beruntun tidak menumpuk persis.
async function addCard(
  kind: CardKind,
  at?: { x: number; y: number },
  extra?: Card['content'],
  dim = defaultSize[kind],
) {
  if (!canWrite.value) return
  const c = at ?? centerWorld()
  const offset = at ? 0 : (placed++ % 5) * 24
  const input: NewCard = {
    id: newId(),
    kind,
    x: clamp(snap(c.x - dim.w / 2 + offset, snapOn.value), -COORD_LIMIT, COORD_LIMIT),
    y: clamp(snap(c.y - dim.h / 2 + offset, snapOn.value), -COORD_LIMIT, COORD_LIMIT),
    w: dim.w,
    h: dim.h,
    color: '',
  }
  if (kind === 'board') input.title = 'Papan tanpa judul'
  else input.content = extra ?? emptyContent(kind)
  try {
    const res = await saving(() => createCard(bid.value, input))
    if (!res) return
    upsert([res.card])
    record({ kind: 'create', ids: [res.card.id] })
    select([res.card.id])
    // Bentuk baru hanya dipilih: teksnya diketik dengan dobel klik, seperti di aplikasi diagram.
    if (kind !== 'link' && kind !== 'shape') startEdit(res.card.id)
    else focusCard(res.card.id)
    announce.value = `${kindLabel[kind]} baru ditambahkan.`
    return res.card
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
    const res = await saving(() => {
      const cur = byId.value.get(id) ?? card
      return patchCards(bid.value, [{ id, content, base_content_version: cur.content_version }])
    })
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

// Draf teks dikirim paling sering tiap 150 ms, supaya orang lain melihat teksnya berubah langsung.
const draftTimers = new Map<string, ReturnType<typeof setTimeout>>()
const draftLatest = new Map<string, Card['content']>()
function sendDraft(id: string, content: Card['content']) {
  draftLatest.set(id, content)
  if (draftTimers.has(id)) return
  draftTimers.set(
    id,
    setTimeout(() => {
      draftTimers.delete(id)
      send({ t: 'draft', card: id, content: draftLatest.get(id) })
    }, 150),
  )
}

function onNoteInput(id: string, doc: PMNode) {
  dirtyNotes.set(id, doc)
  sendDraft(id, { doc })
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

// Teks judul berformat (revisi 49): doc null berarti teks polos tanpa format.
function onHeading(id: string, v: { text: string; doc: PMNode | null }) {
  const card = byId.value.get(id)
  if (!card) return
  queueContent(id, { ...v, level: (card.content as HeadingContent).level ?? 1 })
}

function onHeadingDraft(id: string, v: { text: string; doc: PMNode | null }) {
  const card = byId.value.get(id)
  if (card) sendDraft(id, { ...v, level: (card.content as HeadingContent).level ?? 1 })
}

// Keterangan, judul kolom, dan goresan gambar bebas: isi utuh, dijaga content_version seperti catatan. Goresan juga
// dikirim sebagai draf supaya orang lain melihat gambarnya bertambah langsung.
function onContent(id: string, content: Card['content']) {
  patchLocal(id, { content })
  queueContent(id, content)
}

async function onTable(id: string, op: TableOp) {
  try {
    const res = await saving(() => tableOp(bid.value, id, op))
    if (res) upsert([res.card])
  } catch (e) {
    if (e instanceof ApiError && e.status === 409) showToast(e.message)
  }
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

// Kebalikan operasi checklist, untuk undo: dihitung dari isi sebelum operasi.
function checklistInverse(before: ChecklistContent, op: ChecklistOp): ChecklistOp | null {
  switch (op.op) {
    case 'title':
      return { op: 'title', text: before.title, doc: before.title_doc ?? null }
    case 'add':
      return { op: 'remove', item_id: op.item_id }
    case 'remove': {
      const i = before.items.findIndex((it) => it.id === op.item_id)
      const it = before.items[i]
      return it ? { op: 'add', item_id: it.id, text: it.text, index: i, doc: it.doc ?? null } : null
    }
    case 'set': {
      const it = before.items.find((x) => x.id === op.item_id)
      if (!it) return null
      return {
        op: 'set',
        item_id: it.id,
        text: op.text === undefined ? undefined : it.text,
        doc: op.text === undefined ? undefined : (it.doc ?? null),
        done: op.done === undefined ? undefined : it.done,
      }
    }
    case 'move': {
      const i = before.items.findIndex((it) => it.id === op.item_id)
      return i < 0 ? null : { op: 'move', item_id: op.item_id, index: i }
    }
  }
}

async function onChecklist(id: string, op: ChecklistOp, recordStep = true) {
  const before = byId.value.get(id)?.content as ChecklistContent | undefined
  try {
    const res = await saving(() => checklistOp(bid.value, id, op))
    if (!res) return
    upsert([res.card])
    const inv = before && recordStep ? checklistInverse(before, op) : null
    if (inv) record({ kind: 'checklist', id, ops: { undo: inv, redo: op } })
  } catch {
    // Papan sudah dimuat ulang oleh saving().
  }
}

// Kunci lunak: membuka editor mengambil kunci kartu itu. Kartu yang dikunci orang lain tidak dibuka; kartu yang
// dikunci tab lain milik akun sendiri menawarkan "Edit di sini".
const editBefore = new Map<string, { content: Card['content']; version: number }>()
let pendingForce: string | null = null

// Petunjuk posisi kursor dari dobel klik, dipakai editor yang dibuka (kosong untuk Enter dan tombol Ubah isi).
const editHint = ref<(CaretHint & { id: string }) | null>(null)
// Kartu yang dikunci orang lain dan dibuka hanya baca (dobel klik): teksnya tetap berubah langsung lewat draf.
const readonlyFor = ref<{ id: string; name: string } | null>(null)
watch(selected, (s) => {
  if (readonlyFor.value && !s.has(readonlyFor.value.id)) readonlyFor.value = null
})

function startEdit(id: string, hint?: CaretHint) {
  const kind = byId.value.get(id)?.kind
  if (!canWrite.value || kind === 'link' || kind === 'drawing') return
  const lock = remoteLocks.value[id]
  if (lock) {
    if (lock.id === myId.value) tabPrompt.value = id
    else {
      select([id])
      readonlyFor.value = { id, name: lock.name }
      announce.value = `Hanya baca: ${lock.name} sedang mengedit kartu ini.`
    }
    return
  }
  editHint.value = hint ? { ...hint, id } : null
  send({ t: 'lock', card: id })
  beginEdit(id)
}

function beginEdit(id: string) {
  select([id])
  focusedId.value = id
  editingId.value = id
  const card = byId.value.get(id)
  if (card) editBefore.set(id, { content: card.content, version: card.content_version })
}

function editHere() {
  const id = tabPrompt.value
  if (!id) return
  pendingForce = id
  send({ t: 'lock', card: id, force: true })
}

let lockBeat: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  // Detak kunci tiap 10 detik; tanpa detak 30 detik server melepasnya (koneksi setengah mati).
  lockBeat = setInterval(() => {
    if (editingId.value) send({ t: 'lock', card: editingId.value })
    if (penObject.value) send({ t: 'lock', card: penObject.value })
  }, 10000)
})

async function finishEdit(id: string) {
  flushNotes()
  await (chains.get(id) ?? Promise.resolve()).catch(() => undefined)
  send({ t: 'unlock', card: id })
  const before = editBefore.get(id)
  editBefore.delete(id)
  const after = byId.value.get(id)
  if (
    before &&
    after &&
    after.content_version !== before.version &&
    after.kind !== 'checklist' &&
    after.kind !== 'board'
  ) {
    record({
      kind: 'content',
      id,
      before: before.content,
      after: after.content,
      version: after.content_version,
    })
  }
}

async function stopEdit() {
  const id = editingId.value
  editingId.value = null
  if (!id) return
  finishEdit(id)
  await focusCard(id)
}

async function releaseForOtherTab(id: string) {
  if (editingId.value !== id) {
    send({ t: 'unlock', card: id })
    return
  }
  editingId.value = null
  await finishEdit(id)
  showToast('Kartu ini sekarang diedit di tab lain.')
}

async function deleteIds(ids: string[]) {
  deletingIds = new Set(ids)
  try {
    return await saving(() => deleteCards(bid.value, ids))
  } finally {
    setTimeout(() => (deletingIds = new Set()), 2000)
  }
}

async function removeSelected() {
  if (!canWrite.value || selected.value.size === 0) return
  if (anyLocked.value) {
    lockedToast()
    return
  }
  const ids = [...selected.value]
  if (editingId.value && ids.includes(editingId.value)) editingId.value = null
  const boards = ids.filter((id) => byId.value.get(id)?.kind === 'board').length
  try {
    const res = await deleteIds(ids)
    if (!res) return
    removeLocal(ids)
    record({ kind: 'delete', ids, batch: res.batch })
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

function restoreFrom(info: { batch: string } | null) {
  if (info) restoreDeleted(info.batch)
}

// Tombol Aksi di bar pilihan membuka menu yang sama dengan klik kanan (revisi 52), di bawah tombolnya.
function openBarMenu(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  if (menu.value?.opener === el) {
    closeMenu(true)
    return
  }
  const r = el.getBoundingClientRect()
  openMenu(r.left, r.bottom + 4, el, true)
}

// Palet warna bar pilihan: kotak warna kartu, Tanpa latar (gaya revisi 50), dan Gaya lainnya (panel gaya).
const cardSwatch: Record<string, string> = {
  '': 'var(--color-surface)',
  abu: 'var(--color-requested-bg)',
  hijau: 'var(--color-progress-bg)',
  kuning: 'var(--color-ready-bg)',
  biru: 'var(--color-completed-bg)',
  krem: 'var(--color-tag)',
  merah: 'var(--color-danger-bg)',
}
const cardColorOptions = computed(() =>
  colors.map((c) => ({
    value: c.value ?? '',
    label: `Warna ${c.label.toLowerCase()}`,
    swatch: cardSwatch[c.value ?? ''],
  })),
)
const penOptions = computed(() =>
  penColors.map((c) => ({ value: c.value, label: `Coretan ${c.label}`, swatch: c.css })),
)
const allClear = computed(
  () =>
    selCards.value.length > 0 && selCards.value.every((c) => !colorable(c) || c.style?.bg === 'transparan'),
)
const colorExtras = computed(() => [
  {
    id: 'tanpa-latar',
    label: 'Tanpa latar',
    pressed: allClear.value,
    run: () => applyStyle({ bg: allClear.value ? '' : 'transparan' }),
  },
  { id: 'gaya', label: 'Gaya lainnya...', run: () => (stylePanelOpen.value = true) },
])
// Memilih warna juga mengembalikan latar kartu yang tadinya tanpa latar, supaya warnanya terlihat.
function pickCardColor(v: string) {
  setColor(v)
  if (selCards.value.some((c) => c.style?.bg === 'transparan')) applyStyle({ bg: '' })
}

async function restoreDeleted(batch: string) {
  deletedWhileEditing.value = null
  boardGone.value = null
  try {
    await saving(() => restoreBatch(batch))
    await load(true)
    announce.value = 'Dipulihkan dari Tempat sampah.'
  } catch {
    // Pesan galat sudah tampil di penanda simpan.
  }
}

const selectionColor = computed<string>(() => {
  const list = [...selected.value].map((id) => byId.value.get(id)?.color ?? '')
  return list.length && list.every((c) => c === list[0]) ? (list[0] ?? '') : 'campuran'
})

// applyLayout mengubah tata letak beberapa kartu sekaligus dan mencatatnya untuk undo.
function applyLayout(changes: { id: string; after: Geo }[]) {
  const entries: { id: string; before: Geo; after: Geo }[] = []
  const patches: CardPatch[] = []
  const now = Date.now()
  for (const ch of changes) {
    const c = byId.value.get(ch.id)
    if (!c) continue
    const keys = Object.keys(ch.after) as (keyof Geo)[]
    entries.push({ id: ch.id, before: geo(c, keys), after: ch.after })
    patchLocal(ch.id, ch.after as Partial<Card>)
    recentGeom.set(ch.id, now)
    const p: CardPatch = { id: ch.id }
    for (const k of keys) {
      const v = ch.after[k]
      if (k === 'color') p.color = (v as CardColor | null) ?? ''
      else (p as Record<string, unknown>)[k] = v
    }
    patches.push(p)
  }
  if (entries.length) record({ kind: 'layout', cards: entries })
  sendGeometry(patches)
}

function setColor(value: string) {
  if (!canWrite.value || value === 'campuran') return
  const color = (value || null) as CardColor | null
  applyLayout([...selected.value].map((id) => ({ id, after: { color } })))
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

function align(kind: string) {
  if (anyLocked.value) {
    lockedToast()
    return
  }
  const groups = units([...selected.value], cards.value, groupScope.value)
  const out = arrange(groups, kind as AlignKind)
  if (!out.size) return
  applyLayout([...out].map(([id, pos]) => ({ id, after: pos })))
  announce.value = `${groups.length} kartu atau grup dirapikan.`
}

// ---------------------------------------------------------------------------------------------------------------
// Revisi 50: grup, kunci posisi, gaya kartu, menu klik kanan, dan alat gambar

// Grup yang sedang dimasuki lewat dobel klik; klik di dalamnya memilih satu tingkat di bawahnya. Escape keluar.
const groupScope = ref<string[]>([])
watch(bid, () => (groupScope.value = []))

function unitIds(id: string): string[] {
  const c = byId.value.get(id)
  return c ? unitOf(c, cards.value, groupScope.value) : [id]
}

// Dobel klik pada kartu bergrup masuk satu tingkat ke grupnya dan memilih kartu itu (atau subgrupnya), bukan membuka
// editor. Dobel klik berikutnya baru mengedit. Mengembalikan true bila masuk grup.
function enterGroup(card: Card): boolean {
  const p = card.group_ids ?? []
  const s = groupScope.value
  const inside = s.every((g, i) => p[i] === g)
  const depth = inside ? s.length : 0
  if (p.length <= depth) return false
  groupScope.value = [...p.slice(0, depth + 1)]
  select(unitOf(card, cards.value, groupScope.value))
  focusedId.value = card.id
  announce.value = 'Masuk ke grup. Klik memilih satu anggota; Escape untuk keluar.'
  return true
}

function exitGroup() {
  const s = groupScope.value
  const member = cards.value.find((c) => s.every((g, i) => c.group_ids?.[i] === g))
  groupScope.value = s.slice(0, -1)
  if (member) select(unitOf(member, cards.value, groupScope.value))
  announce.value = 'Keluar dari grup.'
}

// Bingkai grup yang terpilih utuh, mengikuti seret yang sedang berjalan.
const groupFrames = computed(() => {
  const out: Rect[] = []
  const depth = groupScope.value.length
  const seen = new Set<string>()
  for (const id of selected.value) {
    const c = shownById.value.get(id)
    const g = c?.group_ids?.[depth]
    if (!c || !g || seen.has(g)) continue
    seen.add(g)
    const members = shownCards.value
      .filter((m) => m.group_ids?.[depth] === g)
      .map((m) => (dragged(m) ? { ...m, x: m.x + dragOffset.value.dx, y: m.y + dragOffset.value.dy } : m))
    const b = bounds(members)
    if (b) out.push({ x: b.x - 8, y: b.y - 8, w: b.w + 16, h: b.h + 16 })
  }
  return out
})

const selCards = computed(() => [...selected.value].flatMap((id) => byId.value.get(id) ?? []))
const anyLocked = computed(() => selCards.value.some((c) => c.locked))

function lockedToast(n = selCards.value.filter((c) => c.locked).length) {
  showToast(
    `${n} kartu terkunci. Buka kunci dulu (${keyLabel('lock')}) untuk memindah, mengubah ukuran, memutar, atau menghapusnya.`,
  )
}

function groupSelected() {
  if (!canWrite.value) return
  const ids = [...selected.value]
  const map = groupPaths(ids, cards.value, newId())
  if (!map) {
    showToast(
      ids.length < 2
        ? 'Pilih minimal dua kartu untuk dijadikan grup.'
        : 'Grup bersarang paling dalam 8 tingkat.',
    )
    return
  }
  applyLayout([...map].map(([id, g]) => ({ id, after: { group_ids: g } })))
  announce.value = `${map.size} kartu dijadikan satu grup.`
}

function ungroupSelected() {
  if (!canWrite.value) return
  const map = ungroupPaths([...selected.value], cards.value, groupScope.value)
  if (!map.size) return
  applyLayout([...map].map(([id, g]) => ({ id, after: { group_ids: g } })))
  announce.value = 'Grup dipisahkan.'
}

function toggleLock() {
  if (!canWrite.value || !selCards.value.length) return
  const lock = !selCards.value.every((c) => c.locked)
  applyLayout(selCards.value.map((c) => ({ id: c.id, after: { locked: lock } })))
  announce.value = lock
    ? `Posisi ${selCards.value.length} kartu dikunci.`
    : `Kunci posisi ${selCards.value.length} kartu dibuka.`
}

// Gaya: salin dari kartu pertama yang dipilih, tempel ke semua kartu terpilih (hanya bagian yang berlaku untuk jenis
// kartu tujuan). Warna kartu ikut, kecuali untuk bentuk, gambar, dan gambar bebas yang punya warna sendiri.
const copiedStyle = ref<{ style: CardStyle; color: CardColor | null } | null>(null)
const colorable = (c: Card) => c.kind !== 'shape' && c.kind !== 'image' && c.kind !== 'drawing'
function copyStyle() {
  const c = selCards.value[0]
  if (!c) return
  copiedStyle.value = { style: { ...(c.style ?? {}) }, color: c.color }
  announce.value = 'Gaya kartu disalin.'
}
function pasteStyleSel() {
  const s = copiedStyle.value
  if (!canWrite.value || !s || !selCards.value.length) return
  applyLayout(
    selCards.value.map((c) => ({
      id: c.id,
      after: { style: pasteStyle(c.style, s.style, c.kind), ...(colorable(c) ? { color: s.color } : {}) },
    })),
  )
  announce.value = `Gaya ditempel ke ${selCards.value.length} kartu.`
}
const stylePanelOpen = ref(false)
function applyStyle(patch: Partial<Record<StyleKey, string | number>>) {
  applyLayout(
    selCards.value.map((c) => {
      const keys = styleKeys(c.kind)
      const next: Record<string, unknown> = { ...(c.style ?? {}) }
      for (const [k, v] of Object.entries(patch)) if (keys.includes(k as StyleKey)) next[k] = v
      return { id: c.id, after: { style: normalize(next as CardStyle) } }
    }),
  )
}

// Urutan tumpuk: paling depan dan paling belakang melompati semua kartu; maju dan mundur melewati satu kartu lain.
function reorder(kind: 'front' | 'forward' | 'backward' | 'back') {
  if (!canWrite.value || !selCards.value.length) return
  const sel = selected.value
  const sorted = [...cards.value].sort((a, b) => a.z - b.z)
  const others = sorted.filter((c) => !sel.has(c.id))
  const list = sorted.filter((c) => sel.has(c.id))
  const top = sorted.at(-1)?.z ?? 0
  const bottom = sorted[0]?.z ?? 0
  const out = list.map((c, i) => {
    let z = c.z
    if (kind === 'front') z = top + 1 + i
    else if (kind === 'back') z = bottom - list.length + i
    else if (kind === 'forward') {
      const k = others.findIndex((o) => o.z > c.z)
      if (k >= 0) z = others[k + 1] ? (others[k]!.z + others[k + 1]!.z) / 2 + i * 1e-3 : others[k]!.z + 1 + i
    } else {
      const k = others.map((o) => o.z < c.z).lastIndexOf(true)
      if (k >= 0)
        z = k > 0 ? (others[k]!.z + others[k - 1]!.z) / 2 + i * 1e-3 : others[k]!.z - list.length + i
    }
    return { id: c.id, after: { z } }
  })
  applyLayout(out)
  announce.value =
    kind === 'front'
      ? 'Dibawa ke paling depan.'
      : kind === 'back'
        ? 'Dikirim ke paling belakang.'
        : kind === 'forward'
          ? 'Maju satu lapis.'
          : 'Mundur satu lapis.'
}

function matchSize(kind: 'w' | 'h' | 'wh') {
  const list = selCards.value.filter((c) => c.kind !== 'drawing')
  const ref = list[0]
  if (!canWrite.value || !ref || list.length < 2) return
  if (anyLocked.value) {
    lockedToast()
    return
  }
  applyLayout(
    list.slice(1).map((c) => ({
      id: c.id,
      after: kind === 'w' ? { w: ref.w } : kind === 'h' ? { h: ref.h } : { w: ref.w, h: ref.h },
    })),
  )
  announce.value = `Ukuran ${list.length - 1} kartu disamakan dengan kartu yang pertama dipilih.`
}

function tidySelected() {
  if (!canWrite.value) return
  if (anyLocked.value) {
    lockedToast()
    return
  }
  const groups = units([...selected.value], cards.value, groupScope.value)
  applyLayout([...tidy(groups)].map(([id, pos]) => ({ id, after: pos })))
  announce.value = `${groups.length} kartu atau grup dirapikan menjadi grid.`
}

// Putar dan balik bentuk: perubahan isi dengan langkah undo.
function shapeChange(change: (s: ShapeContent) => ShapeContent) {
  const list = selCards.value.filter((c) => c.kind === 'shape')
  for (const c of list) {
    const after = change(c.content as ShapeContent)
    record({ kind: 'content', id: c.id, before: c.content, after, version: c.content_version + 1 })
    onContent(c.id, after)
  }
}
function rotate90() {
  if (!canWrite.value) return
  if (selCards.value.some((c) => c.kind === 'shape' && c.locked)) {
    lockedToast()
    return
  }
  shapeChange((s) => ({ ...s, rotation: ((s.rotation || 0) + 90) % 360 }))
  announce.value = 'Bentuk diputar 90 derajat.'
}
function flip(axis: 'x' | 'y') {
  if (!canWrite.value) return
  shapeChange((s) => (axis === 'x' ? { ...s, flip_x: !s.flip_x } : { ...s, flip_y: !s.flip_y }))
  announce.value = axis === 'x' ? 'Bentuk dibalik mendatar.' : 'Bentuk dibalik tegak.'
}

// Garis yang kedua ujungnya ikut terseret bersama (grup) ikut bergeser titik beloknya.
function shiftBends(moved: Set<string>, dx: number, dy: number) {
  for (const c of connectors.value) {
    if (!c.from || !c.to || !moved.has(c.from) || !moved.has(c.to) || !c.geometry.points.length) continue
    const points = c.geometry.points.map(
      ([x, y]) => [Math.round(x + dx), Math.round(y + dy)] as [number, number],
    )
    void onConnectorUpdate(c.id, { geometry: { ...c.geometry, points } })
  }
}

// Jadikan papan baru: kartu papan baru di pojok kiri atas pilihan, lalu pilihan dipindah ke dalamnya.
async function boardFromSelection() {
  if (!canWrite.value) return
  if (anyLocked.value) {
    showToast(
      `Pilihan berisi kartu terkunci. Buka kunci dulu (${keyLabel('lock')}), lalu jadikan papan baru.`,
    )
    return
  }
  const ids = [...selected.value]
  const b = bounds(selCards.value)
  if (!b || !ids.length) return
  const dim = defaultSize.board
  const card = await addCard('board', { x: b.x + dim.w / 2, y: b.y + dim.h / 2 })
  if (card?.ref) await moveTo(ids, card.ref.id, card.ref.title)
}

function copyCardLink() {
  const id = [...selected.value][0]
  if (!id || !board.value) return
  const href = router.resolve({
    name: 'whiteboard-board',
    params: { id: board.value.id },
    query: { kartu: id },
  }).href
  navigator.clipboard
    ?.writeText(new URL(href, location.origin).toString())
    .then(() => (announce.value = 'Tautan kartu disalin.'))
    .catch(() => showToast('Tautan tidak bisa disalin di browser ini.'))
  announce.value = 'Tautan kartu disalin.'
}

function zoomToSelection() {
  const list = selCards.value.map((c) => shownById.value.get(c.id) ?? c)
  if (!list.length) return
  const phone = size.value.w < 768
  view.value = fitView(
    list,
    size.value.w,
    size.value.h,
    phone ? FIT_TOP_PHONE : 48,
    phone ? FIT_BOTTOM_PHONE : 48,
  )
}

function selectSameKind() {
  const kinds = new Set(selCards.value.map((c) => c.kind))
  const ids = cards.value.filter((c) => kinds.has(c.kind)).map((c) => c.id)
  select(ids)
  announce.value = `${ids.length} kartu sejenis terpilih.`
}

function cutSelection() {
  if (!canWrite.value || !selected.value.size) return
  if (anyLocked.value) {
    lockedToast()
    return
  }
  copySelection()
  void removeSelected()
}

function pasteHere(at?: { x: number; y: number }) {
  if (!canWrite.value) return
  const read = navigator.clipboard?.readText?.()
  if (!read) {
    void pasteText(clipMemory, false, at)
    return
  }
  read
    .then((t) => pasteText(t.startsWith(CLIP_PREFIX) ? t : clipMemory, false, at))
    .catch(() => pasteText(clipMemory, false, at))
}

function toggleGrid() {
  gridOn.value = !gridOn.value
  announce.value = gridOn.value ? 'Grid tampil.' : 'Grid disembunyikan.'
}
function toggleSnap() {
  snapOn.value = !snapOn.value
  announce.value = snapOn.value ? 'Snap ke grid menyala.' : 'Snap ke grid mati.'
}

// Gambar dan berkas: ganti, potong, posisi dan zoom, lihat ukuran penuh, unduh.
const imageDialog = ref<{ mode: 'view' | 'crop' | 'frame'; id: string } | null>(null)
const imageCard = computed(() => (imageDialog.value ? (byId.value.get(imageDialog.value.id) ?? null) : null))
const replaceInput = ref<HTMLInputElement | null>(null)
let replaceTarget: string | null = null
const only = computed(() => (selCards.value.length === 1 ? selCards.value[0] : undefined))

function pickReplace() {
  if (!only.value || !canWrite.value) return
  replaceTarget = only.value.id
  replaceInput.value?.click()
}
function onReplacePicked(e: Event) {
  const el = e.target as HTMLInputElement
  const f = el.files?.[0]
  el.value = ''
  if (f && replaceTarget) void replaceCardFile(replaceTarget, f)
}

// Ganti berkas: posisi, ukuran, gaya, keterangan, grup, kunci, dan garis tetap. Undo memakai berkas lama lewat patch
// file_id (berkas lama tidak dihapus server; job harian membersihkannya setelah tidak dipakai kartu mana pun).
async function replaceCardFile(id: string, file: File) {
  const c = byId.value.get(id)
  if (!c || !canWrite.value) return
  const err = precheckFile(file)
  if (err) {
    showToast(err)
    return
  }
  const image = file.type.startsWith('image/')
  if (c.kind === 'image' && !image) {
    showToast('Kartu gambar hanya bisa diganti dengan gambar (JPEG, PNG, WebP, atau GIF).')
    return
  }
  if (c.kind === 'file' && image) {
    showToast('Gambar ditaruh sebagai kartu gambar. Kartu berkas diganti dengan berkas yang bukan gambar.')
    return
  }
  uploadStatus.value = `Mengganti dengan ${file.name}`
  try {
    const res = await saving(() =>
      replaceFile(
        bid.value,
        id,
        file,
        (p) => (uploadStatus.value = `Mengganti dengan ${file.name} (${Math.round(p * 100)}%)`),
      ),
    )
    if (!res) return
    upsert([res.card])
    record({
      kind: 'layout',
      cards: [{ id, before: { file_id: res.old_file_id }, after: { file_id: res.card.file?.id } }],
    })
    announce.value = `Berkas kartu diganti dengan ${file.name}.`
  } catch (e) {
    if (e instanceof ApiError) showToast(e.message)
  } finally {
    uploadStatus.value = ''
  }
}

function openImage(mode: 'view' | 'crop' | 'frame') {
  if (only.value?.kind === 'image' && only.value.file) imageDialog.value = { mode, id: only.value.id }
}

// Potong bebas mengubah tinggi kartu mengikuti perbandingan potongan, supaya gambar tidak gepeng; posisi dan zoom
// memakai perbandingan bingkai yang ada.
function saveCrop(crop: Crop | null) {
  const c = imageCard.value
  const mode = imageDialog.value?.mode
  imageDialog.value = null
  if (!c || !canWrite.value) return
  const before = c.content as CaptionContent
  const after: CaptionContent = { caption: before.caption, ...(crop ? { crop } : {}) }
  record({ kind: 'content', id: c.id, before, after, version: c.content_version + 1 })
  onContent(c.id, after)
  if (mode === 'crop' && crop && c.file?.width && c.file.height && !c.locked) {
    const pad = before.caption ? 72 : 24
    const h = Math.round((c.w * crop.h * c.file.height) / (crop.w * c.file.width)) + pad
    applyLayout([{ id: c.id, after: { h: clamp(h, MIN_SIZE, MAX_SIZE) } }])
  }
  announce.value = crop ? 'Potongan gambar disimpan.' : 'Gambar ditampilkan utuh.'
}

function downloadOriginal() {
  const f = only.value?.file
  if (!f) return
  const a = document.createElement('a')
  a.href = downloadUrl(f.id)
  a.download = f.name
  a.click()
}

// Kartu anggota tim (revisi 51).
const teamOpen = ref(false)
let teamAt: { x: number; y: number } | undefined
function openTeam(at?: { x: number; y: number }) {
  if (!canWrite.value) return
  teamAt = at
  teamOpen.value = true
}

// Semua orang yang dipilih ditaruh berjajar rapi (grid otomatis) di tengah layar atau di titik "tambah di sini".
async function addPeople(list: PersonInfo[]) {
  teamOpen.value = false
  const dim = personSize(DEFAULT_PERSON)
  const pos = gridAt(list.length, dim, teamAt ?? centerWorld())
  const made: string[] = []
  for (const [i, p] of list.entries()) {
    const at = pos[i]!
    try {
      const res = await saving(() =>
        createCard(bid.value, {
          id: newId(),
          kind: 'person',
          x: clamp(at.x, -COORD_LIMIT, COORD_LIMIT),
          y: clamp(at.y, -COORD_LIMIT, COORD_LIMIT),
          w: dim.w,
          h: dim.h,
          color: '',
          content: { user_id: p.id, ...DEFAULT_PERSON },
        }),
      )
      if (res) {
        upsert([res.card])
        made.push(res.card.id)
      }
    } catch {
      break
    }
  }
  if (made.length) {
    record({ kind: 'create', ids: made })
    select(made)
    announce.value = `${made.length} kartu orang ditambahkan.`
  }
}

// Tampilan banyak kartu orang sekaligus: isi (dengan undo) dan ukuran kartu yang pas untuk tampilan itu. Kartu
// terkunci hanya berganti tampilan, ukurannya tetap.
function setPersonDisplay(patch: Partial<Omit<PersonContent, 'user_id'>>) {
  if (!canWrite.value) return
  const list = selCards.value.filter((c) => c.kind === 'person')
  const sizes: { id: string; after: Geo }[] = []
  for (const c of list) {
    const after = { ...(c.content as PersonContent), ...patch }
    record({ kind: 'content', id: c.id, before: c.content, after, version: c.content_version + 1 })
    onContent(c.id, after)
    if (!c.locked) sizes.push({ id: c.id, after: personSize(after) })
  }
  if (sizes.length) applyLayout(sizes)
  announce.value = `Tampilan ${list.length} kartu orang diganti.`
}

// Satu tabel aksi untuk menu dan pintasan keyboard. Setiap aksi memeriksa sendiri apakah berlaku.
const actions: Partial<Record<ShortcutId, () => void>> = {
  cut: cutSelection,
  copy: copySelection,
  paste: () => pasteHere(),
  duplicate: () => canWrite.value && duplicateSelection(),
  remove: () => void removeSelected(),
  copyStyle,
  pasteStyle: pasteStyleSel,
  stylePanel: () => selCards.value.length && (stylePanelOpen.value = true),
  group: groupSelected,
  ungroup: ungroupSelected,
  lock: toggleLock,
  front: () => reorder('front'),
  forward: () => reorder('forward'),
  backward: () => reorder('backward'),
  back: () => reorder('back'),
  alignLeft: () => canWrite.value && align('left'),
  alignHCenter: () => canWrite.value && align('hcenter'),
  alignRight: () => canWrite.value && align('right'),
  alignTop: () => canWrite.value && align('top'),
  alignVCenter: () => canWrite.value && align('vcenter'),
  alignBottom: () => canWrite.value && align('bottom'),
  spreadH: () => canWrite.value && align('hspread'),
  spreadV: () => canWrite.value && align('vspread'),
  sameWidth: () => matchSize('w'),
  sameHeight: () => matchSize('h'),
  sameSize: () => matchSize('wh'),
  tidy: tidySelected,
  rotate90,
  flipH: () => flip('x'),
  flipV: () => flip('y'),
  connect: () => canWrite.value && selected.value.size >= 2 && void connectInOrder(),
  newBoard: () => void boardFromSelection(),
  moveBoard: () => {
    if (!canWrite.value || !selected.value.size) return
    if (anyLocked.value) {
      showToast(`Pilihan berisi kartu terkunci. Buka kunci dulu (${keyLabel('lock')}), lalu pindahkan lagi.`)
      return
    }
    moveError.value = ''
    moveOpen.value = true
  },
  comments: () => only.value && (commentsFor.value = only.value.id),
  copyLink: copyCardLink,
  exportSel: () => selected.value.size && (exportOpen.value = true),
  zoomSel: zoomToSelection,
  sameKind: selectSameKind,
  replaceFile: pickReplace,
  crop: () => canWrite.value && openImage('crop'),
  frame: () => canWrite.value && openImage('frame'),
  viewFull: () => openImage('view'),
  download: downloadOriginal,
  addNote: () => void addCard('note', menuAt ?? undefined),
  addHeading: () => void addCard('heading', menuAt ?? undefined),
  addChecklist: () => void addCard('checklist', menuAt ?? undefined),
  addTable: () => void addCard('table', menuAt ?? undefined),
  addShape: () => void addCard('shape', menuAt ?? undefined, newShapeContent('rect'), { w: 160, h: 100 }),
  snap: toggleSnap,
}
// Pintasan yang ditangani tabel ini; sisanya (Ctrl+C, Ctrl+D, Ctrl+Z, Ctrl+A, N, G, Delete) tetap di penangan lama.
const KEY_ACTIONS: ShortcutId[] = (Object.keys(actions) as ShortcutId[]).filter(
  (k) => !['copy', 'paste', 'duplicate', 'remove', 'addNote'].includes(k),
)

// Menu klik kanan: hanya aksi yang berlaku untuk pilihan saat itu. Kartu terkunci tidak menawarkan potong, hapus,
// rata, ukuran, dan putar; Admin (hanya baca) hanya aksi baca.
const menu = ref<{
  x: number
  y: number
  items: MenuItem[]
  label: string
  opener: HTMLElement | null
} | null>(null)
let menuAt: { x: number; y: number } | null = null
// Waktu menu dibuka oleh tekan lama, supaya event contextmenu browser yang menyusul tekan lama tidak membukanya lagi.
let pressOpenedAt = -Infinity

function mi(id: ShortcutId, text: string): MenuItem {
  return { id, label: text, key: keyLabel(id), run: actions[id] }
}

function selectionMenu(): MenuItem[] {
  const list = selCards.value
  const w = canWrite.value
  const locked = anyLocked.value
  const n = list.length
  const one = only.value
  const out: MenuItem[] = []
  if (w && !locked) out.push(mi('cut', 'Potong'))
  out.push(mi('copy', 'Salin'))
  if (w) out.push(mi('paste', 'Tempel'), mi('duplicate', 'Duplikat'))
  if (w && !locked) out.push({ ...mi('remove', 'Hapus'), danger: true })
  if (w) {
    out.push(mi('copyStyle', 'Salin gaya'))
    if (copiedStyle.value) out.push(mi('pasteStyle', 'Tempel gaya'))
    out.push(mi('stylePanel', 'Panel gaya'))
    if (units([...selected.value], cards.value, groupScope.value).length >= 2)
      out.push(mi('group', 'Jadikan grup'))
    if (hasGroup([...selected.value], cards.value, groupScope.value)) out.push(mi('ungroup', 'Pisahkan grup'))
    out.push(mi('lock', list.every((c) => c.locked) ? 'Buka kunci posisi' : 'Kunci posisi'))
    out.push({
      id: 'urutan',
      label: 'Urutan',
      children: [
        mi('front', 'Paling depan'),
        mi('forward', 'Maju'),
        mi('backward', 'Mundur'),
        mi('back', 'Paling belakang'),
      ],
    })
  }
  const unitCount = units([...selected.value], cards.value, groupScope.value).length
  if (w && !locked && unitCount >= 2) {
    out.push({
      id: 'rata',
      label: 'Rata',
      children: [
        mi('alignLeft', 'Rata kiri'),
        mi('alignHCenter', 'Rata tengah mendatar'),
        mi('alignRight', 'Rata kanan'),
        mi('alignTop', 'Rata atas'),
        mi('alignVCenter', 'Rata tengah tegak'),
        mi('alignBottom', 'Rata bawah'),
        ...(unitCount >= 3 ? [mi('spreadH', 'Sebar mendatar'), mi('spreadV', 'Sebar tegak')] : []),
      ],
    })
  }
  if (w && !locked && n >= 2) {
    out.push({
      id: 'ukuran',
      label: 'Ukuran',
      children: [
        mi('sameWidth', 'Samakan lebar'),
        mi('sameHeight', 'Samakan tinggi'),
        mi('sameSize', 'Samakan keduanya'),
      ],
    })
    out.push(mi('tidy', 'Rapikan jadi grid'))
  }
  if (w && list.some((c) => c.kind === 'shape')) {
    if (!list.some((c) => c.kind === 'shape' && c.locked)) out.push(mi('rotate90', 'Putar 90 derajat'))
    out.push(mi('flipH', 'Balik mendatar'), mi('flipV', 'Balik tegak'))
  }
  const persons = list.filter((c) => c.kind === 'person')
  if (w && persons.length) {
    const cur = (k: keyof PersonContent, v: string) =>
      persons.every((c) => (c.content as PersonContent)[k] === v)
    out.push({
      id: 'tampilanOrang',
      label: 'Tampilan kartu orang',
      children: [
        ...PERSON_VIEWS.map((o) => ({
          id: `v-${o.value}`,
          label: o.label,
          checked: cur('view', o.value),
          run: () => setPersonDisplay({ view: o.value }),
        })),
        ...PERSON_SIZES.map((o) => ({
          id: `s-${o.value}`,
          label: `Ukuran ${o.label.toLowerCase()}`,
          checked: cur('size', o.value),
          run: () => setPersonDisplay({ size: o.value }),
        })),
        ...PERSON_LAYOUTS.map((o) => ({
          id: `l-${o.value}`,
          label: o.label,
          checked: cur('layout', o.value),
          run: () => setPersonDisplay({ layout: o.value }),
        })),
      ],
    })
  }
  if (w && n >= 2) out.push(mi('connect', 'Hubungkan dengan garis (urutan pilih)'))
  if (w && !locked) out.push(mi('newBoard', 'Jadikan papan baru'), mi('moveBoard', 'Pindahkan ke papan lain'))
  if (one) out.push(mi('comments', 'Komentar'), mi('copyLink', 'Salin tautan kartu'))
  // Dulu di pilihan "Aksi lain" bar pilihan (W3 dan W5); sejak revisi 52 bar pilihan memakai menu ini.
  if (one && connectorsOf(one.id).length)
    out.push({
      id: 'lines',
      label: `Garis (${connectorsOf(one.id).length})`,
      run: () => (connectorIds.value = connectorsOf(one.id).map((c) => c.id)),
    })
  if (one?.kind === 'shape' && w)
    out.push({ id: 'shapestyle', label: 'Gaya bentuk', run: () => (shapeStyleId.value = one.id) })
  if (one?.kind === 'image' && one.file) {
    if (w)
      out.push(
        mi('replaceFile', 'Ganti gambar'),
        mi('crop', 'Potong gambar'),
        mi('frame', 'Atur posisi dan zoom'),
      )
    out.push(mi('viewFull', 'Lihat ukuran penuh'), mi('download', 'Unduh berkas asli'))
  } else if (one?.kind === 'file' && one.file) {
    if (w) out.push(mi('replaceFile', 'Ganti berkas'))
    out.push(mi('download', 'Unduh berkas'))
  }
  out.push(mi('exportSel', 'Ekspor pilihan (PNG atau PDF)'), mi('zoomSel', 'Zoom ke pilihan'))
  out.push(mi('sameKind', 'Pilih semua yang sejenis'))
  return out
}

function emptyMenu(): MenuItem[] {
  const out: MenuItem[] = []
  if (canWrite.value) {
    out.push({
      id: 'pasteHere',
      label: 'Tempel di sini',
      key: keyLabel('paste'),
      run: () => pasteHere(menuAt ?? undefined),
    })
    out.push({
      id: 'tambah',
      label: 'Tambah kartu di sini',
      children: [
        mi('addNote', 'Catatan'),
        mi('addHeading', 'Judul'),
        mi('addChecklist', 'Checklist'),
        mi('addTable', 'Tabel'),
        mi('addShape', 'Bentuk'),
        { id: 'addTeam', label: 'Anggota tim', run: () => openTeam(menuAt ?? undefined) },
      ],
    })
  }
  out.push(
    {
      id: 'selectAll',
      label: 'Pilih semua',
      key: keyLabel('selectAll'),
      run: () => {
        select(cards.value.map((c) => c.id))
        announce.value = `${cards.value.length} kartu terpilih.`
      },
    },
    { id: 'fit', label: 'Pas ke layar', key: keyLabel('fit'), run: fit },
    { id: 'grid', label: 'Grid', key: keyLabel('grid'), checked: gridOn.value, run: toggleGrid },
    { id: 'snap', label: 'Snap ke grid', key: keyLabel('snap'), checked: snapOn.value, run: toggleSnap },
  )
  if (canWrite.value && canUndo.value)
    out.push({ id: 'undo', label: 'Urungkan', key: keyLabel('undo'), run: () => void runStep('undo') })
  if (canWrite.value && canRedo.value)
    out.push({ id: 'redo', label: 'Ulangi', key: keyLabel('redo'), run: () => void runStep('redo') })
  return out
}

function openMenu(x: number, y: number, opener: HTMLElement | null, onCard: boolean) {
  menu.value = onCard
    ? { x, y, opener, items: selectionMenu(), label: `Aksi untuk ${selected.value.size} kartu terpilih` }
    : { x, y, opener, items: emptyMenu(), label: 'Aksi kanvas' }
}

function closeMenu(refocus: boolean) {
  const opener = menu.value?.opener
  menu.value = null
  if (refocus) (opener ?? canvas.value)?.focus({ preventScroll: true })
}

function worldAt(clientX: number, clientY: number) {
  const r = canvas.value!.getBoundingClientRect()
  return toWorld(view.value, clientX - r.left, clientY - r.top)
}

// Klik kanan menggantikan menu browser hanya di kanvas dan kartu yang tidak sedang diedit. Di editor teks, isian,
// dan kartu yang sedang diedit, menu bawaan browser tetap muncul (salin, tempel, ejaan).
function onContextMenu(e: MouseEvent) {
  const t = e.target as HTMLElement
  if (t.closest('input, textarea, select, [contenteditable="true"], [role="textbox"]')) return
  const id = t.closest<HTMLElement>('[data-card]')?.dataset.card
  if (id && id === editingId.value) return
  e.preventDefault()
  // Tekan lama di layar sentuh sudah membuka menu sendiri; event contextmenu browser yang menyusul diabaikan.
  if (performance.now() - pressOpenedAt < 800) return
  if (penOn.value) return
  if (id && byId.value.has(id)) {
    if (!selected.value.has(id)) select(unitIds(id))
    focusedId.value = id
    openMenu(e.clientX, e.clientY, t.closest<HTMLElement>('[data-card]'), true)
  } else {
    menuAt = worldAt(e.clientX, e.clientY)
    openMenu(e.clientX, e.clientY, canvas.value, false)
  }
}

function openMenuFromKeyboard() {
  const id = focusedId.value && selected.value.has(focusedId.value) ? focusedId.value : null
  const el = id ? canvas.value?.querySelector<HTMLElement>(`[data-card="${id}"]`) : null
  if (id && el) {
    const r = el.getBoundingClientRect()
    openMenu(r.left + Math.min(r.width / 2, 40), r.top + Math.min(r.height / 2, 40), el, true)
    return
  }
  const r = canvas.value!.getBoundingClientRect()
  menuAt = worldAt(r.left + r.width / 2, r.top + r.height / 2)
  openMenu(r.left + r.width / 2, r.top + r.height / 2, canvas.value, false)
}

// Tekan lama 500 ms di layar sentuh membuka menu; geser lebih dari 8 px membatalkannya.
let pressTimer: ReturnType<typeof setTimeout> | undefined
let pressStart: { x: number; y: number } | null = null
function startPress(e: PointerEvent, id: string | null) {
  clearTimeout(pressTimer)
  const x = e.clientX
  const y = e.clientY
  pressStart = { x, y }
  pressTimer = setTimeout(() => {
    pressStart = null
    pressOpenedAt = performance.now()
    gesture = null
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
    window.removeEventListener('pointercancel', onUp)
    dragOffset.value = { dx: 0, dy: 0 }
    if (id && byId.value.has(id)) {
      if (!selected.value.has(id)) select(unitIds(id))
      openMenu(x, y, canvas.value?.querySelector<HTMLElement>(`[data-card="${id}"]`) ?? null, true)
    } else {
      menuAt = worldAt(x, y)
      openMenu(x, y, canvas.value, false)
    }
  }, 500)
}
function cancelPress() {
  clearTimeout(pressTimer)
  pressStart = null
}
onBeforeUnmount(() => clearTimeout(pressTimer))

// ---------------------------------------------------------------------------------------------------------------
// Undo dan redo (hanya aksi sendiri)

async function runStep(dir: 'undo' | 'redo') {
  const from = dir === 'undo' ? history.undo : history.redo
  const to = dir === 'undo' ? history.redo : history.undo
  const step = from.pop()
  historyVersion.value++
  if (!step) return
  let skipped = 0
  try {
    switch (step.kind) {
      case 'layout': {
        const r = layoutPatches(step, byId.value, dir)
        skipped = r.skipped
        for (const p of r.patches) {
          const { id, ...rest } = p
          patchLocal(id, {
            ...rest,
            color: rest.color === undefined ? byId.value.get(id)?.color : rest.color || null,
          } as Partial<Card>)
        }
        await sendGeometry(r.patches)
        break
      }
      case 'create':
        if (dir === 'undo') {
          const ids = step.ids.filter((id) => byId.value.has(id))
          if (ids.length) {
            const res = await deleteIds(ids)
            if (res) {
              removeLocal(ids)
              step.batch = res.batch
            }
          }
        } else if (step.batch) {
          await saving(() => restoreBatch(step.batch!))
          await load(true)
        }
        break
      case 'delete':
        if (dir === 'undo') {
          await saving(() => restoreBatch(step.batch))
          await load(true)
        } else {
          const ids = step.ids.filter((id) => byId.value.has(id))
          const res = ids.length ? await deleteIds(ids) : undefined
          if (res) {
            removeLocal(ids)
            step.batch = res.batch
          }
        }
        break
      case 'content': {
        const cur = byId.value.get(step.id)
        if (!cur || cur.content_version !== step.version) {
          skipped = 1
          break
        }
        const res = await saving(() =>
          patchCards(bid.value, [
            {
              id: step.id,
              content: (dir === 'undo' ? step.before : step.after) as Card['content'],
              base_content_version: cur.content_version,
            },
          ]),
        )
        if (res) {
          upsert(res.cards)
          step.version = res.cards[0]!.content_version
        }
        break
      }
      case 'checklist':
        await onChecklist(step.id, (dir === 'undo' ? step.ops.undo : step.ops.redo) as ChecklistOp, false)
        break
      case 'connectors':
        if (dir === 'undo') {
          step.batches = []
          for (const id of step.ids) {
            const res = await saving(() => deleteConnector(bid.value, id))
            if (res) step.batches.push(res.batch)
          }
          removeConnectors(step.ids)
        } else {
          for (const b of step.batches) await saving(() => restoreBatch(b))
          await load(true)
        }
        break
    }
  } catch {
    // Pesan galat sudah tampil di penanda simpan.
  }
  to.push(step)
  historyVersion.value++
  if (skipped > 0) showToast('Sebagian langkah tidak dibatalkan karena sudah diubah orang lain.')
  else announce.value = dir === 'undo' ? 'Langkah dibatalkan.' : 'Langkah diulang.'
}

// ---------------------------------------------------------------------------------------------------------------
// Salin, tempel, duplikat

const CLIP_PREFIX = 'nvm-whiteboard:'
type ClipCard = {
  kind: CardKind
  x: number
  y: number
  w: number
  h: number
  color: CardColor | null
  content: Card['content']
  title?: string
  file_id?: string
  style?: CardStyle
  locked?: boolean
  group_ids?: string[]
}
let clipMemory = ''

function clipOf(ids: string[]): string {
  const list: ClipCard[] = ids
    .map((id) => byId.value.get(id))
    .filter((c): c is Card => !!c)
    .slice(0, 100)
    .map((c) => ({
      kind: c.kind,
      x: c.x,
      y: c.y,
      w: c.w,
      h: c.h,
      color: c.color,
      content: c.content,
      title: c.ref?.title,
      file_id: c.file?.id,
      style: c.style,
      locked: c.locked,
      group_ids: c.group_ids,
    }))
  return CLIP_PREFIX + JSON.stringify({ v: 1, board: bid.value, cards: list })
}

function copySelection() {
  if (selected.value.size === 0) return
  clipMemory = clipOf([...selected.value])
  navigator.clipboard?.writeText(clipMemory).catch(() => undefined)
  announce.value = `${selected.value.size} kartu disalin.`
}

async function pasteText(text: string, sameOffset = false, at?: { x: number; y: number }) {
  if (!canWrite.value || !text.startsWith(CLIP_PREFIX)) return false
  let data: { board?: string; cards?: ClipCard[] }
  try {
    data = JSON.parse(text.slice(CLIP_PREFIX.length))
  } catch {
    return false
  }
  const list = (data.cards ?? []).slice(0, 100)
  const b = bounds(list)
  if (!b) return false
  // Di papan yang sama kartu digeser 24 px; di papan lain kelompoknya ditaruh di tengah layar.
  const c = centerWorld()
  // "Tempel di sini" (menu klik kanan): pojok kiri atas kelompok di titik klik.
  const dx = at
    ? Math.round(at.x - b.x)
    : data.board === bid.value || sameOffset
      ? 24
      : Math.round(c.x - (b.x + b.w / 2))
  const dy = at
    ? Math.round(at.y - b.y)
    : data.board === bid.value || sameOffset
      ? 24
      : Math.round(c.y - (b.y + b.h / 2))
  // Salinan mendapat id grup baru, supaya tidak bergabung ke grup aslinya; gaya dan kunci ikut.
  const paths = remap(
    list.map((k) => k.group_ids ?? []),
    newId,
  )
  const made: string[] = []
  for (const [i, k] of list.entries()) {
    const input: NewCard = {
      id: newId(),
      kind: k.kind,
      x: clamp(k.x + dx, -COORD_LIMIT, COORD_LIMIT),
      y: clamp(k.y + dy, -COORD_LIMIT, COORD_LIMIT),
      w: k.w,
      h: k.h,
      color: k.color ?? '',
    }
    // Kartu papan ditempel sebagai papan baru yang kosong; isinya tidak ikut tersalin.
    if (k.kind === 'board') input.title = `${k.title ?? 'Papan'} (salinan)`.slice(0, 120)
    else input.content = k.content
    if (k.file_id) input.file_id = k.file_id
    if (k.style) input.style = k.style
    if (k.locked) input.locked = true
    if (paths[i]?.length) input.group_ids = paths[i]
    try {
      const res = await saving(() => createCard(bid.value, input))
      if (res) {
        upsert([res.card])
        made.push(res.card.id)
      }
    } catch {
      break
    }
  }
  if (made.length) {
    record({ kind: 'create', ids: made })
    select(made)
    announce.value = `${made.length} kartu ditempel.`
  }
  return true
}

function duplicateSelection() {
  if (selected.value.size === 0) return
  pasteText(clipOf([...selected.value]), true)
}

function onPaste(e: ClipboardEvent) {
  const el = document.activeElement
  if (editingId.value || !canvas.value || !(el === canvas.value || canvas.value.contains(el))) return
  // Tangkapan layar dan berkas dari clipboard menjadi kartu gambar atau berkas di tengah layar.
  const files = pastedFiles(e.clipboardData, new Date())
  // Satu kartu gambar atau berkas terpilih: tempelan mengganti berkasnya (revisi 50), bukan kartu baru.
  const only = selected.value.size === 1 ? byId.value.get([...selected.value][0]!) : undefined
  if (files.length === 1 && canWrite.value && (only?.kind === 'image' || only?.kind === 'file')) {
    e.preventDefault()
    replaceCardFile(only.id, files[0]!)
    return
  }
  if (files.length && canWrite.value) {
    e.preventDefault()
    uploadFiles(files)
    return
  }
  const text = e.clipboardData?.getData('text/plain') || clipMemory
  if (text.startsWith(CLIP_PREFIX)) {
    e.preventDefault()
    pasteText(text)
  }
}
onMounted(() => document.addEventListener('paste', onPaste))

// ---------------------------------------------------------------------------------------------------------------
// Alat: klik menaruh kartu di tengah layar, seret menaruhnya di titik lepas.

type ToolKind = CardKind | 'upload' | 'template' | 'team'
const tools: { kind: ToolKind; label: string; path: string }[] = [
  { kind: 'note', label: 'Catatan', path: 'M5 4h10l4 4v12H5zM15 4v4h4M8 12h8M8 16h5' },
  { kind: 'heading', label: 'Judul', path: 'M6 5v14M18 5v14M6 12h12' },
  { kind: 'checklist', label: 'Checklist', path: 'M4 7l2 2 3-3M4 15l2 2 3-3M12 8h8M12 16h8' },
  { kind: 'board', label: 'Papan', path: 'M4 4h16v16H4zM8 8h3v3H8zM13 8h3M13 11h3M8 14h8' },
  { kind: 'upload', label: 'Berkas', path: 'M12 16V4M7 9l5-5 5 5M4 16v4h16v-4' },
  {
    kind: 'link',
    label: 'Tautan',
    path: 'M10 14a4 4 0 0 0 6 0l3-3a4 4 0 0 0-6-6l-1 1M14 10a4 4 0 0 0-6 0l-3 3a4 4 0 0 0 6 6l1-1',
  },
  { kind: 'column', label: 'Kolom', path: 'M5 4h6v16H5zM13 4h6v10h-6z' },
  { kind: 'drawing', label: 'Gambar', path: 'M4 20l4-1 11-11-3-3L5 16zM14 7l3 3' },
  { kind: 'table', label: 'Tabel', path: 'M4 5h16v14H4zM4 10h16M4 15h16M10 5v14' },
  { kind: 'template', label: 'Template', path: 'M4 4h7v7H4zM13 4h7v7h-7zM4 13h7v7H4zM16.5 13v7M13 16.5h7' },
  {
    kind: 'team',
    label: 'Tim',
    path: 'M9 11a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7M3 20c0-3.3 2.7-6 6-6s6 2.7 6 6M16 4.5a3 3 0 0 1 0 6M18 14c2 .8 3 2.8 3 6',
  },
  {
    kind: 'shape',
    label: 'Bentuk',
    path: 'M3 14h7v7H3zM17.5 3a3.5 3.5 0 1 1 0 7 3.5 3.5 0 0 1 0-7M14 21l3.5-7 3.5 7z',
  },
]

// Alat yang membuka pilihan dulu (berkas, tautan) mengingat titik lepasnya.
let toolAt: { x: number; y: number } | undefined
function useTool(kind: ToolKind, at?: { x: number; y: number }) {
  toolAt = at
  listMode.value = false
  if (kind === 'drawing') {
    if (penOn.value) stopPen()
    else startPen()
    return
  }
  stopPen()
  if (kind === 'upload') fileInput.value?.click()
  else if (kind === 'template') {
    templateError.value = ''
    templateOpen.value = true
  } else if (kind === 'shape') shapePaletteOpen.value = true
  else if (kind === 'team') openTeam(at)
  else if (kind === 'link') {
    linkError.value = ''
    linkOpen.value = true
  } else addCard(kind, at)
}
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
  useTool(d.kind, toWorld(view.value, e.clientX - r.left, e.clientY - r.top))
}
function onToolClick(kind: ToolKind) {
  if (suppressClick) {
    suppressClick = false
    return
  }
  sheetOpen.value = false
  useTool(kind)
}
const toolGhost = computed(() => {
  const d = toolDrag.value
  if (!d?.moved) return null
  return {
    label: tools.find((t) => t.kind === d.kind)?.label ?? '',
    style: { left: `${d.x + 8}px`, top: `${d.y + 8}px` },
  }
})

// ---------------------------------------------------------------------------------------------------------------
// Mode pena (revisi A): coret langsung di kanvas, di atas kartu. Satu sesi pena orang yang sama menjadi satu objek
// gambar; lewat batas (300 goresan, 6.000 titik, atau 4.000 px) sesi otomatis memulai objek baru. Goresan yang
// sedang digambar disiarkan langsung lewat WebSocket, dan disimpan lewat REST saat goresan selesai.

const penOn = ref(false)
const penColor = ref('tinta')
const penWidth = ref(4)
const penEraser = ref(false)
// Objek gambar sesi ini (dikunci selama sesi, supaya penghapus orang lain ditolak dengan nama kita).
const penObject = ref<string | null>(null)
const live = shallowRef<PenStroke | null>(null)
// Goresan yang sudah selesai tetapi belum dijawab server, tetap tampil supaya tidak berkedip.
const pendingInk = shallowRef<PenStroke[]>([])
const remoteInk = ref<Record<number, PenStroke & { at: number; color: string }>>({})
let penChain: Promise<unknown> = Promise.resolve()
let lastInkSent = 0

function startPen() {
  if (!canWrite.value) return
  if (editingId.value) stopEdit()
  select([])
  penOn.value = true
  penEraser.value = false
  announce.value =
    'Mode pena aktif. Coret di kanvas; dua jari untuk menggeser. Escape atau Selesai untuk keluar.'
}

// Escape keluar dari mode pena di mana pun fokusnya (kanvas, toolbar pena, atau halaman), kecuali di dialog.
function onPenEscape(e: KeyboardEvent) {
  if (e.key !== 'Escape' || !penOn.value || (e.target as HTMLElement | null)?.closest?.('dialog')) return
  e.preventDefault()
  stopPen()
}
watch(penOn, (on) => {
  if (on) window.addEventListener('keydown', onPenEscape)
  else window.removeEventListener('keydown', onPenEscape)
})

function stopPen() {
  if (!penOn.value) return
  penOn.value = false
  live.value = null
  if (penObject.value) send({ t: 'unlock', card: penObject.value })
  penObject.value = null
  announce.value = 'Mode pena selesai.'
}

function penPoint(e: PointerEvent): [number, number] {
  const p = local(e)
  const w = toWorld(view.value, p.x, p.y)
  return [Math.round(w.x), Math.round(w.y)]
}

function drawingObjects() {
  return shownCards.value.filter((c) => c.kind === 'drawing') as (Card & { content: DrawingContent })[]
}

async function eraseAt(e: PointerEvent) {
  const [x, y] = penPoint(e)
  const hit = hitStroke(drawingObjects(), x, y, 8 / view.value.zoom)
  if (!hit) return
  const lock = remoteLocks.value[hit.id]
  if (lock && lock.id !== myId.value) {
    showToast(`${lock.name} sedang menggambar di objek ini. Tunggu sampai selesai, lalu hapus.`)
    return
  }
  const obj = byId.value.get(hit.id)
  if (!obj) return
  if ((obj.content as DrawingContent).strokes.length <= 1) {
    if (penObject.value === hit.id) penObject.value = null
    const res = await deleteIds([hit.id]).catch(() => undefined)
    if (res) {
      removeLocal([hit.id])
      record({ kind: 'delete', ids: [hit.id], batch: res.batch })
      showToast('Gambar dipindah ke Tempat sampah.', res.batch)
    }
    return
  }
  try {
    const res = await saving(() => drawingOp(bid.value, hit.id, { op: 'remove', index: hit.index }))
    if (res) upsert([res.card])
  } catch (err) {
    if (err instanceof ApiError && err.status === 409) showToast(err.message)
  }
}

function onPenDown(e: PointerEvent) {
  if (!penOn.value || (e.pointerType === 'mouse' && e.button !== 0)) return
  e.preventDefault()
  markActive()
  // Dua jari: goresan dibatalkan, kanvas digeser dan di-zoom seperti biasa.
  if (trackTouch(e)) {
    live.value = null
    send({ t: 'ink_end' })
    begin(e)
    return
  }
  if (penEraser.value) {
    eraseAt(e)
    return
  }
  try {
    ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  } catch {
    // Pointer sintetis tidak bisa di-capture; gerakan tetap terbaca selama di atas kanvas.
  }
  live.value = { color: penColor.value, width: penWidth.value, points: penPoint(e) }
}

function onPenMove(e: PointerEvent) {
  if (e.pointerType === 'touch' && touches.has(e.pointerId)) touches.set(e.pointerId, local(e))
  if (gesture?.kind === 'pinch') return
  if (penEraser.value && e.buttons === 1) {
    eraseAt(e)
    return
  }
  const s = live.value
  if (!s) return
  const [x, y] = penPoint(e)
  const n = s.points.length
  if (n >= MAX_STROKE_POINTS * 8 || (s.points[n - 2] === x && s.points[n - 1] === y)) return
  live.value = { ...s, points: [...s.points, x, y] }
  const now = performance.now()
  if (now - lastInkSent > 66) {
    lastInkSent = now
    send({
      t: 'ink',
      points: live.value.points.slice(-MAX_STROKE_POINTS * 2),
      color: s.color,
      width: s.width,
    })
  }
}

function onPenUp(e: PointerEvent) {
  if (e.pointerType === 'touch') touches.delete(e.pointerId)
  const s = live.value
  live.value = null
  if (!s) return
  send({ t: 'ink_end' })
  // Toleransi sekitar setengah piksel layar pada zoom sekarang.
  const points = simplify(s.points, 0.5 / view.value.zoom).slice(0, MAX_STROKE_POINTS * 2)
  const stroke = { ...s, points }
  pendingInk.value = [...pendingInk.value, stroke]
  penChain = penChain
    .then(() => saveStroke(stroke))
    .finally(() => {
      pendingInk.value = pendingInk.value.filter((p) => p !== stroke)
    })
}

// Goresan masuk objek sesi bila masih muat; kalau tidak, sesi memulai objek baru.
function fits(obj: Card | undefined, stroke: PenStroke) {
  if (!obj) return false
  const c = obj.content as DrawingContent
  if (c.strokes.length >= MAX_STROKES || drawingPoints(c) + stroke.points.length / 2 > MAX_DRAWING_POINTS)
    return false
  const abs = c.strokes.map((k) => ({ ...k, points: k.points.map((v, i) => v + (i % 2 ? obj.y : obj.x)) }))
  const box = strokesBox([...abs, stroke])
  return !!box && box.w <= DRAWING_SPACE && box.h <= DRAWING_SPACE
}

async function saveStroke(stroke: PenStroke) {
  const obj = penObject.value ? byId.value.get(penObject.value) : undefined
  try {
    if (obj && fits(obj, stroke)) {
      const res = await saving(() => drawingOp(bid.value, obj.id, { op: 'add', stroke }))
      if (res) upsert([res.card])
      return
    }
    const id = newId()
    const res = await saving(() => createDrawing(bid.value, id, stroke))
    if (!res) return
    upsert([res.card])
    record({ kind: 'create', ids: [id] })
    if (penObject.value) send({ t: 'unlock', card: penObject.value })
    // Objek sesi dikunci: penghapus orang lain di objek ini ditolak dengan nama kita.
    penObject.value = penOn.value ? id : null
    if (penObject.value) send({ t: 'lock', card: id })
  } catch (e) {
    if (e instanceof ApiError) showToast(e.message)
  }
}

async function undoStroke() {
  const obj = penObject.value ? byId.value.get(penObject.value) : undefined
  if (!obj) return
  const n = (obj.content as DrawingContent).strokes.length
  await penChain
  if (n <= 1) {
    const res = await deleteIds([obj.id]).catch(() => undefined)
    if (res) {
      removeLocal([obj.id])
      penObject.value = null
    }
    return
  }
  try {
    const res = await saving(() => drawingOp(bid.value, obj.id, { op: 'remove', index: n - 1 }))
    if (res) upsert([res.card])
  } catch {
    // Pesan galat sudah tampil di penanda simpan.
  }
}

function inkPath(points: number[]) {
  let d = ''
  for (let i = 0; i + 1 < points.length; i += 2) d += `${i ? 'L' : 'M'}${points[i]} ${points[i + 1]}`
  return points.length === 2 ? `${d}h0.1` : d
}

const inkLayer = computed(() => [
  ...pendingInk.value,
  ...(live.value ? [live.value] : []),
  ...Object.values(remoteInk.value),
])

// Ganti warna semua goresan objek gambar terpilih.
async function recolorDrawings(color: string) {
  for (const id of selected.value) {
    if (byId.value.get(id)?.kind !== 'drawing') continue
    try {
      const res = await saving(() => drawingOp(bid.value, id, { op: 'color', color }))
      if (res) upsert([res.card])
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) showToast(e.message)
    }
  }
}
const onlyDrawings = computed(
  () => selected.value.size > 0 && [...selected.value].every((id) => byId.value.get(id)?.kind === 'drawing'),
)

// ---------------------------------------------------------------------------------------------------------------
// Berkas dan gambar (W3): pilih, tempel, atau seret berkas dari komputer ke kanvas. Satu berkas per waktu.

const fileInput = ref<HTMLInputElement | null>(null)
const allowedExt = [
  'jpg',
  'jpeg',
  'png',
  'webp',
  'gif',
  'pdf',
  'doc',
  'docx',
  'xls',
  'xlsx',
  'ppt',
  'pptx',
  'txt',
  'csv',
]
const acceptFiles = [...allowedExt.map((e) => `.${e}`), 'image/*'].join(',')
const MAX_FILE = 50 * 1024 * 1024
const uploadStatus = ref('')
let uploadBusy = false
const uploadWaiting: { file: File; at: { x: number; y: number } }[] = []

function precheckFile(f: File): string | null {
  const ext = f.name.includes('.') ? f.name.split('.').pop()!.toLowerCase() : ''
  if (!allowedExt.includes(ext))
    return `${f.name} tidak bisa diunggah. Gunakan JPG, PNG, WebP, GIF, PDF, dokumen Office, TXT, atau CSV.`
  if (f.size === 0) return `${f.name} kosong.`
  if (f.size > MAX_FILE) return `${f.name} berukuran ${formatBytes(f.size)}, melebihi batas 50 MB.`
  return null
}

function onFilePicked(e: Event) {
  const el = e.target as HTMLInputElement
  uploadFiles([...(el.files ?? [])], toolAt)
  el.value = ''
}

function uploadFiles(files: File[], at?: { x: number; y: number }) {
  if (!canWrite.value || !files.length) return
  const base = at ?? centerWorld()
  const bad: string[] = []
  files.forEach((file, i) => {
    const err = precheckFile(file)
    if (err) bad.push(err)
    else uploadWaiting.push({ file, at: { x: base.x + i * 24, y: base.y + i * 24 } })
  })
  if (bad.length) showToast(bad.join(' '))
  void runUploads()
}

async function runUploads() {
  if (uploadBusy) return
  uploadBusy = true
  const made: string[] = []
  try {
    for (let item = uploadWaiting.shift(); item; item = uploadWaiting.shift()) {
      const { file, at } = item
      const dim = defaultSize[file.type.startsWith('image/') ? 'image' : 'file']
      const left = uploadWaiting.length
      uploadStatus.value = `Mengunggah ${file.name}${left ? `, ${left} lagi menunggu` : ''}`
      try {
        const card = await uploadFile(
          bid.value,
          {
            id: newId(),
            x: clamp(snap(at.x - dim.w / 2, snapOn.value), -COORD_LIMIT, COORD_LIMIT),
            y: clamp(snap(at.y - dim.h / 2, snapOn.value), -COORD_LIMIT, COORD_LIMIT),
            w: dim.w,
            h: dim.h,
          },
          file,
          (p) =>
            (uploadStatus.value = `Mengunggah ${file.name} (${Math.round(p * 100)}%)${left ? `, ${left} lagi menunggu` : ''}`),
        ).promise
        upsert([card])
        made.push(card.id)
      } catch (e) {
        showToast(`${file.name}: ${e instanceof ApiError ? e.message : 'unggahan gagal.'}`)
      }
    }
  } finally {
    uploadBusy = false
    uploadStatus.value = ''
  }
  if (made.length) {
    record({ kind: 'create', ids: made })
    select(made)
    announce.value = `${made.length} berkas diunggah.`
  }
}

// Seret berkas dari komputer ke kanvas: kartu muncul di titik lepas.
const fileOver = ref(false)
function hasFiles(e: DragEvent) {
  return !!e.dataTransfer && [...e.dataTransfer.types].includes('Files')
}
function onDragOver(e: DragEvent) {
  if (!canWrite.value || !hasFiles(e)) return
  e.preventDefault()
  fileOver.value = true
}
function onDrop(e: DragEvent) {
  fileOver.value = false
  if (!canWrite.value || !hasFiles(e) || !canvas.value) return
  e.preventDefault()
  const files = [...(e.dataTransfer?.files ?? [])]
  // Berkas dilepas di atas kartu gambar atau berkas yang terpilih: berkasnya diganti (revisi 50).
  const id = (document.elementFromPoint(e.clientX, e.clientY) as HTMLElement | null)?.closest<HTMLElement>(
    '[data-card]',
  )?.dataset.card
  const target = id && selected.value.has(id) ? byId.value.get(id) : undefined
  if (files.length === 1 && (target?.kind === 'image' || target?.kind === 'file')) {
    replaceCardFile(target.id, files[0]!)
    return
  }
  const r = canvas.value.getBoundingClientRect()
  uploadFiles(files, toWorld(view.value, e.clientX - r.left, e.clientY - r.top))
}

// ---------------------------------------------------------------------------------------------------------------
// Tautan (W3): pratinjau diambil server; bila pratinjau gagal, kartu tetap dibuat dengan alamatnya saja.

const linkOpen = ref(false)
const linkBusy = ref(false)
const linkError = ref('')

async function submitLink(url: string) {
  if (!url) return
  // Pesan langsung tanpa memanggil server; server tetap memeriksa hal yang sama.
  if (!/^https?:\/\/[^/\s]/i.test(url)) {
    linkError.value = 'Kartu tautan hanya untuk alamat http:// atau https://.'
    return
  }
  linkBusy.value = true
  linkError.value = ''
  let previewId = ''
  let note = ''
  try {
    const p = (await fetchPreview(url)).preview
    previewId = p.id
    if (p.status !== 'ok') note = p.message
  } catch (e) {
    note = e instanceof ApiError ? e.message : 'Pratinjau tidak bisa diambil.'
  }
  try {
    const res = await createCard(bid.value, {
      id: newId(),
      kind: 'link',
      ...placeAt('link', toolAt),
      color: '',
      content: { url, preview_id: previewId },
    })
    upsert([res.card])
    record({ kind: 'create', ids: [res.card.id] })
    select([res.card.id])
    linkOpen.value = false
    if (note) showToast(`Tautan ditambahkan tanpa pratinjau. ${note}`)
    else announce.value = 'Tautan ditambahkan.'
  } catch (e) {
    linkError.value = e instanceof ApiError ? e.message : 'Tautan gagal ditambahkan. Periksa koneksi.'
  } finally {
    linkBusy.value = false
  }
}

function placeAt(kind: CardKind, at?: { x: number; y: number }) {
  const dim = defaultSize[kind]
  const c = at ?? centerWorld()
  return {
    x: clamp(snap(c.x - dim.w / 2, snapOn.value), -COORD_LIMIT, COORD_LIMIT),
    y: clamp(snap(c.y - dim.h / 2, snapOn.value), -COORD_LIMIT, COORD_LIMIT),
    w: dim.w,
    h: dim.h,
  }
}

// ---------------------------------------------------------------------------------------------------------------
// Bentuk dan diagram (W5)

const shapePaletteOpen = ref(false)
async function addShape(t: ShapeType) {
  shapePaletteOpen.value = false
  const s = SHAPES.find((x) => x.id === t)!
  await addCard('shape', toolAt, newShapeContent(t), { w: s.w, h: s.h })
}

// Panel gaya bentuk: juga jalur keyboard untuk putar dan ukuran.
const shapeStyleId = ref<string | null>(null)
const shapeStyleCard = computed(() =>
  shapeStyleId.value ? (byId.value.get(shapeStyleId.value) ?? null) : null,
)
function onShapeStyle(c: ShapeContent) {
  if (shapeStyleId.value) onContent(shapeStyleId.value, c)
}
function onShapeSize(sz: { w: number; h: number }) {
  const id = shapeStyleId.value
  if (id) applyLayout([{ id, after: sz }])
}

// Sudut sementara selama pegangan putar diseret.
const rotateDraft = ref<{ id: string; deg: number } | null>(null)
const rotations = computed(() => {
  const m = new Map<string, number>()
  for (const c of shownCards.value)
    if (c.kind === 'shape') m.set(c.id, (c.content as ShapeContent).rotation || 0)
  if (rotateDraft.value) m.set(rotateDraft.value.id, rotateDraft.value.deg)
  return m
})

// Garis terpilih di kanvas (klik garisnya): Delete menghapus, Enter membuka panel gaya, Escape melepas.
const selectedConnector = ref<string | null>(null)
watch(selected, (s) => {
  if (s.size) selectedConnector.value = null
})
// Titik belok sementara selama diseret, sebelum disimpan.
const bendDraft = ref<{ id: string; points: [number, number][] } | null>(null)
const lines = computed<DrawnLine[]>(() =>
  visibleConnectors.value.flatMap((c) => {
    const pts = bendDraft.value?.id === c.id ? bendDraft.value.points : undefined
    const r = routeConnector(c, geomMap.value, rotations.value, pts)
    if (!r) return []
    return [
      {
        ...r,
        color: inkColor(c.style.color || 'tinta'),
        dash: c.style.line === 'dashed' ? '8 6' : c.style.line === 'dotted' ? '2 5' : undefined,
        label: c.label,
        label_start: c.label_start,
        label_end: c.label_end,
      },
    ]
  }),
)
const selectedStops = computed(() =>
  canWrite.value && selectedConnector.value
    ? (lines.value.find((l) => l.id === selectedConnector.value)?.stops ?? null)
    : null,
)

// Port kartu terpilih untuk menarik garis baru: hanya satu kartu, bisa ditulis, tidak sedang diedit atau mode pena.
const connectDraft = ref<{ card: string; port: Port; at: Pt } | null>(null)
const PORT_OFFSET = 48
const portHandles = computed(() => {
  if (!canWrite.value || penOn.value || editingId.value || selected.value.size !== 1) return []
  const id = [...selected.value][0]!
  const card = shownById.value.get(id)
  const b = geomMap.value.get(id)
  if (!card || !b || card.kind === 'drawing' || card.kind === 'column' || card.column_id) return []
  const rot = rotations.value.get(id) ?? 0
  // Pegangan port 48 px layar di luar tepi, searah port: di tepi kartu ada pegangan ubah ukuran 44 px yang sama
  // letaknya, dan klik di sana harus mengubah ukuran, bukan menarik garis. Garis tetap menempel di port aslinya.
  const out = PORT_OFFSET / view.value.zoom
  return portsFor(card.kind === 'shape').map((port) => {
    const { p, dir } = portPoint(b, port as Exclude<Port, 'auto'>, rot)
    return { card: id, port, p, h: { x: p.x + dir.x * out, y: p.y + dir.y * out } }
  })
})
const tempLine = computed(() => {
  const d = connectDraft.value
  if (!d) return null
  const b = geomMap.value.get(d.card)
  if (!b) return null
  return { a: portPoint(b, d.port as Exclude<Port, 'auto'>, rotations.value.get(d.card) ?? 0).p, b: d.at }
})

function worldOf(e: PointerEvent): Pt {
  const p = local(e)
  return toWorld(view.value, p.x, p.y)
}

function onPortDown(card: string, port: Port, e: PointerEvent) {
  if (!canWrite.value) return
  gesture = { kind: 'connect', card, port }
  connectDraft.value = { card, port, at: worldOf(e) }
  begin(e)
}

// Kartu paling atas di titik dunia (selain yang dikecualikan), beserta port terdekatnya bila cukup dekat.
function cardAt(at: Pt, except: string): { id: string; port: Port } | null {
  const hit = [...rendered.value]
    .reverse()
    .find(
      (c) =>
        c.id !== except &&
        c.kind !== 'drawing' &&
        at.x >= c.x &&
        at.x <= c.x + c.w &&
        at.y >= c.y &&
        at.y <= c.y + c.h,
    )
  if (!hit) return null
  const b = geomMap.value.get(hit.id) ?? hit
  const rot = rotations.value.get(hit.id) ?? 0
  let port: Port = 'auto'
  let best = 28 / view.value.zoom
  for (const pt of portsFor(hit.kind === 'shape')) {
    const p = portPoint(b, pt as Exclude<Port, 'auto'>, rot).p
    const dist = Math.hypot(p.x - at.x, p.y - at.y)
    if (dist < best) {
      best = dist
      port = pt
    }
  }
  return { id: hit.id, port }
}

async function finishConnect(at: Pt) {
  const d = connectDraft.value
  connectDraft.value = null
  if (!d) return
  const from = byId.value.get(d.card)
  if (!from) return
  const near = portHandles.value.find((h) => Math.hypot(h.h.x - at.x, h.h.y - at.y) < 8 / view.value.zoom)
  if (near && near.card === d.card) return
  const target = cardAt(at, d.card)
  const toShape = target ? byId.value.get(target.id)?.kind === 'shape' : false
  const route = from.kind === 'shape' || toShape ? 'orthogonal' : 'straight'
  try {
    const res = await saving(() =>
      createConnector(bid.value, {
        from: d.card,
        to: target?.id ?? '',
        style: { start: 'none', end: 'arrow', line: 'solid', color: '' },
        geometry: {
          from: { port: d.port },
          to: target ? { port: target.port } : { port: '', x: Math.round(at.x), y: Math.round(at.y) },
          route,
          points: [],
        },
      }),
    )
    if (!res) return
    upsertConnectors([res.connector])
    select([])
    selectedConnector.value = res.connector.id
    announce.value = target ? 'Garis ditambahkan.' : 'Garis ditambahkan dengan ujung bebas.'
  } catch {
    // Pesan galat sudah tampil di penanda simpan.
  }
}

function onBendDown(index: number, insert: boolean, e: PointerEvent) {
  const id = selectedConnector.value
  const c = id ? connectors.value.find((x) => x.id === id) : null
  if (!c || !canWrite.value) return
  const pts = [...(c.geometry.points ?? [])] as [number, number][]
  const at = worldOf(e)
  if (insert) pts.splice(index, 0, [Math.round(at.x), Math.round(at.y)])
  gesture = { kind: 'bend', id: c.id, index }
  bendDraft.value = { id: c.id, points: pts }
  begin(e)
}

async function saveBends(id: string, points: [number, number][]) {
  const c = connectors.value.find((x) => x.id === id)
  if (!c) return
  await onConnectorUpdate(id, { geometry: { ...c.geometry, points } })
}

function onUnbend(index: number) {
  const id = selectedConnector.value
  const c = id ? connectors.value.find((x) => x.id === id) : null
  if (!c) return
  const pts = [...(c.geometry.points ?? [])] as [number, number][]
  pts.splice(index, 1)
  void saveBends(c.id, pts)
  announce.value = 'Titik belok dihapus.'
}

function onRotateStart(card: Card, e: PointerEvent) {
  if (!canWrite.value) return
  const b = geomMap.value.get(card.id) ?? card
  gesture = { kind: 'rotate', id: card.id, c: boxCenter(b) }
  rotateDraft.value = { id: card.id, deg: (card.content as ShapeContent).rotation || 0 }
  send({ t: 'grab', ids: [card.id] })
  begin(e)
}

// Klik garis memilihnya; fokus pindah ke kanvas supaya Delete, Enter, dan Escape langsung berlaku.
function onConnectorSelect(id: string) {
  select([])
  selectedConnector.value = id
  canvas.value?.focus()
  announce.value = 'Garis terpilih. Enter membuka gaya garis, Delete menghapus.'
}

function onConnectorKey(e: KeyboardEvent): boolean {
  const id = selectedConnector.value
  if (!id) return false
  if (e.key === 'Escape') selectedConnector.value = null
  else if (e.key === 'Enter') connectorIds.value = [id]
  else if ((e.key === 'Delete' || e.key === 'Backspace') && canWrite.value) {
    void onConnectorRemove(id)
    selectedConnector.value = null
  } else return false
  e.preventDefault()
  return true
}

// ---------------------------------------------------------------------------------------------------------------
// Garis penghubung (W3)

const connectorIds = ref<string[] | null>(null)
const dialogConnectors = computed(() =>
  (connectorIds.value ?? []).flatMap((id) => connectors.value.find((c) => c.id === id) ?? []),
)
const visibleConnectors = computed(() =>
  connectors.value.filter((c) => (!c.from || byId.value.has(c.from)) && (!c.to || byId.value.has(c.to))),
)
function connectorsOf(id: string) {
  return visibleConnectors.value.filter((c) => c.from === id || c.to === id)
}
function cardName(id: string) {
  const c = byId.value.get(id)
  return c ? cardLabel(c) : 'kartu'
}

// Hubungkan kartu terpilih berurutan sesuai urutan pilih (revisi 50): 1 ke 2, 2 ke 3, dan seterusnya. Satu langkah
// undo untuk semua garis baru.
async function connectInOrder() {
  const ids = [...selected.value]
  const made: string[] = []
  for (let i = 0; i + 1 < ids.length; i++) {
    const id = await connectPair(ids[i]!, ids[i + 1]!)
    if (id) made.push(id)
  }
  if (made.length) {
    record({ kind: 'connectors', ids: made, batches: [] })
    announce.value =
      made.length === 1 ? 'Garis ditambahkan.' : `${made.length} garis ditambahkan sesuai urutan pilih.`
  }
  // Dua kartu, satu garis: dialog Garis langsung terbuka untuk label dan gaya, seperti sejak W3.
  if (ids.length === 2 && made.length === 1) connectorIds.value = made
}

async function connectPair(from: string, to: string): Promise<string | null> {
  try {
    // Bentuk dan kartu orang (bagan organisasi) memakai garis siku dengan panah.
    const shape = [from, to].some((id) => ['shape', 'person'].includes(byId.value.get(id)?.kind ?? ''))
    const res = await saving(() =>
      createConnector(bid.value, {
        from,
        to,
        ...(shape
          ? {
              style: { start: 'none', end: 'arrow', line: 'solid', color: '' },
              geometry: { from: { port: 'auto' }, to: { port: 'auto' }, route: 'orthogonal', points: [] },
            }
          : {}),
      }),
    )
    if (!res) return null
    upsertConnectors([res.connector])
    return res.connector.id
  } catch {
    // Pesan galat sudah tampil di penanda simpan.
    return null
  }
}

async function onConnectorUpdate(id: string, patch: ConnectorUpdate) {
  try {
    const res = await saving(() => updateConnector(bid.value, id, patch))
    if (res) upsertConnectors([res.connector])
  } catch {
    // Pesan galat sudah tampil di penanda simpan.
  }
}

async function onConnectorRemove(id: string) {
  try {
    const res = await saving(() => deleteConnector(bid.value, id))
    if (!res) return
    removeConnectors([id])
    connectorIds.value = (connectorIds.value ?? []).filter((x) => x !== id)
    if (connectorIds.value.length === 0) connectorIds.value = null
    showToast('Garis dipindah ke Tempat sampah.', res.batch)
  } catch {
    // Pesan galat sudah tampil di penanda simpan.
  }
}

// ---------------------------------------------------------------------------------------------------------------
// Pindah ke papan lain (W3): dari dialog, atau dengan melepas kartu di atas kartu papan.

const moveOpen = ref(false)
const moveBusy = ref(false)
const moveError = ref('')
const movingRefs = computed(() => [...selected.value].flatMap((id) => byId.value.get(id)?.ref?.id ?? []))

// Isi kolom ikut kolomnya di server, jadi hanya kartu teratas yang dikirim.
function topLevel(ids: string[]) {
  const set = new Set(ids)
  return ids.filter((id) => {
    const col = byId.value.get(id)?.column_id
    return !col || !set.has(col)
  })
}

async function moveTo(ids: string[], to: string, title: string) {
  const list = topLevel(ids)
  if (!list.length) return false
  // Kartu terkunci tidak boleh pindah papan (keputusan Manajer, revisi 51); server juga menolaknya.
  const locked = ids.filter((id) => byId.value.get(id)?.locked).length
  if (locked) {
    moveOpen.value = false
    showToast(`${locked} kartu terkunci. Buka kunci dulu (${keyLabel('lock')}), lalu pindahkan lagi.`)
    return false
  }
  moveBusy.value = true
  moveError.value = ''
  try {
    const res = await moveCards(bid.value, list, to)
    removeLocal(res.moved.moved)
    removeConnectors(res.moved.connectors_gone)
    moveOpen.value = false
    showToast(`${res.moved.moved.length} kartu dipindah ke papan ${title}.`, undefined, to)
    return true
  } catch (e) {
    moveError.value = e instanceof ApiError ? e.message : 'Kartu gagal dipindah. Periksa koneksi.'
    if (!moveOpen.value) showToast(moveError.value)
    return false
  } finally {
    moveBusy.value = false
  }
}

function moveFromDialog(to: string) {
  moveTo([...selected.value], to, 'tujuan')
}

// ---------------------------------------------------------------------------------------------------------------
// Komentar (W3)

const commentsFor = ref<string | null>(null)
const commentsReload = ref(0)
const commentsCard = computed(() => (commentsFor.value ? (byId.value.get(commentsFor.value) ?? null) : null))

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

function goParent() {
  boardGone.value = null
  const p = parentCrumb.value
  router.push(p ? crumbLink(crumbs.value.length - 2) : { name: 'whiteboard' })
}

// ---------------------------------------------------------------------------------------------------------------
// Minimap: persegi per kartu dan bingkai layar, klik memindah tampilan ke titik itu.

const MINI_W = 160
const MINI_H = 100
const minimap = computed(() => {
  if (shownCards.value.length === 0 || size.value.w === 0) return null
  const v = view.value
  const tl = toWorld(v, 0, 0)
  const screen = { x: tl.x, y: tl.y, w: size.value.w / v.zoom, h: size.value.h / v.zoom }
  const b = bounds([...shownCards.value, screen])!
  const scale = Math.min(MINI_W / b.w, MINI_H / b.h)
  const map = (r: { x: number; y: number; w: number; h: number }) => ({
    x: (r.x - b.x) * scale,
    y: (r.y - b.y) * scale,
    w: Math.max(1, r.w * scale),
    h: Math.max(1, r.h * scale),
  })
  return { rects: shownCards.value.map((c) => ({ id: c.id, ...map(c) })), screen: map(screen), b, scale }
})

function onMinimapClick(e: MouseEvent) {
  const m = minimap.value
  if (!m) return
  const r = (e.currentTarget as SVGElement).getBoundingClientRect()
  centerOn(m.b.x + (e.clientX - r.left) / m.scale, m.b.y + (e.clientY - r.top) / m.scale)
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
      cancelled: boolean
    }
  | {
      kind: 'resize'
      startX: number
      startY: number
      id: string
      cancelled: boolean
      handle: Handle
      rot: number
      start: Rect
    }
  | { kind: 'rotate'; id: string; c: Pt }
  | { kind: 'connect'; card: string; port: Port }
  | { kind: 'bend'; id: string; index: number }
  | { kind: 'pan'; startX: number; startY: number; view: Viewport; tapCard: string | null }
  | { kind: 'box'; startX: number; startY: number; additive: boolean; base: Set<string> }
  | { kind: 'pinch' }

let gesture: Gesture | null = null
const touches = new Map<number, { x: number; y: number }>()
let pinchStart: { dist: number; mid: { x: number; y: number }; view: Viewport } | null = null
const dragOffset = ref({ dx: 0, dy: 0 })
// dx dan dy dipakai pegangan kiri dan atas (W5), yang menggeser kartu sambil mengubah ukurannya.
const resizeDelta = ref({ dx: 0, dy: 0, dw: 0, dh: 0 })
const box = ref<{ x: number; y: number; w: number; h: number } | null>(null)
const gestureResizeId = ref<string | null>(null)
const spaceHeld = ref(false)
const panMode = ref(false)
let lastMoveSent = 0

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
  selectedConnector.value = null
  // Klik di area kosong keluar dari grup yang sedang dimasuki.
  groupScope.value = []
  if (e.pointerType === 'touch') startPress(e, null)
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
  if (e.pointerType === 'touch') startPress(e, card.id)
  if (trackTouch(e)) return
  const p = local(e)
  // Sentuhan pada kartu yang belum terpilih menggeser kanvas, supaya papan penuh kartu tetap bisa digeser di
  // ponsel. Ketukan tanpa gerak memilih kartu (kartu papan langsung dibuka); ketukan pada kartu terpilih membukanya.
  if (e.pointerType === 'touch' && !selected.value.has(card.id)) {
    gesture = { kind: 'pan', startX: p.x, startY: p.y, view: { ...view.value }, tapCard: card.id }
    begin(e)
    return
  }
  // Klik memilih seluruh grup (atau satu tingkat di bawah grup yang sedang dimasuki); Shift menambah atau melepas
  // seluruh unit itu.
  const unit = unitIds(card.id)
  if (e.shiftKey) {
    const next = new Set(selected.value)
    const has = next.has(card.id)
    for (const id of unit) {
      if (has) next.delete(id)
      else next.add(id)
    }
    selected.value = next
    return
  }
  if (!selected.value.has(card.id)) select(unit)
  gesture = {
    kind: 'move',
    startX: p.x,
    startY: p.y,
    ids: canWrite.value ? [...selected.value] : [],
    started: false,
    anchor: card.id,
    touchTap: e.pointerType === 'touch',
    cancelled: false,
  }
  begin(e)
}

function onResizeStart(card: Card, e: PointerEvent, handle: Handle = 'se') {
  if (!canWrite.value) return
  const p = local(e)
  const start = shownById.value.get(card.id) ?? card
  gesture = {
    kind: 'resize',
    startX: p.x,
    startY: p.y,
    id: card.id,
    cancelled: false,
    handle,
    rot: card.kind === 'shape' ? (card.content as ShapeContent).rotation || 0 : 0,
    start: { x: start.x, y: start.y, w: start.w, h: start.h },
  }
  gestureResizeId.value = card.id
  send({ t: 'grab', ids: [card.id] })
  begin(e)
}

// Jawaban grab: kartu yang sedang dipegang orang lain dikeluarkan dari seret; bila kartu yang dipegang sendiri
// ditolak, seret dibatalkan dan kartu kembali ke tempatnya.
function onGrabResult(granted: string[], denied: { id: string; user: Who }[]) {
  const g = gesture
  if (!denied?.length) return
  const deniedIds = new Set(denied.map((d) => d.id))
  const who = denied[0]!.user.name
  if (g?.kind === 'move') {
    if (deniedIds.has(g.anchor) || granted.length === 0) {
      g.cancelled = true
      g.ids = []
      dragOffset.value = { dx: 0, dy: 0 }
    } else g.ids = g.ids.filter((id) => !deniedIds.has(id))
  } else if (g?.kind === 'resize' && deniedIds.has(g.id)) {
    g.cancelled = true
    resizeDelta.value = { dx: 0, dy: 0, dw: 0, dh: 0 }
  } else if (g?.kind === 'rotate' && deniedIds.has(g.id)) {
    gesture = null
    rotateDraft.value = null
  }
  // Seret yang dibatalkan melepas kartu lain yang sempat diberikan.
  if (g?.kind === 'move' && g.cancelled && granted.length) send({ t: 'release', ids: granted })
  showToast(`${who} sedang memindahkan kartu ini.`)
}

function sendMoves(force = false) {
  const now = performance.now()
  if (!force && now - lastMoveSent < 40) return
  lastMoveSent = now
  const g = gesture
  if (g?.kind === 'move' && !g.cancelled) {
    const { dx, dy } = dragOffset.value
    const moves = g.ids.flatMap((id) => {
      const c = shownById.value.get(id)
      return c ? [{ id, x: c.x + dx, y: c.y + dy, w: c.w, h: c.h }] : []
    })
    if (moves.length) send({ t: 'move', moves })
  } else if (g?.kind === 'resize' && !g.cancelled) {
    const c = byId.value.get(g.id)
    if (c)
      send({
        t: 'move',
        moves: [
          {
            id: c.id,
            x: c.x + resizeDelta.value.dx,
            y: c.y + resizeDelta.value.dy,
            w: c.w + resizeDelta.value.dw,
            h: c.h + resizeDelta.value.dh,
          },
        ],
      })
  }
}

function onMove(e: PointerEvent) {
  if (pressStart && Math.hypot(e.clientX - pressStart.x, e.clientY - pressStart.y) > 8) cancelPress()
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
      const hit = shownCards.value.filter((c) => intersects(rect, c)).map((c) => c.id)
      selected.value = new Set([...gesture.base, ...expand(hit, cards.value, groupScope.value)])
      return
    }
    case 'move': {
      if (gesture.cancelled) return
      if (!gesture.started && Math.hypot(p.x - gesture.startX, p.y - gesture.startY) < 4) return
      if (!gesture.started) {
        gesture.started = true
        // Kartu terkunci tidak bisa digeser; seret yang memuatnya dibatalkan utuh supaya grup tidak terpecah.
        const locked = gesture.ids.filter((id) => byId.value.get(id)?.locked).length
        if (locked) {
          gesture.cancelled = true
          gesture.ids = []
          lockedToast(locked)
          return
        }
        // Grup bergerak utuh: grab semua atau tidak sama sekali (server menolak bila satu anggota dipegang orang lain).
        const all = gesture.ids.some((id) => byId.value.get(id)?.group_ids?.length)
        if (gesture.ids.length) send({ t: 'grab', ids: gesture.ids, all })
      }
      if (gesture.ids.length === 0) return
      const dx = (p.x - gesture.startX) / z
      const dy = (p.y - gesture.startY) / z
      // Snap memakai kartu yang dipegang sebagai jangkar; kartu lain ikut dengan jarak yang sama.
      const anchor = shownById.value.get(gesture.anchor)
      if (!anchor) return
      dragOffset.value = {
        dx: snap(anchor.x + dx, snapOn.value) - anchor.x,
        dy: snap(anchor.y + dy, snapOn.value) - anchor.y,
      }
      sendMoves()
      return
    }
    case 'resize': {
      if (gesture.cancelled) return
      const card = byId.value.get(gesture.id)
      if (!card) return
      if (gesture.handle === 'se' && card.kind !== 'shape') {
        const w = clamp(snap(card.w + (p.x - gesture.startX) / z, snapOn.value), MIN_SIZE, MAX_SIZE)
        const h = clamp(snap(card.h + (p.y - gesture.startY) / z, snapOn.value), MIN_SIZE, MAX_SIZE)
        resizeDelta.value = { dx: 0, dy: 0, dw: w - card.w, dh: h - card.h }
      } else {
        const s0 = gesture.start
        const d = { x: (p.x - gesture.startX) / z, y: (p.y - gesture.startY) / z }
        let r = resizeBox(s0, gesture.rot, gesture.handle, d, e.shiftKey, MIN_SIZE, MAX_SIZE)
        // Snap grid hanya untuk bentuk tegak: tepi yang bergerak ditarik ke grid, tepi seberangnya diam.
        if (gridOn.value && !gesture.rot && !e.shiftKey) {
          const right = s0.x + s0.w
          const bottom = s0.y + s0.h
          const x = gesture.handle.includes('w') ? snap(r.x, true) : r.x
          const y = gesture.handle.includes('n') ? snap(r.y, true) : r.y
          const w = gesture.handle.includes('w')
            ? right - x
            : gesture.handle.includes('e')
              ? snap(r.x + r.w, true) - x
              : r.w
          const h = gesture.handle.includes('n')
            ? bottom - y
            : gesture.handle.includes('s')
              ? snap(r.y + r.h, true) - y
              : r.h
          if (w >= MIN_SIZE && h >= MIN_SIZE) r = { x, y, w, h }
        }
        resizeDelta.value = { dx: r.x - s0.x, dy: r.y - s0.y, dw: r.w - s0.w, dh: r.h - s0.h }
      }
      sendMoves()
      return
    }
    case 'rotate': {
      rotateDraft.value = {
        id: gesture.id,
        deg: angleFrom(gesture.c, toWorld(view.value, p.x, p.y), e.shiftKey),
      }
      return
    }
    case 'connect': {
      if (connectDraft.value)
        connectDraft.value = { ...connectDraft.value, at: toWorld(view.value, p.x, p.y) }
      return
    }
    case 'bend': {
      const b = bendDraft.value
      if (!b) return
      const w = toWorld(view.value, p.x, p.y)
      const pts = [...b.points]
      pts[gesture.index] = [Math.round(snap(w.x, snapOn.value)), Math.round(snap(w.y, snapOn.value))]
      bendDraft.value = { id: b.id, points: pts }
      return
    }
  }
}

function onUp(e: PointerEvent) {
  cancelPress()
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
    // Ketukan pertama hanya memilih (juga kartu papan); dobel ketuk membuka editor atau papannya.
    select([g.tapCard])
    tapSelectAt = performance.now()
    lastTap = { id: g.tapCard, at: performance.now() }
    return
  }
  if (g.kind === 'move') {
    const { dx, dy } = dragOffset.value
    dragOffset.value = { dx: 0, dy: 0 }
    if (!g.started) {
      // Klik sekali hanya memilih. Dobel klik ditangani event dblclick di kartu; dobel ketuk di layar sentuh di sini.
      const card = byId.value.get(g.anchor)
      if (g.touchTap) {
        const double = lastTap?.id === g.anchor && performance.now() - lastTap.at < 350
        lastTap = double ? null : { id: g.anchor, at: performance.now() }
        if (double && card?.kind === 'board') openBoard(card)
        else if (double && card && !enterGroup(card))
          startEdit(g.anchor, caretHint(e.clientX, e.clientY, e.target))
      }
      return
    }
    if (g.ids.length) send({ t: 'release', ids: g.ids })
    if (g.cancelled || (dx === 0 && dy === 0)) return
    if (dropOn(g.ids, toWorld(view.value, local(e).x, local(e).y), dx, dy)) return
    const changes = g.ids.flatMap((id) => {
      const c = shownById.value.get(id)
      return c
        ? [
            {
              id,
              after: {
                x: clamp(c.x + dx, -COORD_LIMIT, COORD_LIMIT),
                y: clamp(c.y + dy, -COORD_LIMIT, COORD_LIMIT),
              },
            },
          ]
        : []
    })
    announce.value = `${changes.length} kartu dipindah.`
    applyLayout(changes)
    shiftBends(new Set(g.ids), dx, dy)
  }
  if (g.kind === 'resize') {
    const { dx, dy, dw, dh } = resizeDelta.value
    resizeDelta.value = { dx: 0, dy: 0, dw: 0, dh: 0 }
    gestureResizeId.value = null
    send({ t: 'release', ids: [g.id] })
    const c = byId.value.get(g.id)
    if (g.cancelled || !c || (dx === 0 && dy === 0 && dw === 0 && dh === 0)) return
    applyLayout([
      {
        id: g.id,
        after:
          dx || dy
            ? {
                x: Math.round(c.x + dx),
                y: Math.round(c.y + dy),
                w: Math.round(c.w + dw),
                h: Math.round(c.h + dh),
              }
            : { w: c.w + dw, h: c.h + dh },
      },
    ])
  }
  if (g.kind === 'rotate') {
    send({ t: 'release', ids: [g.id] })
    const r = rotateDraft.value
    const c = byId.value.get(g.id)
    if (r && c && c.kind === 'shape' && r.deg !== (c.content as ShapeContent).rotation) {
      onContent(c.id, { ...(c.content as ShapeContent), rotation: r.deg })
      announce.value = `Bentuk diputar ${r.deg} derajat.`
    }
    // Sudut sementara dilepas setelah isi lokal berubah, supaya bentuk tidak sempat kembali ke sudut lama.
    rotateDraft.value = null
  }
  if (g.kind === 'connect') void finishConnect(worldOf(e))
  if (g.kind === 'bend') {
    const b = bendDraft.value
    if (b) {
      void saveBends(b.id, b.points).finally(() => {
        if (bendDraft.value?.id === b.id) bendDraft.value = null
      })
      announce.value = 'Titik belok garis disimpan.'
    }
  }
}

// Lepas seret (W3): di atas kartu papan memindah kartu ke papan itu; di atas kolom memasukkan kartu ke kolom (atau
// mengubah urutannya); kartu yang keluar dari kolomnya ditaruh di titik lepas. Mengembalikan true bila ditangani.
function dropOn(ids: string[], at: { x: number; y: number }, dx: number, dy: number): boolean {
  const moving = new Set(ids)
  const hit = [...shownCards.value]
    .reverse()
    .find(
      (c) =>
        !moving.has(c.id) &&
        !(c.column_id && moving.has(c.column_id)) &&
        (c.kind === 'board' || c.kind === 'column') &&
        at.x >= c.x &&
        at.x <= c.x + c.w &&
        at.y >= c.y &&
        at.y <= c.y + c.h,
    )
  if (hit?.kind === 'board' && hit.ref) {
    moveTo(ids, hit.ref.id, hit.ref.title)
    return true
  }
  const list = ids.flatMap((id) => byId.value.get(id) ?? [])
  if (hit?.kind === 'column' && list.every((c) => c.kind !== 'column')) {
    const siblings = shownCards.value.filter((c) => c.column_id === hit.id && !moving.has(c.id))
    const base = columnOrderAt(siblings, at.y)
    sendGeometry(
      list.map((c, i) => ({ id: c.id, column_id: hit.id, col_order: base + i / (list.length + 1) })),
    )
    announce.value = `${list.length} kartu dimasukkan ke kolom.`
    return true
  }
  const leaving = list.filter((c) => c.column_id && !moving.has(c.column_id))
  if (!leaving.length) return false
  sendGeometry(
    list.map((c) => {
      const s = shownById.value.get(c.id) ?? c
      const p: CardPatch = {
        id: c.id,
        x: clamp(s.x + dx, -COORD_LIMIT, COORD_LIMIT),
        y: clamp(s.y + dy, -COORD_LIMIT, COORD_LIMIT),
      }
      if (c.column_id && !moving.has(c.column_id)) {
        p.column_id = ''
        p.w = s.w
      }
      return p
    }),
  )
  announce.value = `${leaving.length} kartu dikeluarkan dari kolom.`
  return true
}

// Ketukan yang memilih kartu memunculkan bar pilihan di bawah jari; event click dari ketukan yang sama lalu jatuh
// ke tombol bar yang baru muncul. Klik di bar diabaikan sebentar setelah ketukan pilih.
// Dobel klik membuka editor dengan kursor di posisi klik (kartu papan: membuka papannya). Ditangani di kanvas karena
// pointer capture saat klik mengalihkan dblclick ke elemen kanvas; kartu dicari dari titik klik. Area kosong: tidak
// ada apa-apa, supaya tidak ada kartu tak sengaja.
function onCanvasDblClick(e: MouseEvent) {
  if (penOn.value) return
  const el = document.elementFromPoint(e.clientX, e.clientY)
  const id = el?.closest<HTMLElement>('[data-card]')?.dataset.card
  const card = id ? byId.value.get(id) : undefined
  if (!card || editingId.value === card.id) return
  if (card.kind === 'board') openBoard(card)
  else if (!enterGroup(card)) startEdit(card.id, caretHint(e.clientX, e.clientY, el))
}

let tapSelectAt = 0
let lastTap: { id: string; at: number } | null = null
function guardTap(e: MouseEvent) {
  if (performance.now() - tapSelectAt < 500) {
    e.preventDefault()
    e.stopPropagation()
  }
}

// Roda di kanvas: Ctrl atau Cmd (juga pinch touchpad, yang dikirim browser sebagai wheel dengan ctrlKey) zoom di
// sekitar kursor; tanpa itu menggeser kanvas, Shift ke samping. Listener native { passive: false } supaya
// preventDefault menahan zoom halaman browser. Event digabung per frame lewat requestAnimationFrame, tanpa transisi.
let wheelFrame = 0
let wheelZoom = 1
let wheelPan = { dx: 0, dy: 0 }
let wheelAt = { x: 0, y: 0 }

// Wheel biasa di dalam editor atau bagian yang bisa digulir (mis. tabel lebar) menggulir bagian itu, bukan kanvas.
function scrollsInside(target: EventTarget | null, dx: number, dy: number) {
  for (let el = target as HTMLElement | null; el && el !== canvas.value; el = el.parentElement) {
    if (el.isContentEditable) return true
    const s = getComputedStyle(el)
    const canY = /(auto|scroll)/.test(s.overflowY) && el.scrollHeight > el.clientHeight
    const canX = /(auto|scroll)/.test(s.overflowX) && el.scrollWidth > el.clientWidth
    if ((canY && dy !== 0) || (canX && dx !== 0)) return true
  }
  return false
}

function onWheel(e: WheelEvent) {
  markActive()
  const zoomKey = e.ctrlKey || e.metaKey
  let { dx, dy } = wheelPixels(e, size.value.h || 800)
  if (!zoomKey && e.shiftKey && dx === 0) [dx, dy] = [dy, 0]
  if (!zoomKey && scrollsInside(e.target, dx, dy)) return
  e.preventDefault()
  const r = canvas.value!.getBoundingClientRect()
  if (zoomKey) {
    wheelZoom *= wheelZoomFactor(dy)
    wheelAt = { x: e.clientX - r.left, y: e.clientY - r.top }
  } else {
    wheelPan = { dx: wheelPan.dx + dx, dy: wheelPan.dy + dy }
  }
  if (!wheelFrame) wheelFrame = requestAnimationFrame(applyWheel)
}

function applyWheel() {
  wheelFrame = 0
  let v = view.value
  if (wheelZoom !== 1) v = zoomAt(v, wheelAt.x, wheelAt.y, v.zoom * wheelZoom)
  if (wheelPan.dx || wheelPan.dy) v = { ...v, x: v.x - wheelPan.dx, y: v.y - wheelPan.dy }
  wheelZoom = 1
  wheelPan = { dx: 0, dy: 0 }
  view.value = v
}

watch(canvas, (el, old) => {
  old?.removeEventListener('wheel', onWheel)
  el?.addEventListener('wheel', onWheel, { passive: false })
})

function resetZoom() {
  zoomBy(1 / view.value.zoom)
  announce.value = 'Zoom 100%.'
}

// ---------------------------------------------------------------------------------------------------------------
// Keyboard

let nudgeTimer: ReturnType<typeof setTimeout> | undefined
const nudgeStart = new Map<string, Geo>()

function nudge(dx: number, dy: number) {
  if (anyLocked.value) {
    lockedToast()
    return
  }
  for (const id of selected.value) {
    const c = byId.value.get(id)
    if (!c || c.column_id) continue
    if (!nudgeStart.has(id)) nudgeStart.set(id, { x: c.x, y: c.y })
    patchLocal(id, {
      x: clamp(c.x + dx, -COORD_LIMIT, COORD_LIMIT),
      y: clamp(c.y + dy, -COORD_LIMIT, COORD_LIMIT),
    })
  }
  announce.value = `${selected.value.size} kartu digeser.`
  clearTimeout(nudgeTimer)
  nudgeTimer = setTimeout(() => {
    const entries = [...nudgeStart].flatMap(([id, before]) => {
      const c = byId.value.get(id)
      return c ? [{ id, before, after: { x: c.x, y: c.y } }] : []
    })
    nudgeStart.clear()
    if (entries.length) record({ kind: 'layout', cards: entries })
    sendGeometry(entries.map((e) => ({ id: e.id, x: e.after.x, y: e.after.y })))
  }, 400)
}

// Shift dan panah atas atau bawah pada kartu di kolom menukar urutannya dengan tetangga (jalur keyboard untuk seret
// di dalam kolom).
function reorderInColumn(c: Card, dir: 'up' | 'down') {
  const sib = shownCards.value.filter((s) => s.column_id === c.column_id).sort((a, b) => a.y - b.y)
  const i = sib.findIndex((s) => s.id === c.id)
  const other = sib[dir === 'up' ? i - 1 : i + 1]
  if (i < 0 || !other) return
  const a = c.col_order ?? 0
  const b = other.col_order ?? 0
  const patches: CardPatch[] =
    a === b
      ? [{ id: c.id, col_order: dir === 'up' ? b - 1 : b + 1 }]
      : [
          { id: c.id, col_order: b },
          { id: other.id, col_order: a },
        ]
  for (const p of patches) patchLocal(p.id, { col_order: p.col_order })
  sendGeometry(patches)
  announce.value = dir === 'up' ? 'Kartu naik satu di kolom.' : 'Kartu turun satu di kolom.'
}

const arrows: Record<string, Direction> = {
  ArrowUp: 'up',
  ArrowDown: 'down',
  ArrowLeft: 'left',
  ArrowRight: 'right',
}

function onKeydown(e: KeyboardEvent) {
  markActive()
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
  if (onConnectorKey(e)) return
  // Menu klik kanan lewat keyboard: Shift+F10 atau tombol Menu.
  if (e.key === 'ContextMenu' || (e.shiftKey && e.key === 'F10')) {
    e.preventDefault()
    openMenuFromKeyboard()
    return
  }
  const sc = findShortcut(e, KEY_ACTIONS)
  if (sc && actions[sc]) {
    e.preventDefault()
    actions[sc]!()
    return
  }
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
      const only = selected.value.size === 1 ? byId.value.get([...selected.value][0]!) : undefined
      if (only?.column_id && (dir === 'up' || dir === 'down')) {
        reorderInColumn(only, dir)
        return
      }
      const step = mod ? GRID * 8 : GRID
      nudge(
        dir === 'left' ? -step : dir === 'right' ? step : 0,
        dir === 'up' ? -step : dir === 'down' ? step : 0,
      )
      return
    }
    const from = focusedId.value ? shownById.value.get(focusedId.value) : undefined
    const next = from ? nearestInDirection(from, shownCards.value, dir) : ordered.value[0]
    if (next) {
      select([next.id])
      focusCard(next.id)
    }
    return
  }
  if (mod) {
    const k = e.key.toLowerCase()
    if (k === 'a') {
      e.preventDefault()
      select(cards.value.map((c) => c.id))
      announce.value = `${cards.value.length} kartu terpilih.`
    } else if (k === 'z' && !e.shiftKey && canWrite.value) {
      e.preventDefault()
      runStep('undo')
    } else if (((k === 'z' && e.shiftKey) || k === 'y') && canWrite.value) {
      e.preventDefault()
      runStep('redo')
    } else if (k === 'c') {
      e.preventDefault()
      copySelection()
    } else if (k === 'd' && canWrite.value) {
      e.preventDefault()
      duplicateSelection()
    } else if (k === '0') {
      e.preventDefault()
      fit()
    } else if (k === '=' || k === '+') {
      e.preventDefault()
      zoomBy(1.25)
    } else if (k === '-') {
      e.preventDefault()
      zoomBy(0.8)
    }
    // Ctrl+V ditangani event paste, supaya isi clipboard sistem bisa dibaca.
    return
  }
  if (e.altKey) return
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
      // Di dalam grup: Escape keluar satu tingkat dan memilih grup itu utuh.
      if (groupScope.value.length) {
        e.preventDefault()
        exitGroup()
        return
      }
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
      toggleGrid()
  }
}

function onKeyup(e: KeyboardEvent) {
  if (e.key === ' ') spaceHeld.value = false
}

// ---------------------------------------------------------------------------------------------------------------
// Riwayat

const historyOpen = ref(false)
const activity = shallowRef<Activity[]>([])
const historyState = ref<'loading' | 'error' | 'ok'>('loading')
const historyError = ref('')

async function openHistory() {
  historyOpen.value = true
  historyState.value = 'loading'
  try {
    activity.value = (await boardActivity(bid.value)).activity
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
    if (editingId.value) send({ t: 'unlock', card: editingId.value })
    selected.value = new Set()
    editingId.value = null
    focusedId.value = null
    titleEditing.value = false
    searchOpen.value = false
    remoteLocks.value = {}
    remoteGrabs.value = {}
    remoteSel.value = {}
    remoteMoves.value = new Map()
    cursors.value = {}
    connectors.value = []
    connectorIds.value = null
    commentsFor.value = null
    moveOpen.value = false
    history.undo.length = 0
    history.redo.length = 0
    historyVersion.value++
    closeSearch()
    load()
  },
  { immediate: true },
)

function onVisibility() {
  checkIdle()
}
onMounted(() => document.addEventListener('visibilitychange', onVisibility))

onBeforeUnmount(() => {
  stopPen()
  window.removeEventListener('keydown', onPenEscape)
  flushNotes()
  if (editingId.value) send({ t: 'unlock', card: editingId.value })
  socket?.stop()
  abort?.abort()
  canvas.value?.removeEventListener('wheel', onWheel)
  cancelAnimationFrame(wheelFrame)
  ro?.disconnect()
  clearTimeout(toastTimer)
  clearInterval(sweepTimer)
  clearInterval(lockBeat)
  endToolDrag()
  document.removeEventListener('pointerdown', onDocPointer)
  document.removeEventListener('paste', onPaste)
  document.removeEventListener('visibilitychange', onVisibility)
  window.removeEventListener('online', onOnline)
  window.removeEventListener('beforeunload', onBeforeUnload)
  window.removeEventListener('pointermove', onMove)
  window.removeEventListener('pointerup', onUp)
  window.removeEventListener('pointercancel', onUp)
})

// Tombol teks bar pilihan, sama dengan toolbar pena (WbPenBar).
const barBtn =
  'inline-flex min-h-11 shrink-0 items-center gap-1 justify-center rounded-card px-3 text-sm font-medium hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark aria-expanded:bg-canvas disabled:opacity-50'
const btn =
  'inline-flex min-h-11 items-center justify-center gap-1.5 rounded-card border border-line bg-surface px-3 font-medium hover:bg-canvas active:translate-y-px disabled:cursor-not-allowed disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
const iconBtn =
  'inline-flex size-11 shrink-0 items-center justify-center rounded-card border border-line bg-surface hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark aria-pressed:bg-accent aria-pressed:text-on-accent disabled:opacity-50'
// Tombol toggle bergaya sama dengan btn; keadaan tertekan memakai warna accent seperti iconBtn.
const pressed = 'aria-pressed:bg-accent aria-pressed:text-on-accent aria-pressed:hover:bg-accent-hover'
const toolBtn =
  'flex min-h-14 w-full flex-col items-center justify-center gap-0.5 rounded-card px-1 text-xs text-ink hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-mark touch-none'
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <!-- Bar atas: lokasi, judul, cari, status simpan, riwayat. -->
    <div class="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-line bg-surface px-3 py-1.5">
      <nav aria-label="Lokasi papan" class="hidden min-w-0 max-xl:order-first md:block">
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
        class="max-xl:order-first md:hidden"
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

      <!-- Di bawah 1280 px lokasi dan judul mengisi baris pertama, kontrol di baris kedua (pemisah basis-full di bawah),
           supaya judul selalu terbaca dan tinggi bar tidak berubah saat penanda simpan berganti teks. -->
      <div class="flex min-w-0 flex-1 items-center gap-1 max-xl:order-first">
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

      <div aria-hidden="true" class="hidden basis-full max-xl:order-first max-xl:block"></div>

      <!-- Kehadiran (W2): orang lain di papan ini, warna tetap per orang, redup saat diam. Klik mengikuti kursornya. -->
      <ul v-if="others.length" aria-label="Orang lain di papan ini" class="flex items-center">
        <li v-for="p in others.slice(0, 5)" :key="p.id">
          <button
            type="button"
            class="inline-flex size-11 items-center justify-center rounded-full focus-visible:outline-2 focus-visible:outline-mark"
            :class="p.idle ? 'opacity-50' : ''"
            :data-collab="p.color"
            :aria-label="`${p.name}${p.editing ? ', sedang mengedit' : ''}${p.idle ? ', diam' : ''}${p.tabs > 1 ? `, ${p.tabs} tab` : ''}. Ikuti kursornya`"
            :title="`${p.name}${p.idle ? ' (diam)' : ''}`"
            data-testid="wb-presence"
            @click="follow(p)"
          >
            <span class="wb-presence-ring inline-flex rounded-full">
              <UserAvatar :user-id="p.id" :name="p.name" :avatar-id="p.avatar_id" size="sm" decorative />
            </span>
          </button>
        </li>
        <li v-if="others.length > 5" class="px-1 text-sm text-ink-muted">+{{ others.length - 5 }}</li>
      </ul>

      <div ref="searchBox" class="relative flex items-center gap-2" @keydown="onSearchKey">
        <button
          type="button"
          :class="iconBtn"
          class="md:hidden"
          :aria-expanded="searchOpen"
          :aria-controls="`${uid}-search`"
          aria-label="Cari papan dan isi kartu"
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
          <label :for="`${uid}-q`" class="sr-only">Cari papan dan isi kartu</label>
          <input
            :id="`${uid}-q`"
            v-model="query"
            type="search"
            enterkeyhint="search"
            maxlength="100"
            placeholder="Cari papan dan kartu"
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
            <p
              v-else-if="results && results.length === 0 && cardHits.length === 0"
              role="status"
              class="p-3 text-sm text-ink-muted"
            >
              Tidak ada papan atau kartu yang cocok.
            </p>
            <template v-else-if="results">
              <p v-if="results.length" :id="`${uid}-qb`" class="px-3 pt-2 text-sm font-semibold">Papan</p>
              <ul v-if="results.length" :aria-labelledby="`${uid}-qb`">
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
              <p v-if="cardHits.length" :id="`${uid}-qc`" class="px-3 pt-2 text-sm font-semibold">Kartu</p>
              <ul v-if="cardHits.length" :aria-labelledby="`${uid}-qc`" data-testid="wb-search-cards">
                <li v-for="h in cardHits" :key="h.id">
                  <RouterLink
                    :to="{ name: 'whiteboard-board', params: { id: h.board_id }, query: { kartu: h.id } }"
                    class="flex min-h-11 flex-col justify-center px-3 py-2 hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-mark"
                    @click="closeSearch"
                  >
                    <span class="break-words [overflow-wrap:anywhere]">{{ h.snippet }}</span>
                    <span class="text-sm text-ink-muted">{{ kindLabel[h.kind] }} di {{ h.board }}</span>
                  </RouterLink>
                </li>
              </ul>
            </template>
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
      <!-- Di ponsel Riwayat dan Daftar hanya ikon: bar atas 360 px yang penuh pindah ke dua baris saat penanda
           simpan berubah teks, dan kanvas yang bergeser di tengah klik membuat klik mendarat di elemen lain. -->
      <button
        type="button"
        :class="btn"
        class="max-md:w-11 max-md:px-0"
        title="Riwayat papan"
        @click="openHistory"
      >
        <svg
          aria-hidden="true"
          viewBox="0 0 24 24"
          class="size-5 md:hidden"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
        >
          <path d="M12 7v5l3 2M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18" />
        </svg>
        <span class="max-md:sr-only">Riwayat</span>
      </button>
      <!-- W4: Tampilan Daftar (jalur pembaca layar dan ponsel kecil), ekspor, dan tanda template. -->
      <button
        type="button"
        :class="[btn, pressed]"
        :aria-pressed="listMode ? 'true' : 'false'"
        title="Tampilkan kartu sebagai daftar"
        class="max-md:w-11 max-md:px-0"
        @click="listMode = !listMode"
      >
        <svg
          aria-hidden="true"
          viewBox="0 0 24 24"
          class="size-5 md:hidden"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
        >
          <path d="M9 6h11M9 12h11M9 18h11M4 6h.01M4 12h.01M4 18h.01" />
        </svg>
        <span class="max-md:sr-only">Daftar</span>
      </button>
      <button v-if="state === 'ok'" type="button" :class="btn" class="max-md:hidden" @click="openExport">
        Ekspor
      </button>
      <button
        v-if="canWrite && board && !board.is_root"
        type="button"
        :class="[btn, pressed]"
        class="max-md:hidden"
        :aria-pressed="board.is_template ? 'true' : 'false'"
        title="Tandai papan ini sebagai template yang bisa disalin dari papan lain"
        @click="toggleTemplate"
      >
        Jadikan template
      </button>
      <span
        v-else-if="board?.is_template"
        class="inline-flex min-h-11 items-center rounded-chip border border-line px-2 text-sm"
      >
        Template
      </span>
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
          v-show="state === 'ok' && !listMode"
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
          @pointermove="onCanvasPointerMove"
          @keydown="onKeydown"
          @keyup="onKeyup"
          @dblclick="onCanvasDblClick"
          @contextmenu="onContextMenu"
          @dragover="onDragOver"
          @dragleave="fileOver = false"
          @drop="onDrop"
        >
          <div class="wb-world absolute top-0 left-0 origin-top-left" :style="worldStyle">
            <WbConnectorLayer
              :lines="lines"
              :selected="selectedConnector"
              :zoom="view.zoom"
              @select="onConnectorSelect"
              @open="connectorIds = [$event]"
            />
            <!-- Bingkai grup terpilih (revisi 50): putus-putus di sekeliling seluruh anggotanya. -->
            <div
              v-for="(f, i) in groupFrames"
              :key="`grup-${i}`"
              aria-hidden="true"
              class="wb-group-frame pointer-events-none absolute top-0 left-0"
              :style="{
                transform: `translate(${f.x}px, ${f.y}px)`,
                width: `${f.w}px`,
                height: `${f.h}px`,
                '--wb-ring': `${2 / view.zoom}px`,
              }"
              data-testid="wb-group-frame"
            />
            <template v-for="c in rendered" :key="c.id">
              <WbInk
                v-if="c.kind === 'drawing'"
                :ref="(inst) => setCardRef(c.id, inst)"
                :card="c"
                :selected="selected.has(c.id)"
                :focusable="rovingId === c.id"
                :zoom="view.zoom"
                :dx="dragged(c) ? dragOffset.dx : 0"
                :dy="dragged(c) ? dragOffset.dy : 0"
                :pen="penOn"
                :remote="remoteFor(c.id)"
                @focus="focusedId = c.id"
                @grab="onCardGrab(c, $event)"
              />
              <WbCard
                v-else
                :ref="(inst) => setCardRef(c.id, inst)"
                :card="c"
                :selected="selected.has(c.id)"
                :editing="editingId === c.id"
                :can-write="canWrite"
                :focusable="rovingId === c.id"
                :zoom="view.zoom"
                :dx="(dragged(c) ? dragOffset.dx : 0) + (gestureResizeId === c.id ? resizeDelta.dx : 0)"
                :dy="(dragged(c) ? dragOffset.dy : 0) + (gestureResizeId === c.id ? resizeDelta.dy : 0)"
                :dw="gestureResizeId === c.id ? resizeDelta.dw : 0"
                :dh="gestureResizeId === c.id ? resizeDelta.dh : 0"
                :rotation="rotateDraft?.id === c.id ? rotateDraft.deg : null"
                :remote="remoteFor(c.id)"
                :caret="editHint?.id === c.id ? editHint : null"
                :readonly-by="readonlyFor?.id === c.id ? readonlyFor.name : null"
                @focus="focusedId = c.id"
                @grab="onCardGrab(c, $event)"
                @resize="(e: PointerEvent, h?: Handle) => onResizeStart(c, e, h)"
                @rotate="onRotateStart(c, $event)"
                @live="sendDraft(c.id, $event)"
                @close="stopEdit"
                @note="onNoteInput(c.id, $event)"
                @heading="onHeading(c.id, $event)"
                @draft="onHeadingDraft(c.id, $event)"
                @rename="onRenameRef(c.id, $event)"
                @checklist="onChecklist(c.id, $event)"
                @table="onTable(c.id, $event)"
                @content="onContent(c.id, $event)"
                @comments="commentsFor = c.id"
              />
            </template>
            <WbDiagramHandles
              v-if="portHandles.length || selectedStops || tempLine"
              :zoom="view.zoom"
              :ports="portHandles"
              :stops="selectedStops"
              :temp="tempLine"
              @port="onPortDown"
              @bend="(i: number, e: PointerEvent) => onBendDown(i, false, e)"
              @insert="(i: number, e: PointerEvent) => onBendDown(i, true, e)"
              @unbend="onUnbend"
            />
            <!-- Goresan yang sedang digambar (sendiri dan orang lain) dan yang belum dijawab server. -->
            <svg
              v-if="inkLayer.length"
              aria-hidden="true"
              class="pointer-events-none absolute top-0 left-0 z-[1000000002] overflow-visible"
              data-testid="wb-ink-live"
              width="1"
              height="1"
            >
              <path
                v-for="(k, i) in inkLayer"
                :key="i"
                :d="inkPath(k.points)"
                fill="none"
                :stroke-width="k.width"
                stroke-linecap="round"
                stroke-linejoin="round"
                :style="{ stroke: inkColor(k.color) }"
              />
            </svg>
          </div>
          <!-- Mode pena: lapisan penangkap penunjuk di atas semua kartu. -->
          <div
            v-if="penOn"
            class="absolute inset-0 z-[15] touch-none"
            :class="penEraser ? 'cursor-cell' : 'cursor-crosshair'"
            data-testid="wb-pen-surface"
            @pointerdown="onPenDown"
            @pointermove="onPenMove"
            @pointerup="onPenUp"
            @pointercancel="onPenUp"
          />
          <div
            v-for="cur in cursorList"
            :key="cur.key"
            aria-hidden="true"
            class="pointer-events-none absolute top-0 left-0 z-10 flex items-start"
            :style="cur.style"
            :data-collab="cur.color"
            data-testid="wb-cursor"
          >
            <svg viewBox="0 0 16 16" class="wb-cursor-arrow size-4"><path d="M1 1l5 14 2-6 6-2z" /></svg>
            <!-- Di dekat tepi kanan label pindah ke kiri panah, di dekat tepi bawah naik ke atasnya, supaya tidak
                 terpotong kanvas. -->
            <span
              class="wb-remote-label absolute rounded-chip px-1.5 py-0.5 text-xs font-medium whitespace-nowrap"
              :class="[cur.flipX ? 'right-5' : 'left-5', cur.flipY ? 'bottom-1' : 'top-0']"
            >
              {{ cur.name }}
            </span>
          </div>
          <div
            v-if="box"
            aria-hidden="true"
            class="pointer-events-none absolute border border-dashed border-mark bg-mark/10"
            :style="{ left: `${box.x}px`, top: `${box.y}px`, width: `${box.w}px`, height: `${box.h}px` }"
          />
          <div
            v-if="fileOver"
            aria-hidden="true"
            class="pointer-events-none absolute inset-2 rounded-card border-2 border-dashed border-mark"
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

        <WbPenBar
          v-if="state === 'ok' && penOn && !listMode"
          :color="penColor"
          :width="penWidth"
          :eraser="penEraser"
          :can-undo="!!penObject"
          @color="penColor = $event"
          @width="penWidth = $event"
          @eraser="penEraser = $event"
          @undo="undoStroke"
          @done="stopPen"
        />
        <!-- Bar pilihan (revisi 52): palet warna, Aksi (menu yang sama dengan klik kanan), Hapus. Panel format menempel di
             kanannya (slot wb-format-slot) saat mengedit teks, dan turun ke baris kedua bila tidak muat. -->
        <div
          v-if="state === 'ok' && canWrite && selected.size > 0 && !penOn && !listMode"
          class="pointer-events-none absolute top-2 right-2 left-2 z-20 flex flex-wrap items-start gap-2"
        >
          <div
            class="pointer-events-auto flex max-w-full flex-wrap items-center gap-0.5 rounded-card border border-line bg-surface p-1 shadow-menu"
            data-testid="wb-selbar"
            @click.capture="guardTap"
          >
            <WbPicker
              v-if="onlyDrawings"
              label="Warna coretan terpilih"
              compact
              class="border-0"
              :options="penOptions"
              :model-value="null"
              @update:model-value="recolorDrawings"
            />
            <WbPicker
              v-else
              label="Warna kartu terpilih"
              compact
              class="border-0"
              :options="cardColorOptions"
              :model-value="selectionColor === 'campuran' ? null : selectionColor"
              :mixed="selectionColor === 'campuran'"
              :extras="colorExtras"
              @update:model-value="pickCardColor"
            />
            <span aria-hidden="true" class="mx-0.5 h-6 w-px bg-line" />
            <button
              v-if="selected.size === 1 && byId.get([...selected][0]!)?.kind !== 'link'"
              type="button"
              :class="barBtn"
              @click="startEdit([...selected][0]!)"
            >
              {{ byId.get([...selected][0]!)?.kind === 'board' ? 'Ubah judul' : 'Ubah isi' }}
            </button>
            <button
              type="button"
              :class="barBtn"
              aria-haspopup="menu"
              :aria-expanded="menu?.opener?.dataset.menuButton === 'bar' ? 'true' : 'false'"
              data-menu-button="bar"
              @click="openBarMenu"
            >
              Aksi
              <svg
                aria-hidden="true"
                viewBox="0 0 16 16"
                class="size-4"
                fill="none"
                stroke="currentColor"
                stroke-width="1.75"
              >
                <path d="M4 6l4 4 4-4" />
              </svg>
            </button>
            <button type="button" :class="barBtn" class="text-urgent" @click="removeSelected">
              Hapus {{ selected.size }} kartu
            </button>
          </div>
          <div id="wb-format-slot" class="pointer-events-auto max-w-full empty:hidden" />
        </div>

        <!-- Terputus (W2): penanda tetap, bukan toast; perubahan menunggu di antrean sampai tersambung lagi. -->
        <p
          v-if="state === 'ok' && (sockState !== 'open' || netDown)"
          class="absolute top-2 left-1/2 z-30 max-w-[calc(100%-1rem)] -translate-x-1/2 rounded-card border border-line bg-surface px-3 py-2 text-sm shadow-menu"
          role="status"
          data-testid="wb-offline"
        >
          Terputus, menyambung ulang. Perubahan disimpan saat tersambung lagi.
        </p>

        <!-- Minimap (mulai 768 px): dekoratif untuk pembaca layar; navigasi aksesibel lewat panah dan Pas ke layar. -->
        <svg
          v-if="state === 'ok' && minimap && !listMode"
          aria-hidden="true"
          :width="MINI_W"
          :height="MINI_H"
          class="absolute right-2 bottom-16 z-20 hidden cursor-pointer rounded-card border border-line bg-surface md:block"
          @click="onMinimapClick"
        >
          <rect
            v-for="r in minimap.rects"
            :key="r.id"
            :x="r.x"
            :y="r.y"
            :width="r.w"
            :height="r.h"
            class="wb-mini-card"
          />
          <rect
            :x="minimap.screen.x"
            :y="minimap.screen.y"
            :width="minimap.screen.w"
            :height="minimap.screen.h"
            class="wb-mini-screen"
          />
        </svg>

        <WbListView
          v-if="state === 'ok' && listMode"
          class="absolute inset-0"
          :cards="shownCards"
          :can-write="canWrite"
          :board-title="board?.title ?? ''"
          @show="listShow"
          @edit="listEdit"
          @open="listOpen"
          @move="listMove"
          @remove="listRemove"
          @checklist="onChecklist"
        />

        <!-- Kontrol tampilan di kanan bawah; tombol Tambah di kiri bawah untuk ponsel. -->
        <div
          v-if="state === 'ok' && !listMode"
          class="absolute right-2 bottom-2 z-20 flex flex-wrap items-center justify-end gap-1 rounded-card border border-line bg-surface p-1 shadow-menu max-md:max-w-[calc(100%-9rem)]"
        >
          <template v-if="canWrite">
            <button
              type="button"
              :class="iconBtn"
              class="border-transparent"
              aria-label="Urungkan langkah"
              title="Urungkan (Ctrl Z)"
              :disabled="!canUndo"
              @click="runStep('undo')"
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
                <path d="M9 14L4 9l5-5M4 9h10a6 6 0 0 1 0 12h-3" />
              </svg>
            </button>
            <button
              type="button"
              :class="iconBtn"
              class="border-transparent"
              aria-label="Ulangi langkah"
              title="Ulangi (Ctrl Shift Z)"
              :disabled="!canRedo"
              @click="runStep('redo')"
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
                <path d="M15 14l5-5-5-5M20 9H10a6 6 0 0 0 0 12h3" />
              </svg>
            </button>
          </template>
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
          <button
            type="button"
            class="inline-flex min-h-11 min-w-14 items-center justify-center rounded-card px-1 text-sm tabular-nums hover:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark"
            :aria-label="`Zoom ${zoomPercent}%, kembali ke 100%`"
            title="Kembali ke 100%"
            data-testid="wb-zoom"
            @click="resetZoom"
          >
            {{ zoomPercent }}%
          </button>
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
          v-if="state === 'ok' && !listMode"
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

        <p
          v-if="uploadStatus"
          role="status"
          class="absolute top-2 right-2 z-30 max-w-[calc(100%-1rem)] rounded-card border border-line bg-surface px-3 py-2 text-sm shadow-menu"
          data-testid="wb-upload"
        >
          {{ uploadStatus }}
        </p>
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
            <RouterLink
              v-if="toast.board"
              :to="{ name: 'whiteboard-board', params: { id: toast.board } }"
              :class="btn"
              @click="toast = null"
            >
              Buka papan
            </RouterLink>
            <button type="button" :class="btn" @click="toast = null">Tutup</button>
          </div>
        </div>
      </div>
    </div>

    <p :id="`${uid}-help`" class="sr-only">
      Panah: pindah antar kartu. Shift dan panah: geser kartu terpilih. Enter: ubah isi atau buka papan.
      Delete: hapus. Ctrl A: pilih semua. Ctrl Z dan Ctrl Shift Z: urungkan dan ulangi. Ctrl C, Ctrl V, Ctrl
      D: salin, tempel, duplikat. N: catatan baru. Ctrl 0: pas ke layar. Tahan Spasi untuk menggeser kanvas.
      Di kolom, Shift dan panah atas atau bawah mengubah urutan kartu. Tempel gambar atau berkas dengan Ctrl
      V. Garis, komentar, dan pindah papan ada di bar kartu terpilih.
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
        <button type="button" :class="btn" class="col-span-2 min-h-14" @click="openExport">Ekspor</button>
        <button
          v-if="canWrite && board && !board.is_root"
          type="button"
          :class="[btn, pressed]"
          class="col-span-2 min-h-14"
          :aria-pressed="board.is_template ? 'true' : 'false'"
          @click="toggleTemplate"
        >
          Jadikan template
        </button>
        <button type="button" :class="btn" class="col-span-2 min-h-14" @click="openTrash">
          Tempat sampah
        </button>
      </div>
    </BaseDialog>

    <WbTrashPanel :open="trashOpen" @close="trashOpen = false" @restored="onRestored" />
    <WbShapePalette :open="shapePaletteOpen" @close="shapePaletteOpen = false" @pick="addShape" />
    <WbShapeStyle
      :open="shapeStyleCard !== null"
      :content="shapeStyleCard?.kind === 'shape' ? (shapeStyleCard.content as ShapeContent) : null"
      :w="shapeStyleCard?.w ?? 0"
      :h="shapeStyleCard?.h ?? 0"
      :can-write="canWrite"
      @close="shapeStyleId = null"
      @content="onShapeStyle"
      @size="onShapeSize"
    />
    <WbTemplateDialog
      :open="templateOpen"
      :current="bid"
      :busy="templateBusy"
      :error="templateError"
      @close="templateOpen = false"
      @pick="placeTemplate"
    />
    <WbContextMenu
      v-if="menu"
      :items="menu.items"
      :x="menu.x"
      :y="menu.y"
      :label="menu.label"
      @close="closeMenu"
    />
    <WbStylePanel
      :open="stylePanelOpen"
      :cards="selCards"
      :can-write="canWrite"
      @close="stylePanelOpen = false"
      @apply="applyStyle"
      @person="setPersonDisplay"
    />
    <WbTeamDialog :open="teamOpen" @close="teamOpen = false" @add="addPeople" />
    <WbImageDialog
      :mode="imageDialog?.mode ?? null"
      :card="imageCard"
      @close="imageDialog = null"
      @save="saveCrop"
    />
    <input
      ref="replaceInput"
      type="file"
      class="sr-only"
      tabindex="-1"
      aria-hidden="true"
      :accept="acceptFiles"
      @change="onReplacePicked"
    />
    <WbExportDialog
      :open="exportOpen"
      :title="board?.title ?? 'papan'"
      :cards="shownCards"
      :connectors="connectors"
      :selected="selected"
      @close="exportOpen = false"
    />

    <input
      ref="fileInput"
      type="file"
      multiple
      :accept="acceptFiles"
      class="sr-only"
      tabindex="-1"
      aria-hidden="true"
      @change="onFilePicked"
    />
    <WbLinkDialog
      :open="linkOpen"
      :busy="linkBusy"
      :error="linkError"
      @close="linkOpen = false"
      @submit="submitLink"
    />
    <WbConnectorDialog
      :open="connectorIds !== null"
      :connectors="dialogConnectors"
      :name-of="cardName"
      :is-shape="(id: string) => byId.get(id)?.kind === 'shape'"
      :can-write="canWrite"
      @close="connectorIds = null"
      @update="onConnectorUpdate"
      @remove="onConnectorRemove"
    />
    <WbMoveDialog
      :open="moveOpen"
      :current="bid"
      :moving="movingRefs"
      :busy="moveBusy"
      :error="moveError"
      @close="moveOpen = false"
      @move="moveFromDialog"
    />
    <WbCommentsPanel
      :open="commentsFor !== null"
      :board="bid"
      :card="commentsCard"
      :can-write="canWrite"
      :reload="commentsReload"
      @close="commentsFor = null"
      @card="upsert([$event])"
    />

    <BaseDialog
      :open="tabPrompt !== null"
      title="Kartu ini sedang kamu edit di tab lain"
      @close="tabPrompt = null"
    >
      <div class="flex flex-col gap-4 px-5 py-4">
        <p>Teksnya ikut berubah di sini selama kamu mengetik di tab lain. Pindahkan editornya ke tab ini?</p>
        <div class="flex flex-wrap justify-end gap-2">
          <button type="button" :class="btn" @click="tabPrompt = null">Batal</button>
          <button type="button" :class="btn" @click="editHere">Edit di sini</button>
        </div>
      </div>
    </BaseDialog>

    <BaseDialog
      :open="deletedWhileEditing !== null"
      title="Kartu ini dihapus"
      @close="deletedWhileEditing = null"
    >
      <div class="flex flex-col gap-4 px-5 py-4">
        <p>
          Kartu ini dipindah ke Tempat sampah oleh {{ deletedWhileEditing?.by }}. Ketikan terakhirmu ikut
          tersimpan.
        </p>
        <div class="flex flex-wrap justify-end gap-2">
          <button type="button" :class="btn" @click="deletedWhileEditing = null">Tutup</button>
          <button v-if="canWrite" type="button" :class="btn" @click="restoreFrom(deletedWhileEditing)">
            Pulihkan kartu
          </button>
        </div>
      </div>
    </BaseDialog>

    <BaseDialog :open="boardGone !== null" title="Papan ini dihapus" @close="goParent">
      <div class="flex flex-col gap-4 px-5 py-4">
        <p>Papan ini dipindah ke Tempat sampah oleh {{ boardGone?.by }}.</p>
        <div class="flex flex-wrap justify-end gap-2">
          <button type="button" :class="btn" @click="goParent">Ke papan induk</button>
          <button v-if="canWrite" type="button" :class="btn" @click="restoreFrom(boardGone)">
            Pulihkan papan
          </button>
        </div>
      </div>
    </BaseDialog>

    <BaseDialog :open="historyOpen" title="Riwayat papan" variant="panel" @close="historyOpen = false">
      <div class="min-h-0 flex-1 overflow-y-auto px-5 py-4">
        <p v-if="historyState === 'loading'" role="status" class="text-ink-muted">Memuat riwayat...</p>
        <p v-else-if="historyState === 'error'" role="alert" class="text-danger">{{ historyError }}</p>
        <p v-else-if="activity.length === 0" class="text-ink-muted">Belum ada riwayat.</p>
        <ol v-else class="flex flex-col gap-3">
          <li v-for="a in activity" :key="a.id" class="border-b border-line pb-3 last:border-b-0">
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
