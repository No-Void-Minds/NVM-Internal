<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useId } from 'vue'
import type { Editor } from '@tiptap/core'
import { ALIGNS, FONT_SIZES, clearFormat, setAlign, setFontSize } from './wbFormat'
import { HIGHLIGHTS, TEXT_COLORS, colorClass, highlightClass } from '../../utils/wbRich'

// Panel format teks (revisi 49, ditata ulang revisi 52). Sejak revisi 52 panel ini menempel di bar pilihan kiri atas
// kanvas (slot #wb-format-slot, di kanan tombol Hapus, turun ke baris kedua bila tidak muat), jadi tidak pernah
// melayang di atas teks yang sedang diedit. Gayanya sama dengan toolbar pena. Di layar sentuh panel menempel di atas
// keyboard virtual (visualViewport) dan digulir menyamping. Tanpa slot (mis. bar pilihan tidak ada) panel diletakkan
// di atas atau bawah baris yang dipilih seperti revisi 49. Tombol tidak mengambil fokus saat diklik (mousedown
// prevent), jadi pilihan teks tetap ada; keyboard masuk lewat Alt+F10 dan Escape kembali ke teks.
const props = defineProps<{ editor: Editor; mode: 'slot' | 'float' }>()

type Panel = '' | 'size' | 'color' | 'highlight' | 'align' | 'link' | 'more'
const uid = useId()
const bar = ref<HTMLElement | null>(null)
const row = ref<HTMLElement | null>(null)
const panelEl = ref<HTMLElement | null>(null)
const pos = ref<{ left: number; top: number; width?: number }>({ left: -9999, top: -9999 })
const docked = ref(false)
const panel = ref<Panel>('')
const roving = ref(0)
const tick = ref(0)
const linkDraft = ref('')
const linkError = ref('')
const linkInput = ref<HTMLInputElement | null>(null)

const allowedLink = (url: string) => /^(https?:\/\/|mailto:)/i.test(url.trim())
// dodge: panel di slot bar pilihan akan menutupi kartu yang sedang diedit (kartu di tepi atas kanvas), jadi panel
// pindah menempel di tepi bawah kanvas. Kanvas tidak digeser.
const dodge = ref(false)
const fixed = computed(() => docked.value || props.mode === 'float' || dodge.value)

const state = computed(() => {
  void tick.value
  const e = props.editor
  const style = e.getAttributes('textStyle')
  const block = e.isActive('heading') ? e.getAttributes('heading') : e.getAttributes('paragraph')
  return {
    color: (style.color as string | undefined) ?? null,
    size: (style.fontSize as string | undefined) ?? null,
    highlight: e.isActive('highlight') ? ((e.getAttributes('highlight').color as string) ?? 'kuning') : null,
    align: (block.textAlign as string | undefined) ?? 'left',
    heading: e.isActive('heading') ? Number(e.getAttributes('heading').level) : 0,
    canSink: e.can().sinkListItem('listItem'),
    canLift: e.can().liftListItem('listItem'),
  }
})

