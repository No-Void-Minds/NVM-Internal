import type { ActivityCell } from '../api/activities'

// Jumlah jam yang sudah terisi: satu slot dihitung terisi bila Aktivitas atau Output ada isinya.
export function filledHours(cells: ActivityCell[]): number {
  return cells.filter((c) => c.activity.trim() !== '' || c.output.trim() !== '').length
}

// Status simpan per sel, dipakai kartu di papan task dan halaman Aktivitas.
export const saveLabel: Record<'saving' | 'saved' | 'error', string> = {
  saving: 'Menyimpan...',
  saved: 'Tersimpan',
  error: 'Gagal disimpan',
}
