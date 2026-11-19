-- name: ListTimelineEvents :many
-- Semua event yang beririsan dengan rentang [from, to], termasuk event rentang yang mulai sebelum from atau
-- berakhir sesudah to. end_date NULL berarti satu hari.
SELECT e.id, e.category, e.title, e.description, e.start_date, e.end_date, e.user_id, e.absence_type,
       u.name AS user_name
FROM timeline_events e
LEFT JOIN users u ON u.id = e.user_id
WHERE NOT e.is_deleted
  AND e.start_date <= @to_date::date
  AND coalesce(e.end_date, e.start_date) >= @from_date::date
ORDER BY e.start_date, e.category, e.id;

-- name: GetTimelineEvent :one
SELECT id, category, title, description, start_date, end_date, user_id, absence_type
FROM timeline_events WHERE id = @id AND NOT is_deleted
FOR UPDATE;

-- name: InsertTimelineEvent :one
INSERT INTO timeline_events (category, title, description, start_date, end_date, user_id, absence_type,
                             created_by, updated_by)
VALUES (@category, sqlc.narg(title), sqlc.narg(description), @start_date, sqlc.narg(end_date),
        sqlc.narg(user_id), sqlc.narg(absence_type), @actor_id, @actor_id)
RETURNING id;

-- name: UpdateTimelineEvent :execrows
UPDATE timeline_events
SET category = @category, title = sqlc.narg(title), description = sqlc.narg(description),
    start_date = @start_date, end_date = sqlc.narg(end_date), user_id = sqlc.narg(user_id),
    absence_type = sqlc.narg(absence_type), updated_by = @actor_id
WHERE id = @id AND NOT is_deleted;

-- name: DeleteTimelineEvent :execrows
UPDATE timeline_events SET is_deleted = true, updated_by = @actor_id WHERE id = @id AND NOT is_deleted;

-- name: GetActiveUserName :one
SELECT name FROM users WHERE id = @id AND is_active;

-- name: ListTimelineNotes :many
SELECT id, title, body FROM timeline_notes WHERE NOT is_deleted ORDER BY position, id;

-- name: GetTimelineNote :one
SELECT id, title, body FROM timeline_notes WHERE id = @id AND NOT is_deleted FOR UPDATE;

-- name: CountTimelineNotes :one
SELECT count(*) FROM timeline_notes WHERE NOT is_deleted;

-- name: NextTimelineNotePosition :one
SELECT coalesce(max(position) + 1, 0)::int FROM timeline_notes WHERE NOT is_deleted;

-- name: InsertTimelineNote :one
INSERT INTO timeline_notes (title, body, position, created_by, updated_by)
VALUES (@title, @body, @position, @actor_id, @actor_id)
RETURNING id;

-- name: UpdateTimelineNote :execrows
UPDATE timeline_notes SET title = @title, body = @body, updated_by = @actor_id WHERE id = @id AND NOT is_deleted;

-- name: DeleteTimelineNote :execrows
UPDATE timeline_notes SET is_deleted = true, updated_by = @actor_id WHERE id = @id AND NOT is_deleted;

-- name: CountTimelineNotesIn :one
SELECT count(*) FROM timeline_notes WHERE NOT is_deleted AND id = ANY(@ids::bigint[]);

-- name: SetTimelineNotePosition :exec
UPDATE timeline_notes SET position = @position WHERE id = @id AND NOT is_deleted;