type Tool = {
  id: string
  label: string
  key?: string
  path: string
  // Kelompok: pemisah digambar di antara kelompok yang berbeda.
  group: number
  active?: () => boolean
  disabled?: () => boolean
  run?: () => void
  panel?: Exclude<Panel, ''>
}
const c = () => props.editor.chain().focus()
const isOn = (name: string) => () => props.editor.isActive(name)
const tools: Tool[] = [
  {
    id: 'bold',
    group: 1,
    label: 'Tebal',
    key: 'Ctrl+B',
    path: 'M7 5h6a3.5 3.5 0 0 1 0 7H7zM7 12h7a3.5 3.5 0 0 1 0 7H7z',
    active: isOn('bold'),
    run: () => c().toggleBold().run(),
  },
  {
    id: 'italic',
    group: 1,
    label: 'Miring',
    key: 'Ctrl+I',
    path: 'M11 5h6M7 19h6M14 5l-4 14',
    active: isOn('italic'),
    run: () => c().toggleItalic().run(),
  },
  {
    id: 'underline',
    group: 1,
    label: 'Garis bawah',
    key: 'Ctrl+U',
    path: 'M7 5v6a5 5 0 0 0 10 0V5M5 20h14',
    active: isOn('underline'),
    run: () => c().toggleUnderline().run(),
  },
  {
    id: 'strike',
    group: 1,
    label: 'Coret',
    key: 'Ctrl+Shift+S',
    path: 'M5 12h14M16 7a4 3 0 0 0-8 0c0 4 8 3 8 7a4 3 0 0 1-8 0',
    active: isOn('strike'),
    run: () => c().toggleStrike().run(),
  },
  {
    id: 'size',
    group: 2,
    label: 'Ukuran dan judul',
    path: 'M4 7V5h10v2M9 5v14M7 19h4M14 12h6M17 12v7',
    panel: 'size',
  },
  { id: 'color', group: 2, label: 'Warna teks', path: 'M6 19L12 5l6 14M8.5 14h7', panel: 'color' },
  {
    id: 'highlight',
    group: 2,
    label: 'Stabilo',
    path: 'M9 15l-4 4h5l2-2M9 15l7-10 4 3-7 10z',
    panel: 'highlight',
  },
  { id: 'align', group: 2, label: 'Perataan', path: 'M4 6h16M4 10h10M4 14h16M4 18h10', panel: 'align' },
  {
    id: 'bullet',
    group: 3,
    label: 'Daftar berbutir',
    key: 'Ctrl+Shift+8',
    path: 'M9 6h11M9 12h11M9 18h11M4.5 6h.01M4.5 12h.01M4.5 18h.01',
    active: isOn('bulletList'),
    run: () => c().toggleBulletList().run(),
  },
  {
    id: 'ordered',
    group: 3,
    label: 'Daftar bernomor',
    key: 'Ctrl+Shift+7',
    path: 'M10 6h10M10 12h10M10 18h10M4 5h1v4M4 9h2M4 15h2l-2 3h2',
    active: isOn('orderedList'),
    run: () => c().toggleOrderedList().run(),
  },
  {
    id: 'indent',
    group: 3,
    label: 'Menjorok masuk',
    key: 'Tab',
    path: 'M4 6h16M10 10h10M10 14h10M4 18h16M4 9l3 3-3 3',
    disabled: () => !state.value.canSink,
    run: () => c().sinkListItem('listItem').run(),
  },
  {
    id: 'outdent',
    group: 3,
    label: 'Menjorok keluar',
    key: 'Shift+Tab',
    path: 'M4 6h16M10 10h10M10 14h10M4 18h16M7 9l-3 3 3 3',
    disabled: () => !state.value.canLift,
    run: () => c().liftListItem('listItem').run(),
  },
  {
    id: 'quote',
    group: 4,
    label: 'Kutipan',
    key: 'Ctrl+Shift+B',
    path: 'M6 7v10M10 9h9M10 13h9M10 17h6',
    active: isOn('blockquote'),
    run: () => c().toggleBlockquote().run(),
  },
  {
    id: 'hr',
    group: 4,
    label: 'Garis pemisah',
    key: 'Ctrl+Shift+Enter',
    path: 'M4 12h16M8 7h8M8 17h8',
    run: () => c().setHorizontalRule().run(),
  },
  {
    id: 'link',
    group: 4,
    label: 'Tautan',
    key: 'Ctrl+K',
    path: 'M10 14a4 4 0 0 0 5.66 0l3-3a4 4 0 0 0-5.66-5.66l-1 1M14 10a4 4 0 0 0-5.66 0l-3 3a4 4 0 0 0 5.66 5.66l1-1',
    active: isOn('link'),
    panel: 'link',
  },
  {
    id: 'clear',
    group: 4,
    label: 'Hapus format',
    key: 'Ctrl+\\',
    path: 'M6 5h12M12 5l-3 14M15 14l5 5M20 14l-5 5',
    run: () => clearFormat(props.editor),
  },
  {
    id: 'more',
    group: 5,
    label: 'Lainnya (kode, superskrip, subskrip)',
    path: 'M6 12h.01M12 12h.01M18 12h.01',
    panel: 'more',
  },
]
// Isi "Lainnya": format yang jarang dipakai.
const moreTools = [
  { id: 'code', label: 'Kode', key: 'Ctrl+E', active: isOn('code'), run: () => c().toggleCode().run() },
  {
    id: 'sup',
    label: 'Superskrip',
    key: 'Ctrl+.',
    active: isOn('superscript'),
    run: () => c().toggleMark('superscript').run(),
  },
  {
    id: 'sub',
    label: 'Subskrip',
    key: 'Ctrl+,',
    active: isOn('subscript'),
    run: () => c().toggleMark('subscript').run(),
  },
]
const panelLabel: Record<Exclude<Panel, ''>, string> = {
  size: 'Ukuran dan judul',
  color: 'Warna teks',
  highlight: 'Stabilo',
  align: 'Perataan',
  link: 'Tautan',
  more: 'Format lainnya',
}

