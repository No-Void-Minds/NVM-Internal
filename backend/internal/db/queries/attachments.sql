-- name: TaskIsActive :one
SELECT EXISTS (SELECT 1 FROM tasks WHERE id = @id AND NOT is_deleted);

-- name: InsertAttachment :one
-- id dikembalikan sebagai teks karena langsung dipakai sebagai nama berkas di disk.
INSERT INTO task_attachments (task_id, kind, original_name, mime_type, size_bytes, checksum, uploaded_by)
VALUES (@task_id, @kind, @original_name, @mime_type, @size_bytes, @checksum, @uploaded_by)
RETURNING id::text;