// Penggambar ekspor papan ke kanvas (W4). Digambar ulang dari data kartu, bukan tangkapan DOM, karena kartu di luar
// layar tidak di-mount (culling) dan tangkapan DOM butuh pustaka tambahan. Hasilnya teks polos per baris (format
// huruf di dalam paragraf tidak ikut), gambar dari thumbnail, tabel sebagai kisi, goresan, dan garis penghubung.
// Warna selalu dari token mode terang, supaya PNG dan PDF tetap layak dicetak walau aplikasi sedang di mode gelap.
import { fileUrl } from '../../api/whiteboard'
import { EXPORT_PAD, EXPORT_SCALE, exportScale, wrapText } from '../../utils/wbExport'
import { openShape, rotatedBounds, routeConnector, shapePath, type ShapeContent } from '../../utils/wbShapes'
import {
  bounds,
  cardLines,
  type CaptionContent,
  type Card,
  type Connector,
  type DrawingContent,
  type Rect,
  type TableContent,
} from '../../utils/whiteboard'
import { penColors } from './wbColors'

const PAD = 12
const RADIUS = 4
const cardBg: Record<string, string> = {
  abu: '--color-requested-bg',
  hijau: '--color-progress-bg',
  kuning: '--color-ready-bg',
  biru: '--color-completed-bg',
  krem: '--color-tag',
  merah: '--color-danger-bg',
}

// Token mode terang: atribut tema gelap dilepas sebentar di dalam satu task, jadi tidak ada repaint di antaranya.
function lightTokens(): (name: string) => string {
  const root = document.documentElement
  const theme = root.getAttribute('data-theme')
  if (theme) root.removeAttribute('data-theme')
  const cs = getComputedStyle(root)
  const cache = new Map<string, string>()
  const names = [
    '--color-canvas',
    '--color-surface',
    '--color-ink',
    '--color-ink-muted',
    '--color-line',
    ...Object.values(cardBg),
    ...penColors.map((p) => p.css.slice(4, -1)),
  ]
  for (const n of names) cache.set(n, cs.getPropertyValue(n).trim())
  const font = getComputedStyle(document.body).fontFamily
  if (theme) root.setAttribute('data-theme', theme)
  cache.set('font', font)
  return (n) => cache.get(n) || '#1a1c20'
}

function ink(tok: (n: string) => string, value: string) {
  const css = penColors.find((c) => c.value === value)?.css ?? 'var(--color-ink)'
  return tok(css.slice(4, -1))
}

type Laid = {
  card: Card
  rect: Rect
  text: { text: string; size: number; bold?: boolean; muted?: boolean }[]
}

function fontOf(tok: (n: string) => string, size: number, bold?: boolean) {
  return `${bold ? '700 ' : ''}${size}px ${tok('font')}`
}

function loadImage(src: string): Promise<HTMLImageElement | null> {
  return new Promise((resolve) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = () => resolve(null)
    img.src = src
  })
}

export type ExportResult = { canvas: HTMLCanvasElement; scale: number; reduced: boolean }

