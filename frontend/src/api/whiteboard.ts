import { ApiError, apiGet, apiPost, apiPut, notifyUnauthorized, request } from './client'
import type {
  Activity,
  Card,
  CardColor,
  CardKind,
  Connector,
  ConnectorGeometry,
  ConnectorStyle,
  PMNode,
  PreviewInfo,
} from '../utils/whiteboard'

// Whiteboard (docs/whiteboard.md). Semua adalah papan: "root" adalah papan akar bersama. can_write dari server:
// Staf dan Manajer menulis, Admin hanya membaca.

export type Crumb = { id: string; title: string }
export type SearchResult = { id: string; title: string; path: string }
// Hasil pencarian isi kartu (W4): board adalah jalur papan tempat kartu itu, dari papan akar.
export type CardHit = { id: string; board_id: string; board: string; kind: CardKind; snippet: string }
export type BoardView = {
  id: string
  title: string
  is_root: boolean
  is_template: boolean
  version: number
  seq: number
  crumbs: Crumb[]
  cards: Card[]
  connectors: Connector[]
}
export type TrashBatch = {
  batch: string
  deleted_at: string
  purge_at: string
  deleted_by_name: string
  board_title: string
  boards: string[]
  cards: number
  connectors: number
  excerpts: string[]
  blocked_by?: string
}

// Patch per field: field yang tidak disebut tidak berubah. color "" mengosongkan warna.
export type CardPatch = {
  id: string
  x?: number
  y?: number
  w?: number
  h?: number
  z?: number
  color?: CardColor | ''
  content?: Card['content']
  base_content_version?: number
  // "" mengeluarkan kartu dari kolom.
  column_id?: string
  col_order?: number
}

export type NewCard = {
  id: string
  kind: CardKind
  x: number
  y: number
  w: number
  h: number
  color: CardColor | ''
  content?: Card['content']
  // Hanya untuk kartu papan: judul papan baru.
  title?: string
  // Kartu gambar atau berkas hasil salin tempel memakai berkas yang sudah ada.
  file_id?: string
}

export type ChecklistOp =
  | { op: 'title'; text: string; doc?: PMNode | null }
  | { op: 'add'; item_id: string; text: string; index?: number; doc?: PMNode | null }
  | { op: 'set'; item_id: string; text?: string; done?: boolean; doc?: PMNode | null }
  | { op: 'remove'; item_id: string }
  | { op: 'move'; item_id: string; index: number }

export function searchBoards(q: string, signal?: AbortSignal) {
  return apiGet<{ results: SearchResult[]; cards: CardHit[] }>(
    `/whiteboard/search?q=${encodeURIComponent(q)}`,
    signal,
  )
}

// Template papan (W4): papan mana pun selain papan akar bisa ditandai template, lalu disalin ke papan lain.
export type TemplateInfo = { id: string; title: string; path: string; cards: number }
export function listTemplates(signal?: AbortSignal) {
  return apiGet<{ templates: TemplateInfo[] }>('/whiteboard/templates', signal)
}
export function setTemplate(id: string, isTemplate: boolean) {
  return apiPut<{ board: { id: string; is_template: boolean } }>(`/whiteboard/boards/${id}/template`, {
    is_template: isTemplate,
  })
}
export function fromTemplate(
  board: string,
  input: { template_id: string; id: string; x: number; y: number; w: number; h: number },
) {
  return apiPost<{ card: Card }>(`/whiteboard/boards/${board}/from-template`, input)
}

export function getBoard(id: string, signal?: AbortSignal) {
  return apiGet<{ board: BoardView; can_write: boolean }>(`/whiteboard/boards/${id}`, signal)
}

export function renameBoard(id: string, title: string, version: number) {
  return apiPut<{ board: { id: string; title: string; version: number } }>(`/whiteboard/boards/${id}`, {
    title,
    version,
  })
}

export function boardActivity(id: string) {
  return apiGet<{ activity: Activity[] }>(`/whiteboard/boards/${id}/activity`)
}

export function createCard(board: string, input: NewCard) {
  return apiPost<{ card: Card }>(`/whiteboard/boards/${board}/cards`, input)
}

export function patchCards(board: string, cards: CardPatch[]) {
  return apiPost<{ cards: Card[] }>(`/whiteboard/boards/${board}/cards/patch`, { cards })
}

export function deleteCards(board: string, ids: string[]) {
  return apiPost<{ batch: string }>(`/whiteboard/boards/${board}/cards/delete`, { ids })
}

export function checklistOp(board: string, card: string, op: ChecklistOp) {
  return apiPost<{ card: Card }>(`/whiteboard/boards/${board}/cards/${card}/checklist`, op)
}

export function listTrash(signal?: AbortSignal) {
  return apiGet<{ batches: TrashBatch[]; days: number; can_write: boolean }>('/whiteboard/trash', signal)
}

export function restoreBatch(batch: string) {
  return apiPost<{ restored: { board_id: string; title: string } }>(`/whiteboard/trash/${batch}/restore`)
}

export type TableOp =
  | { op: 'set'; row: number; col: number; text: string; doc?: PMNode | null }
  | { op: 'add_row' | 'add_col'; index?: number }
  | { op: 'remove_row'; index: number }
  | { op: 'remove_col'; index: number }
  | { op: 'header'; on: boolean }
  | { op: 'title'; text: string; doc?: PMNode | null }

