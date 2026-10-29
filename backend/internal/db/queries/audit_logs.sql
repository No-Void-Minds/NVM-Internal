-- name: ListAuditLogs :many
-- Sasaran aksi digabung ke nama akun atau divisi supaya halaman Audit Log bisa menulis
-- "Menonaktifkan akun Staf C" tanpa query tambahan per baris.
SELECT a.id, a.occurred_at, a.action, a.entity, a.entity_id, a.old_value, a.new_value, a.ip_address,
       a.user_id AS actor_id, actor.name AS actor_name,
       target_user.name AS target_user_name,
       target_division.name AS target_division_name,
       target_label.name AS target_label_name
       
FROM audit_logs a
LEFT JOIN users actor ON actor.id = a.user_id
LEFT JOIN users target_user ON a.entity = 'user' AND target_user.id::text = a.entity_id
LEFT JOIN divisions target_division ON a.entity = 'division' AND target_division.id::text = a.entity_id
LEFT JOIN labels target_label ON a.entity = 'label' AND target_label.id::text = a.entity_id
WHERE (sqlc.narg(from_time)::timestamptz IS NULL OR a.occurred_at >= sqlc.narg(from_time)::timestamptz)
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR a.occurred_at < sqlc.narg(to_time)::timestamptz)
  AND (sqlc.narg(actor_id)::bigint IS NULL OR a.user_id = sqlc.narg(actor_id)::bigint)
  AND (cardinality(@actions::text[]) = 0 OR a.action = ANY(@actions::text[]))
ORDER BY a.occurred_at DESC, a.id DESC
LIMIT @row_limit::int OFFSET @row_offset::int;