function title(t: { label: string; key?: string }) {
  return t.key ? `${t.label} (${t.key})` : t.label
}

async function press(t: Tool, i: number) {
  roving.value = i
  if (t.panel) await togglePanel(t.panel)
  else {
    panel.value = ''
    t.run?.()
  }
}

async function togglePanel(p: Exclude<Panel, ''>, focusInside = false) {
  if (panel.value === p) {
    panel.value = ''
    return
  }
  panel.value = p
  await nextTick()
  if (p === 'link') {
    linkDraft.value = (props.editor.getAttributes('link').href as string | undefined) ?? ''
    linkError.value = ''
    linkInput.value?.focus()
  } else if (focusInside) panelButtons()[0]?.focus()
}

function openLink() {
  roving.value = tools.findIndex((t) => t.id === 'link')
  if (panel.value !== 'link') void togglePanel('link')
}

function applyLink() {
  const url = linkDraft.value.trim()
  if (!url) {
    c().extendMarkRange('link').unsetLink().run()
    panel.value = ''
    return
  }
  if (!allowedLink(url)) {
    linkError.value = 'Tautan harus diawali http://, https://, atau mailto:.'
    return
  }
  c().extendMarkRange('link').setLink({ href: url }).run()
  panel.value = ''
}

function pick(fn: () => void) {
  fn()
  panel.value = ''
  // Kembali ke teks langsung: commands.focus() TipTap 3 menunggu satu frame, jadi keyboard sempat tertinggal di panel.
  props.editor.view.focus()
}

const buttons = () => [...(row.value?.querySelectorAll<HTMLButtonElement>('[data-tb]') ?? [])]
const panelButtons = () => [...(panelEl.value?.querySelectorAll<HTMLButtonElement>('button') ?? [])]
function focusFirst() {
  const list = buttons()
  ;(list[roving.value] ?? list[0])?.focus()
}

// Panah berpindah antar tombol toolbar (roving tabindex) dan antar pilihan di dalam popover (Enter memilih);
// Escape menutup popover lalu kembali ke tombolnya, Escape berikutnya kembali ke teks.
function onKeydown(e: KeyboardEvent) {
  e.stopPropagation()
  if (e.key === 'Escape') {
    e.preventDefault()
    if (panel.value) {
      panel.value = ''
      focusFirst()
    } else props.editor.view.focus() // sinkron: commands.focus() TipTap 3 menunggu satu frame
    return
  }
  const inPanel = panelEl.value?.contains(document.activeElement)
  if (inPanel && (e.target as HTMLElement).tagName === 'INPUT') return
  const list = inPanel ? panelButtons() : buttons()
  const at = list.indexOf(document.activeElement as HTMLButtonElement)
  if (at < 0) {
    if (!inPanel && e.key === 'ArrowDown' && panel.value) {
      e.preventDefault()
      panelButtons()[0]?.focus()
    }
    return
  }
  let next = -1
  if (e.key === 'ArrowRight' || (inPanel && e.key === 'ArrowDown')) next = (at + 1) % list.length
  else if (e.key === 'ArrowLeft' || (inPanel && e.key === 'ArrowUp'))
    next = (at - 1 + list.length) % list.length
  else if (e.key === 'Home') next = 0
  else if (e.key === 'End') next = list.length - 1
  else if (!inPanel && e.key === 'ArrowDown' && panel.value) {
    e.preventDefault()
    panelButtons()[0]?.focus()
    return
  }
  if (next < 0) return
  e.preventDefault()
  if (!inPanel) roving.value = next
  list[next]?.focus()
}

let frame = 0
function schedule() {
  if (frame) return
  frame = requestAnimationFrame(() => {
    frame = 0
    place()
  })
}

