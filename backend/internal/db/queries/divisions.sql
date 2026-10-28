-- name: ListDivisions :many
SELECT d.id, d.name, d.is_active,
       count(u.id) FILTER (WHERE u.is_active)::int AS active_users
FROM divisions d
LEFT JOIN users u ON u.division_id = d.id
GROUP BY d.id
ORDER BY d.is_active DESC, lower(d.name);

-- name: GetDivisionForUpdate :one
SELECT id, name, is_active FROM divisions WHERE id = @id FOR UPDATE;

-- name: CreateDivision :one
INSERT INTO divisions (name) VALUES (@name) RETURNING id;

-- name: RenameDivision :exec
UPDATE divisions SET name = @name WHERE id = @id;

-- name: SetDivisionActive :exec
UPDATE divisions SET is_active = @is_active WHERE id = @id;