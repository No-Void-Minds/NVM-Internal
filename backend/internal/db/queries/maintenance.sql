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
-- Berkas Whiteboard (W3, migrasi 000024) juga di folder yang sama, dengan thumbnail <id>.thumb.jpg di sebelahnya.
SELECT id::text FROM task_attachments
UNION ALL
SELECT id::text FROM activity_attachments
UNION ALL
SELECT id::text FROM wb_files;

-- name: ListAvatarFileIDs :many
SELECT avatar_id::text FROM users WHERE avatar_id IS NOT NULL;
