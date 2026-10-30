export const MAX_ATTACHMENT_BYTES = 50 * 1024 * 1024

// Sama dengan daftar izin di backend (attachment/filetype.go). Pemeriksaan di sini hanya supaya pengguna langsung
// tahu sebelum menunggu unggahan selesai; keputusan akhirnya tetap di backend, karena browser bisa dilewati.
const allowedExtensions = [
  'pdf',
  'csv',
  'txt',
  'doc',
  'docx',
  'xls',
  'xlsx',
  'ppt',
  'pptx',
  'jpg',
  'jpeg',
  'png',
]

// MIME gambar ikut dicantumkan supaya di ponsel pemilih berkas menawarkan kamera dan galeri.
export const acceptAttr = [...allowedExtensions.map((e) => `.${e}`), 'image/jpeg', 'image/png'].join(',')

export const allowedText = 'PDF, CSV, TXT, Word, Excel, PowerPoint, JPG, PNG. Maksimal 50 MB.'

const number = new Intl.NumberFormat('id-ID', { maximumFractionDigits: 1 })

export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${number.format(n / 1024)} KB`
  return `${number.format(n / 1024 / 1024)} MB`
}

export function precheck(file: File): string | null {
  const dot = file.name.lastIndexOf('.')
  const ext = dot > 0 ? file.name.slice(dot + 1).toLowerCase() : ''
  if (!allowedExtensions.includes(ext)) {
    return `Tipe berkas tidak diizinkan: ${file.name}. Gunakan PDF, dokumen Office, CSV, TXT, JPG, atau PNG.`
  }
  if (file.size === 0) return `${file.name} kosong.`
  if (file.size > MAX_ATTACHMENT_BYTES) {
    return `${file.name} berukuran ${formatBytes(file.size)}, melebihi batas 50 MB.`
  }
  return null
}

export function isImage(mime: string): boolean {
  return mime.startsWith('image/')
}
