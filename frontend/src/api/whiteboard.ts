import { apiGet, apiPost, apiPut } from './client'
import type { Activity, Card, CardColor, CardKind } from '../utils/whiteboard'

// Whiteboard (docs/whiteboard.md). Semua adalah papan: "root" adalah papan akar bersama. can_write dari server:
// Staf dan Manajer menulis, Admin hanya membaca.

export type Crumb = { id: string; title: string }
export type SearchResult = { id: string; title: string; path: string }
export type BoardView = {
  id: string
  title: string
  is_root: boolean
  version: number
  seq: number
  crumbs: Crumb[]
  cards: Card[]
}
export type TrashBatch = {
  batch: string
  deleted_at: string
  purge_at: string
  deleted_by_name: string
  board_title: string
  boards: string[]
  cards: number
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
}

export type ChecklistOp =
  | { op: 'title'; text: string }
  | { op: 'add'; item_id: string; text: string; index?: number }
  | { op: 'set'; item_id: string; text?: string; done?: boolean }
  | { op: 'remove'; item_id: string }
  | { op: 'move'; item_id: string; index: number }

export function searchBoards(q: string, signal?: AbortSignal) {
  return apiGet<{ results: SearchResult[] }>(`/whiteboard/search?q=${encodeURIComponent(q)}`, signal)
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
