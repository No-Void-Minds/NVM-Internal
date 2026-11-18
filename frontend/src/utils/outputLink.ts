// Tautan di kolom Output Aktivitas. Berkas ini sengaja tanpa impor supaya test unit Node bisa memuatnya langsung.

// Output yang berupa tautan ditampilkan sebagai tautan. Hanya http dan https, supaya javascript: atau data:
// tidak pernah jadi href. Dipakai tanpa v-html, jadi teks lain tetap teks biasa.
export function outputLink(value: string): string | null {
  const v = value.trim()
  if (v.includes(' ') || v.includes('\n')) return null
  if (!/^https?:\/\//i.test(v)) return null
  try {
    const url = new URL(v)
    return url.protocol === 'http:' || url.protocol === 'https:' ? url.href : null
  } catch {
    return null
  }
}

export type LinkKind = 'drive-file' | 'drive-folder' | 'docs' | 'sheets' | 'slides' | 'host'

export type OutputKind = { kind: LinkKind; label: string; driveFileId: string | null }

const kindLabel: Record<Exclude<LinkKind, 'host'>, string> = {
  'drive-file': 'File Drive',
  'drive-folder': 'Folder Drive',
  docs: 'Google Docs',
  sheets: 'Spreadsheet',
  slides: 'Slide',
}

// Jenis tautan output dikenali dari URL saja, tanpa memanggil jaringan, untuk ikon dan label chip.
// Selain lima pola Google di bawah, labelnya nama host seperti sebelumnya.
export function outputKind(value: string): OutputKind | null {
  const href = outputLink(value)
  if (!href) return null
  const url = new URL(href)
  const path = url.pathname
  const named = (kind: Exclude<LinkKind, 'host'>, driveFileId: string | null = null): OutputKind => ({
    kind,
    label: kindLabel[kind],
    driveFileId,
  })
  if (url.hostname === 'drive.google.com') {
    const file = /^\/file\/d\/([\w-]+)/.exec(path)
    if (file) return named('drive-file', file[1])
    if (/^\/drive\/(u\/\d+\/)?folders\/[\w-]+/.test(path)) return named('drive-folder')
  }
  if (url.hostname === 'docs.google.com') {
    if (path.startsWith('/document/')) return named('docs')
    if (path.startsWith('/spreadsheets/')) return named('sheets')
    if (path.startsWith('/presentation/')) return named('slides')
  }
  return { kind: 'host', label: url.host, driveFileId: null }
}

// Thumbnail Drive hanya muncul bila browser pengguna login Google dan punya akses; selain itu permintaannya
// dialihkan ke halaman login dan <img> gagal, lalu dialog menyembunyikan area gambarnya.
export function driveThumbnail(id: string): string {
  return `https://drive.google.com/thumbnail?id=${encodeURIComponent(id)}&sz=w480`
}
