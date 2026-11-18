import type { TaskHistoryItem } from '../api/tasks'
import { formatDay, statusLabel, type TaskStatus } from './task'
import { freqLabel } from './recurrence'

const fieldName: Record<string, string> = {
  title: 'judul',
  description: 'deskripsi',
  urgency: 'urgency',
  start_date: 'tanggal mulai',
  due_date: 'deadline',
  deliverable_url: 'tautan deliverable',
  division_id: 'divisi',
  // Baris riwayat sebelum Sprint 7 masih memakai assignee_ids; dipertahankan supaya tetap terbaca.
  assignee_ids: 'PIC',
}

// describeTaskHistory menulis satu baris riwayat seperti di Stitch, mis. "memindahkan ke Ready".
// Nama pelaku ditulis terpisah oleh komponen supaya bisa ditebalkan.
// Sejak revisi 8 berkas bisa menempel di sub task, jadi riwayat perlu menyebut tempatnya.
function attachmentWhere(v: Record<string, unknown>): string {
  if (v.subtask_id) return 'lampiran sub task'
  return v.kind === 'brief' ? 'lampiran brief' : 'berkas deliverable'
}

export function describeTaskHistory(h: TaskHistoryItem): string {
  const after = h.new_value ?? {}
  switch (h.action) {
    case 'task_created':
      return 'membuat task'
    case 'task_status_changed': {
      const to = after.status as TaskStatus
      const reopened = subtaskRevisionText(after)
      if (reopened) return reopened
      return `memindahkan ke ${statusLabel[to] ?? String(to)}`
    }
    case 'task_updated': {
      const fields = Object.keys(after).map((k) => fieldName[k] ?? k)
      return fields.length ? `mengubah ${fields.join(', ')}` : 'mengubah task'
    }
    case 'task_labels_changed':
      return 'mengubah label'
    case 'task_deliverable_changed':
      return after.deliverable_url ? 'mengisi tautan deliverable' : 'menghapus tautan deliverable'
    case 'task_deleted':
      return 'menghapus task'
    case 'attachment_added':
      return `mengunggah ${attachmentWhere(after)} ${String(after.name ?? '')}`
    case 'attachment_deleted': {
      const before = h.old_value ?? {}
      return `menghapus ${attachmentWhere(before)} ${String(before.name ?? '')}`
    }
    case 'subtask_added':
      return `menambah sub task "${String(after.title ?? '')}"`
    case 'subtask_removed':
      return `menghapus sub task "${String((h.old_value ?? {}).title ?? '')}"`
    // Satu kejadian untuk nama dan PIC sekalian (revisi 7), jadi teksnya mengikuti apa yang benar-benar berubah.
    case 'subtask_renamed': {
      const prev = h.old_value ?? {}
      if (prev.title !== after.title) {
        return `mengganti nama sub task "${String(prev.title ?? '')}" menjadi "${String(after.title ?? '')}"`
      }
      if (prev.pic_id !== after.pic_id) return `mengubah PIC sub task "${String(after.title ?? '')}"`
      // Revisi 16: tenggat ikut dicatat. Riwayat lama tanpa due_date terbaca sama (undefined) dan jatuh ke bawah.
      if ((prev.due_date ?? null) !== (after.due_date ?? null)) {
        return after.due_date
          ? `mengubah deadline sub task "${String(after.title ?? '')}" menjadi ${formatDay(String(after.due_date))}`
          : `menghapus deadline sub task "${String(after.title ?? '')}"`
      }
      // Revisi 18: catatan hasil dan tautan hasil, disimpan bersama dari kotak Hasil.
      const name = String(after.title ?? '')
      const noteChanged = (prev.result_note ?? null) !== (after.result_note ?? null)
      const linkChanged = (prev.deliverable_url ?? '') !== (after.deliverable_url ?? '')
      if (noteChanged && linkChanged) return `mengubah catatan dan tautan hasil sub task "${name}"`
      if (noteChanged) {
        return after.result_note
          ? `mengubah catatan hasil sub task "${name}"`
          : `menghapus catatan hasil sub task "${name}"`
      }
      if (linkChanged) {
        return after.deliverable_url
          ? `mengubah tautan hasil sub task "${name}"`
          : `menghapus tautan hasil sub task "${name}"`
      }
      return `mengubah sub task "${name}"`
    }
    case 'task_pics_changed':
      return 'mengubah PIC task'
    // Kejadian lama dari sebelum revisi 7. Teksnya dipertahankan untuk riwayat yang sudah tercatat.
    case 'subtask_assignees_changed':
      return `mengubah PIC sub task "${String(after.title ?? '')}"`
    case 'subtask_checked':
      return `mencentang sub task "${String(after.title ?? '')}"`
    case 'subtask_unchecked':
      return `melepas centang sub task "${String(after.title ?? '')}"`
    case 'subtask_reordered':
      return 'mengubah urutan sub task'
    // Revisi 19: task berulang. Kemunculan dibuat aktor Sistem, jadi nama pelakunya kosong.
    case 'series_created':
      return `menjadikan task ini berulang (${freqWord(after.freq)})`
    case 'series_updated':
      return 'mengubah jadwal dan templat task berulang'
    case 'series_stopped':
      return 'menghentikan pengulangan'
    case 'occurrence_created':
      return `membuat kemunculan ${after.occurrence_date ? formatDay(String(after.occurrence_date)) : ''}`.trim()
    default:
      return h.action
  }
}

export function historyNote(h: TaskHistoryItem): string | null {
  const note = h.new_value?.note
  return typeof note === 'string' && note ? note : null
}

function freqWord(v: unknown): string {
  return v === 'daily' || v === 'weekly' || v === 'monthly' ? freqLabel[v].toLowerCase() : String(v ?? '')
}

// Revisi 22: PIC sub task mengembalikan task ke On Progress dengan membuka lagi sub task miliknya. Kalimatnya
// memuat alasan, jadi alasan disimpan di kunci reason (bukan note) supaya tidak tampil dua kali.
export function subtaskRevisionText(after: Record<string, unknown>): string | null {
  const list = after.reopened_subtasks
  if (!Array.isArray(list) || list.length === 0) return null
  const titles = list.map((s) => String((s as { title?: unknown }).title ?? '')).join(', ')
  const reason = typeof after.reason === 'string' && after.reason ? ` Alasan: ${after.reason}.` : ''
  return `mengembalikan ke On Progress untuk revisi: ${titles}.${reason}`
}
