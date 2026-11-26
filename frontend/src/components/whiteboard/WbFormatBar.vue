<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useId } from 'vue'
import type { Editor } from '@tiptap/core'
import { ALIGNS, FONT_SIZES, clearFormat, setAlign, setFontSize } from './wbFormat'
import { HIGHLIGHTS, TEXT_COLORS, colorClass, highlightClass } from '../../utils/wbRich'

// Toolbar format melayang (revisi 49) untuk editor teks yang sedang aktif. Diletakkan di atas baris yang dipilih,
// pindah ke bawahnya bila di atas tidak cukup ruang, dan selalu di dalam layar. Di layar sentuh menempel di atas
// keyboard virtual (visualViewport) dan digulir menyamping. Tombol tidak mengambil fokus saat diklik (mousedown
// prevent), jadi pilihan teks tetap ada; keyboard masuk lewat Alt+F10 dan Escape kembali ke teks.
const props = defineProps<{ editor: Editor }>()

const uid = useId()
const bar = ref<HTMLElement | null>(null)
const row = ref<HTMLElement | null>(null)
const pos = ref<{ left: number; top: number; width?: number }>({ left: -9999, top: -9999 })
const docked = ref(false)
const panel = ref<'' | 'size' | 'color' | 'highlight' | 'align' | 'link'>('')
const roving = ref(0)
const tick = ref(0)
const linkDraft = ref('')
const linkError = ref('')
const linkInput = ref<HTMLInputElement | null>(null)

