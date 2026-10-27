// Hanya path di dalam aplikasi yang boleh jadi tujuan setelah login,
// supaya ?kembali= tidak bisa dipakai untuk mengalihkan pengguna ke situs lain.
export function safeRedirect(value: unknown): string | null {
  if (typeof value !== 'string') return null
  if (!value.startsWith('/') || value.startsWith('//') || value.startsWith('/\\')) return null
  return value
}