export type PenStroke = { color: string; width: number; points: number[] }
export type DrawingOp =
  { op: 'add'; stroke: PenStroke } | { op: 'remove'; index: number } | { op: 'color'; color: string }

// Goresan pertama sesi pena membuat objek gambar; titik dalam koordinat dunia, kotak batasnya dihitung server.
export function createDrawing(board: string, id: string, stroke: PenStroke) {
  return apiPost<{ card: Card }>(`/whiteboard/boards/${board}/drawings`, { id, stroke })
}

export function drawingOp(board: string, card: string, op: DrawingOp) {
  return apiPost<{ card: Card }>(`/whiteboard/boards/${board}/cards/${card}/drawing`, op)
}

export function tableOp(board: string, card: string, op: TableOp) {
  return apiPost<{ card: Card }>(`/whiteboard/boards/${board}/cards/${card}/table`, op)
}

export type Preview = PreviewInfo & { message: string }
export function fetchPreview(url: string) {
  return apiPost<{ preview: Preview }>('/whiteboard/preview', { url })
}

export function fileUrl(id: string, thumb = false) {
  return `/api/whiteboard/files/${id}${thumb ? '?thumb=1' : ''}`
}

export type UploadHandle = { promise: Promise<Card>; abort: () => void }

// XMLHttpRequest supaya progres unggahan terbaca, sama dengan lampiran task. Kartu dibuat server bersama berkasnya.
export function uploadFile(
  board: string,
  at: { id: string; x: number; y: number; w: number; h: number },
  file: File,
  onProgress: (fraction: number) => void,
): UploadHandle {
  const xhr = new XMLHttpRequest()
  const q = new URLSearchParams({
    card_id: at.id,
    x: String(Math.round(at.x)),
    y: String(Math.round(at.y)),
    w: String(at.w),
    h: String(at.h),
  })
  const promise = new Promise<Card>((resolve, reject) => {
    xhr.open('POST', `/api/whiteboard/boards/${board}/files?${q}`)
    xhr.responseType = 'json'
    xhr.setRequestHeader('Accept', 'application/json')
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded / e.total)
    }
    xhr.onload = () => {
      const body = xhr.response as { card?: Card; error?: { code?: string; message?: string } } | null
      if (xhr.status === 201 && body?.card) {
        resolve(body.card)
        return
      }
      if (xhr.status === 401) notifyUnauthorized()
      if (!body?.error?.code) {
        const message =
          xhr.status === 413 ? 'Berkas melebihi batas 50 MB.' : 'Server aplikasi tidak merespons.'
        reject(new ApiError(xhr.status, 'backend_unreachable', message))
        return
      }
      reject(new ApiError(xhr.status, body.error.code, body.error.message ?? 'Unggahan gagal. Coba lagi.'))
    }
    xhr.onerror = () =>
      reject(new ApiError(0, 'network', 'Tidak bisa terhubung. Periksa koneksi ke jaringan kantor.'))
    xhr.onabort = () => reject(new DOMException('Unggahan dibatalkan', 'AbortError'))
    const form = new FormData()
    form.append('file', file)
    xhr.send(form)
  })
  return { promise, abort: () => xhr.abort() }
}

export type BoardOption = { id: string; title: string; parent_id: string | null; is_root: boolean }
export function listBoards() {
  return apiGet<{ boards: BoardOption[] }>('/whiteboard/boards')
}

export type MoveResult = {
  from: string
  to: string
  moved: string[]
  connectors_gone: string[]
  batch: string
}
export function moveCards(board: string, ids: string[], to: string) {
  return apiPost<{ moved: MoveResult }>(`/whiteboard/boards/${board}/move`, { ids, to })
}

// from atau to kosong berarti ujung bebas; titiknya di geometry (W5).
export function createConnector(
  board: string,
  input: {
    from: string
    to: string
    style?: Partial<ConnectorStyle>
    label?: string
    geometry?: ConnectorGeometry
  },
) {
  return apiPost<{ connector: Connector }>(`/whiteboard/boards/${board}/connectors`, input)
}

export function updateConnector(
  board: string,
  id: string,
  patch: {
    style?: ConnectorStyle
    label?: string
    label_start?: string
    label_end?: string
    geometry?: ConnectorGeometry
  },
) {
  return apiPut<{ connector: Connector }>(`/whiteboard/boards/${board}/connectors/${id}`, patch)
}

export function deleteConnector(board: string, id: string) {
  return request<{ batch: string }>('DELETE', `/whiteboard/boards/${board}/connectors/${id}`)
}

export type Comment = {
  id: string
  card_id: string
  user_id: number
  user_name: string
  avatar_id: string | null
  body: string
  created_at: string
  edited_at: string | null
}

export function listComments(board: string, card: string, signal?: AbortSignal) {
  return apiGet<{ comments: Comment[]; me: number; can_moderate: boolean }>(
    `/whiteboard/boards/${board}/cards/${card}/comments`,
    signal,
  )
}

export function addComment(board: string, card: string, body: string) {
  return apiPost<{ card: Card }>(`/whiteboard/boards/${board}/cards/${card}/comments`, { body })
}

export function editComment(id: string, body: string) {
  return apiPut<{ card: Card }>(`/whiteboard/comments/${id}`, { body })
}

export function deleteComment(id: string) {
  return request<{ card: Card }>('DELETE', `/whiteboard/comments/${id}`)
}
