// Tambah bentuk tersambung (revisi 57d1): letak bentuk baru di sisi bentuk asal tanpa menimpa kartu lain. Bagian murni
// tanpa Vue supaya bisa diuji dengan node --test.

export type Dir = 'right' | 'down' | 'left' | 'up'
type Rect = { x: number; y: number; w: number; h: number }

export const QUICK_GAP = 80
// Port garis untuk tiap arah: keluar dari sisi bentuk asal, masuk dari sisi bentuk baru yang menghadapnya.
export const DIR_PORTS: Record<Dir, { from: string; to: string }> = {
  right: { from: 'right', to: 'left' },
  left: { from: 'left', to: 'right' },
  down: { from: 'bottom', to: 'top' },
  up: { from: 'top', to: 'bottom' },
}
const overlaps = (a: Rect, b: Rect, pad: number) =>
  a.x < b.x + b.w + pad && b.x < a.x + a.w + pad && a.y < b.y + b.h + pad && b.y < a.y + a.h + pad

// Mula-mula QUICK_GAP dari tepi bentuk asal dan sejajar tengahnya. Bila menimpa kartu lain (dengan jarak 16 px),
// geser menjauh searah arahnya per satu ukuran bentuk; sesudah 8 langkah, coba geser tegak lurus bergantian kiri dan
// kanan (atau atas dan bawah) dari letak pertama. Hasil terakhir dipakai bila semua langkah terisi.
export function placeNext(src: Rect, dir: Dir, size: { w: number; h: number }, obstacles: Rect[]): Rect {
  const horiz = dir === 'right' || dir === 'left'
  const sign = dir === 'right' || dir === 'down' ? 1 : -1
  const base: Rect = horiz
    ? {
        x: sign > 0 ? src.x + src.w + QUICK_GAP : src.x - QUICK_GAP - size.w,
        y: src.y + src.h / 2 - size.h / 2,
        ...size,
      }
    : {
        x: src.x + src.w / 2 - size.w / 2,
        y: sign > 0 ? src.y + src.h + QUICK_GAP : src.y - QUICK_GAP - size.h,
        ...size,
      }
  const free = (r: Rect) => !obstacles.some((o) => overlaps(r, o, 16))
  const along = (horiz ? size.w : size.h) + 40
  const across = (horiz ? size.h : size.w) + 40
  const tries: Rect[] = []
  for (let k = 0; k < 8; k++)
    tries.push(horiz ? { ...base, x: base.x + sign * k * along } : { ...base, y: base.y + sign * k * along })
  for (let k = 1; k <= 6; k++)
    for (const s of [1, -1])
      tries.push(horiz ? { ...base, y: base.y + s * k * across } : { ...base, x: base.x + s * k * across })
  const hit = tries.find(free) ?? tries.at(-1)!
  return { x: Math.round(hit.x), y: Math.round(hit.y), w: size.w, h: size.h }
}
