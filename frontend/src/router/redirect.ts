// Tujuan setelah login atau ganti password paksa (?kembali=). Yang diterima hanya path di dalam aplikasi, atau URL
// https ke host yang tercantum di VITE_ALLOWED_RETURN_HOSTS (dashboard dan studio yang dijaga forward_auth, revisi 5).
// Selain itu diabaikan, supaya ?kembali= tidak bisa dipakai mengalihkan pengguna ke situs lain.
export type ReturnTarget = { internal: string } | { external: string }

// Berkas ini sengaja tanpa import, supaya bisa diuji langsung dengan Node di luar Vite (import.meta.env kosong di sana).
const configuredHosts = (import.meta.env?.VITE_ALLOWED_RETURN_HOSTS ?? '')
  .split(',')
  .map((h: string) => h.trim().toLowerCase())
  .filter(Boolean)

export function returnTarget(value: unknown, allowedHosts: string[] = configuredHosts): ReturnTarget | null {
  if (typeof value !== 'string' || value === '') return null
  if (value.startsWith('/')) {
    // "//host" dan "/\host" dibaca browser sebagai alamat situs lain.
    if (value.startsWith('//') || value.startsWith('/\\')) return null
    // "/" adalah halaman login itu sendiri; kembali ke sana akan berputar tanpa henti.
    if (value === '/' || value.startsWith('/?') || value.startsWith('/#')) return null
    return { internal: value }
  }
  let url: URL
  try {
    url = new URL(value)
  } catch {
    return null
  }
  // Port dan userinfo ditolak supaya yang lolos hanya host yang persis tercantum, lewat port HTTPS bawaan.
  if (url.protocol !== 'https:' || url.username || url.password || url.port) return null
  return allowedHosts.includes(url.hostname.toLowerCase()) ? { external: url.href } : null
}
