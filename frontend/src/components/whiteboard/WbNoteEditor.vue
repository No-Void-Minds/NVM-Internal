<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, shallowRef, useId } from 'vue'
import { Editor } from '@tiptap/core'
import StarterKit from '@tiptap/starter-kit'
import type { PMNode } from '../../utils/whiteboard'

// Editor kartu catatan, dimuat hanya saat kartu dibuka untuk diedit (dynamic import), jadi halaman lain dan kartu
// yang tidak diedit tidak membawa TipTap. injectCSS false wajib: TipTap bawaannya menyisipkan tag <style>, yang
// diblokir CSP style-src 'self'. Gaya editor ada di style.css (.wb-editor).
const props = defineProps<{ doc: PMNode; label: string; caret?: { textIndex?: number } | null }>()
const emit = defineEmits<{ input: [doc: PMNode]; close: [] }>()

const host = ref<HTMLElement | null>(null)
const editor = shallowRef<Editor | null>(null)
// Dipakai tombol toolbar untuk aria-pressed; diperbarui di setiap transaksi.
const active = ref<Record<string, boolean>>({})
const linkOpen = ref(false)
const linkDraft = ref('')
const linkError = ref('')
const linkId = useId()

const allowedLink = (url: string) => /^(https?:\/\/|mailto:)/i.test(url.trim())

function syncActive() {
  const e = editor.value
  if (!e) return
  active.value = {
    bold: e.isActive('bold'),
    italic: e.isActive('italic'),
    underline: e.isActive('underline'),
    strike: e.isActive('strike'),
    bulletList: e.isActive('bulletList'),
    orderedList: e.isActive('orderedList'),
    heading: e.isActive('heading', { level: 1 }),
    link: e.isActive('link'),
  }
}

