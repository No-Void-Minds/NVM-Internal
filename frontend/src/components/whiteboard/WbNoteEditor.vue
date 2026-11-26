<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import { Editor, Extension } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import type { PMNode } from '../../utils/whiteboard'
import WbFormatBar from './WbFormatBar.vue'
import { formatKeys, formatMarks } from './wbFormat'

// Editor teks berformat Whiteboard: isi catatan dan setiap field berformat (revisi 49). Dimuat hanya saat sebuah
// field dibuka untuk diedit (dynamic import), dan hanya satu yang terpasang sekaligus (WbRichField), jadi kartu lain
// dirender dari JSON. injectCSS false wajib: TipTap bawaannya menyisipkan tag <style>, yang diblokir CSP
// style-src 'self'. Gaya editor ada di style.css (.wb-editor). Panel formatnya menempel di bar pilihan (revisi 52).
const props = defineProps<{
  doc: PMNode
  label: string
  caret?: { textIndex?: number } | null
  // Satu baris: Enter tidak membuat paragraf baru tetapi dikirim ke pemilik field (simpan, butir berikutnya).
  singleLine?: boolean
  selectAll?: boolean
  editorClass?: string
  // Aturan ketik Markdown ("+ " jadi daftar, "# " jadi judul). Mati di field kecil dan teks bentuk, karena "+ id"
  // dan "- simpan()" adalah notasi UML dan teks biasa, bukan perintah format; catatan tetap memakainya.
  markdown?: boolean
}>()
const emit = defineEmits<{ input: [doc: PMNode]; close: []; enter: [] }>()

const host = ref<HTMLElement | null>(null)
const editor = shallowRef<Editor | null>(null)
const bar = ref<InstanceType<typeof WbFormatBar> | null>(null)
// Revisi 52: panel format menempel di slot bar pilihan (kiri atas kanvas). Di layar sentuh panel menempel di atas
// keyboard virtual, jadi dipasang di body; tanpa slot (mis. bar pilihan tidak tampil) panel melayang seperti revisi 49.
const slotted = !matchMedia('(pointer: coarse)').matches && !!document.getElementById('wb-format-slot')

const allowedLink = (url: string) => /^(https?:\/\/|mailto:)/i.test(url.trim())

onMounted(() => {
  const enter = () => (emit('enter'), true)
  const extensions = [
    StarterKit.configure({
      codeBlock: false,
      dropcursor: false,
      // Gapcursor memakai animasi kedip tanpa henti; dimatikan supaya tidak ada animasi berulang.
      gapcursor: false,
      hardBreak: props.singleLine ? false : undefined,
      heading: { levels: [1, 2, 3] },
      link: {
        openOnClick: false,
        autolink: true,
        defaultProtocol: 'https',
        isAllowedUri: (url) => allowedLink(url),
        HTMLAttributes: { rel: 'noopener noreferrer nofollow', target: '_blank' },
      },
    }),
    ...formatMarks,
    formatKeys({ onLink: () => bar.value?.openLink(), onToolbar: () => bar.value?.focusFirst() }),
  ]
  if (props.singleLine)
    extensions.push(
      Extension.create({
        name: 'wbSingleLine',
        priority: 1000,
        addKeyboardShortcuts: () => ({ Enter: enter, 'Shift-Enter': enter, 'Mod-Enter': enter }),
      }),
    )
  editor.value = new Editor({
    element: host.value!,
    injectCSS: false,
    enableInputRules: props.markdown !== false,
    content: props.doc,
    // Dobel klik menaruh kursor di titik klik (onCreate); tanpa itu (Enter) di akhir teks.
    autofocus: props.caret?.textIndex !== undefined ? false : props.selectAll ? 'all' : 'end',
    onCreate: ({ editor: e }) => {
      const want = props.caret?.textIndex
      if (want === undefined) {
        // Fokus dan kursor dipasang langsung, bukan menunggu satu frame seperti autofocus TipTap, supaya ketikan cepat
        // sesudah klik pada field yang baru dibuka tidak hilang.
        if (props.selectAll) e.commands.selectAll()
        else e.commands.setTextSelection(e.state.doc.content.size)
        e.view.focus()
        return
      }
      // Indeks karakter teks tampilan dipetakan ke posisi dokumen: jumlah karakter text node sebelumnya.
      let seen = 0
      let at: number | null = null
      e.state.doc.descendants((node, pos) => {
        if (at !== null) return false
        if (!node.isText) return true
        const len = node.text?.length ?? 0
        if (want <= seen + len) at = pos + (want - seen)
        seen += len
        return false
      })
      e.chain()
        .focus(at ?? 'end')
        .run()
    },
    extensions,
    editorProps: {
      attributes: {
        class: `wb-editor outline-none ${props.editorClass ?? 'min-h-16'}`,
        role: 'textbox',
        'aria-multiline': props.singleLine ? 'false' : 'true',
        'aria-label': props.label,
      },
    },
    onUpdate: ({ editor: e }) => emit('input', e.getJSON() as PMNode),
  })
})

onBeforeUnmount(() => editor.value?.destroy())

// Escape keluar dari editor.
function onKeydown(e: KeyboardEvent) {
  if (e.key !== 'Escape') return
  e.preventDefault()
  e.stopPropagation()
  emit('close')
}

defineExpose({ current: () => (editor.value?.getJSON() as PMNode | undefined) ?? null })
</script>

<template>
  <div @keydown="onKeydown" @pointerdown.stop>
    <div ref="host" />
    <Teleport :to="slotted ? '#wb-format-slot' : 'body'">
      <WbFormatBar v-if="editor" ref="bar" :editor="editor" :mode="slotted ? 'slot' : 'float'" />
    </Teleport>
  </div>
</template>
