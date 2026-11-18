-- name: ListTaskFilesForDrive :many
-- Lampiran task yang belum pernah terunggah ke Drive. Lampiran dan task yang ditandai terhapus dilewati,
-- tetapi salinannya di Drive tidak ikut dihapus (keputusan revisi 25).
SELECT a.id::text AS id, a.task_id, t.title AS task_title, t.created_at AS task_created_at,
       a.kind, a.original_name, a.size_bytes,
       a.subtask_id, s.title AS subtask_title, s.position AS subtask_position,
       -- COALESCE wajib: subselect ini NULL untuk lampiran task induk dan sub task tanpa PIC, dan sqlc
       -- menganggap kolomnya NOT NULL karena users.name NOT NULL.
       COALESCE((SELECT u.name FROM subtask_assignees sa JOIN users u ON u.id = sa.user_id
        WHERE sa.subtask_id = a.subtask_id ORDER BY u.name LIMIT 1), '')::text AS subtask_pic
FROM task_attachments a
JOIN tasks t ON t.id = a.task_id
LEFT JOIN task_subtasks s ON s.id = a.subtask_id
WHERE NOT a.is_deleted AND NOT t.is_deleted
  AND NOT EXISTS (
    SELECT 1 FROM drive_exports d
    WHERE d.source_kind = 'task_attachment' AND d.source_id = a.id::text
  )
ORDER BY t.created_at, a.task_id, a.uploaded_at, a.id;

-- name: ListActivityFilesForDrive :many
-- Lampiran Aktivitas Harian yang belum pernah terunggah ke Drive.
SELECT a.id::text AS id, a.user_id, u.name AS user_name, a.activity_date, a.slot,
       a.original_name, a.size_bytes
FROM activity_attachments a
JOIN users u ON u.id = a.user_id
WHERE NOT EXISTS (
    SELECT 1 FROM drive_exports d
    WHERE d.source_kind = 'activity_attachment' AND d.source_id = a.id::text
  )
ORDER BY a.activity_date, a.user_id, a.slot, a.created_at, a.id;

-- name: GetDriveFolder :one
SELECT folder_path FROM drive_folders WHERE scope = @scope AND source_id = @source_id;

-- name: ListDriveFolderPaths :many
-- Nama folder yang sudah dipakai di satu folder induk, untuk memeriksa bentrok nama.
SELECT folder_path FROM drive_folders WHERE scope = @scope AND folder_path LIKE @prefix || '%';

-- name: InsertDriveFolder :exec
INSERT INTO drive_folders (scope, source_id, folder_path) VALUES (@scope, @source_id, @folder_path)
ON CONFLICT (scope, source_id) DO NOTHING;

-- name: ListDrivePathsIn :many
-- Berkas yang sudah terunggah di dalam satu folder, untuk memeriksa bentrok nama berkas.
SELECT drive_path FROM drive_exports WHERE drive_path LIKE @prefix || '%';

-- name: InsertDriveExport :exec
INSERT INTO drive_exports (source_kind, source_id, drive_path, size_bytes)
VALUES (@source_kind, @source_id, @drive_path, @size_bytes)
ON CONFLICT (source_kind, source_id) DO NOTHING;

-- name: InsertDriveExportRun :exec
INSERT INTO drive_export_runs (ok, files, bytes, message) VALUES (@ok, @files, @bytes, @message);

-- name: GetLastDriveExportRun :one
SELECT finished_at, ok, files, bytes, message FROM drive_export_runs
ORDER BY finished_at DESC, id DESC LIMIT 1;

-- name: CountDriveSkipped :one
-- Alasan lampiran task tidak ikut disalin, dihitung di database. Dipakai ringkasan uji kering, supaya
-- "tidak ada berkas baru" selalu punya sebabnya yang terbaca, bukan diam saja.
SELECT
  count(*) FILTER (WHERE a.is_deleted)::bigint AS lampiran_dihapus,
  count(*) FILTER (WHERE NOT a.is_deleted AND t.is_deleted)::bigint AS task_dihapus,
  count(*) FILTER (
    WHERE NOT a.is_deleted AND NOT t.is_deleted
      AND EXISTS (
        SELECT 1 FROM drive_exports d
        WHERE d.source_kind = 'task_attachment' AND d.source_id = a.id::text
      )
  )::bigint AS sudah_tercatat
FROM task_attachments a
JOIN tasks t ON t.id = a.task_id;

-- name: CountActivityFilesSkipped :one
SELECT count(*)::bigint AS sudah_tercatat
FROM activity_attachments a
WHERE EXISTS (
  SELECT 1 FROM drive_exports d
  WHERE d.source_kind = 'activity_attachment' AND d.source_id = a.id::text
);
