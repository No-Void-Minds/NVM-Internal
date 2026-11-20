import type { ActivityCell } from '../api/activities'

// Jumlah jam yang sudah terisi, sama dengan hitungan backend: Aktivitas, Output, atau berkas. Jam tersambung
// membawa isi jam kepalanya dari server, jadi ikut terhitung (revisi 33).
export function filledHours(cells: ActivityCell[]): number {
  return cells.filter((c) => c.activity.trim() !== '' || c.output.trim() !== '' || c.files.length > 0).length
}

// Jumlah baris satu rantai, dihitung dari jam kepalanya. 1 untuk jam yang tidak tersambung.
export function chainLength(cells: ActivityCell[], slot: number): number {
  return cells.filter((c) => c.head === slot).length
}

// Status simpan per sel, dipakai kartu di papan task dan halaman Aktivitas.
export const saveLabel: Record<'saving' | 'saved' | 'error', string> = {
  saving: 'Menyimpan...',
  saved: 'Tersimpan',
  error: 'Gagal disimpan',
}