function place() {
  tick.value++
  const el = bar.value
  if (!el) return
  const vv = window.visualViewport
  docked.value = matchMedia('(pointer: coarse)').matches && !!vv
  if (props.mode === 'slot' && !docked.value) {
    placeInSlot(el)
    return
  }
  if (!fixed.value) return
  const bh = el.offsetHeight
  if (docked.value && vv) {
    pos.value = { left: vv.offsetLeft, top: vv.offsetTop + vv.height - bh, width: vv.width }
    return
  }
  const view = props.editor.view
  const { from, to } = props.editor.state.selection
  let a: { top: number; bottom: number; left: number; right: number }
  let b: typeof a
  try {
    a = view.coordsAtPos(from)
    b = view.coordsAtPos(to)
  } catch {
    a = b = view.dom.getBoundingClientRect()
  }
  const vw = document.documentElement.clientWidth
  const vh = window.innerHeight
  const bw = el.offsetWidth
  let y = Math.min(a.top, b.top) - bh - 8
  if (y < 8) y = Math.max(a.bottom, b.bottom) + 8
  y = Math.max(8, Math.min(y, vh - bh - 8))
  const cx = a.top === b.top ? (a.left + b.right) / 2 : a.left
  pos.value = { left: Math.round(Math.max(8, Math.min(cx - bw / 2, vw - bw - 8))), top: Math.round(y) }
}

function hits(a: DOMRect | { x: number; y: number; width: number; height: number }, b: DOMRect) {
  return a.x < b.right && b.x < a.x + a.width && a.y < b.bottom && b.y < a.y + a.height
}
function placeInSlot(el: HTMLElement) {
  const slot = document.getElementById('wb-format-slot')?.getBoundingClientRect()
  const card = props.editor.view.dom.closest('[data-card]')?.getBoundingClientRect()
  const canvas = document.querySelector('.wb-canvas')?.getBoundingClientRect()
  if (!slot || !card || !canvas) {
    dodge.value = false
    return
  }
  const natural = { x: slot.x, y: slot.y, width: el.offsetWidth, height: el.offsetHeight }
  const bottom = {
    x: canvas.x + 8,
    y: canvas.bottom - el.offsetHeight - 8,
    width: el.offsetWidth,
    height: el.offsetHeight,
  }
  dodge.value = hits(natural, card) && !hits(bottom, card)
  if (dodge.value) pos.value = { left: Math.round(bottom.x), top: Math.round(bottom.y) }
}

let ro: ResizeObserver | undefined
onMounted(() => {
  props.editor.on('transaction', schedule)
  props.editor.on('focus', schedule)
  window.addEventListener('resize', schedule)
  window.addEventListener('scroll', schedule, true)
  window.addEventListener('wheel', schedule, { capture: true, passive: true })
  window.visualViewport?.addEventListener('resize', schedule)
  window.visualViewport?.addEventListener('scroll', schedule)
  ro = new ResizeObserver(schedule)
  if (bar.value) ro.observe(bar.value)
  place()
})
onBeforeUnmount(() => {
  props.editor.off('transaction', schedule)
  props.editor.off('focus', schedule)
  window.removeEventListener('resize', schedule)
  window.removeEventListener('scroll', schedule, true)
  window.removeEventListener('wheel', schedule, { capture: true })
  window.visualViewport?.removeEventListener('resize', schedule)
  window.visualViewport?.removeEventListener('scroll', schedule)
  ro?.disconnect()
  cancelAnimationFrame(frame)
})

defineExpose({ focusFirst, openLink })

// Sama dengan toolbar pena (WbPenBar): tombol 44 px, hover canvas, tertekan accent.
const btn =
  'inline-flex size-11 shrink-0 items-center justify-center rounded-card text-ink hover:bg-canvas aria-pressed:bg-accent aria-pressed:text-on-accent aria-expanded:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-50'
const opt =
  'inline-flex min-h-11 shrink-0 items-center gap-2 rounded-card px-3 text-sm text-ink hover:bg-canvas aria-pressed:bg-accent aria-pressed:text-on-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
const swatch =
  'relative inline-flex size-11 shrink-0 items-center justify-center rounded-card hover:bg-canvas aria-pressed:outline-2 aria-pressed:outline-mark focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
</script>

