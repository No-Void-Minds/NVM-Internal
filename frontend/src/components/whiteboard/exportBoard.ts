// Penggambar ekspor papan ke kanvas (W4). Digambar ulang dari data kartu, bukan tangkapan DOM, karena kartu di luar
// layar tidak di-mount (culling) dan tangkapan DOM butuh pustaka tambahan. Hasilnya teks dengan format huruf (tebal,
// miring, garis bawah, coret, warna, stabilo, ukuran, dan perataan; revisi 49), gambar dari thumbnail, tabel sebagai
// kisi, goresan, dan garis penghubung. Sejak revisi 50 juga gaya kartu (latar, garis tepi, sudut, bayangan tegas,
// transparansi, jarak dalam), potongan gambar, dan bentuk yang dibalik.
// Warna selalu dari token mode terang, supaya PNG dan PDF tetap layak dicetak walau aplikasi sedang di mode gelap.
import { fileUrl } from '../../api/whiteboard'
import { EXPORT_PAD, EXPORT_SCALE, exportScale } from '../../utils/wbExport'
import { HIGHLIGHTS, TEXT_COLORS, wrapRuns, type Run } from '../../utils/wbRich'
import {
  openShape,
  rotatedBounds,
  routeConnector,
  shapeMarks,
  shapePath,
  solidShape,
  strokeScale,
  FRAGMENT_TAB,
  classStereotype,
  guardLanes,
  underlineText,
  laneOrient,
  textBox,
  type ShapeContent,
  ERD_HEAD,
  ERD_ROW,
  keyLabel,
  defaultText,
  fitImage,
  frameBox,
  frameImage,
  isFrame,
} from '../../utils/wbShapes'
import { BAND, laneBoxes } from '../../utils/wbContainers'
import { stackOrder } from '../../utils/wbStack'
import {
  bounds,
  cardLines,
  fieldLines,
  type CaptionContent,
  type Card,
  type Connector,
  type DrawingContent,
  type Rect,
  type TableContent,
  type TextLine,
} from '../../utils/whiteboard'
import { penColors } from './wbColors'
import { AVATAR_PX, type PersonContent } from '../../utils/wbPerson'
import { initials } from '../../utils/person'

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
    ...TEXT_COLORS.map((c) => c.token),
    ...HIGHLIGHTS.map((c) => c.token),
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

type Tok = (n: string) => string

// Tampilan kartu dari gaya revisi 50 (utils/wbStyle.ts), dengan nilai yang sama dengan kelas wb-st-* di style.css.
function lookOf(card: Card) {
  const st = card.style ?? {}
  const pen = penColors.find((c) => c.value === st.border_color)
  return {
    alpha: st.opacity && st.opacity < 100 && card.kind !== 'board' ? st.opacity / 100 : 1,
    clear: st.bg === 'transparan' && card.kind !== 'image' && card.kind !== 'shape',
    border: st.border === 'tanpa' ? 0 : st.border === 'sedang' ? 2 : st.border === 'tebal' ? 4 : 1,
    borderColor: pen ? pen.css.slice(4, -1) : '--color-line',
    dash: st.border_style === 'putus' ? [8, 5] : st.border_style === 'titik' ? [2, 4] : [],
    radius: st.radius === 'siku' ? 0 : st.radius === 'sedang' ? 12 : st.radius === 'bulat' ? 24 : RADIUS,
    shadow: st.shadow === 'tegas',
    pad: card.kind === 'image' ? PAD : st.padding === 'rapat' ? 6 : st.padding === 'lega' ? 20 : PAD,
  }
}
// Satu baris hasil pecah: run berformat dan apakah baris terakhir paragrafnya (rata kiri-kanan tidak meregangkannya).
type Drawn = TextLine & { runs: Run[]; last: boolean }
type Laid = { card: Card; rect: Rect; text: Drawn[] }

function fontOf(tok: Tok, size: number, bold?: boolean) {
  return `${bold ? '700 ' : ''}${size}px ${tok('font')}`
}

