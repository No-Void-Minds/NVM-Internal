import { apiGet } from './client'

export type AuditCategory = 'auth' | 'account' | 'division' | 'label' | 'task'

export type AuditEntry = {
  id: number
  occurred_at: string
  action: string
  actor: { id: string; name: string } | null
  target: { type: string; id: string; name?: string } | null
  old_value: Record<string, unknown> | null
  new_value: Record<string, unknown> | null
  ip: string
}

export type AuditFilter = {
  from?: string
  to?: string
  userId?: number
  category?: AuditCategory
  page: number
}

export type AuditPage = { entries: AuditEntry[]; has_more: boolean; page: number }

export function listAuditLogs(f: AuditFilter): Promise<AuditPage> {
  const params = new URLSearchParams({ page: String(f.page) })
  if (f.from) params.set('from', f.from)
  if (f.to) params.set('to', f.to)
  if (f.userId !== undefined) params.set('user_id', String(f.userId))
  if (f.category) params.set('category', f.category)
  return apiGet<AuditPage>(`/audit-logs?${params}`)
}
