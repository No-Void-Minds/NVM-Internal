// Ekspor papan ke PNG dan PDF (W4), bagian murni yang bisa diuji dengan node --test. Penggambarnya ada di
// components/whiteboard/exportBoard.ts.

// Batas kanvas browser: Safari ponsel menolak kanvas di atas sekitar 16,7 juta piksel, dan sisi di atas 16.384 px
// tidak aman di semua peramban. Papan yang lebih besar diekspor dengan skala diperkecil.
export const MAX_EXPORT_PIXELS = 16_000_000
export const MAX_EXPORT_SIDE = 16_384
export const EXPORT_PAD = 32
// Skala bawaan 2x supaya teks tetap tajam saat diperbesar atau dicetak.
export const EXPORT_SCALE = 2

export function exportScale(w: number, h: number, wanted = EXPORT_SCALE): number {
  if (!(w > 0 && h > 0)) return wanted
  return Math.min(wanted, Math.sqrt(MAX_EXPORT_PIXELS / (w * h)), MAX_EXPORT_SIDE / w, MAX_EXPORT_SIDE / h)
}

// PDF satu halaman berisi satu gambar JPEG (DCTDecode, jadi byte JPEG dipakai apa adanya). Ukuran halaman mengikuti
// gambar pada 96 dpi (96 piksel = 72 poin), dibatasi 14.400 poin per sisi (batas halaman PDF).
export function pdfFromJpeg(jpeg: Uint8Array, pxW: number, pxH: number): Uint8Array<ArrayBuffer> {
  const k = Math.min(0.75, 14400 / pxW, 14400 / pxH)
  const pw = (pxW * k).toFixed(2)
  const ph = (pxH * k).toFixed(2)
  const enc = new TextEncoder()
  const parts: Uint8Array[] = []
  const offsets: number[] = []
  let len = 0
  const add = (b: Uint8Array | string) => {
    const u = typeof b === 'string' ? enc.encode(b) : b
    parts.push(u)
    len += u.length
  }
  const obj = (n: number, body: string) => {
    offsets[n] = len
    add(`${n} 0 obj\n${body}\nendobj\n`)
  }
  // "%PDF-1.4", lalu baris komentar berbyte di atas 127 yang menandai berkas biner bagi alat transfer.
  add(
    new Uint8Array([
      0x25, 0x50, 0x44, 0x46, 0x2d, 0x31, 0x2e, 0x34, 0x0a, 0x25, 0xe2, 0xe3, 0xcf, 0xd3, 0x0a,
    ]),
  )
  obj(1, '<< /Type /Catalog /Pages 2 0 R >>')
  obj(2, '<< /Type /Pages /Kids [3 0 R] /Count 1 >>')
  obj(
    3,
    `<< /Type /Page /Parent 2 0 R /MediaBox [0 0 ${pw} ${ph}] /Resources << /XObject << /Im0 4 0 R >> >> /Contents 5 0 R >>`,
  )
  offsets[4] = len
  add(
    `4 0 obj\n<< /Type /XObject /Subtype /Image /Width ${pxW} /Height ${pxH} /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length ${jpeg.length} >>\nstream\n`,
  )
  add(jpeg)
  add('\nendstream\nendobj\n')
  const draw = `q ${pw} 0 0 ${ph} 0 0 cm /Im0 Do Q`
  obj(5, `<< /Length ${draw.length} >>\nstream\n${draw}\nendstream`)
  const xref = len
  add(
    'xref\n0 6\n0000000000 65535 f \n' +
      offsets
        .slice(1)
        .map((o) => `${String(o).padStart(10, '0')} 00000 n \n`)
        .join(''),
  )
  add(`trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n${xref}\n%%EOF\n`)
  const out = new Uint8Array(len)
  let at = 0
  for (const p of parts) {
    out.set(p, at)
    at += p.length
  }
  return out
}

// Nama berkas ekspor dari judul papan: huruf kecil, angka, dan tanda hubung saja.
export function exportName(title: string, ext: 'png' | 'pdf'): string {
  const base =
    title
      .normalize('NFKD')
      .replace(/[̀-ͯ]/g, '')
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '')
      .slice(0, 60) || 'papan'
  return `${base}.${ext}`
}

// Memecah teks menjadi baris selebar maxW menurut measure (lebar piksel). Baris baru di teks tetap baris baru;
// kata yang lebih lebar dari satu baris dipotong per huruf, supaya URL panjang tidak meluap keluar kartu.
export function wrapText(text: string, maxW: number, measure: (s: string) => number): string[] {
  const out: string[] = []
  for (const para of text.split('\n')) {
    let line = ''
    for (const word of para.split(/(?<=\s)/)) {
      if (measure((line + word).trimEnd()) <= maxW) {
        line += word
        continue
      }
      if (line.trim()) {
        out.push(line.trimEnd())
        line = ''
      }
      for (const ch of word) {
        if (line && measure(line + ch) > maxW) {
          out.push(line)
          line = ''
        }
        line += ch
      }
    }
    out.push(line.trimEnd())
  }
  return out
}