function runSize(size: number, r: Run) {
  return size * (r.scale ?? 1) * (r.sup || r.sub ? 0.75 : 1)
}
function runFont(tok: Tok, size: number, r: Run, bold?: boolean) {
  const family = r.code ? 'ui-monospace, monospace' : tok('font')
  return `${r.italic ? 'italic ' : ''}${r.bold || bold ? '700 ' : ''}${runSize(size, r)}px ${family}`
}

function wrapLine(ctx: CanvasRenderingContext2D, tok: Tok, l: TextLine, maxW: number): Drawn[] {
  if (l.rule) return [{ ...l, runs: [], last: true }]
  const measure = (r: Run, t: string) => {
    ctx.font = runFont(tok, l.size, r, l.bold)
    return ctx.measureText(t).width
  }
  const lines = wrapRuns(l.runs ?? [{ text: l.text }], maxW, measure)
  return lines.map((runs, i) => ({ ...l, runs, last: i === lines.length - 1 }))
}

function lineHeight(l: Drawn) {
  return l.size * Math.max(1, ...l.runs.map((r) => r.scale ?? 1)) * 1.4
}

// Menggambar satu baris di kotak selebar maxW mulai x, dengan perataan baris itu (atau align bila tidak ada).
function drawLine(
  ctx: CanvasRenderingContext2D,
  tok: Tok,
  l: Drawn,
  x: number,
  y: number,
  maxW: number,
  color: string,
  align = 'left',
) {
  const lh = lineHeight(l)
  if (l.rule) {
    ctx.fillStyle = tok('--color-line')
    ctx.fillRect(x, y + lh / 2, maxW, 1)
    return
  }
  const widths = l.runs.map((r) => {
    ctx.font = runFont(tok, l.size, r, l.bold)
    return ctx.measureText(r.text).width
  })
  const free = Math.max(0, maxW - widths.reduce((a, b) => a + b, 0))
  const al = l.align ?? align
  const spaces =
    al === 'justify' && !l.last ? l.runs.reduce((n, r) => n + (r.text.match(/ /g)?.length ?? 0), 0) : 0
  const gap = spaces ? free / spaces : 0
  let cx = x + (al === 'center' ? free / 2 : al === 'right' ? free : 0)
  const base = y + lh * 0.75
  ctx.textBaseline = 'alphabetic'
  for (const r of l.runs) {
    ctx.font = runFont(tok, l.size, r, l.bold)
    const fs = runSize(l.size, r)
    const by = base + (r.sup ? -fs * 0.5 : r.sub ? fs * 0.25 : 0)
    for (const piece of gap ? r.text.split(/(?<= )/) : [r.text]) {
      const w = ctx.measureText(piece).width
      if (r.highlight) {
        ctx.fillStyle = tok(r.highlight)
        ctx.fillRect(cx, y + 1, w, lh - 2)
      }
      ctx.fillStyle = r.color ? tok(r.color) : color
      ctx.fillText(piece, cx, by)
      if (r.underline) ctx.fillRect(cx, by + Math.max(1, fs * 0.12), w, Math.max(1, fs / 14))
      if (r.strike) ctx.fillRect(cx, by - fs * 0.3, w, Math.max(1, fs / 14))
      cx += w + (piece.endsWith(' ') ? gap : 0)
    }
  }
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
    const pad = lookOf(card).pad
    const text = cardLines(card).flatMap((l) => wrapLine(measure, tok, l, Math.max(20, card.w - pad * 2)))
    let need = pad * 2 + text.reduce((n, l) => n + lineHeight(l), 0)
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
  // Label bentuk kecil (event, gateway, aktor) ada di bawah kotaknya; batas gambar ikut memuatnya.
  const box = bounds(
    laid.flatMap((l) => {
      const rr = rotatedBounds(l.rect, rot.get(l.card.id) ?? 0)
      if (l.card.kind !== 'shape' || !(l.card.content as ShapeContent).text) return [rr]
      const t = textBox((l.card.content as ShapeContent).shape, l.rect.w, l.rect.h)
      return [rr, { x: l.rect.x + t.x, y: l.rect.y + t.y, w: t.w, h: Math.max(t.h, 20) }]
    }),
  )
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
  // Jenis bentuk per kartu: kotak port sama dengan layar (batang fork, lingkaran di kotak tidak persegi).
  const shapeKinds = new Map(
    laid.flatMap((l) =>
      l.card.kind === 'shape' ? [[l.card.id, (l.card.content as ShapeContent).shape]] : [],
    ) as [string, string][],
  )
  // Id baris tabel ERD (revisi 57a): garis yang menempel ke baris tertentu, sama dengan layar.
  const tableRows = new Map(
    laid.flatMap((l) =>
      l.card.kind === 'shape' && (l.card.content as ShapeContent).shape === 'erd_table'
        ? [[l.card.id, ((l.card.content as ShapeContent).rows ?? []).map((r) => r.id)]]
        : [],
    ) as [string, string[]][],
  )
  const images = await Promise.all(
    laid.map((l) =>
      (l.card.kind === 'image' || l.card.kind === 'shape') && l.card.file?.thumb
        ? loadImage(fileUrl(l.card.file.id, true))
        : l.card.kind === 'person' && l.card.person?.avatar_url
          ? loadImage(l.card.person.avatar_url)
          : Promise.resolve(null),
    ),
  )

  // Urutan sama dengan kanvas (revisi 53): garis di antara kartu sesuai z-nya, bukan di bawah semuanya.
  const index = new Map(laid.map((l, i) => [l.card.id, i]))
  const lineOf = new Map(connectors.map((c) => [c.id, c]))
  for (const layer of stackOrder(
    laid.map((l) => l.card),
    connectors,
  )) {
    if (layer.line) {
      // save dan restore: gaya teks garis (baseline, font) tidak bocor ke kartu yang digambar sesudahnya.
      ctx.save()
      drawConnector(lineOf.get(layer.id)!)
      ctx.restore()
      continue
    }
    const i = index.get(layer.id)!
    const { card, rect, text } = laid[i]!
    const look = lookOf(card)
    ctx.save()
    ctx.globalAlpha = look.alpha
    drawCard(i, card, rect, text, look)
    ctx.restore()
  }
  return { canvas, scale, reduced: scale < EXPORT_SCALE }

  // Jalur, ujung, dan posisi label sama dengan garis di layar (routeConnector).
  function drawConnector(c: Connector) {
    if (!ctx) return
    const r = routeConnector(c, geom, rot, undefined, shapeKinds, tableRows)
    if (!r) return
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

  function drawCard(i: number, card: Card, rect: Rect, text: Drawn[], look: ReturnType<typeof lookOf>) {
    if (!ctx) return
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
      return
    }
    if (card.kind === 'shape') {
      if (look.shadow) setShadow(ctx)
      drawShape(ctx, tok, card.content as ShapeContent, rect, card.file ? images[i]! : null)
      return
    }
    ctx.beginPath()
    ctx.roundRect(rect.x, rect.y, rect.w, rect.h, look.radius)
    if (!look.clear) {
      if (look.shadow) setShadow(ctx)
      ctx.fillStyle = tok(card.color ? cardBg[card.color]! : '--color-surface')
      ctx.fill()
      ctx.shadowColor = 'transparent'
    }
    if (look.border) {
      ctx.strokeStyle = tok(look.borderColor)
      ctx.lineWidth = look.border
      ctx.setLineDash(look.dash)
      ctx.stroke()
      ctx.setLineDash([])
    }

    const PAD = look.pad
    if (card.kind === 'person') {
      drawPerson(ctx, tok, card, rect, text, images[i] ?? null, PAD)
      return
    }
    let y = rect.y + PAD
    const img = images[i]
    if (img) {
      const cap = (card.content as CaptionContent).caption ? 48 : 0
      const ih = Math.max(20, rect.h - cap)
      const k = Math.min(rect.w / img.width, ih / img.height)
      ctx.save()
      ctx.beginPath()
      ctx.roundRect(rect.x, rect.y, rect.w, rect.h, look.radius)
      ctx.clip()
      // Potongan (revisi 50): bagian potongan saja yang mengisi area gambar kartu, sama dengan tampilan di layar.
      const crop = (card.content as CaptionContent).crop
      if (crop) {
        ctx.drawImage(
          img,
          crop.x * img.width,
          crop.y * img.height,
          crop.w * img.width,
          crop.h * img.height,
          rect.x,
          rect.y,
          rect.w,
          ih,
        )
      } else
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
    for (const l of text) {
      drawLine(
        ctx,
        tok,
        l,
        rect.x + PAD,
        y,
        rect.w - PAD * 2,
        tok(l.muted ? '--color-ink-muted' : '--color-ink'),
      )
      y += lineHeight(l)
    }
    if (card.kind === 'table') {
      const t = card.content as TableContent
      const rows = t.rows
      const header = t.header
      const cols = Math.max(1, ...rows.map((r) => r.length))
      const cw = (rect.w - PAD * 2) / cols
      ctx.strokeStyle = tok('--color-line')
      rows.forEach((row, r) => {
        for (let c = 0; c < cols; c++) {
          const x = rect.x + PAD + c * cw
          ctx.strokeStyle = tok('--color-line')
          ctx.strokeRect(x, y, cw, 28)
          // Sel hanya satu baris di ekspor: baris pertama sel, dipotong selebar sel.
          const first = fieldLines(row[c] ?? '', t.docs?.[r]?.[c], { size: 13, bold: header && r === 0 })[0]
          const line = first && wrapLine(ctx, tok, first, cw - 8)[0]
          if (line) drawLine(ctx, tok, line, x + 4, y + 4, cw - 8, tok('--color-ink'))
        }
        y += 28
      })
    }
    ctx.restore()
  }
}

// Kartu orang (revisi 51): foto bulat (atau inisial) dan baris nama, jabatan, divisi, dan status nonaktif, di samping
// atau di bawah foto sesuai susunan kartu.
function drawPerson(
  ctx: CanvasRenderingContext2D,
  tok: Tok,
  card: Card,
  rect: Rect,
  text: Drawn[],
  img: HTMLImageElement | null,
  pad: number,
) {
  const c = card.content as PersonContent
  const a = AVATAR_PX[c.size] ?? 56
  const side = c.layout !== 'atas'
  const ax = side ? rect.x + pad : rect.x + (rect.w - a) / 2
  const ay = side ? rect.y + (rect.h - a) / 2 : rect.y + pad
  ctx.save()
  ctx.beginPath()
  ctx.arc(ax + a / 2, ay + a / 2, a / 2, 0, Math.PI * 2)
  ctx.clip()
  if (img) {
    // object-fit cover: sisi pendek mengisi lingkaran, tengah gambar dipakai.
    const k = Math.max(a / img.width, a / img.height)
    ctx.drawImage(
      img,
      ax + (a - img.width * k) / 2,
      ay + (a - img.height * k) / 2,
      img.width * k,
      img.height * k,
    )
  } else {
    ctx.fillStyle = tok('--color-canvas')
    ctx.fillRect(ax, ay, a, a)
    ctx.fillStyle = tok('--color-ink')
    ctx.font = fontOf(tok, Math.round(a * 0.36), true)
    ctx.textAlign = 'center'
    ctx.textBaseline = 'middle'
    ctx.fillText(initials(card.person?.name ?? '?'), ax + a / 2, ay + a / 2)
    ctx.textAlign = 'left'
  }
  ctx.restore()
  const tx = side ? ax + a + 12 : rect.x + pad
  const tw = side ? rect.x + rect.w - pad - tx : rect.w - pad * 2
  const total = text.reduce((n, l) => n + lineHeight(l), 0)
  let y = side ? rect.y + (rect.h - total) / 2 : ay + a + 8
  for (const l of text) {
    drawLine(
      ctx,
      tok,
      l,
      tx,
      y,
      tw,
      tok(l.muted ? '--color-ink-muted' : '--color-ink'),
      side ? 'left' : 'center',
    )
    y += lineHeight(l)
  }
}

// Bayangan tegas: hanya untuk kartu yang perlu menonjol (hierarki), sama dengan .wb-st-sh2.
function setShadow(ctx: CanvasRenderingContext2D) {
  ctx.shadowColor = 'rgba(26, 28, 32, 0.26)'
  ctx.shadowBlur = 8
  ctx.shadowOffsetY = 4
}

const shapeFill: Record<string, string> = { putih: '--color-surface', ...cardBg }

// Bentuk diputar terhadap pusatnya, digambar dari path yang sama dengan layar, lalu teksnya dipecah per baris di
// tengah bentuk. Kelas UML: nama tebal, garis pemisah, atribut, garis pemisah, metode.
function drawShape(
  ctx: CanvasRenderingContext2D,
  tok: Tok,
  c: ShapeContent,
  r: Box,
  // Gambar bingkai (revisi 57b), sudah dimuat.
  img: HTMLImageElement | null = null,
) {
  const framed = !!img && isFrame(c.shape)
  const sw = c.stroke_width * strokeScale(c.shape)
  ctx.save()
  ctx.translate(r.x + r.w / 2, r.y + r.h / 2)
  ctx.rotate(((c.rotation || 0) * Math.PI) / 180)
  ctx.translate(-r.w / 2, -r.h / 2)
  const pw = Math.max(1, r.w - sw)
  const ph = Math.max(1, r.h - sw)
  const path = new Path2D(shapePath(c.shape, pw, ph))
  ctx.save()
  ctx.translate(sw / 2, sw / 2)
  // Balik mendatar dan tegak (revisi 50): hanya path bentuknya, teks digambar sesudahnya tanpa dibalik.
  if (c.flip_x || c.flip_y) {
    ctx.translate(c.flip_x ? pw : 0, c.flip_y ? ph : 0)
    ctx.scale(c.flip_x ? -1 : 1, c.flip_y ? -1 : 1)
  }
  if (solidShape(c.shape)) {
    ctx.fillStyle = ink(tok, c.stroke)
    ctx.fill(path)
  } else if (!openShape(c.shape) && c.fill) {
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
  // Tanda di dalam bentuk (ikon BPMN), sama dengan WbShape.vue.
  for (const m of framed && c.shape === 'wf_image' ? [] : shapeMarks(c, pw, ph)) {
    const mp = new Path2D(m.d)
    const paint = (v: 'line' | 'surface' | 'none' | undefined) =>
      v === 'line' ? ink(tok, c.stroke) : v === 'surface' ? tok('--color-surface') : null
    const fillWith = paint(m.fill)
    if (fillWith) {
      ctx.fillStyle = fillWith
      ctx.fill(mp)
    }
    const strokeWith = m.stroke === 'none' ? null : (paint(m.stroke) ?? ink(tok, c.stroke))
    if (strokeWith) {
      ctx.strokeStyle = strokeWith
      ctx.lineWidth = c.stroke_width * (m.width ?? 1)
      ctx.setLineDash(m.dash ? [c.stroke_width * 3, c.stroke_width * 3] : [])
      ctx.stroke(mp)
      ctx.setLineDash([])
    }
  }
  ctx.restore()
  // Gambar bingkai di layar bingkainya, dengan cara isi dan posisi yang sama dengan object-fit di layar.
  if (framed && img) {
    const box = frameBox(c.shape, r.w, r.h, sw)
    const at = fitImage(box, img.naturalWidth || 1, img.naturalHeight || 1, frameImage(c))
    ctx.save()
    ctx.beginPath()
    ctx.rect(box.x, box.y, box.w, box.h)
    ctx.clip()
    ctx.drawImage(img, at.x, at.y, at.w, at.h)
    ctx.restore()
  }
  const fs = c.font_size
  const ink0 = tok('--color-ink')
  const lay = (text: string, doc: Parameters<typeof fieldLines>[1], maxW: number, bold?: boolean) =>
    fieldLines(text, doc, { size: fs, bold }).flatMap((l) => wrapLine(ctx, tok, l, maxW))
  if (c.shape === 'uml_class') {
    let y = 6
    // Varian kelas (revisi 54e), sama dengan WbShape.vue: stereotip, nama miring untuk abstrak, enum dua bagian.
    const stereo = classStereotype(c.variant)
    if (stereo)
      for (const l of lay(stereo, null, r.w - 16)) {
        drawLine(ctx, tok, l, 8, y, r.w - 16, ink0, 'center')
        y += lineHeight(l)
      }
    for (const l0 of lay(c.name || 'Kelas', c.name_doc, r.w - 16, true)) {
      const l = c.variant === 'abstract' ? { ...l0, runs: l0.runs.map((x) => ({ ...x, italic: true })) } : l0
      drawLine(ctx, tok, l, 8, y, r.w - 16, ink0, 'center')
      y += lineHeight(l)
    }
    const parts: [string, (typeof c)['attributes_doc']][] = [[c.attributes ?? '', c.attributes_doc]]
    if (c.variant !== 'enum') parts.push([c.methods ?? '', c.methods_doc])
    for (const [part, doc] of parts) {
      y += 4
      ctx.strokeStyle = ink(tok, c.stroke)
      ctx.lineWidth = sw
      ctx.beginPath()
      ctx.moveTo(0, y)
      ctx.lineTo(r.w, y)
      ctx.stroke()
      y += 6
      for (const l of lay(part, doc, r.w - 16)) {
        if (y + lineHeight(l) > r.h) break
        drawLine(ctx, tok, l, 8, y, r.w - 16, ink0)
        y += lineHeight(l)
      }
    }
  } else if (c.shape === 'erd_table') {
    // Tabel ERD (revisi 57a), sama dengan WbErdTable.vue: nama di kepala, lalu kunci (ikon kunci untuk PK), nama
    // kolom, dan tipe per baris. Teks yang terlalu panjang dipotong di baris pertamanya.
    const first = (text: string, size: number, maxW: number, bold?: boolean) =>
      wrapLine(ctx, tok, { text, size, bold }, Math.max(10, maxW))[0]
    const muted = tok('--color-ink-muted')
    const head = first(c.name || 'tabel', 14, r.w - 16, true)
    if (head) drawLine(ctx, tok, head, 8, (ERD_HEAD - lineHeight(head)) / 2, r.w - 16, ink0, 'center')
    const rows = c.rows ?? []
    if (!rows.length) {
      const l = first('Belum ada kolom', 14, r.w - 16)
      if (l) drawLine(ctx, tok, l, 8, ERD_HEAD + (ERD_ROW - lineHeight(l)) / 2, r.w - 16, muted)
    }
    rows.forEach((row, i) => {
      const y = ERD_HEAD + ERD_ROW * i
      const pk = row.keys.includes('pk')
      if (pk) {
        ctx.save()
        ctx.translate(8, y + ERD_ROW / 2 - 7)
        ctx.scale(14 / 16, 14 / 16)
        ctx.strokeStyle = ink0
        ctx.lineWidth = 1.75
        ctx.lineCap = 'round'
        ctx.beginPath()
        ctx.arc(5, 8, 3, 0, Math.PI * 2)
        ctx.moveTo(8, 8)
        ctx.lineTo(14.5, 8)
        ctx.moveTo(12, 8)
        ctx.lineTo(12, 11)
        ctx.moveTo(14.5, 8)
        ctx.lineTo(14.5, 10)
        ctx.stroke()
        ctx.restore()
      }
      const keys = first(keyLabel(row.keys), 12, 44, true)
      if (keys && row.keys.length)
        drawLine(ctx, tok, keys, pk ? 26 : 8, y + (ERD_ROW - lineHeight(keys)) / 2, 44, ink0)
      const typeText = row.type + (row.nullable ? ' null' : '')
      ctx.font = runFont(tok, 12, { text: typeText })
      const typeW = Math.min(r.w * 0.45, typeText ? ctx.measureText(typeText).width + 24 : 0)
      const type = typeText ? first(typeText, 12, typeW) : null
      if (type)
        drawLine(ctx, tok, type, r.w - 8 - typeW, y + (ERD_ROW - lineHeight(type)) / 2, typeW, muted, 'right')
      const nameX = 8 + 64 + 6
      const name = first(row.name, 14, r.w - nameX - typeW - 14, pk)
      if (name)
        drawLine(ctx, tok, name, nameX, y + (ERD_ROW - lineHeight(name)) / 2, r.w - nameX - typeW - 14, ink0)
    })
  } else if (c.text && !(framed && c.text === defaultText(c.shape))) {
    // Area teks dari definisi bentuk (revisi 54), sama dengan layar: judul di atas untuk batas sistem, label di bawah
    // untuk event dan gateway.
    const t = textBox(c.shape, r.w, r.h)
    // Nama pool ditulis tegak (revisi 54b): kotak teks diputar -90 derajat di pita kirinya.
    const tw = t.vertical ? t.h : t.w
    const th = t.vertical ? t.w : t.h
    ctx.save()
    if (t.vertical) {
      ctx.translate(t.x, t.y + t.h)
      ctx.rotate(-Math.PI / 2)
    } else ctx.translate(t.x, t.y)
    const lines = lay(c.text, c.doc, Math.max(20, tw - 8), t.valign === 'top' || !!t.vertical)
    const total = lines.reduce((n, l) => n + lineHeight(l), 0)
    let y = t.valign === 'top' || t.y >= r.h ? 0 : (th - total) / 2
    // Atribut kunci ERD (revisi 54e): teksnya bergaris bawah.
    const under = underlineText(c.shape)
    for (const l0 of lines) {
      const l = under ? { ...l0, runs: l0.runs.map((x) => ({ ...x, underline: true })) } : l0
      drawLine(ctx, tok, l, 4, y, Math.max(20, tw - 8), ink0, t.vertical ? 'center' : c.align)
      y += lineHeight(l)
    }
    ctx.restore()
  }
  // Label lane, sama dengan WbShape.vue: tegak di pita kedua pool, mendatar di pita kedua swimlane.
  const orient = laneOrient(c.shape)
  // Fragmen sequence (revisi 54d): operator di tab dan [syarat] di kiri atas tiap operand, sama dengan WbShape.vue.
  if (guardLanes(c.shape)) {
    const op = lay(c.variant || 'alt', null, FRAGMENT_TAB.w - 12, true)[0]
    if (op) drawLine(ctx, tok, op, 8, 3, FRAGMENT_TAB.w - 12, ink0, 'left')
    laneBoxes(c.lanes ?? [], r.w, r.h, 'rows', 0).forEach((b, i) => {
      if (!b.label) return
      const g = lay(`[${b.label}]`, null, Math.max(20, b.w - 16))[0]
      if (g)
        drawLine(ctx, tok, g, 8, b.y + (i ? 4 : FRAGMENT_TAB.h + 4), Math.max(20, b.w - 16), ink0, 'left')
    })
  } else if (orient)
    for (const b of laneBoxes(c.lanes ?? [], r.w, r.h, orient)) {
      if (!b.label) continue
      ctx.save()
      const len = orient === 'rows' ? b.h : b.w
      if (orient === 'rows') {
        ctx.translate(BAND, b.y + b.h)
        ctx.rotate(-Math.PI / 2)
      } else ctx.translate(b.x, BAND)
      const ls = lay(b.label, null, Math.max(20, len - 8))
      const total = ls.reduce((n, l) => n + lineHeight(l), 0)
      let y = (BAND - total) / 2
      for (const l of ls.slice(0, 2)) {
        drawLine(ctx, tok, l, 4, y, Math.max(20, len - 8), ink0, 'center')
        y += lineHeight(l)
      }
      ctx.restore()
    }
  ctx.restore()
}

type Box = { x: number; y: number; w: number; h: number }

export function canvasBlob(canvas: HTMLCanvasElement, type: string, quality?: number): Promise<Blob | null> {
  return new Promise((resolve) => canvas.toBlob(resolve, type, quality))
}
