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
SELECT id::text FROM task_attachments;

-- name: ListAvatarFileIDs :many
SELECT avatar_id::text FROM users WHERE avatar_id IS NOT NULL;