<template>
  <div
    ref="bar"
    data-testid="wb-formatbar"
    class="flex flex-col gap-1 border border-line bg-surface p-1 shadow-menu"
    :class="[
      fixed ? 'fixed z-[60]' : 'max-w-full rounded-card',
      docked ? 'rounded-none' : fixed ? 'max-w-[min(calc(100vw-1rem),34rem)] rounded-card' : '',
    ]"
    :style="
      fixed
        ? { left: `${pos.left}px`, top: `${pos.top}px`, width: pos.width ? `${pos.width}px` : undefined }
        : undefined
    "
    @mousedown="($event.target as HTMLElement).closest('input') || $event.preventDefault()"
    @pointerdown.stop
    @keydown="onKeydown"
  >
    <div
      ref="row"
      role="toolbar"
      aria-label="Format teks"
      class="flex items-center gap-0.5"
      :class="docked ? 'flex-nowrap overflow-x-auto overscroll-x-contain' : 'flex-wrap'"
    >
      <template v-for="(t, i) in tools" :key="t.id">
        <span
          v-if="i > 0 && tools[i - 1]!.group !== t.group"
          aria-hidden="true"
          class="mx-0.5 h-6 w-px shrink-0 bg-line"
        />
        <button
          type="button"
          data-tb
          :class="btn"
          :tabindex="i === roving ? 0 : -1"
          :aria-label="t.label"
          :aria-keyshortcuts="t.key?.replace(/Ctrl/g, 'Control')"
          :title="title(t)"
          :aria-pressed="t.active ? (t.active() ? 'true' : 'false') : undefined"
          :aria-expanded="t.panel ? panel === t.panel : undefined"
          :aria-controls="t.panel ? `${uid}-panel` : undefined"
          :aria-haspopup="t.panel ? 'true' : undefined"
          :disabled="t.disabled?.()"
          @click="press(t, i)"
        >
          <span
            v-if="t.id === 'color'"
            aria-hidden="true"
            class="text-lg leading-none font-bold"
            :class="colorClass(state.color)"
            >A</span
          >
          <svg
            v-else
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
        </button>
      </template>
    </div>

    <div
      v-if="panel"
      :id="`${uid}-panel`"
      ref="panelEl"
      role="group"
      :aria-label="panelLabel[panel]"
      class="flex flex-wrap gap-0.5 border-t border-line pt-1"
    >
      <template v-if="panel === 'size'">
        <button
          v-for="s in FONT_SIZES"
          :key="s.label"
          type="button"
          :class="opt"
          :aria-pressed="!state.heading && state.size === s.value ? 'true' : 'false'"
          :title="
            s.value === 'besar'
              ? 'Perbesar (Ctrl+Shift+.)'
              : s.value === 'kecil'
                ? 'Perkecil (Ctrl+Shift+,)'
                : s.label
          "
          @click="pick(() => (editor.chain().focus().setParagraph().run(), setFontSize(editor, s.value)))"
        >
          {{ s.label }}
        </button>
        <button
          v-for="lv in [1, 2, 3] as const"
          :key="lv"
          type="button"
          :class="opt"
          :aria-pressed="state.heading === lv ? 'true' : 'false'"
          :title="`Judul ${lv} (Ctrl+Alt+${lv})`"
          :aria-keyshortcuts="`Control+Alt+${lv}`"
          @click="pick(() => editor.chain().focus().toggleHeading({ level: lv }).run())"
        >
          Judul {{ lv }}
        </button>
      </template>
      <!-- Palet kotak: nama warna di aria-label dan tooltip, pilihan aktif ditandai centang. -->
      <template v-else-if="panel === 'color'">
        <button
          type="button"
          :class="swatch"
          aria-label="Warna bawaan"
          title="Warna bawaan"
          :aria-pressed="!state.color ? 'true' : 'false'"
          @click="pick(() => editor.chain().focus().setMark('textStyle', { color: null }).run())"
        >
          <span aria-hidden="true" class="text-lg font-bold">A</span>
          <svg
            v-if="!state.color"
            aria-hidden="true"
            viewBox="0 0 16 16"
            class="absolute right-0.5 bottom-0.5 size-4 rounded-full bg-surface"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M4 8.5l2.5 2.5L12 5.5" />
          </svg>
        </button>
        <button
          v-for="col in TEXT_COLORS"
          :key="col.value"
          type="button"
          :class="swatch"
          :aria-label="`Warna ${col.label.toLowerCase()}`"
          :title="col.label"
          :aria-pressed="state.color === col.value ? 'true' : 'false'"
          @click="pick(() => editor.chain().focus().setMark('textStyle', { color: col.value }).run())"
        >
          <span aria-hidden="true" class="text-lg font-bold" :class="colorClass(col.value)">A</span>
          <svg
            v-if="state.color === col.value"
            aria-hidden="true"
            viewBox="0 0 16 16"
            class="absolute right-0.5 bottom-0.5 size-4 rounded-full bg-surface text-ink"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M4 8.5l2.5 2.5L12 5.5" />
          </svg>
        </button>
      </template>
      <template v-else-if="panel === 'highlight'">
        <button
          type="button"
          :class="swatch"
          aria-label="Tanpa stabilo"
          title="Tanpa stabilo"
          :aria-pressed="!state.highlight ? 'true' : 'false'"
          @click="pick(() => editor.chain().focus().unsetMark('highlight').run())"
        >
          <svg
            aria-hidden="true"
            viewBox="0 0 24 24"
            class="size-5"
            fill="none"
            stroke="currentColor"
            stroke-width="1.75"
          >
            <path d="M5 5l14 14M5 12h14" />
          </svg>
        </button>
        <button
          v-for="hl in HIGHLIGHTS"
          :key="hl.value"
          type="button"
          :class="swatch"
          :aria-label="`Stabilo ${hl.label.toLowerCase()}`"
          :title="hl.value === 'kuning' ? 'Kuning (Ctrl+Shift+H)' : hl.label"
          :aria-pressed="state.highlight === hl.value ? 'true' : 'false'"
          @click="pick(() => editor.chain().focus().setMark('highlight', { color: hl.value }).run())"
        >
          <span aria-hidden="true" class="rounded-chip px-1.5 font-bold" :class="highlightClass(hl.value)"
            >A</span
          >
          <svg
            v-if="state.highlight === hl.value"
            aria-hidden="true"
            viewBox="0 0 16 16"
            class="absolute right-0.5 bottom-0.5 size-4 rounded-full bg-surface text-ink"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M4 8.5l2.5 2.5L12 5.5" />
          </svg>
        </button>
      </template>
      <template v-else-if="panel === 'align'">
        <button
          v-for="al in ALIGNS"
          :key="al.value"
          type="button"
          :class="opt"
          :aria-pressed="state.align === al.value ? 'true' : 'false'"
          :title="`${al.label} (${al.key})`"
          :aria-keyshortcuts="al.key.replace('Ctrl', 'Control')"
          @click="pick(() => setAlign(editor, al.value))"
        >
          {{ al.label }}
        </button>
      </template>
      <template v-else-if="panel === 'more'">
        <button
          v-for="m in moreTools"
          :key="m.id"
          type="button"
          :class="opt"
          :aria-pressed="m.active() ? 'true' : 'false'"
          :title="title(m)"
          :aria-keyshortcuts="m.key.replace('Ctrl', 'Control')"
          @click="pick(m.run)"
        >
          {{ m.label }}
        </button>
      </template>
      <form v-else class="flex w-full flex-wrap items-start gap-2" novalidate @submit.prevent="applyLink">
        <label :for="`${uid}-url`" class="sr-only">Alamat tautan</label>
        <input
          :id="`${uid}-url`"
          ref="linkInput"
          v-model="linkDraft"
          type="url"
          inputmode="url"
          placeholder="https://"
          maxlength="2048"
          :aria-invalid="linkError ? 'true' : undefined"
          :aria-describedby="linkError ? `${uid}-err` : undefined"
          class="min-h-11 min-w-0 flex-1 basis-40 rounded-card border border-line bg-surface px-2"
        />
        <button
          type="submit"
          class="min-h-11 rounded-card border border-line bg-surface px-3 font-medium hover:bg-canvas"
        >
          Pasang tautan
        </button>
        <p v-if="linkError" :id="`${uid}-err`" role="alert" class="w-full text-sm text-danger">
          {{ linkError }}
        </p>
      </form>
    </div>
  </div>
</template>
