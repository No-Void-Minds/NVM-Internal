import type { TaskHistoryItem } from '../api/tasks'
import { statusLabel, type TaskStatus } from './task'

const fieldName: Record<string, string> = {
  title: 'judul',
  description: 'deskripsi',
  urgency: 'urgency',
  start_date: 'tanggal mulai',
  due_date: 'tenggat',
  deliverable_url: 'tautan deliverable',
  division_id: 'divisi',
  // Baris riwayat sebelum Sprint 7 masih memakai assignee_ids; dipertahankan supaya tetap terbaca.
  assignee_ids: 'PIC',
}

// describeTaskHistory menulis satu baris riwayat seperti di Stitch, mis. "memindahkan ke Ready".
// Nama pelaku ditulis terpisah oleh komponen supaya bisa ditebalkan.
export function describeTaskHistory(h: TaskHistoryItem): string {
  const after = h.new_value ?? {}
  switch (h.action) {
    case 'task_created':
      return 'membuat task'
    case 'task_status_changed': {
      const to = after.status as TaskStatus
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
      return `mengunggah ${after.kind === 'brief' ? 'lampiran brief' : 'berkas deliverable'} ${String(after.name ?? '')}`
    case 'attachment_deleted': {
      const before = h.old_value ?? {}
      return `menghapus ${before.kind === 'brief' ? 'lampiran brief' : 'berkas deliverable'} ${String(before.name ?? '')}`
    }
    case 'subtask_added':
      return `menambah sub task "${String(after.title ?? '')}"`
    case 'subtask_removed':
      return `menghapus sub task "${String((h.old_value ?? {}).title ?? '')}"`
    case 'subtask_renamed':
      return `mengganti nama sub task "${String((h.old_value ?? {}).title ?? '')}" menjadi "${String(after.title ?? '')}"`
    case 'subtask_assignees_changed':
      return `mengubah PIC sub task "${String(after.title ?? '')}"`
    case 'subtask_checked':
      return `mencentang sub task "${String(after.title ?? '')}"`
    case 'subtask_unchecked':
      return `melepas centang sub task "${String(after.title ?? '')}"`
    case 'subtask_reordered':
      return 'mengubah urutan sub task'
    default:
      return h.action
  }
}

export function historyNote(h: TaskHistoryItem): string | null {
  const note = h.new_value?.note
  return typeof note === 'string' && note ? note : null
}
