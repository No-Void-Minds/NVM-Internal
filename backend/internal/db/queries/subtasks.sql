-- name: ListSubtasks :many
SELECT st.id, st.task_id, st.title, st.position, st.done_at, u.name AS done_by_name
FROM task_subtasks st
LEFT JOIN users u ON u.id = st.done_by
WHERE st.task_id = ANY(@task_ids::text[])
ORDER BY st.task_id, st.position;

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

-- name: GetSubtask :one
SELECT id, title, done_at FROM task_subtasks WHERE id = @id AND task_id = @task_id;

-- name: IsSubtaskAssignee :one
SELECT EXISTS (SELECT 1 FROM subtask_assignees WHERE subtask_id = @subtask_id AND user_id = @user_id);

-- name: CheckSubtask :exec
UPDATE task_subtasks SET done_at = now(), done_by = @user_id WHERE id = @id AND task_id = @task_id;

-- name: UncheckSubtask :exec
UPDATE task_subtasks SET done_at = NULL, done_by = NULL WHERE id = @id AND task_id = @task_id;

-- name: CountOpenSubtasks :one
-- Dipakai syarat On Progress ke Ready. Task tanpa sub task menghasilkan 0, jadi tidak tertahan.
SELECT count(*) FROM task_subtasks WHERE task_id = @task_id AND done_at IS NULL;

-- name: CountSubtasksByTask :many
-- Satu query untuk semua task di papan, bukan satu per kartu.
SELECT task_id, count(*) AS total, count(done_at) AS done
FROM task_subtasks
WHERE task_id = ANY(@task_ids::text[])
GROUP BY task_id;