onMounted(() => {
  editor.value = new Editor({
    element: host.value!,
    injectCSS: false,
    content: props.doc,
    // Dobel klik menaruh kursor di titik klik (onCreate); tanpa itu (Enter) di akhir teks.
    autofocus: props.caret?.textIndex !== undefined ? false : 'end',
    onCreate: ({ editor: e }) => {
      const want = props.caret?.textIndex
      if (want === undefined) return
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
    extensions: [
      StarterKit.configure({
        codeBlock: false,
        horizontalRule: false,
        dropcursor: false,
        // Gapcursor memakai animasi kedip tanpa henti; dimatikan supaya tidak ada animasi berulang.
        gapcursor: false,
        heading: { levels: [1, 2, 3] },
        link: {
          openOnClick: false,
          autolink: true,
          defaultProtocol: 'https',
          isAllowedUri: (url) => allowedLink(url),
          HTMLAttributes: { rel: 'noopener noreferrer nofollow', target: '_blank' },
        },
      }),
    ],
    editorProps: {
      attributes: {
        class: 'wb-editor min-h-16 outline-none',
        role: 'textbox',
        'aria-multiline': 'true',
        'aria-label': props.label,
      },
    },
    onUpdate: ({ editor: e }) => emit('input', e.getJSON() as PMNode),
    onTransaction: syncActive,
  })
  syncActive()
})

onBeforeUnmount(() => editor.value?.destroy())

type Command = 'bold' | 'italic' | 'underline' | 'strike' | 'bulletList' | 'orderedList' | 'heading'
function run(cmd: Command) {
  const c = editor.value?.chain().focus()
  if (!c) return
  switch (cmd) {
    case 'bold':
      return c.toggleBold().run()
    case 'italic':
      return c.toggleItalic().run()
    case 'underline':
      return c.toggleUnderline().run()
    case 'strike':
      return c.toggleStrike().run()
    case 'bulletList':
      return c.toggleBulletList().run()
    case 'orderedList':
      return c.toggleOrderedList().run()
    case 'heading':
      return c.toggleHeading({ level: 1 }).run()
  }
}

function openLink() {
  linkDraft.value = (editor.value?.getAttributes('link').href as string | undefined) ?? ''
  linkError.value = ''
  linkOpen.value = true
}

function applyLink() {
  const url = linkDraft.value.trim()
  if (!url) {
    editor.value?.chain().focus().extendMarkRange('link').unsetLink().run()
    linkOpen.value = false
    return
  }
  if (!allowedLink(url)) {
    linkError.value = 'Tautan harus diawali http://, https://, atau mailto:.'
    return
  }
  editor.value?.chain().focus().extendMarkRange('link').setLink({ href: url }).run()
  linkOpen.value = false
}

// Escape keluar dari editor; Escape di dalam isian tautan hanya menutup isian itu.
function onKeydown(e: KeyboardEvent) {
  if (e.key !== 'Escape') return
  e.preventDefault()
  e.stopPropagation()
  if (linkOpen.value) {
    linkOpen.value = false
    editor.value?.commands.focus()
    return
  }
  emit('close')
}

const tools: { cmd: Command; label: string; path: string }[] = [
  { cmd: 'bold', label: 'Tebal', path: 'M7 5h6a3.5 3.5 0 0 1 0 7H7zM7 12h7a3.5 3.5 0 0 1 0 7H7z' },
  { cmd: 'italic', label: 'Miring', path: 'M11 5h6M7 19h6M14 5l-4 14' },
  { cmd: 'underline', label: 'Garis bawah', path: 'M7 5v6a5 5 0 0 0 10 0V5M5 20h14' },
  { cmd: 'strike', label: 'Coret', path: 'M5 12h14M16 7a4 3 0 0 0-8 0c0 4 8 3 8 7a4 3 0 0 1-8 0' },
  { cmd: 'heading', label: 'Subjudul', path: 'M6 5v14M18 5v14M6 12h12' },
  {
    cmd: 'bulletList',
    label: 'Daftar berbutir',
    path: 'M9 6h11M9 12h11M9 18h11M4.5 6h.01M4.5 12h.01M4.5 18h.01',
  },
  {
    cmd: 'orderedList',
    label: 'Daftar bernomor',
    path: 'M10 6h10M10 12h10M10 18h10M4 5h1v4M4 9h2M4 15h2l-2 3h2',
  },
]
const toolBtn =
  'inline-flex size-11 items-center justify-center rounded-card text-ink hover:bg-canvas aria-pressed:bg-accent aria-pressed:text-on-accent focus-visible:outline-2 focus-visible:outline-mark'
</script>

<template>
  <div class="flex flex-col gap-2" @keydown="onKeydown" @pointerdown.stop>
    <div role="toolbar" aria-label="Format teks" class="flex flex-wrap gap-0.5 border-b border-line pb-1">
      <button
        v-for="t in tools"
        :key="t.cmd"
        type="button"
        :class="toolBtn"
        :aria-label="t.label"
        :aria-pressed="active[t.cmd] ? 'true' : 'false'"
        :title="t.label"
        @mousedown.prevent
        @click="run(t.cmd)"
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
      </button>
      <button
        type="button"
        :class="toolBtn"
        aria-label="Tautan"
        :aria-pressed="active.link ? 'true' : 'false'"
        :aria-expanded="linkOpen"
        :aria-controls="linkId"
        title="Tautan"
        @mousedown.prevent
        @click="openLink"
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
            d="M10 14a4 4 0 0 0 5.66 0l3-3a4 4 0 0 0-5.66-5.66l-1 1M14 10a4 4 0 0 0-5.66 0l-3 3a4 4 0 0 0 5.66 5.66l1-1"
          />
        </svg>
      </button>
    </div>
    <form
      v-if="linkOpen"
      :id="linkId"
      class="flex flex-wrap items-start gap-2"
      novalidate
      @submit.prevent="applyLink"
    >
      <label :for="`${linkId}-url`" class="sr-only">Alamat tautan</label>
      <input
        :id="`${linkId}-url`"
        v-model="linkDraft"
        type="url"
        inputmode="url"
        placeholder="https://"
        maxlength="2048"
        :aria-invalid="linkError ? 'true' : undefined"
        :aria-describedby="linkError ? `${linkId}-err` : undefined"
        class="min-h-11 min-w-0 flex-1 basis-40 rounded-card border border-line bg-surface px-2"
      />
      <button
        type="submit"
        class="min-h-11 rounded-card border border-line bg-surface px-3 font-medium hover:bg-canvas"
      >
        Pasang tautan
      </button>
      <p v-if="linkError" :id="`${linkId}-err`" role="alert" class="w-full text-sm text-danger">
        {{ linkError }}
      </p>
    </form>
    <div ref="host" />
  </div>
</template>