export async function renderBoard(cards: Card[], connectors: Connector[]): Promise<ExportResult | null> {
  const tok = lightTokens()
  const measure = document.createElement('canvas').getContext('2d')
  if (!measure) return null

  // Tinggi kartu teks mengikuti isinya, seperti di layar (min-height), supaya teks panjang tidak terpotong.
  const laid: Laid[] = cards.map((card) => {
    const text: Laid['text'] = []
    for (const l of cardLines(card)) {
      measure.font = fontOf(tok, l.size, l.bold)
      for (const t of wrapText(l.text, Math.max(20, card.w - PAD * 2), (x) => measure.measureText(x).width))
        text.push({ ...l, text: t })
    }
    let need = PAD * 2 + text.reduce((n, l) => n + l.size * 1.4, 0)
    if (card.kind === 'table') need += ((card.content as TableContent).rows.length || 1) * 28
    if (card.kind === 'image' || card.kind === 'shape') need = card.h
    return {
      card,
      text,
      rect: { x: card.x, y: card.y, w: card.w, h: Math.max(card.h, card.kind === 'drawing' ? 0 : need) },
    }
  })
  const rot = new Map(
    laid.flatMap((l) =>
      l.card.kind === 'shape' ? [[l.card.id, (l.card.content as ShapeContent).rotation || 0]] : [],
    ) as [string, number][],
  )
  // Bentuk yang diputar bisa keluar dari kotak tegaknya; batas gambar memakai kotak setelah diputar.
  const box = bounds(laid.map((l) => rotatedBounds(l.rect, rot.get(l.card.id) ?? 0)))
  if (!box) return null
  const w = box.w + EXPORT_PAD * 2
  const h = box.h + EXPORT_PAD * 2
  const scale = exportScale(w, h)
  const canvas = document.createElement('canvas')
  canvas.width = Math.max(1, Math.floor(w * scale))
  canvas.height = Math.max(1, Math.floor(h * scale))
  const ctx = canvas.getContext('2d')
  if (!ctx) return null
  ctx.scale(scale, scale)
  ctx.translate(EXPORT_PAD - box.x, EXPORT_PAD - box.y)
  ctx.fillStyle = tok('--color-canvas')
  ctx.fillRect(box.x - EXPORT_PAD, box.y - EXPORT_PAD, w, h)

  const geom = new Map(laid.map((l) => [l.card.id, l.rect]))
  // Jalur, ujung, dan posisi label sama dengan lapisan garis di layar (routeConnector).
  for (const c of connectors) {
    const r = routeConnector(c, geom, rot)
    if (!r) continue
    const color = ink(tok, c.style.color || 'tinta')
    ctx.strokeStyle = color
    ctx.lineWidth = 2
    ctx.lineJoin = 'round'
    ctx.setLineDash(c.style.line === 'dashed' ? [8, 6] : c.style.line === 'dotted' ? [2, 5] : [])
    ctx.stroke(new Path2D(r.d))
    ctx.setLineDash([])
    for (const h of r.heads) {
      const path = new Path2D(h.d)
      if (h.fill !== 'none') {
        ctx.fillStyle = h.fill === 'line' ? color : tok('--color-surface')
        ctx.fill(path)
      }
      ctx.stroke(path)
    }
    for (const [text, at, size] of [
      [c.label, r.mid, 13],
      [c.label_start, { x: r.near.start.x, y: r.near.start.y - 12 }, 12],
      [c.label_end, { x: r.near.end.x, y: r.near.end.y - 12 }, 12],
    ] as const) {
      if (!text) continue
      ctx.font = fontOf(tok, size)
      const tw = ctx.measureText(text).width
      ctx.fillStyle = tok('--color-canvas')
      ctx.fillRect(at.x - tw / 2 - 4, at.y - size * 0.75, tw + 8, size * 1.5)
      ctx.fillStyle = tok('--color-ink')
      ctx.textBaseline = 'middle'
      ctx.fillText(text, at.x - tw / 2, at.y)
    }
  }

  const images = await Promise.all(
    laid.map((l) =>
      l.card.kind === 'image' && l.card.file?.thumb
        ? loadImage(fileUrl(l.card.file.id, true))
        : Promise.resolve(null),
    ),
  )

  const order = laid.map((_, i) => i).sort((i, j) => laid[i]!.card.z - laid[j]!.card.z)
  for (const i of order) {
    const { card, rect, text } = laid[i]!
    if (card.kind === 'drawing') {
      ctx.lineCap = 'round'
      ctx.lineJoin = 'round'
      for (const s of (card.content as DrawingContent).strokes) {
        ctx.strokeStyle = ink(tok, s.color)
        ctx.lineWidth = s.width
        ctx.beginPath()
        for (let k = 0; k + 1 < s.points.length; k += 2) {
          const x = card.x + s.points[k]!
          const y = card.y + s.points[k + 1]!
          if (k) ctx.lineTo(x, y)
          else ctx.moveTo(x, y)
        }
        if (s.points.length === 2) ctx.lineTo(card.x + s.points[0]! + 0.1, card.y + s.points[1]!)
        ctx.stroke()
      }
      continue
    }
    if (card.kind === 'shape') {
      drawShape(ctx, tok, card.content as ShapeContent, rect)
      continue
    }
    ctx.beginPath()
    ctx.roundRect(rect.x, rect.y, rect.w, rect.h, RADIUS)
    ctx.fillStyle = tok(card.color ? cardBg[card.color]! : '--color-surface')
    ctx.fill()
    ctx.strokeStyle = tok('--color-line')
    ctx.lineWidth = 1
    ctx.stroke()

    let y = rect.y + PAD
    const img = images[i]
    if (img) {
      const cap = (card.content as CaptionContent).caption ? 48 : 0
      const ih = Math.max(20, rect.h - cap)
      const k = Math.min(rect.w / img.width, ih / img.height)
      ctx.save()
      ctx.beginPath()
      ctx.roundRect(rect.x, rect.y, rect.w, rect.h, RADIUS)
      ctx.clip()
      ctx.drawImage(
        img,
        rect.x + (rect.w - img.width * k) / 2,
        rect.y + (ih - img.height * k) / 2,
        img.width * k,
        img.height * k,
      )
      ctx.restore()
      y = rect.y + ih + 8
    }

    ctx.save()
    ctx.beginPath()
    ctx.rect(rect.x, rect.y, rect.w, rect.h)
    ctx.clip()
    ctx.textBaseline = 'top'
    for (const l of text) {
      ctx.font = fontOf(tok, l.size, l.bold)
      ctx.fillStyle = tok(l.muted ? '--color-ink-muted' : '--color-ink')
      ctx.fillText(l.text, rect.x + PAD, y)
      y += l.size * 1.4
    }
    if (card.kind === 'table') {
      const rows = (card.content as TableContent).rows
      const header = (card.content as TableContent).header
      const cols = Math.max(1, ...rows.map((r) => r.length))
      const cw = (rect.w - PAD * 2) / cols
      ctx.strokeStyle = tok('--color-line')
      rows.forEach((row, r) => {
        for (let c = 0; c < cols; c++) {
          const x = rect.x + PAD + c * cw
          ctx.strokeRect(x, y, cw, 28)
          ctx.font = fontOf(tok, 13, header && r === 0)
          ctx.fillStyle = tok('--color-ink')
          const cell = wrapText(row[c] ?? '', cw - 8, (x) => ctx.measureText(x).width)[0] ?? ''
          ctx.fillText(cell, x + 4, y + 7)
        }
        y += 28
      })
    }
    ctx.restore()
  }
  return { canvas, scale, reduced: scale < EXPORT_SCALE }
}