const allowedLink = (url: string) => /^(https?:\/\/|mailto:)/i.test(url.trim())

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
  active?: () => boolean
  disabled?: () => boolean
  run?: () => void
  panel?: 'size' | 'color' | 'highlight' | 'align' | 'link'
}
const c = () => props.editor.chain().focus()
const tools: Tool[] = [
  {
    id: 'bold',
    label: 'Tebal',
    key: 'Ctrl+B',
    path: 'M7 5h6a3.5 3.5 0 0 1 0 7H7zM7 12h7a3.5 3.5 0 0 1 0 7H7z',
    active: () => props.editor.isActive('bold'),
    run: () => c().toggleBold().run(),
  },
  {
    id: 'italic',
    label: 'Miring',
    key: 'Ctrl+I',
    path: 'M11 5h6M7 19h6M14 5l-4 14',
    active: () => props.editor.isActive('italic'),
    run: () => c().toggleItalic().run(),
  },
  {
    id: 'underline',
    label: 'Garis bawah',
    key: 'Ctrl+U',
    path: 'M7 5v6a5 5 0 0 0 10 0V5M5 20h14',
    active: () => props.editor.isActive('underline'),
    run: () => c().toggleUnderline().run(),
  },
  {
    id: 'strike',
    label: 'Coret',
    key: 'Ctrl+Shift+S',
    path: 'M5 12h14M16 7a4 3 0 0 0-8 0c0 4 8 3 8 7a4 3 0 0 1-8 0',
    active: () => props.editor.isActive('strike'),
    run: () => c().toggleStrike().run(),
  },
  {
    id: 'code',
    label: 'Kode',
    key: 'Ctrl+E',
    path: 'M9 7l-5 5 5 5M15 7l5 5-5 5',
    active: () => props.editor.isActive('code'),
    run: () => c().toggleCode().run(),
  },
  {
    id: 'sup',
    label: 'Superskrip',
    key: 'Ctrl+.',
    path: 'M4 8l8 10M12 8l-8 10M16 9l3-4h1v5M16 10h5',
    active: () => props.editor.isActive('superscript'),
    run: () => c().toggleMark('superscript').run(),
  },
  {
    id: 'sub',
    label: 'Subskrip',
    key: 'Ctrl+,',
    path: 'M4 5l8 10M12 5L4 15M16 15l3-4h1v5M16 16h5',
    active: () => props.editor.isActive('subscript'),
    run: () => c().toggleMark('subscript').run(),
  },
  { id: 'size', label: 'Ukuran dan judul', path: 'M4 7V5h10v2M9 5v14M7 19h4M14 12h6M17 12v7', panel: 'size' },
  { id: 'color', label: 'Warna teks', path: 'M6 19L12 5l6 14M8.5 14h7', panel: 'color' },
  { id: 'highlight', label: 'Stabilo', path: 'M9 15l-4 4h5l2-2M9 15l7-10 4 3-7 10z', panel: 'highlight' },
  { id: 'align', label: 'Perataan', path: 'M4 6h16M4 10h10M4 14h16M4 18h10', panel: 'align' },
  {
    id: 'bullet',
    label: 'Daftar berbutir',
    key: 'Ctrl+Shift+8',
    path: 'M9 6h11M9 12h11M9 18h11M4.5 6h.01M4.5 12h.01M4.5 18h.01',
    active: () => props.editor.isActive('bulletList'),
    run: () => c().toggleBulletList().run(),
  },
  {
    id: 'ordered',
    label: 'Daftar bernomor',
    key: 'Ctrl+Shift+7',
    path: 'M10 6h10M10 12h10M10 18h10M4 5h1v4M4 9h2M4 15h2l-2 3h2',
    active: () => props.editor.isActive('orderedList'),
    run: () => c().toggleOrderedList().run(),
  },
  {
    id: 'indent',
    label: 'Menjorok masuk',
    key: 'Tab',
    path: 'M4 6h16M10 10h10M10 14h10M4 18h16M4 9l3 3-3 3',
    disabled: () => !state.value.canSink,
    run: () => c().sinkListItem('listItem').run(),
  },
  {
    id: 'outdent',
    label: 'Menjorok keluar',
    key: 'Shift+Tab',
    path: 'M4 6h16M10 10h10M10 14h10M4 18h16M7 9l-3 3 3 3',
    disabled: () => !state.value.canLift,
    run: () => c().liftListItem('listItem').run(),
  },
  {
    id: 'quote',
    label: 'Kutipan',
    key: 'Ctrl+Shift+B',
    path: 'M6 7v10M10 9h9M10 13h9M10 17h6',
    active: () => props.editor.isActive('blockquote'),
    run: () => c().toggleBlockquote().run(),
  },
  {
    id: 'hr',
    label: 'Garis pemisah',
    key: 'Ctrl+Shift+Enter',
    path: 'M4 12h16M8 7h8M8 17h8',
    run: () => c().setHorizontalRule().run(),
  },
  {
    id: 'link',
    label: 'Tautan',
    key: 'Ctrl+K',
    path: 'M10 14a4 4 0 0 0 5.66 0l3-3a4 4 0 0 0-5.66-5.66l-1 1M14 10a4 4 0 0 0-5.66 0l-3 3a4 4 0 0 0 5.66 5.66l1-1',
    active: () => props.editor.isActive('link'),
    panel: 'link',
  },
  {
    id: 'clear',
    label: 'Hapus format',
    key: 'Ctrl+\\',
    path: 'M6 5h12M12 5l-3 14M15 14l5 5M20 14l-5 5',
    run: () => clearFormat(props.editor),
  },
]
const panelLabel = {
  size: 'Ukuran dan judul',
  color: 'Warna teks',
  highlight: 'Stabilo',
  align: 'Perataan',
  link: 'Tautan',
}

function title(t: Tool) {
  return t.key ? `${t.label} (${t.key})` : t.label
}

function press(t: Tool, i: number) {
  roving.value = i
  if (t.panel) togglePanel(t.panel)
  else {
    panel.value = ''
    t.run?.()
  }
}

async function togglePanel(p: NonNullable<Tool['panel']>) {
  if (panel.value === p) {
    panel.value = ''
    return
  }
  panel.value = p
  if (p === 'link') {
    linkDraft.value = (props.editor.getAttributes('link').href as string | undefined) ?? ''
    linkError.value = ''
    await nextTick()
    linkInput.value?.focus()
  }
}

function openLink() {
  roving.value = tools.findIndex((t) => t.id === 'link')
  if (panel.value !== 'link') togglePanel('link')
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
}

const buttons = () => [...(row.value?.querySelectorAll<HTMLButtonElement>('[data-tb]') ?? [])]
function focusFirst() {
  const list = buttons()
  ;(list[roving.value] ?? list[0])?.focus()
}

