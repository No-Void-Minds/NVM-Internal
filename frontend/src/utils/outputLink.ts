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
  return href ? kindOf(href) : null
}

function kindOf(href: string): OutputKind {
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

export type OutputPart =
  { type: 'text'; text: string } | ({ type: 'link'; href: string; url: string } & OutputKind)

// Kandidat URL berhenti di spasi, baris baru, dan koma (pemisah yang dipakai tim di Output), juga di kutip dan
// kurung sudut yang tidak pernah jadi bagian tautan yang ditempel.
const urlPattern = /https?:\/\/[^\s,<>"']+/gi

// Tanda baca di ujung kalimat ikut tertangkap pola di atas, jadi dibuang dari ujung URL. Kurung tutup hanya
// dibuang bila tidak punya pasangan di dalam URL, supaya tautan seperti .../Foo_(bar) tetap utuh.
function trimTail(raw: string): string {
  let u = raw
  for (;;) {
    const last = u.at(-1)
    if (last && '.;:!?'.includes(last)) u = u.slice(0, -1)
    else if (last === ')' && (u.match(/\(/g)?.length ?? 0) < (u.match(/\)/g)?.length ?? 0)) u = u.slice(0, -1)
    else if (last === ']' && (u.match(/\[/g)?.length ?? 0) < (u.match(/]/g)?.length ?? 0)) u = u.slice(0, -1)
    else return u
  }
}

// Revisi 35: memecah Output menjadi teks dan tautan sesuai urutan aslinya. Setiap URL http atau https jadi
// tautan berlabel (File Drive, Spreadsheet, atau nama host); label yang sama diberi nomor mulai dari yang kedua.
// Potongan teks yang isinya hanya pemisah (spasi, baris baru, koma) dibuang, supaya deretan tautan tidak
// diselingi koma lepas. Isi Output yang tersimpan tidak diubah; ini hanya untuk tampilan.
export function outputParts(value: string): OutputPart[] {
  const parts: OutputPart[] = []
  const seen = new Map<string, number>()
  let at = 0
  const pushText = (text: string) => {
    if (text.replace(/[\s,]/g, '') !== '') parts.push({ type: 'text', text })
  }
  for (const m of value.matchAll(urlPattern)) {
    const raw = trimTail(m[0])
    const start = m.index
    const href = outputLink(raw)
    if (!href) continue
    pushText(value.slice(at, start))
    const kind = kindOf(href)
    const n = (seen.get(kind.label) ?? 0) + 1
    seen.set(kind.label, n)
    parts.push({ type: 'link', href, url: raw, ...kind, label: n > 1 ? `${kind.label} ${n}` : kind.label })
    at = start + raw.length
  }
  pushText(value.slice(at))
  return parts
}
