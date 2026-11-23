// Berkas dari Ctrl+V (revisi 36). Tanpa impor supaya test unit Node bisa memuatnya langsung.

const imageExt: Record<string, string> = {
  'image/png': 'png',
  'image/jpeg': 'jpg',
  'image/webp': 'webp',
  'image/gif': 'gif',
}

// YYYYMMDD-HHMMSS menurut jam perangkat, sama dengan jam yang dilihat orang yang menempel.
export function pasteStamp(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}-${p(d.getHours())}${p(d.getMinutes())}${p(d.getSeconds())}`
}

// Berkas dari clipboard. Kosong berarti tempelannya teks, jadi pemanggil membiarkan browser menempelkannya ke
// isian seperti biasa. Tangkapan layar dari clipboard selalu bernama generik (image.png) atau tanpa nama, jadi
// diberi nama tempel-<waktu>.<ext>, dengan -2, -3 untuk beberapa gambar sekaligus. Ekstensi mengikuti jenis
// gambarnya karena backend menolak isi yang tidak cocok dengan ekstensi. Berkas bernama dari penjelajah berkas
// tetap memakai namanya sendiri.
// data cukup berbentuk { files }, bukan DataTransfer, supaya test unit Node (tanpa tipe DOM) bisa memakainya.
export function pastedFiles(data: { files: ArrayLike<File> } | null, now: Date): File[] {
  const files = Array.from(data?.files ?? [])
  let n = 0
  return files.map((f) => {
    const ext = imageExt[f.type]
    if (!ext || (f.name && !/^image\.\w+$/i.test(f.name))) return f
    n++
    const suffix = n > 1 ? `-${n}` : ''
    return new File([f], `tempel-${pasteStamp(now)}${suffix}.${ext}`, { type: f.type })
  })
}
