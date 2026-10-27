export type Role = 'admin' | 'manager' | 'staff'

const labels: Record<Role, string> = { admin: 'Admin', manager: 'Manajer', staff: 'Staf' }

export function roleLabel(role: Role): string {
  return labels[role]
}