const shapeFill: Record<string, string> = { putih: '--color-surface', ...cardBg }

// Bentuk diputar terhadap pusatnya, digambar dari path yang sama dengan layar, lalu teksnya dipecah per baris di
// tengah bentuk. Kelas UML: nama tebal, garis pemisah, atribut, garis pemisah, metode.
function drawShape(ctx: CanvasRenderingContext2D, tok: (n: string) => string, c: ShapeContent, r: Box) {
  const sw = c.stroke_width
  ctx.save()
  ctx.translate(r.x + r.w / 2, r.y + r.h / 2)
  ctx.rotate(((c.rotation || 0) * Math.PI) / 180)
  ctx.translate(-r.w / 2, -r.h / 2)
  const path = new Path2D(shapePath(c.shape, Math.max(1, r.w - sw), Math.max(1, r.h - sw)))
  ctx.save()
  ctx.translate(sw / 2, sw / 2)
  if (!openShape(c.shape) && c.fill) {
    ctx.fillStyle = tok(shapeFill[c.fill] ?? '--color-surface')
    ctx.fill(path)
  }
  ctx.strokeStyle = ink(tok, c.stroke)
  ctx.lineWidth = sw
  ctx.lineJoin = 'round'
  ctx.lineCap = 'round'
  ctx.setLineDash(c.stroke_style === 'dashed' ? [sw * 4, sw * 3] : [])
  ctx.stroke(path)
  ctx.setLineDash([])
  ctx.restore()
  ctx.fillStyle = tok('--color-ink')
  ctx.textBaseline = 'top'
  const fs = c.font_size
  const lh = fs * 1.3
  const fit = (t: string, maxW: number) => wrapText(t, maxW, (x) => ctx.measureText(x).width)
  if (c.shape === 'uml_class') {
    let y = 6
    ctx.font = fontOf(tok, fs, true)
    for (const l of fit(c.name || 'Kelas', r.w - 16)) {
      ctx.fillText(l, (r.w - ctx.measureText(l).width) / 2, y)
      y += lh
    }
    ctx.font = fontOf(tok, fs)
    for (const part of [c.attributes ?? '', c.methods ?? '']) {
      y += 4
      ctx.strokeStyle = ink(tok, c.stroke)
      ctx.lineWidth = sw
      ctx.beginPath()
      ctx.moveTo(0, y)
      ctx.lineTo(r.w, y)
      ctx.stroke()
      y += 6
      for (const l of fit(part, r.w - 16)) {
        if (y + lh > r.h) break
        ctx.fillText(l, 8, y)
        y += lh
      }
    }
  } else if (c.text) {
    ctx.font = fontOf(tok, fs)
    const lines = fit(c.text, r.w * 0.7)
    let y = (r.h - lines.length * lh) / 2
    for (const l of lines) {
      const tw = ctx.measureText(l).width
      const x = c.align === 'left' ? r.w * 0.15 : c.align === 'right' ? r.w * 0.85 - tw : (r.w - tw) / 2
      ctx.fillText(l, x, y)
      y += lh
    }
  }
  ctx.restore()
}

type Box = { x: number; y: number; w: number; h: number }

export function canvasBlob(canvas: HTMLCanvasElement, type: string, quality?: number): Promise<Blob | null> {
  return new Promise((resolve) => canvas.toBlob(resolve, type, quality))
}
