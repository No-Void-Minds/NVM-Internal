// Penggambar ekspor papan ke kanvas (W4). Digambar ulang dari data kartu, bukan tangkapan DOM, karena kartu di luar
// layar tidak di-mount (culling) dan tangkapan DOM butuh pustaka tambahan. Hasilnya teks polos per baris (format
// huruf di dalam paragraf tidak ikut), gambar dari thumbnail, tabel sebagai kisi, goresan, dan garis penghubung.
// Warna selalu dari token mode terang, supaya PNG dan PDF tetap layak dicetak walau aplikasi sedang di mode gelap.
import { fileUrl } from '../../api/whiteboard'
import { EXPORT_PAD, EXPORT_SCALE, exportScale, wrapText } from '../../utils/wbExport'
import {
  bounds,
  cardLines,
  edgePoint,
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
    if (card.kind === 'image') need = card.h
    return {
      card,
      text,
      rect: { x: card.x, y: card.y, w: card.w, h: Math.max(card.h, card.kind === 'drawing' ? 0 : need) },
    }
  })
  const box = bounds(laid.map((l) => l.rect))
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
  ctx.lineWidth = 2
  for (const c of connectors) {
    const a = c.from ? geom.get(c.from) : undefined
    const b = c.to ? geom.get(c.to) : undefined
    if (!a || !b) continue
    const p = edgePoint(a, { x: b.x + b.w / 2, y: b.y + b.h / 2 })
    const q = edgePoint(b, { x: a.x + a.w / 2, y: a.y + a.h / 2 })
    const color = ink(tok, c.style.color || 'tinta')
    ctx.strokeStyle = color
    ctx.fillStyle = color
    ctx.setLineDash(c.style.line === 'dashed' ? [8, 6] : c.style.line === 'dotted' ? [2, 5] : [])
    ctx.beginPath()
    ctx.moveTo(p.x, p.y)
    ctx.lineTo(q.x, q.y)
    ctx.stroke()
    ctx.setLineDash([])
    // Ujung selain "tidak ada" digambar sebagai panah, sama dengan lapisan garis di layar.
    for (const [tip, from, kind] of [
      [q, p, c.style.end],
      [p, q, c.style.start],
    ] as const) {
      if (kind === 'none') continue
      const len = Math.hypot(tip.x - from.x, tip.y - from.y) || 1
      const ux = (tip.x - from.x) / len
      const uy = (tip.y - from.y) / len
      const bx = tip.x - ux * 12
      const by = tip.y - uy * 12
      ctx.beginPath()
      ctx.moveTo(tip.x, tip.y)
      ctx.lineTo(bx - uy * 6, by + ux * 6)
      ctx.lineTo(bx + uy * 6, by - ux * 6)
      ctx.closePath()
      ctx.fill()
    }
    if (c.label) {
      ctx.font = fontOf(tok, 13)
      const mx = (p.x + q.x) / 2
      const my = (p.y + q.y) / 2
      const tw = ctx.measureText(c.label).width
      ctx.fillStyle = tok('--color-surface')
      ctx.fillRect(mx - tw / 2 - 4, my - 10, tw + 8, 20)
      ctx.fillStyle = tok('--color-ink')
      ctx.textBaseline = 'middle'
      ctx.fillText(c.label, mx - tw / 2, my)
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

export function canvasBlob(canvas: HTMLCanvasElement, type: string, quality?: number): Promise<Blob | null> {
  return new Promise((resolve) => canvas.toBlob(resolve, type, quality))
}
