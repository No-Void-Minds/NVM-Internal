-- name: ListSubtasks :many
SELECT id, task_id, title, position
FROM task_subtasks
WHERE task_id = ANY(@task_ids::text[])
ORDER BY task_id, position;

-- name: ListSubtaskAssignees :many
SELECT sa.subtask_id, u.id, u.name, u.username
FROM subtask_assignees sa
JOIN users u ON u.id = sa.user_id
WHERE sa.subtask_id = ANY(@subtask_ids::bigint[])
ORDER BY sa.subtask_id, sa.assigned_at, u.id;

-- name: CreateSubtask :one
INSERT INTO task_subtasks (task_id, title, position)
VALUES (@task_id, @title, @position)
RETURNING id;

-- name: UpdateSubtask :exec
-- task_id ikut dicocokkan supaya sub task milik task lain tidak bisa diubah lewat ID yang salah.
UPDATE task_subtasks SET title = @title, position = @position
WHERE id = @id AND task_id = @task_id;

-- name: DeleteSubtasks :exec
-- PIC sub task ikut terhapus lewat ON DELETE CASCADE.
DELETE FROM task_subtasks WHERE task_id = @task_id AND id = ANY(@ids::bigint[]);

-- name: AddSubtaskAssignee :exec
-- clock_timestamp, bukan now(), supaya urutan PIC dalam satu transaksi tetap terbaca: PIC pertama menentukan divisi task.
INSERT INTO subtask_assignees (subtask_id, user_id, assigned_by, assigned_at)
VALUES (@subtask_id, @user_id, @assigned_by, clock_timestamp());

-- name: RemoveSubtaskAssignees :exec
DELETE FROM subtask_assignees WHERE subtask_id = @subtask_id AND user_id = ANY(@user_ids::bigint[]);

-- name: FirstPICDivision :one
-- Divisi task mengikuti PIC pertama: sub task paling awal yang punya PIC, lalu PIC yang dipasang lebih dulu.
-- Tidak ada baris berarti task belum punya PIC.
SELECT u.division_id
FROM task_subtasks st
JOIN subtask_assignees sa ON sa.subtask_id = st.id
JOIN users u ON u.id = sa.user_id
WHERE st.task_id = @task_id
ORDER BY st.position, sa.assigned_at, u.id
LIMIT 1;