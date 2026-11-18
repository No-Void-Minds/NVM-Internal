-- name: CountExpiredSessions :one
SELECT count(*) FROM user_sessions WHERE created_at < @age_before;

-- name: DeleteExpiredSessions :execrows
-- Sesi yang sudah melewati umur mutlak tidak bisa dipakai lagi (touchSession menolaknya), jadi aman dihapus.
-- Sejak revisi 8 tidak ada lagi batas idle, jadi last_seen_at tidak ikut menentukan.
DELETE FROM user_sessions WHERE created_at < @age_before;

-- name: CountOldAuditLogs :one
SELECT count(*) FROM audit_logs WHERE action = ANY(@actions::text[]) AND occurred_at < @before;

-- name: DeleteOldAuditLogs :execrows
DELETE FROM audit_logs WHERE action = ANY(@actions::text[]) AND occurred_at < @before;

-- name: ListAttachmentFileIDs :many
-- Termasuk lampiran is_deleted: berkasnya sengaja disimpan, jadi bukan berkas yatim.
-- Lampiran Aktivitas Harian (revisi 24) menumpang folder attachments/ yang sama, jadi wajib ikut di sini:
-- tanpa UNION ini job pembersih berkas yatim akan menghapus semua berkasnya.
SELECT id::text FROM task_attachments
UNION ALL
SELECT id::text FROM activity_attachments;

-- name: ListAvatarFileIDs :many
SELECT avatar_id::text FROM users WHERE avatar_id IS NOT NULL;
