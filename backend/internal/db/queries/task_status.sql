-- name: SetTaskStatus :one
-- solved_date diisi saat Completed dan dikosongkan saat keluar dari Completed (FR-TASK-10, AC-08).
UPDATE tasks
SET status      = @status::task_status,
    solved_date = CASE WHEN @status::task_status = 'completed' THEN now() ELSE NULL END,
    version     = version + 1
WHERE id = @id
RETURNING version, solved_date;