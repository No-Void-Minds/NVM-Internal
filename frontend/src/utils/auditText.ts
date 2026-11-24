import type { AuditEntry } from '../api/audit'
import { roleLabel, type Role } from './roles'
import { statusLabel, type TaskStatus } from './task'
import { subtaskRevisionText } from './taskHistory'

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
const accessLabel: Record<string, string> = {
  can_manage_timeline: '"Kelola Timeline"',
  can_manage_notes: '"Kelola Important Notes"',
}
const onOff = (v: unknown) => (v === true ? 'aktif' : 'nonaktif')

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
      // Revisi 38: setiap akses tambahan yang berubah disebut dengan nilai sebelum dan sesudahnya.
      for (const [key, label] of Object.entries(accessLabel)) {
        if (key in after) changes.push(`akses ${label} ${onOff(before[key])} menjadi ${onOff(after[key])}`)
      }
      return `Mengubah akun ${target}: ${changes.join(', ')}`
    }
    case 'profile_updated': {
      // Pelaku dan sasaran selalu akun yang sama, karena profil hanya bisa diubah pemiliknya.
      const changes: string[] = []
      if ('name' in after) changes.push(`nama "${str(before.name)}" menjadi "${str(after.name)}"`)
      if ('job_title' in after) {
        const old = before.job_title ? `"${str(before.job_title)}"` : 'kosong'
        changes.push(
          after.job_title ? `jabatan ${old} menjadi "${str(after.job_title)}"` : `jabatan ${old} dikosongkan`,
        )
      }
      if (after.avatar === 'diganti') changes.push('mengganti foto profil')
      if (after.avatar === 'dihapus') changes.push('menghapus foto profil')
      return `Mengubah profil sendiri: ${changes.join(', ')}`
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
    case 'label_created':
      return `Membuat label ${str(after.name)}`
    case 'label_updated':
      return `Mengubah label ${target}`
    case 'label_activated':
      return `Mengaktifkan label ${target}`
    case 'label_deactivated':
      return `Menonaktifkan label ${target}`
    case 'task_created':
      return `Membuat task ${e.target?.id ?? ''}`
    case 'series_created':
      return `Menjadikan task ${e.target?.id ?? ''} berulang`
    case 'series_updated':
      // Revisi 29: perubahan sub task di kemunculan terbaru ikut memperbarui templat.
      if (after.source === 'subtask')
        return `Memperbarui sub task templat task berulang dari ${e.target?.id ?? ''}`
      return `Mengubah jadwal task berulang dari ${e.target?.id ?? ''}`
    case 'series_stopped':
      return `Menghentikan pengulangan task ${e.target?.id ?? ''}`
    case 'activity_updated': {
      // Hanya perubahan Manajer pada isian orang lain yang tercatat (revisi 21).
      const nama = str(after.name) || target
      return `Mengubah aktivitas ${nama} ${str(after.date)} jam ${str(after.time)}`
    }
    case 'occurrence_created':
      return `Membuat kemunculan task berulang ${e.target?.id ?? ''}`
    case 'task_updated':
      return `Mengubah task ${e.target?.id ?? ''}`
    case 'task_deleted':
      return `Menghapus task ${e.target?.id ?? ''}`
    case 'task_labels_changed':
      return `Mengubah label task ${e.target?.id ?? ''}`
    case 'task_deliverable_changed':
      return `Mengubah tautan deliverable ${e.target?.id ?? ''}`
    case 'task_status_changed': {
      // Revisi 22: kalimat yang sama dengan riwayat task, ditambah ID task.
      const reopened = subtaskRevisionText(after)
      if (reopened) return reopened.replace('mengembalikan ke', `Mengembalikan ${e.target?.id ?? ''} ke`)
      return `Memindahkan ${e.target?.id ?? ''} ke ${statusLabel[after.status as TaskStatus] ?? str(after.status)}`
    }
    case 'attachment_added':
      return `Mengunggah ${str(after.name)} ke ${e.target?.id ?? ''}`
    case 'attachment_deleted':
      return `Menghapus lampiran dari ${e.target?.id ?? ''}`
    case 'subtask_added':
      return `Menambah sub task "${str(after.title)}" ke ${e.target?.id ?? ''}`
    case 'subtask_removed':
      return `Menghapus sub task "${str(before.title)}" dari ${e.target?.id ?? ''}`
    // Satu kejadian untuk nama dan PIC sekalian (revisi 7), jadi teksnya mengikuti apa yang benar-benar berubah.
    case 'subtask_renamed':
      if (before.title !== after.title) {
        return `Mengganti nama sub task "${str(before.title)}" menjadi "${str(after.title)}" di ${e.target?.id ?? ''}`
      }
      return `Mengubah PIC sub task "${str(after.title)}" di ${e.target?.id ?? ''}`
    case 'task_pics_changed':
      return `Mengubah PIC task ${e.target?.id ?? ''}`
    // Kejadian lama dari sebelum revisi 7. Teksnya dipertahankan untuk riwayat yang sudah tercatat.
    case 'subtask_assignees_changed':
      return `Mengubah PIC sub task "${str(after.title)}" di ${e.target?.id ?? ''}`
    case 'subtask_checked':
      return `Mencentang sub task "${str(after.title)}" di ${e.target?.id ?? ''}`
    case 'subtask_unchecked':
      return `Melepas centang sub task "${str(after.title)}" di ${e.target?.id ?? ''}`
    case 'subtask_reordered':
      return `Mengubah urutan sub task ${e.target?.id ?? ''}`
    // Revisi 30. Kehadiran dicatat dengan nama orangnya saja, tanpa alasan: Admin juga membaca audit log.
    case 'timeline_event_created':
    case 'timeline_event_updated':
    case 'timeline_event_deleted': {
      const v = e.action === 'timeline_event_deleted' ? before : after
      const verb = { created: 'Membuat', updated: 'Mengubah', deleted: 'Menghapus' }[e.action.slice(15)]
      const what = v.category === 'kehadiran' ? `kehadiran ${str(v.name)}` : `"${str(v.title)}"`
      return `${verb} kartu Timeline ${what} (${str(v.start_date)})`
    }
    case 'timeline_note_created':
      return `Menambah catatan penting "${str(after.title)}"`
    case 'timeline_note_updated':
      return `Mengubah catatan penting "${str(after.title)}"`
    case 'timeline_note_deleted':
      return `Menghapus catatan penting "${str(before.title)}"`
    // Whiteboard: hanya hapus, pulihkan, dan hapus permanen yang masuk audit log; perubahan isi ada di riwayat papan.
    case 'wb_deleted':
    case 'wb_restored': {
      const boards = Array.isArray(after.boards) ? after.boards.map(String) : []
      const verb =
        e.action === 'wb_deleted' ? 'Memindahkan ke Tempat sampah' : 'Memulihkan dari Tempat sampah'
      const what = boards.length
        ? `papan "${boards[0]}"${boards.length > 1 ? ` beserta ${boards.length - 1} papan di dalamnya` : ''}`
        : `${str(after.cards)} kartu`
      return `${verb} ${what} di papan Whiteboard "${str(after.board)}"`
    }
    case 'wb_purged':
      return `Menghapus permanen isi Tempat sampah Whiteboard (${str(after.boards)} folder atau papan, ${str(after.cards)} kartu)`
    default:
      return e.action
  }
}
