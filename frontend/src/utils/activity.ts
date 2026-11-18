import type { ActivityCell } from '../api/activities'

// Jumlah jam yang sudah terisi: satu slot dihitung terisi bila Aktivitas atau Output ada isinya.
export function filledHours(cells: ActivityCell[]): number {
  return cells.filter((c) => c.activity.trim() !== '' || c.output.trim() !== '').length
}

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

// Nama host tautan output, dipakai sebagai label chip di halaman Aktivitas supaya URL panjang tidak
// merusak lebar kolom.
export function outputHost(value: string): string {
  const href = outputLink(value)
  if (!href) return ''
  try {
    return new URL(href).host
  } catch {
    return ''
  }
}

// Status simpan per sel, dipakai kartu di papan task dan halaman Aktivitas.
export const saveLabel: Record<'saving' | 'saved' | 'error', string> = {
  saving: 'Menyimpan...',
  saved: 'Tersimpan',
  error: 'Gagal disimpan',
}
