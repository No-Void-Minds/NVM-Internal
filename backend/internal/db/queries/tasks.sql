-- name: GetEligiblePICs :many
-- PIC hanya akun aktif dengan peran Staf atau Manajer (keputusan Sprint 3).
SELECT id, division_id FROM users
WHERE id = ANY(@ids::bigint[]) AND is_active AND role IN ('staff', 'manager');

-- name: GetLabelsByIDs :many
SELECT id, division_id, is_active FROM labels WHERE id = ANY(@ids::bigint[]);

-- name: CreateTask :one
INSERT INTO tasks (title, description, division_id, requested_by, urgency, start_date, due_date, deliverable_url)
VALUES (@title, sqlc.narg(description), @division_id, @requested_by, @urgency,
        sqlc.narg(start_date), sqlc.narg(due_date), sqlc.narg(deliverable_url))
RETURNING id;

-- name: AddTaskAssignee :exec
-- clock_timestamp, bukan now(), supaya urutan PIC dalam satu transaksi tetap terbaca: PIC pertama = divisi task.
INSERT INTO task_assignees (task_id, user_id, assigned_by, assigned_at)
VALUES (@task_id, @user_id, @assigned_by, clock_timestamp());

-- name: AddTaskLabel :exec
INSERT INTO task_labels (task_id, label_id) VALUES (@task_id, @label_id);

-- name: GetTaskDetail :one
SELECT t.id, t.title, t.description, t.status, t.urgency, t.division_id, d.name AS division_name,
       t.requested_by, r.name AS requested_by_name, t.request_date, t.start_date, t.due_date,
       t.solved_date, t.deliverable_url, t.is_migrated, t.version
FROM tasks t
JOIN divisions d ON d.id = t.division_id
JOIN users r ON r.id = t.requested_by
WHERE t.id = @id AND NOT t.is_deleted;

-- name: ListTasks :many
-- Daftar sederhana untuk Sprint 3; papan dengan filter lengkap dibuat di Sprint 4.
SELECT t.id, t.title, t.status, t.urgency, t.division_id, d.name AS division_name,
       t.start_date, t.due_date, t.solved_date, t.version
FROM tasks t
JOIN divisions d ON d.id = t.division_id
WHERE NOT t.is_deleted
  AND (sqlc.narg(assignee_id)::bigint IS NULL
       OR EXISTS (SELECT 1 FROM task_assignees ta WHERE ta.task_id = t.id AND ta.user_id = sqlc.narg(assignee_id)::bigint))
ORDER BY t.created_at DESC;

-- name: ListTaskAssignees :many
SELECT ta.task_id, u.id, u.name, u.username
FROM task_assignees ta
JOIN users u ON u.id = ta.user_id
WHERE ta.task_id = ANY(@task_ids::text[])
ORDER BY ta.task_id, ta.assigned_at, u.id;

-- name: ListTaskLabels :many
SELECT tl.task_id, l.id, l.name, l.color, l.is_active
FROM task_labels tl
JOIN labels l ON l.id = tl.label_id
WHERE tl.task_id = ANY(@task_ids::text[])
ORDER BY tl.task_id, lower(l.name);

-- name: ListTaskHistory :many
SELECT a.id, a.occurred_at, a.action, a.old_value, a.new_value, actor.name AS actor_name
FROM audit_logs a
LEFT JOIN users actor ON actor.id = a.user_id
WHERE a.entity = 'task' AND a.entity_id = @task_id::text
ORDER BY a.occurred_at DESC, a.id DESC;

-- name: ListBoardTasks :many
-- Kolomnya sama persis dengan ListTasks supaya hasilnya bisa dikonversi dan dilengkapi PIC serta label
-- dengan kode yang sama. Completed hanya yang selesai sejak @completed_since; sisanya hanya ada di Arsip.
SELECT t.id, t.title, t.status, t.urgency, t.division_id, d.name AS division_name,
       t.start_date, t.due_date, t.solved_date, t.version
FROM tasks t
JOIN divisions d ON d.id = t.division_id
WHERE NOT t.is_deleted
  AND (t.status <> 'completed' OR t.solved_date >= @completed_since::timestamptz)
  AND ((cardinality(@pic_ids::bigint[]) = 0 AND cardinality(@pic_usernames::text[]) = 0)
       OR EXISTS (SELECT 1 FROM task_assignees ta JOIN users u ON u.id = ta.user_id
                  WHERE ta.task_id = t.id
                    AND (ta.user_id = ANY(@pic_ids::bigint[]) OR lower(u.username) = ANY(@pic_usernames::text[]))))
  AND (sqlc.narg(division_id)::bigint IS NULL OR t.division_id = sqlc.narg(division_id)::bigint)
  AND (sqlc.narg(label_id)::bigint IS NULL
       OR EXISTS (SELECT 1 FROM task_labels tl WHERE tl.task_id = t.id AND tl.label_id = sqlc.narg(label_id)::bigint))
  AND (sqlc.narg(urgency)::task_urgency IS NULL OR t.urgency = sqlc.narg(urgency)::task_urgency)
  AND (sqlc.narg(due)::text IS NULL
       OR (sqlc.narg(due)::text = 'overdue' AND t.due_date < @today::date AND t.status <> 'completed')
       OR (sqlc.narg(due)::text = 'week' AND t.due_date BETWEEN @today::date AND (@today::date) + 6)
       OR (sqlc.narg(due)::text = 'none' AND t.due_date IS NULL))
-- Bobot urgency, bukan urutan abjad (FR-TASK-14). Kolom Completed diurutkan ulang di Go berdasarkan solved_date.
ORDER BY CASE t.urgency WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END,
         t.due_date NULLS LAST, t.created_at;

-- name: ListPeople :many
-- Calon PIC untuk filter papan dan form task: aturannya sama dengan GetEligiblePICs.
SELECT u.id, u.name, u.username, u.role, d.name AS division_name
FROM users u
JOIN divisions d ON d.id = u.division_id
WHERE u.is_active AND u.role IN ('staff', 'manager')
ORDER BY lower(u.name);