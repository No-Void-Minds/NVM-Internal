// Pintasan keyboard Whiteboard revisi 50: satu tabel untuk menu klik kanan (teks pintasan) dan penangan keyboard
// kanvas. Dicocokkan lewat KeyboardEvent.code (tombol fisik), jadi Shift+2 dan Alt+Shift+huruf tetap terbaca walau
// karakter yang dihasilkan berbeda per tata letak keyboard. Pintasan yang sudah ada sebelum revisi 50 (Ctrl+C,
// Ctrl+D, Ctrl+Z, Ctrl+A, Ctrl+0, N, G, Delete) tetap ditangani penangan lama; di sini hanya teksnya untuk menu.

export type Combo = { ctrl?: boolean; alt?: boolean; shift?: boolean; code: string }

export const SHORTCUTS = {
  cut: { ctrl: true, code: 'KeyX' },
  copy: { ctrl: true, code: 'KeyC' },
  paste: { ctrl: true, code: 'KeyV' },
  duplicate: { ctrl: true, code: 'KeyD' },
  remove: { code: 'Delete' },
  copyStyle: { ctrl: true, alt: true, code: 'KeyC' },
  pasteStyle: { ctrl: true, alt: true, code: 'KeyV' },
  stylePanel: { ctrl: true, alt: true, code: 'KeyS' },
  group: { ctrl: true, code: 'KeyG' },
  ungroup: { ctrl: true, shift: true, code: 'KeyG' },
  lock: { ctrl: true, alt: true, code: 'KeyL' },
  front: { ctrl: true, shift: true, code: 'BracketRight' },
  forward: { ctrl: true, code: 'BracketRight' },
  backward: { ctrl: true, code: 'BracketLeft' },
  back: { ctrl: true, shift: true, code: 'BracketLeft' },
  alignLeft: { alt: true, shift: true, code: 'KeyA' },
  alignHCenter: { alt: true, shift: true, code: 'KeyH' },
  alignRight: { alt: true, shift: true, code: 'KeyD' },
  alignTop: { alt: true, shift: true, code: 'KeyW' },
  alignVCenter: { alt: true, shift: true, code: 'KeyV' },
  alignBottom: { alt: true, shift: true, code: 'KeyS' },
  spreadH: { alt: true, shift: true, code: 'KeyX' },
  spreadV: { alt: true, shift: true, code: 'KeyY' },
  sameWidth: { alt: true, shift: true, code: 'Digit1' },
  sameHeight: { alt: true, shift: true, code: 'Digit2' },
  sameSize: { alt: true, shift: true, code: 'Digit3' },
  tidy: { alt: true, shift: true, code: 'KeyG' },
  rotate90: { shift: true, code: 'KeyR' },
  flipH: { shift: true, code: 'KeyH' },
  flipV: { shift: true, code: 'KeyV' },
  connect: { alt: true, shift: true, code: 'KeyK' },
  // Revisi 57d1: tambah bentuk tersambung (ditangani penangan Ctrl lama di kanvas).
  quickAdd: { ctrl: true, code: 'Enter' },
  newBoard: { alt: true, shift: true, code: 'KeyB' },
  moveBoard: { alt: true, shift: true, code: 'KeyM' },
  comments: { ctrl: true, alt: true, code: 'KeyM' },
  copyLink: { alt: true, shift: true, code: 'KeyU' },
  exportSel: { alt: true, shift: true, code: 'KeyE' },
  zoomSel: { shift: true, code: 'Digit2' },
  sameKind: { alt: true, shift: true, code: 'KeyJ' },
  replaceFile: { alt: true, shift: true, code: 'KeyI' },
  crop: { alt: true, shift: true, code: 'KeyC' },
  frame: { alt: true, shift: true, code: 'KeyZ' },
  viewFull: { alt: true, shift: true, code: 'KeyO' },
  download: { alt: true, shift: true, code: 'KeyN' },
  addNote: { code: 'KeyN' },
  addHeading: { code: 'KeyJ' },
  addChecklist: { code: 'KeyC' },
  addTable: { code: 'KeyT' },
  addShape: { code: 'KeyB' },
  selectAll: { ctrl: true, code: 'KeyA' },
  fit: { ctrl: true, code: 'Digit0' },
  grid: { code: 'KeyG' },
  snap: { shift: true, code: 'KeyG' },
  undo: { ctrl: true, code: 'KeyZ' },
  redo: { ctrl: true, code: 'KeyY' },
} satisfies Record<string, Combo>

export type ShortcutId = keyof typeof SHORTCUTS

const KEY_NAMES: Record<string, string> = {
  BracketRight: ']',
  BracketLeft: '[',
  Delete: 'Delete',
}

// label: teks pintasan untuk menu dan tooltip, mis. "Ctrl+Alt+C".
export function label(id: ShortcutId): string {
  const c: Combo = SHORTCUTS[id]
  const key = KEY_NAMES[c.code] ?? c.code.replace(/^Key|^Digit/, '')
  return [c.ctrl && 'Ctrl', c.alt && 'Alt', c.shift && 'Shift', key].filter(Boolean).join('+')
}

type KeyLike = { ctrlKey: boolean; metaKey: boolean; altKey: boolean; shiftKey: boolean; code: string }

export function matches(e: KeyLike, id: ShortcutId): boolean {
  const c: Combo = SHORTCUTS[id]
  return (
    e.code === c.code &&
    (e.ctrlKey || e.metaKey) === !!c.ctrl &&
    e.altKey === !!c.alt &&
    e.shiftKey === !!c.shift
  )
}

// find: pintasan revisi 50 yang cocok dengan tombol ini, dari daftar yang diizinkan penangan.
export function find(e: KeyLike, ids: readonly ShortcutId[]): ShortcutId | null {
  return ids.find((id) => matches(e, id)) ?? null
}