// Navigasi panah antar tombol toolbar (roving tabindex); Escape menutup panel lalu kembali ke teks.
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
  const list = buttons()
  const at = list.indexOf(document.activeElement as HTMLButtonElement)
  if (at < 0) return
  let next = -1
  if (e.key === 'ArrowRight' || e.key === 'ArrowDown') next = (at + 1) % list.length
  else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') next = (at - 1 + list.length) % list.length
  else if (e.key === 'Home') next = 0
  else if (e.key === 'End') next = list.length - 1
  if (next < 0) return
  e.preventDefault()
  roving.value = next
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
  const top = Math.min(a.top, b.top)
  const bottom = Math.max(a.bottom, b.bottom)
  // Di atas baris yang dipilih; bila tidak muat, di bawahnya. Keduanya tidak menutupi baris itu.
  let y = top - bh - 8
  if (y < 8) y = bottom + 8
  y = Math.max(8, Math.min(y, vh - bh - 8))
  const cx = a.top === b.top ? (a.left + b.right) / 2 : a.left
  const x = Math.max(8, Math.min(cx - bw / 2, vw - bw - 8))
  pos.value = { left: Math.round(x), top: Math.round(y) }
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

const btn =
  'inline-flex size-11 shrink-0 items-center justify-center rounded-card text-ink hover:bg-canvas aria-pressed:bg-accent aria-pressed:text-on-accent aria-expanded:bg-canvas focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark disabled:opacity-50'
const opt =
  'inline-flex min-h-11 shrink-0 items-center gap-2 rounded-card px-3 text-sm text-ink hover:bg-canvas aria-pressed:bg-accent aria-pressed:text-on-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-mark'
</script>

<template>
  <div
    ref="bar"
    data-testid="wb-formatbar"
    class="fixed z-[60] flex flex-col gap-1 border border-line bg-surface p-1 shadow-menu"
    :class="docked ? 'rounded-none' : 'max-w-[min(calc(100vw-1rem),34rem)] rounded-card'"
    :style="{ left: `${pos.left}px`, top: `${pos.top}px`, width: pos.width ? `${pos.width}px` : undefined }"
    @mousedown="($event.target as HTMLElement).closest('input') || $event.preventDefault()"
    @pointerdown.stop
    @keydown="onKeydown"
  >
    <div
      ref="row"
      role="toolbar"
      aria-label="Format teks"
      class="flex gap-0.5"
      :class="docked ? 'flex-nowrap overflow-x-auto overscroll-x-contain' : 'flex-wrap'"
    >
      <button
        v-for="(t, i) in tools"
        :key="t.id"
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
    </div>

    <div
      v-if="panel"
      :id="`${uid}-panel`"
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
      <template v-else-if="panel === 'color'">
        <button
          type="button"
          :class="opt"
          :aria-pressed="!state.color ? 'true' : 'false'"
          @click="pick(() => editor.chain().focus().setMark('textStyle', { color: null }).run())"
        >
          Bawaan
        </button>
        <button
          v-for="col in TEXT_COLORS"
          :key="col.value"
          type="button"
          :class="opt"
          :aria-pressed="state.color === col.value ? 'true' : 'false'"
          @click="pick(() => editor.chain().focus().setMark('textStyle', { color: col.value }).run())"
        >
          <span aria-hidden="true" class="font-bold" :class="colorClass(col.value)">A</span>{{ col.label }}
        </button>
      </template>
      <template v-else-if="panel === 'highlight'">
        <button
          type="button"
          :class="opt"
          :aria-pressed="!state.highlight ? 'true' : 'false'"
          @click="pick(() => editor.chain().focus().unsetMark('highlight').run())"
        >
          Tanpa stabilo
        </button>
        <button
          v-for="hl in HIGHLIGHTS"
          :key="hl.value"
          type="button"
          :class="opt"
          :aria-pressed="state.highlight === hl.value ? 'true' : 'false'"
          :title="hl.value === 'kuning' ? 'Kuning (Ctrl+Shift+H)' : hl.label"
          @click="pick(() => editor.chain().focus().setMark('highlight', { color: hl.value }).run())"
        >
          <span aria-hidden="true" class="rounded-chip px-1" :class="highlightClass(hl.value)">A</span
          >{{ hl.label }}
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
