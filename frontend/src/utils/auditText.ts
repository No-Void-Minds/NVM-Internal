import type { AuditEntry } from '../api/audit'
import { roleLabel, type Role } from './roles'

// Durasi per tahap kuncian, sama dengan backend (lockout.go).
const lockMinutes: Record<number, number> = { 1: 5, 2: 15, 3: 30 }

const failReason: Record<string, string> = {
  wrong_password: 'password salah',
  locked: 'akun sedang dikunci',
  inactive: 'akun nonaktif',
}

function str(v: unknown): string {
  return typeof v === 'string' ? v : String(v ?? '')
}

function role(v: unknown): string {
  return typeof v === 'string' && ['admin', 'manager', 'staff'].includes(v) ? roleLabel(v as Role) : str(v)
}

// describeAudit mengubah satu baris audit log menjadi kalimat, mis. "Menonaktifkan akun Staf C".
// divisionName dipakai untuk perubahan divisi akun, yang di audit log hanya menyimpan id divisi.
export function describeAudit(e: AuditEntry, divisionName: (id: number) => string): string {
  const target = e.target?.name ?? `#${e.target?.id ?? ''}`
  const before = e.old_value ?? {}
  const after = e.new_value ?? {}

  switch (e.action) {
    case 'login_success':
      return 'Masuk'
    case 'login_failed':
      if (after.reason === 'unknown_user')
        return `Gagal masuk dengan username tidak terdaftar "${str(after.username)}"`
      return `Gagal masuk, ${failReason[str(after.reason)] ?? 'alasan tidak tercatat'}`
    case 'account_locked': {
      const minutes = lockMinutes[Number(after.level)]
      return minutes
        ? `Gagal masuk 5 kali, akun dikunci ${minutes} menit`
        : 'Gagal masuk 5 kali, akun dikunci'
    }
    case 'logout':
      return 'Keluar dari semua perangkat'
    case 'password_changed':
      return 'Mengganti password sendiri'
    case 'password_change_failed':
      return 'Gagal mengganti password, password lama salah'
    case 'user_created':
      return `Membuat akun ${target} sebagai ${role(after.role)}`
    case 'user_updated': {
      const changes: string[] = []
      if ('name' in after) changes.push(`nama "${str(before.name)}" menjadi "${str(after.name)}"`)
      if ('role' in after) changes.push(`peran ${role(before.role)} menjadi ${role(after.role)}`)
      if ('division_id' in after) {
        changes.push(
          `divisi ${divisionName(Number(before.division_id))} menjadi ${divisionName(Number(after.division_id))}`,
        )
      }
      return `Mengubah akun ${target}: ${changes.join(', ')}`
    }
    case 'user_activated':
      return `Mengaktifkan akun ${target}`
    case 'user_deactivated':
      return `Menonaktifkan akun ${target}`
    case 'password_reset':
      return `Mengatur ulang password ${target}`
    case 'user_unlocked':
      return `Membuka kunci akun ${target}`
    case 'division_created':
      return `Membuat divisi ${str(after.name)}`
    case 'division_renamed':
      return `Mengganti nama divisi "${str(before.name)}" menjadi "${str(after.name)}"`
    case 'division_activated':
      return `Mengaktifkan divisi ${target}`
    case 'division_deactivated':
      return `Menonaktifkan divisi ${target}`
    default:
      return e.action
  }
}
