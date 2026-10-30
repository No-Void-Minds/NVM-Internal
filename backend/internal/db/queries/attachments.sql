-- name: TaskIsActive :one
SELECT EXISTS (SELECT 1 FROM tasks WHERE id = @id AND NOT is_deleted);

-- name: InsertAttachment :one
-- id dikembalikan sebagai teks karena langsung dipakai sebagai nama berkas di disk.
INSERT INTO task_attachments (task_id, kind, original_name, mime_type, size_bytes, checksum, uploaded_by)
VALUES (@task_id, @kind, @original_name, @mime_type, @size_bytes, @checksum, @uploaded_by)
RETURNING id::text;

-- name: ListTaskAttachments :many
SELECT a.id::text AS id, a.kind, a.original_name, a.mime_type, a.size_bytes, a.uploaded_at,
       u.name AS uploaded_by_name
FROM task_attachments a
JOIN users u ON u.id = a.uploaded_by
WHERE a.task_id = @task_id AND NOT a.is_deleted
ORDER BY a.uploaded_at, a.id;

-- name: GetAttachment :one
-- Lampiran dari task yang dihapus ikut tersembunyi, sama dengan task-nya.
SELECT a.id::text AS id, a.kind, a.original_name, a.mime_type, a.size_bytes, a.uploaded_at
FROM task_attachments a
JOIN tasks t ON t.id = a.task_id
WHERE a.id = @id::uuid AND a.task_id = @task_id AND NOT a.is_deleted AND NOT t.is_deleted;

-- name: SoftDeleteAttachment :exec
UPDATE task_attachments SET is_deleted = true
WHERE id = @id::uuid AND task_id = @task_id AND NOT is_deleted;