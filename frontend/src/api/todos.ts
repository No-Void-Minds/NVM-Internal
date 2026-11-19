import { apiGet, apiPost, apiPut, request } from './client'

// To Do pribadi (revisi 28). Semua endpoint memakai akun yang login; tidak ada endpoint untuk To Do orang lain.
export type TodoItem = {
  id: number
  title: string
  done: boolean
}

export type TodoList = { todos: TodoItem[]; max: number }

export function listTodos(signal?: AbortSignal): Promise<TodoList> {
  return apiGet<TodoList>('/todos', signal)
}

export function addTodo(title: string): Promise<{ todo: TodoItem }> {
  return apiPost<{ todo: TodoItem }>('/todos', { title })
}

export function renameTodo(id: number, title: string) {
  return apiPut<void>(`/todos/${id}`, { title })
}

export function setTodoDone(id: number, done: boolean) {
  return apiPut<void>(`/todos/${id}/done`, { done })
}

export function deleteTodo(id: number) {
  return request<void>('DELETE', `/todos/${id}`)
}

// Urutan dikirim sebagai daftar id. Satu id milik orang lain membuat seluruh permintaan ditolak.
export function reorderTodos(ids: number[]) {
  return apiPut<void>('/todos/order', { ids })
}

export function clearDoneTodos(): Promise<{ deleted: number }> {
  return apiPost<{ deleted: number }>('/todos/clear-done')
}
