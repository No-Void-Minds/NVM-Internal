import { Extension, Mark, type Editor } from '@tiptap/core'
import {
  alignClass,
  alignSet,
  colorClass,
  colorSet,
  highSet,
  highlightClass,
  sizeClass,
  sizeSet,
} from '../../utils/wbRich'

// Format teks revisi 49. Warna, stabilo, ukuran, dan perataan disimpan sebagai nama dari daftar (server memakai daftar
// yang sama) dan dirender sebagai kelas CSS statis di style.css, bukan atribut style: CSP style-src 'self' memblokir
// atribut style yang dipasang ProseMirror lewat setAttribute. Kontras setiap pilihan dihitung di docs/sprint.md
// (Sprint 12h); warna yang gagal di salah satu latar kartu tidak ada di daftar.

export const FONT_SIZES = [
  { value: 'kecil', label: 'Kecil' },
  { value: null, label: 'Normal' },
  { value: 'besar', label: 'Besar' },
  { value: 'sangat_besar', label: 'Sangat besar' },
] as const
export const ALIGNS = [
  { value: 'left', label: 'Rata kiri', key: 'Ctrl+Shift+L' },
  { value: 'center', label: 'Rata tengah', key: 'Ctrl+Shift+E' },
  { value: 'right', label: 'Rata kanan', key: 'Ctrl+Shift+R' },
  { value: 'justify', label: 'Rata kiri-kanan', key: 'Ctrl+Shift+J' },
] as const

function fromClass(el: HTMLElement, prefix: string, set: Set<string>) {
  for (const c of el.classList)
    if (c.startsWith(prefix) && set.has(c.slice(prefix.length))) return c.slice(prefix.length)
  return null
}

const TextStyle = Mark.create({
  name: 'textStyle',
  addAttributes() {
    return {
      color: { default: null, parseHTML: (el) => fromClass(el, 'wb-c-', colorSet), renderHTML: () => ({}) },
      fontSize: {
        default: null,
        parseHTML: (el) => fromClass(el, 'wb-fs-', sizeSet),
        renderHTML: () => ({}),
      },
    }
  },
  parseHTML() {
    return [{ tag: 'span', getAttrs: (el) => (el.className.match(/wb-(c|fs)-/) ? {} : false) }]
  },
  renderHTML({ mark }) {
    const cls = [colorClass(mark.attrs.color), sizeClass(mark.attrs.fontSize)].filter(Boolean).join(' ')
    return ['span', cls ? { class: cls } : {}, 0]
  },
})

const Highlight = Mark.create({
  name: 'highlight',
  addAttributes() {
    return {
      color: {
        default: 'kuning',
        parseHTML: (el) => fromClass(el, 'wb-h-', highSet),
        renderHTML: () => ({}),
      },
    }
  },
  parseHTML() {
    return [{ tag: 'mark' }]
  },
  renderHTML({ mark }) {
    return ['mark', { class: highlightClass(mark.attrs.color) || 'wb-h-kuning' }, 0]
  },
  addKeyboardShortcuts() {
    return { 'Mod-Shift-h': () => this.editor.commands.toggleMark('highlight', { color: 'kuning' }) }
  },
})

const Superscript = Mark.create({
  name: 'superscript',
  excludes: 'subscript',
  parseHTML: () => [{ tag: 'sup' }],
  renderHTML: () => ['sup', 0],
  addKeyboardShortcuts() {
    return { 'Mod-.': () => this.editor.commands.toggleMark('superscript') }
  },
})

const Subscript = Mark.create({
  name: 'subscript',
  excludes: 'superscript',
  parseHTML: () => [{ tag: 'sub' }],
  renderHTML: () => ['sub', 0],
  addKeyboardShortcuts() {
    return { 'Mod-,': () => this.editor.commands.toggleMark('subscript') }
  },
})

export function setAlign(editor: Editor, value: string) {
  const textAlign = alignSet.has(value) ? value : null
  editor
    .chain()
    .focus()
    .updateAttributes('paragraph', { textAlign })
    .updateAttributes('heading', { textAlign })
    .run()
}

export function setFontSize(editor: Editor, fontSize: string | null) {
  editor.chain().focus().setMark('textStyle', { fontSize }).run()
}
function stepSize(editor: Editor, dir: 1 | -1) {
  const scale = ['kecil', null, 'besar', 'sangat_besar']
  const cur = (editor.getAttributes('textStyle').fontSize as string | undefined) ?? null
  const next = scale[Math.max(0, Math.min(scale.length - 1, scale.indexOf(cur) + dir))] ?? null
  setFontSize(editor, next)
  return true
}

export function clearFormat(editor: Editor) {
  editor.chain().focus().unsetAllMarks().clearNodes().updateAttributes('paragraph', { textAlign: null }).run()
}

// Pintasan yang tidak dibawa StarterKit. onLink dan onToolbar diisi editor (dialog tautan dan Alt+F10 ada di toolbar).
export function formatKeys(hooks: { onLink: () => void; onToolbar: () => void }) {
  return Extension.create({
    name: 'wbFormatKeys',
    addGlobalAttributes() {
      return [
        {
          types: ['paragraph', 'heading'],
          attributes: {
            textAlign: {
              default: null,
              parseHTML: (el) => fromClass(el, 'wb-al-', alignSet),
              renderHTML: (a) => (alignClass(a.textAlign) ? { class: alignClass(a.textAlign) } : {}),
            },
          },
        },
      ]
    },
    addKeyboardShortcuts() {
      const e = this.editor
      const align = (v: string) => () => (setAlign(e, v), true)
      return {
        'Mod-Shift-l': align('left'),
        'Mod-Shift-e': align('center'),
        'Mod-Shift-r': align('right'),
        'Mod-Shift-j': align('justify'),
        'Mod->': () => stepSize(e, 1),
        'Mod-<': () => stepSize(e, -1),
        'Mod-Shift-Enter': () => e.commands.setHorizontalRule(),
        'Mod-k': () => (hooks.onLink(), true),
        'Mod-\\': () => (clearFormat(e), true),
        'Alt-F10': () => (hooks.onToolbar(), true),
      }
    },
  })
}

export const formatMarks = [TextStyle, Highlight, Superscript, Subscript]
