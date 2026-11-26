// Posisi kursor untuk dobel klik edit: titik layar yang diklik, butir checklist atau sel tabel di titik itu, dan
// offset karakter di teks tampilannya. Editor yang terbuka menaruh kursor teks di posisi yang sama.
// textIndex: urutan karakter di seluruh teks catatan (WbRichText merender teks dokumen apa adanya sebagai text node).
export type CaretHint = {
  x: number
  y: number
  item?: string
  cell?: string
  offset?: number
  textIndex?: number
}

type CaretDoc = Document & {
  caretPositionFromPoint?: (x: number, y: number) => { offsetNode: Node; offset: number } | null
}

export function caretHint(x: number, y: number, target: EventTarget | null): CaretHint {
  const el = target as HTMLElement | null
  const hint: CaretHint = { x, y }
  const item = el?.closest<HTMLElement>('[data-item-text]')
  const cell = el?.closest<HTMLElement>('[data-cell-text]')
  if (item) hint.item = item.dataset.itemText
  if (cell) hint.cell = cell.dataset.cellText
  const d = document as CaretDoc
  const pos = d.caretPositionFromPoint
    ? d.caretPositionFromPoint(x, y)
    : (() => {
        const r = document.caretRangeFromPoint?.(x, y)
        return r ? { offsetNode: r.startContainer, offset: r.startOffset } : null
      })()
  // Offset hanya dipakai bila titiknya ada di teks tampilan (bukan placeholder seperti "Judul kosong").
  if (
    pos &&
    pos.offsetNode.nodeType === Node.TEXT_NODE &&
    pos.offsetNode.parentElement?.closest('[data-caret]')
  )
    hint.offset = pos.offset - leadingSpace(pos.offsetNode.textContent ?? '')
  const rich = el?.closest<HTMLElement>('[data-richtext]')
  if (pos && rich && pos.offsetNode.nodeType === Node.TEXT_NODE && rich.contains(pos.offsetNode)) {
    let index = 0
    const walk = document.createTreeWalker(rich, NodeFilter.SHOW_TEXT)
    for (let n = walk.nextNode(); n && n !== pos.offsetNode; n = walk.nextNode())
      index += n.textContent?.length ?? 0
    hint.textIndex = index + pos.offset
  }
  return hint
}

// Interpolasi template Vue menyisakan spasi di awal teks; offset dihitung dari teks yang terlihat.
function leadingSpace(t: string) {
  return t.length - t.trimStart().length
}
