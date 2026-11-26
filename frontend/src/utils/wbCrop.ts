// Potongan gambar kartu (revisi 50): pecahan 0 sampai 1 dari sisi berkas asli. Mode "atur posisi dan zoom" memakai
// potongan yang sama dengan perbandingan sisi bingkai kartu, jadi kartu dan ekspor cukup membaca satu parameter.
// Fungsi murni untuk node --test.
import type { Crop } from './whiteboard.ts'

// frameCrop: potongan dari zoom (1 sampai 5) dan posisi (0 sampai 1) di bingkai berperbandingan fa = lebar / tinggi.
export function frameCrop(iw: number, ih: number, fa: number, zoom: number, px: number, py: number): Crop {
  const ia = iw / ih
  let w = 1
  let h = 1
  // Pada zoom 1 gambar menutupi bingkai (seperti object-fit cover): sisi yang lebih panjang terpotong.
  if (ia > fa) w = fa / ia
  else h = ia / fa
  w /= zoom
  h /= zoom
  return { x: px * (1 - w), y: py * (1 - h), w, h }
}

// frameOf: kebalikan frameCrop, untuk membuka dialog dari potongan yang tersimpan.
export function frameOf(iw: number, ih: number, fa: number, crop: Crop | null | undefined) {
  if (!crop) return { zoom: 1, px: 0.5, py: 0.5 }
  const base = frameCrop(iw, ih, fa, 1, 0, 0)
  const zoom = Math.min(5, Math.max(1, base.w / crop.w))
  return {
    zoom,
    px: crop.w < 1 ? crop.x / (1 - crop.w) : 0.5,
    py: crop.h < 1 ? crop.y / (1 - crop.h) : 0.5,
  }
}

// cropBox: posisi gambar utuh di dalam bingkai (persen), dipakai kartu: bingkai menampilkan bagian crop saja.
export function cropBox(crop: Crop) {
  return {
    width: `${100 / crop.w}%`,
    height: `${100 / crop.h}%`,
    left: `${(-crop.x / crop.w) * 100}%`,
    top: `${(-crop.y / crop.h) * 100}%`,
  }
}

// clampCrop menjaga potongan di dalam gambar dan minimal 2% sisinya (aturan server).
export function clampCrop(c: Crop): Crop {
  const w = Math.min(1, Math.max(0.02, c.w))
  const h = Math.min(1, Math.max(0.02, c.h))
  return { x: Math.min(1 - w, Math.max(0, c.x)), y: Math.min(1 - h, Math.max(0, c.y)), w, h }
}
