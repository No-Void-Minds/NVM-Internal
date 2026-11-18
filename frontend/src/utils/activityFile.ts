import { formatBytes } from './attachment'

// Lampiran Aktivitas Harian (revisi 24). Batas 95 MB karena Cloudflare menolak unggahan di atas 100 MB.
export const MAX_ACTIVITY_FILE_BYTES = 95 * 1024 * 1024
export const MAX_ACTIVITY_FILES_PER_SLOT = 10

// Sama dengan daftar izin di backend (attachment.activityRules). Pemeriksaan di sini hanya supaya pengguna
// langsung tahu sebelum menunggu unggahan selesai; keputusan akhirnya tetap di backend.
const allowedExtensions = [
  'jpg',
  'jpeg',
  'png',
  'webp',
  'gif',
  'mp4',
  'mov',
  'webm',
  'pdf',
  'doc',
  'docx',
  'xls',
  'xlsx',
  'ppt',
  'pptx',
  'txt',
  'csv',
  'zip',
]

// MIME gambar dan video ikut dicantumkan supaya di ponsel pemilih berkas menawarkan kamera dan galeri.
export const activityAcceptAttr = [...allowedExtensions.map((e) => `.${e}`), 'image/*', 'video/*'].join(',')

export const activityAllowedText =
  'Gambar, video, PDF, dokumen Office, TXT, CSV, atau ZIP. Maksimal 95 MB per berkas, 10 berkas per jam.'

export function activityFilePrecheck(file: File): string | null {
  const dot = file.name.lastIndexOf('.')
  const ext = dot > 0 ? file.name.slice(dot + 1).toLowerCase() : ''
  if (!allowedExtensions.includes(ext)) {
    return `Jenis berkas tidak didukung: ${file.name}. Yang didukung: gambar, video, PDF, dokumen Office, TXT, CSV, dan ZIP.`
  }
  if (file.size === 0) return `${file.name} kosong.`
  if (file.size > MAX_ACTIVITY_FILE_BYTES) {
    return `${file.name} berukuran ${formatBytes(file.size)}, melebihi batas 95 MB.`
  }
  return null
}

export function isVideo(mime: string): boolean {
  return mime.startsWith('video/')
}

// Nama berkas tangkapan layar yang ditempel (Ctrl+V): jam dan nomor urut supaya mudah dibedakan. Ekstensi
// mengikuti jenis gambarnya, karena backend menolak isi yang tidak cocok dengan ekstensi.
export function pasteName(time: string, index: number, ext = 'png'): string {
  const jam = time.slice(0, 5).replace(':', '.')
  return `tangkapan-${jam}-${index}.${ext}`
}
