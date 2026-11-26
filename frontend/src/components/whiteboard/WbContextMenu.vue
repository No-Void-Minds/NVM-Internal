<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, useId } from 'vue'

// Menu klik kanan Whiteboard (revisi 50). Satu komponen per tingkat: submenu memakai komponen ini lagi. role="menu"
// dengan menuitem; panah atas dan bawah pindah item, panah kanan (atau Enter, Spasi) membuka submenu, panah kiri
// menutupnya, Escape menutup satu tingkat, Tab menutup semua. Posisi selalu di dalam layar: dibalik ke kiri atau ke
// atas bila tidak muat. Item hanya berisi aksi yang berlaku untuk pilihan saat itu (disusun halaman).
export type MenuItem = {
  id: string
  label: string
  key?: string
  run?: () => void
  children?: MenuItem[]
  // Saklar (grid, snap): menuitemcheckbox dengan aria-checked.
  checked?: boolean
  danger?: boolean
}

const props = withDefaults(
  defineProps<{
    items: MenuItem[]
    x: number
    y: number
    label: string
    level?: number
    anchorWidth?: number
    parentId?: string
  }>(),
  { level: 0, anchorWidth: 0, parentId: undefined },
)
// close: semua tingkat ditutup (aksi dijalankan, Tab, klik di luar); back: hanya tingkat ini (panah kiri, Escape).
const emit = defineEmits<{ close: [refocus: boolean]; back: [] }>()

const uid = useId()
const menu = ref<HTMLElement | null>(null)
const pos = ref({ left: props.x, top: props.y })
const open = ref<{ index: number; x: number; y: number; w: number } | null>(null)
const active = ref(0)

function buttons() {
  return [...(menu.value?.querySelectorAll<HTMLElement>(':scope > li > [role^="menuitem"]') ?? [])]
}
function focusAt(i: number) {
  const list = buttons()
  if (!list.length) return
  active.value = (i + list.length) % list.length
  list[active.value]?.focus()
}

// Dijepit 8 px di dalam layar; submenu dibalik ke kiri induknya bila di kanan tidak cukup ruang.
function place() {
  const el = menu.value
  if (!el) return
  const vw = document.documentElement.clientWidth
  const vh = window.innerHeight
  const w = el.offsetWidth
  const h = el.offsetHeight
  let left = props.x
  if (left + w > vw - 8) left = props.level > 0 ? props.x - props.anchorWidth - w : vw - w - 8
  let top = props.y
  if (top + h > vh - 8) top = props.level > 0 ? vh - h - 8 : Math.max(8, props.y - h)
  pos.value = { left: Math.max(8, left), top: Math.max(8, Math.min(top, vh - h - 8)) }
}

async function openSub(i: number, focus: boolean) {
  const item = props.items[i]
  const btn = buttons()[i]
  if (!item?.children?.length || !btn) return
  const r = btn.getBoundingClientRect()
  open.value = { index: i, x: r.right, y: r.top, w: r.width }
  if (focus) {
    await nextTick()
    document
      .querySelector<HTMLElement>(`[data-wb-menu="${uid}-${props.level + 1}"] [role^="menuitem"]`)
      ?.focus()
  }
}

function choose(i: number) {
  const item = props.items[i]
  if (!item) return
  if (item.children?.length) {
    openSub(i, true)
    return
  }
  emit('close', false)
  item.run?.()
}

function onKeydown(e: KeyboardEvent) {
  e.stopPropagation()
  const i = buttons().indexOf(document.activeElement as HTMLElement)
  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      focusAt(i + 1)
      return
    case 'ArrowUp':
      e.preventDefault()
      focusAt(i - 1)
      return
    case 'Home':
      e.preventDefault()
      focusAt(0)
      return
    case 'End':
      e.preventDefault()
      focusAt(-1)
      return
    case 'ArrowRight':
      if (props.items[i]?.children?.length) {
        e.preventDefault()
        openSub(i, true)
      }
      return
    case 'ArrowLeft':
      if (props.level > 0) {
        e.preventDefault()
        emit('back')
      }
      return
    case 'Escape':
      e.preventDefault()
      if (props.level > 0) emit('back')
      else emit('close', true)
      return
    case 'Tab':
      e.preventDefault()
      emit('close', true)
      return
    case 'Enter':
    case ' ':
      e.preventDefault()
      if (i >= 0) choose(i)
  }
}

function closeSub() {
  const i = open.value?.index ?? 0
  open.value = null
  nextTick(() => focusAt(i))
}

function onDocDown(e: PointerEvent) {
  if (props.level > 0) return
  const t = e.target as HTMLElement
  if (!t.closest('[data-wb-menu]')) emit('close', false)
}

onMounted(async () => {
  await nextTick()
  place()
  focusAt(0)
  if (props.level === 0) document.addEventListener('pointerdown', onDocDown, true)
})
onBeforeUnmount(() => document.removeEventListener('pointerdown', onDocDown, true))

const item =
  'flex min-h-11 w-full items-center gap-4 rounded-card px-3 text-left text-sm text-ink hover:bg-canvas focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-mark aria-expanded:bg-canvas'
</script>

<template>
  <ul
    ref="menu"
    role="menu"
    :aria-label="label"
    :data-wb-menu="`${parentId ?? uid}-${level}`"
    data-testid="wb-menu"
    class="fixed z-[70] max-h-[calc(100dvh-1rem)] min-w-56 overflow-y-auto rounded-card border border-line bg-surface p-1 shadow-menu"
    :style="{ left: `${pos.left}px`, top: `${pos.top}px` }"
    @keydown="onKeydown"
    @contextmenu.prevent
  >
    <li v-for="(it, i) in items" :key="it.id" role="none">
      <button
        type="button"
        :role="it.checked === undefined ? 'menuitem' : 'menuitemcheckbox'"
        :aria-checked="it.checked === undefined ? undefined : it.checked ? 'true' : 'false'"
        :aria-haspopup="it.children ? 'menu' : undefined"
        :aria-expanded="it.children ? open?.index === i : undefined"
        :aria-keyshortcuts="it.key?.replace(/Ctrl/g, 'Control')"
        :tabindex="i === active ? 0 : -1"
        :class="[item, it.danger ? 'text-urgent' : '']"
        @click="choose(i)"
        @pointerenter="it.children ? openSub(i, false) : open && (open = null)"
        @focus="active = i"
      >
        <span
          v-if="it.checked !== undefined"
          aria-hidden="true"
          class="inline-flex size-4 shrink-0 items-center justify-center rounded-sm border border-line"
        >
          <svg
            v-if="it.checked"
            viewBox="0 0 16 16"
            class="size-3"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M3 8.5l3 3 7-7" />
          </svg>
        </span>
        <span class="flex-1">{{ it.label }}</span>
        <span v-if="it.key" aria-hidden="true" class="text-xs text-ink-muted">{{ it.key }}</span>
        <svg
          v-if="it.children"
          aria-hidden="true"
          viewBox="0 0 16 16"
          class="size-4 text-ink-muted"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
        >
          <path d="M6 3l5 5-5 5" />
        </svg>
      </button>
    </li>
  </ul>
  <WbContextMenu
    v-if="open && items[open.index]?.children"
    :parent-id="uid"
    :items="items[open.index]!.children!"
    :x="open.x"
    :y="open.y"
    :anchor-width="open.w"
    :label="items[open.index]!.label"
    :level="level + 1"
    @close="emit('close', $event)"
    @back="closeSub"
  />
</template>
