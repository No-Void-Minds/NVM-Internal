-- name: ListSubtasks :many
SELECT st.id, st.task_id, st.title, st.description, st.reference_url, st.deliverable_url,
       st.position, st.done_at, st.created_by, st.due_date, st.result_note,
       u.name AS done_by_name, c.name AS created_by_name
FROM task_subtasks st
LEFT JOIN users u ON u.id = st.done_by
LEFT JOIN users c ON c.id = st.created_by
WHERE st.task_id = ANY(@task_ids::text[])
ORDER BY st.task_id, st.position;

-- name: ListSubtaskAssignees :many
SELECT sa.subtask_id, u.id, u.name, u.username, u.avatar_id
FROM subtask_assignees sa
JOIN users u ON u.id = sa.user_id
WHERE sa.subtask_id = ANY(@subtask_ids::bigint[])
ORDER BY sa.subtask_id, sa.assigned_at, u.id;

-- name: CreateSubtask :one
INSERT INTO task_subtasks (task_id, title, description, reference_url, deliverable_url, position, created_by, due_date)
VALUES (@task_id, @title, @description, @reference_url, @deliverable_url, @position, @created_by, sqlc.narg(due_date))
RETURNING id;

-- name: UpdateSubtask :exec
-- task_id ikut dicocokkan supaya sub task milik task lain tidak bisa diubah lewat ID yang salah.
UPDATE task_subtasks SET title = @title, position = @position
WHERE id = @id AND task_id = @task_id;

-- name: DeleteSubtasks :exec
-- PIC sub task ikut terhapus lewat ON DELETE CASCADE.
DELETE FROM task_subtasks WHERE task_id = @task_id AND id = ANY(@ids::bigint[]);

-- name: AddSubtaskAssignee :exec
-- Sejak revisi 7 isinya satu orang: PIC yang ditugaskan mengerjakan sub task ini, dipilih dari PIC task.
-- Sub task yang belum ditugaskan tidak punya baris di sini.
INSERT INTO subtask_assignees (subtask_id, user_id, assigned_by, assigned_at)
VALUES (@subtask_id, @user_id, @assigned_by, clock_timestamp());

-- name: RemoveSubtaskAssignees :exec
DELETE FROM subtask_assignees WHERE subtask_id = @subtask_id AND user_id = ANY(@user_ids::bigint[]);

-- name: FirstPICDivision :one
-- Divisi task mengikuti PIC task yang dipasang lebih dulu. Tidak ada baris berarti task belum punya PIC.
SELECT u.division_id
FROM task_assignees tas
JOIN users u ON u.id = tas.user_id
WHERE tas.task_id = @task_id
ORDER BY tas.assigned_at, u.id
LIMIT 1;

-- name: GetSubtask :one
SELECT id, title, description, reference_url, deliverable_url, done_at, created_by, due_date, result_note
FROM task_subtasks WHERE id = @id AND task_id = @task_id;

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

-- name: ListSubtasksWithOtherManagerPIC :many
-- Sub task yang punya PIC Manajer selain si pemanggil. Dipakai aturan: Manajer tidak boleh menyentuh centang milik Manajer lain.
SELECT DISTINCT sa.subtask_id
FROM subtask_assignees sa
JOIN users u ON u.id = sa.user_id
WHERE sa.subtask_id = ANY(@subtask_ids::bigint[]) AND u.role = 'manager' AND u.id <> @user_id;
-- name: SubtaskSlot :one
-- Jumlah sub task dan posisi berikutnya, dipakai saat PIC atau Manajer menambah satu sub task.
-- Posisi dihitung dari max, bukan dari jumlah, karena penghapusan meninggalkan lubang.
SELECT count(*)::int AS total, COALESCE(max(position) + 1, 0)::int AS next_position
FROM task_subtasks WHERE task_id = @task_id;

-- name: CountSubtaskAssignees :one
SELECT count(*)::int FROM subtask_assignees WHERE subtask_id = @subtask_id;

-- name: RenameSubtask :exec
-- Nama, deskripsi, tautan, tenggat, dan catatan hasil; urutan tidak ikut berubah saat PIC atau Manajer
-- memperbaiki isinya.
UPDATE task_subtasks
SET title = @title, description = @description, reference_url = @reference_url, deliverable_url = @deliverable_url,
    due_date = sqlc.narg(due_date), result_note = sqlc.narg(result_note)
WHERE id = @id AND task_id = @task_id;

-- name: ClearSubtaskAssignees :exec
-- Dipakai saat PIC sub task diganti: barisnya diganti utuh, bukan ditambah, karena satu sub task satu PIC.
DELETE FROM subtask_assignees WHERE subtask_id = @subtask_id;

-- name: DropSubtaskAssigneesOfUsers :exec
-- Dipakai saat Manajer melepas seseorang dari PIC task: sub task yang ditugaskan ke dia ikut dilepas,
-- supaya tidak ada sub task yang dipegang orang yang sudah tidak ada di task ini.
DELETE FROM subtask_assignees sa
USING task_subtasks st
WHERE sa.subtask_id = st.id AND st.task_id = @task_id AND sa.user_id = ANY(@user_ids::bigint[]);

-- name: ListTasksWithSubtaskPIC :many
-- Revisi 17: task mana yang sub task-nya dipegang user ini, untuk menghitung move_to papan dan Arsip sekaligus.
SELECT DISTINCT st.task_id
FROM task_subtasks st
JOIN subtask_assignees sa ON sa.subtask_id = st.id
WHERE st.task_id = ANY(@task_ids::text[]) AND sa.user_id = @user_id;
