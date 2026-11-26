// Kartu anggota tim Whiteboard (revisi 51). Fungsi murni untuk node --test. Kartu hanya menyimpan user_id dan
// tampilan; nama, foto, jabatan, dan divisi datang dari server (card.person) dan tidak pernah disalin ke isi kartu.

export type PersonView = 'foto_nama' | 'foto_nama_jabatan' | 'lengkap'
export type PersonSize = 'kecil' | 'sedang' | 'besar'
export type PersonLayout = 'atas' | 'samping'
export type PersonContent = { user_id: number; view: PersonView; size: PersonSize; layout: PersonLayout }
// Hanya data yang sudah terlihat di aplikasi: tanpa email, username, atau peran.
export type PersonInfo = {
  id: number
  name: string
  job_title: string | null
  division_id?: number
  division: string
  active: boolean
  avatar_url: string | null
}

export const PERSON_VIEWS: { value: PersonView; label: string }[] = [
  { value: 'foto_nama', label: 'Foto dan nama' },
  { value: 'foto_nama_jabatan', label: 'Foto, nama, jabatan' },
  { value: 'lengkap', label: 'Lengkap (dengan divisi)' },
]
export const PERSON_SIZES: { value: PersonSize; label: string }[] = [
  { value: 'kecil', label: 'Kecil' },
  { value: 'sedang', label: 'Sedang' },
  { value: 'besar', label: 'Besar' },
]
export const PERSON_LAYOUTS: { value: PersonLayout; label: string }[] = [
  { value: 'samping', label: 'Foto di samping' },
  { value: 'atas', label: 'Foto di atas' },
]

export const DEFAULT_PERSON: Omit<PersonContent, 'user_id'> = {
  view: 'foto_nama_jabatan',
  size: 'sedang',
  layout: 'samping',
}

// Ukuran foto (px dunia) per ukuran kartu; foto selalu persegi dengan object-fit cover.
export const AVATAR_PX: Record<PersonSize, number> = { kecil: 40, sedang: 56, besar: 80 }

// personSize: ukuran kartu yang pas untuk tampilan itu; dipakai saat kartu dibuat dan saat tampilannya diganti.
export function personSize(c: Omit<PersonContent, 'user_id'>): { w: number; h: number } {
  const lines = c.view === 'foto_nama' ? 1 : c.view === 'foto_nama_jabatan' ? 2 : 3
  const a = AVATAR_PX[c.size]
  const line = c.size === 'kecil' ? 18 : c.size === 'sedang' ? 20 : 24
  if (c.layout === 'samping') {
    const w = c.size === 'kecil' ? 200 : c.size === 'sedang' ? 260 : 320
    return { w, h: Math.max(a, lines * line) + 26 }
  }
  const w = c.size === 'kecil' ? 150 : c.size === 'sedang' ? 180 : 220
  return { w, h: a + lines * line + 34 }
}

// personLines: baris teks kartu (nama, jabatan, divisi, status nonaktif) sesuai tampilan. Status nonaktif selalu
// berupa teks, bukan hanya warna.
export function personLines(c: PersonContent, p: PersonInfo | undefined): string[] {
  if (!p) return ['Pengguna tidak ditemukan']
  const out = [p.name]
  if (c.view !== 'foto_nama' && p.job_title) out.push(p.job_title)
  if (c.view === 'lengkap' && p.division) out.push(p.division)
  if (!p.active) out.push('Nonaktif')
  return out
}

export function personLabel(_c: PersonContent, p: PersonInfo | undefined): string {
  if (!p) return 'Orang: pengguna tidak ditemukan'
  return `Orang: ${[p.name, p.job_title, p.division, p.active ? '' : 'nonaktif'].filter(Boolean).join(', ')}`
}

// gridAt: posisi kiri atas setiap kartu baru, berjajar rapi (kolom sebanyak akar jumlah dibulatkan ke atas) dan
// berpusat di titik tujuan.
export function gridAt(
  n: number,
  size: { w: number; h: number },
  center: { x: number; y: number },
  gap = 24,
) {
  const cols = Math.max(1, Math.ceil(Math.sqrt(n)))
  const rows = Math.ceil(n / cols)
  const totalW = cols * size.w + (cols - 1) * gap
  const totalH = rows * size.h + (rows - 1) * gap
  return Array.from({ length: n }, (_, i) => ({
    x: Math.round(center.x - totalW / 2 + (i % cols) * (size.w + gap)),
    y: Math.round(center.y - totalH / 2 + Math.floor(i / cols) * (size.h + gap)),
  }))
}
