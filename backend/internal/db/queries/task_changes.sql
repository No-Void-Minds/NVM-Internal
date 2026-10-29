-- name: GetTaskForUpdate :one
SELECT id, title, description, status, urgency, division_id, start_date, due_date, deliverable_url, version
FROM tasks
WHERE id = @id AND NOT is_deleted
FOR UPDATE;

-- name: IsTaskAssignee :one
SELECT EXISTS (SELECT 1 FROM task_assignees WHERE task_id = @task_id AND user_id = @user_id);

-- name: ListAssigneeIDs :many
SELECT user_id FROM task_assignees WHERE task_id = @task_id ORDER BY assigned_at, user_id;

-- name: DeleteTaskAssignees :exec
DELETE FROM task_assignees WHERE task_id = @task_id;

-- name: ListTaskLabelScopes :many
SELECT tl.label_id, l.division_id
FROM task_labels tl
JOIN labels l ON l.id = tl.label_id
WHERE tl.task_id = @task_id;

-- name: DeleteTaskLabels :exec
DELETE FROM task_labels WHERE task_id = @task_id;

-- name: UpdateTaskFields :one
UPDATE tasks
SET title           = @title,
    description     = sqlc.narg(description),
    urgency         = @urgency,
    division_id     = @division_id,
    start_date      = sqlc.narg(start_date),
    due_date        = sqlc.narg(due_date),
    deliverable_url = sqlc.narg(deliverable_url),
    version         = version + 1
WHERE id = @id
RETURNING version;

-- name: SetTaskDeliverable :one
UPDATE tasks SET deliverable_url = sqlc.narg(deliverable_url), version = version + 1
WHERE id = @id
RETURNING version;

-- name: BumpTaskVersion :one
UPDATE tasks SET version = version + 1 WHERE id = @id RETURNING version;

-- name: SoftDeleteTask :exec
UPDATE tasks SET is_deleted = true, version = version + 1 WHERE id = @id;