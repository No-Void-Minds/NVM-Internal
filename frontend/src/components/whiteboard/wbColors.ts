// Warna pena gambar bebas dan garis penghubung. Nama warna sama dengan warna kartu (backend menerima nama yang sama),
// tetapi digambar dengan token teks (fg) yang gelap, bukan token latar kartu yang pucat, supaya kontrasnya
// terhadap kanvas dan kartu putih minimal 3:1 (elemen non-teks).
export const penColors: { value: string; label: string; css: string }[] = [
  { value: 'tinta', label: 'tinta', css: 'var(--color-ink)' },
  { value: 'abu', label: 'abu', css: 'var(--color-requested-fg)' },
  { value: 'hijau', label: 'hijau', css: 'var(--color-progress-fg)' },
  { value: 'kuning', label: 'kuning', css: 'var(--color-ready-fg)' },
  { value: 'biru', label: 'biru', css: 'var(--color-collab-2)' },
  { value: 'krem', label: 'krem', css: 'var(--color-high)' },
  { value: 'merah', label: 'merah', css: 'var(--color-danger)' },
]

export function inkColor(value: string): string {
  return penColors.find((c) => c.value === value)?.css ?? 'var(--color-ink)'
}
