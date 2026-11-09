import type { SubtaskInput, TaskDetail } from '../api/tasks'

export type SubtaskDraft = { key: string; id: number; title: string; picIds: number[] }

// Sama dengan batas di backend (task/subtasks.go).
export const maxSubtasks = 50

let counter = 0
// key hanya untuk v-for. id database tidak bisa dipakai karena sub task baru belum punya id.
export function newKey(): string {
  counter += 1
  return `sub-${counter}`
}

export function emptyDraft(): SubtaskDraft {
  return { key: newKey(), id: 0, title: '', picIds: [] }
}

export function draftsFromTask(t: TaskDetail | null): SubtaskDraft[] {
  return (t?.subtasks ?? []).map((s) => ({
    key: newKey(),
    id: s.id,
    title: s.title,
    picIds: s.assignees.map((a) => a.id),
  }))
}

export function toPayload(drafts: SubtaskDraft[]): SubtaskInput[] {
  return drafts.map((d) =>
    d.id
      ? { id: d.id, title: d.title.trim(), pic_ids: d.picIds }
      : { title: d.title.trim(), pic_ids: d.picIds },
  )
}
