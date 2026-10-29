-- name: ListLabels :many
SELECT l.id, l.name, l.color, l.division_id, d.name AS division_name, l.is_active,
       (SELECT count(*) FROM task_labels tl
          JOIN tasks t ON t.id = tl.task_id
         WHERE tl.label_id = l.id AND NOT t.is_deleted)::int AS task_count
FROM labels l
LEFT JOIN divisions d ON d.id = l.division_id
ORDER BY l.is_active DESC, lower(l.name);

-- name: GetLabelForUpdate :one
SELECT id, name, color, division_id, is_active FROM labels WHERE id = @id FOR UPDATE;

-- name: CreateLabel :one
INSERT INTO labels (name, color, division_id)
VALUES (@name, @color, sqlc.narg(division_id))
RETURNING id;

-- name: UpdateLabel :exec
UPDATE labels
SET name = @name, color = @color, division_id = sqlc.narg(division_id)
WHERE id = @id;

-- name: SetLabelActive :exec
UPDATE labels SET is_active = @is_active WHERE id = @id;