import { defineComponent, h, type PropType, type VNodeChild } from 'vue'
import type { PMNode } from '../../utils/whiteboard'
import { alignClass, colorClass, highlightClass, sizeClass } from '../../utils/wbRich'

// Merender dokumen catatan dari JSON dengan h(), tanpa v-html: teks selalu jadi text node, dan hanya node serta
// mark dari daftar putih yang punya elemen. Server sudah membersihkan isinya; pemeriksaan href di sini lapisan
// kedua bila data lama atau rusak sampai ke klien.

function safeHref(v: unknown): string | null {
  if (typeof v !== 'string') return null
  try {
    const u = new URL(v)
    return u.protocol === 'http:' || u.protocol === 'https:' || u.protocol === 'mailto:' ? v : null
  } catch {
    return null
  }
}

function text(node: PMNode): VNodeChild {
  let out: VNodeChild = node.text ?? ''
  for (const m of node.marks ?? []) {
    switch (m.type) {
      case 'bold':
        out = h('strong', out)
        break
      case 'italic':
        out = h('em', out)
        break
      case 'underline':
        out = h('u', out)
        break
      case 'strike':
        out = h('s', out)
        break
      case 'superscript':
        out = h('sup', out)
        break
      case 'subscript':
        out = h('sub', out)
        break
      case 'textStyle': {
        // Kelas dari daftar tetap (wbFormat.ts); nilai lain tidak menghasilkan kelas.
        const cls = [colorClass(m.attrs?.color), sizeClass(m.attrs?.fontSize)].filter(Boolean).join(' ')
        if (cls) out = h('span', { class: cls }, out)
        break
      }
      case 'highlight': {
        const cls = highlightClass(m.attrs?.color)
        if (cls) out = h('mark', { class: cls }, out)
        break
      }
      case 'code':
        out = h('code', { class: 'rounded-chip bg-canvas px-1 font-mono text-[0.9em]' }, out)
        break
      case 'link': {
        const href = safeHref(m.attrs?.href)
        if (href)
          out = h(
            'a',
            {
              href,
              target: '_blank',
              rel: 'noopener noreferrer nofollow',
              class: 'underline underline-offset-2',
              // Klik tautan tidak memulai seret atau memilih kartu.
              onPointerdown: (e: Event) => e.stopPropagation(),
            },
            out,
          )
        break
      }
    }
  }
  return out
}

function children(node: PMNode): VNodeChild[] {
  return (node.content ?? []).map(render)
}

function render(node: PMNode): VNodeChild {
  switch (node.type) {
    case 'text':
      return text(node)
    case 'hardBreak':
      return h('br')
    case 'paragraph':
      return h('p', { class: ['min-h-[1.5em]', alignClass(node.attrs?.textAlign)] }, children(node))
    case 'heading': {
      const level = node.attrs?.level === 2 ? 2 : node.attrs?.level === 3 ? 3 : 1
      const cls = level === 1 ? 'text-lg font-bold' : level === 2 ? 'text-base font-bold' : 'font-semibold'
      return h(`h${level + 2}`, { class: [cls, alignClass(node.attrs?.textAlign)] }, children(node))
    }
    case 'blockquote':
      return h('blockquote', { class: 'border-l-2 border-line pl-3' }, children(node))
    case 'bulletList':
      return h('ul', { class: 'list-disc pl-5' }, children(node))
    case 'orderedList': {
      const start = typeof node.attrs?.start === 'number' ? node.attrs.start : 1
      return h('ol', { class: 'list-decimal pl-5', start }, children(node))
    }
    case 'horizontalRule':
      return h('hr', { class: 'my-2 border-line' })
    case 'listItem':
      return h('li', children(node))
    default:
      return null
  }
}

export default defineComponent({
  name: 'WbRichText',
  props: { doc: { type: Object as PropType<PMNode>, required: true } },
  setup(props) {
    return () =>
      h('div', { class: 'wb-richtext space-y-1 break-words', 'data-richtext': '' }, children(props.doc))
  },
})
