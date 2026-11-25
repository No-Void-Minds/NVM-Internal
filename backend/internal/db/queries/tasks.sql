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


-- name: AddTaskLabel :exec
INSERT INTO task_labels (task_id, label_id) VALUES (@task_id, @label_id);

-- name: GetTaskDetail :one
SELECT t.id, t.title, t.description, t.status, t.urgency, t.division_id, d.name AS division_name,
       t.requested_by, r.name AS requested_by_name, t.request_date, t.start_date, t.due_date,
       t.solved_date, t.deliverable_url, t.is_migrated, t.version, t.series_id, t.occurrence_date
FROM tasks t
JOIN divisions d ON d.id = t.division_id
JOIN users r ON r.id = t.requested_by
WHERE t.id = @id AND NOT t.is_deleted;

-- name: ListTasks :many
-- Daftar sederhana untuk Sprint 3; papan dengan filter lengkap dibuat di Sprint 4.
SELECT t.id, t.title, t.status, t.urgency, t.division_id, d.name AS division_name,
       t.start_date, t.due_date, t.solved_date, t.version,
       t.series_id, t.occurrence_date, s.freq AS series_freq, t.requested_by
FROM tasks t
JOIN divisions d ON d.id = t.division_id
LEFT JOIN task_series s ON s.id = t.series_id
WHERE NOT t.is_deleted
  AND (sqlc.narg(assignee_id)::bigint IS NULL
       OR EXISTS (SELECT 1 FROM task_assignees tas
                  WHERE tas.task_id = t.id AND tas.user_id = sqlc.narg(assignee_id)::bigint)
       -- Revisi 15: "Task saya" juga memuat task yang sub task-nya dipegang orang ini.
       OR EXISTS (SELECT 1 FROM task_subtasks st
                  JOIN subtask_assignees sa ON sa.subtask_id = st.id
                  WHERE st.task_id = t.id AND sa.user_id = sqlc.narg(assignee_id)::bigint)
       -- Revisi 47: sama dengan papan sejak revisi 45, task yang diminta orang ini juga termasuk.
       OR t.requested_by = sqlc.narg(assignee_id)::bigint)
ORDER BY t.created_at DESC;

-- name: ListTaskAssignees :many
-- PIC dipilih Manajer di task (revisi 6). Urutan pemasangan dipertahankan: PIC pertama menentukan divisi task.
SELECT tas.task_id, u.id, u.name, u.username, u.avatar_id
FROM task_assignees tas
JOIN users u ON u.id = tas.user_id
WHERE tas.task_id = ANY(@task_ids::text[])
ORDER BY tas.task_id, tas.assigned_at, u.id;

-- name: ListTaskLabels :many
SELECT tl.task_id, l.id, l.name, l.color, l.is_active
FROM task_labels tl
JOIN labels l ON l.id = tl.label_id
WHERE tl.task_id = ANY(@task_ids::text[])
ORDER BY tl.task_id, lower(l.name);

-- name: ListTaskHistory :many
SELECT a.id, a.occurred_at, a.action, a.old_value, a.new_value, actor.name AS actor_name, a.user_id AS actor_id
FROM audit_logs a
LEFT JOIN users actor ON actor.id = a.user_id
WHERE a.entity = 'task' AND a.entity_id = @task_id::text
ORDER BY a.occurred_at DESC, a.id DESC;

-- name: ListBoardTasks :many
-- Kolomnya sama persis dengan ListTasks supaya hasilnya bisa dikonversi dan dilengkapi PIC serta label
-- dengan kode yang sama. Completed hanya yang selesai sejak @completed_since; sisanya hanya ada di Arsip.
SELECT t.id, t.title, t.status, t.urgency, t.division_id, d.name AS division_name,
       t.start_date, t.due_date, t.solved_date, t.version,
       t.series_id, t.occurrence_date, s.freq AS series_freq, t.requested_by
FROM tasks t
JOIN divisions d ON d.id = t.division_id
LEFT JOIN task_series s ON s.id = t.series_id
WHERE NOT t.is_deleted
  AND (t.status <> 'completed' OR t.solved_date >= @completed_since::timestamptz)
  -- Revisi 15: filter PIC juga cocok lewat PIC sub task. Dua EXISTS, jadi task yang cocok lewat dua jalur
  -- tetap satu baris. Blok ini sama di ListBoardTasks dan ListArchiveTasks.
  AND ((cardinality(@pic_ids::bigint[]) = 0 AND cardinality(@pic_usernames::text[]) = 0)
       OR EXISTS (SELECT 1 FROM task_assignees tas
                  JOIN users u ON u.id = tas.user_id
                  WHERE tas.task_id = t.id
                    AND (tas.user_id = ANY(@pic_ids::bigint[]) OR lower(u.username) = ANY(@pic_usernames::text[])))
       OR EXISTS (SELECT 1 FROM task_subtasks st
                  JOIN subtask_assignees sa ON sa.subtask_id = st.id
                  JOIN users u ON u.id = sa.user_id
                  WHERE st.task_id = t.id
                    AND (sa.user_id = ANY(@pic_ids::bigint[]) OR lower(u.username) = ANY(@pic_usernames::text[])))
       -- Revisi 45: "Task saya" juga memuat task yang diminta orang ini (requested_by). Tetap satu baris per task.
       OR t.requested_by = sqlc.narg(requester_id)::bigint)
  AND (sqlc.narg(division_id)::bigint IS NULL OR t.division_id = sqlc.narg(division_id)::bigint)
  AND (sqlc.narg(label_id)::bigint IS NULL
       OR EXISTS (SELECT 1 FROM task_labels tl WHERE tl.task_id = t.id AND tl.label_id = sqlc.narg(label_id)::bigint))
  AND (sqlc.narg(urgency)::task_urgency IS NULL OR t.urgency = sqlc.narg(urgency)::task_urgency)
  AND (sqlc.narg(due)::text IS NULL
       OR (sqlc.narg(due)::text = 'overdue' AND t.due_date < @today::date AND t.status <> 'completed')
       OR (sqlc.narg(due)::text = 'today' AND t.due_date = @today::date)
       OR (sqlc.narg(due)::text = 'week' AND t.due_date BETWEEN @today::date AND (@today::date) + 6)
       OR (sqlc.narg(due)::text = 'none' AND t.due_date IS NULL))
         -- Blok kata kunci ini harus sama dengan ListArchiveTasks. Diubah di satu tempat, ubah di tempat lain.
  AND (sqlc.narg(pattern)::text IS NULL OR (
        t.id ILIKE sqlc.narg(pattern)::text
        OR t.title ILIKE sqlc.narg(pattern)::text
        OR t.description ILIKE sqlc.narg(pattern)::text
        OR t.deliverable_url ILIKE sqlc.narg(pattern)::text
        OR EXISTS (SELECT 1 FROM users r WHERE r.id = t.requested_by AND r.name ILIKE sqlc.narg(pattern)::text)
        OR EXISTS (SELECT 1 FROM task_subtasks st
                   WHERE st.task_id = t.id AND st.title ILIKE sqlc.narg(pattern)::text)
        OR EXISTS (SELECT 1 FROM task_assignees tas
                   JOIN users u ON u.id = tas.user_id
                   WHERE tas.task_id = t.id AND u.name ILIKE sqlc.narg(pattern)::text)
        OR EXISTS (SELECT 1 FROM task_labels tl JOIN labels l ON l.id = tl.label_id
                   WHERE tl.task_id = t.id AND l.name ILIKE sqlc.narg(pattern)::text)
        OR EXISTS (SELECT 1 FROM task_attachments ta
                   WHERE ta.task_id = t.id AND NOT ta.is_deleted AND ta.original_name ILIKE sqlc.narg(pattern)::text)))
-- Bobot urgency, bukan urutan abjad (FR-TASK-14). Kolom Completed diurutkan ulang di Go berdasarkan solved_date.
ORDER BY CASE t.urgency WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END,
         t.due_date NULLS LAST, t.created_at;

-- name: ListPeople :many
-- Calon PIC untuk filter papan dan form task: aturannya sama dengan GetEligiblePICs.
SELECT u.id, u.name, u.username, u.role, d.name AS division_name, u.avatar_id
FROM users u
JOIN divisions d ON d.id = u.division_id
WHERE u.is_active AND u.role IN ('staff', 'manager')
ORDER BY lower(u.name);

-- name: ListArchiveTasks :many
-- Kolom sama dengan ListTasks, ditambah total_count: jumlah seluruh hasil sebelum LIMIT, supaya halaman
-- dan total didapat dari satu query. Pola @pattern sudah berisi tanda % dan wildcard pengguna sudah di-escape.
SELECT t.id, t.title, t.status, t.urgency, t.division_id, d.name AS division_name,
       t.start_date, t.due_date, t.solved_date, t.version,
       t.series_id, t.occurrence_date, s.freq AS series_freq, t.requested_by,
       count(*) OVER () AS total_count
FROM tasks t
JOIN divisions d ON d.id = t.division_id
LEFT JOIN task_series s ON s.id = t.series_id
WHERE NOT t.is_deleted AND t.status = 'completed'
  -- Revisi 19: "Lihat kemunculan lain" dari detail task berulang.
  AND (sqlc.narg(series_id)::bigint IS NULL OR t.series_id = sqlc.narg(series_id)::bigint)
  -- Revisi 15: filter PIC juga cocok lewat PIC sub task. Dua EXISTS, jadi task yang cocok lewat dua jalur
  -- tetap satu baris. Blok ini sama di ListBoardTasks dan ListArchiveTasks.
  AND ((cardinality(@pic_ids::bigint[]) = 0 AND cardinality(@pic_usernames::text[]) = 0)
       OR EXISTS (SELECT 1 FROM task_assignees tas
                  JOIN users u ON u.id = tas.user_id
                  WHERE tas.task_id = t.id
                    AND (tas.user_id = ANY(@pic_ids::bigint[]) OR lower(u.username) = ANY(@pic_usernames::text[])))
       OR EXISTS (SELECT 1 FROM task_subtasks st
                  JOIN subtask_assignees sa ON sa.subtask_id = st.id
                  JOIN users u ON u.id = sa.user_id
                  WHERE st.task_id = t.id
                    AND (sa.user_id = ANY(@pic_ids::bigint[]) OR lower(u.username) = ANY(@pic_usernames::text[])))
       -- Revisi 47: Arsip "Task saya" mengikuti papan (revisi 45), termasuk task yang diminta orang ini.
       OR t.requested_by = sqlc.narg(requester_id)::bigint)
  AND (sqlc.narg(division_id)::bigint IS NULL OR t.division_id = sqlc.narg(division_id)::bigint)
  AND (sqlc.narg(label_id)::bigint IS NULL
       OR EXISTS (SELECT 1 FROM task_labels tl WHERE tl.task_id = t.id AND tl.label_id = sqlc.narg(label_id)::bigint))
  AND (sqlc.narg(urgency)::task_urgency IS NULL OR t.urgency = sqlc.narg(urgency)::task_urgency)
  AND (sqlc.narg(solved_from)::timestamptz IS NULL OR t.solved_date >= sqlc.narg(solved_from)::timestamptz)
  AND (sqlc.narg(solved_to)::timestamptz IS NULL OR t.solved_date < sqlc.narg(solved_to)::timestamptz)
  AND (sqlc.narg(pattern)::text IS NULL OR (
        t.id ILIKE sqlc.narg(pattern)::text
        OR t.title ILIKE sqlc.narg(pattern)::text
        OR t.description ILIKE sqlc.narg(pattern)::text
        OR t.deliverable_url ILIKE sqlc.narg(pattern)::text
        OR EXISTS (SELECT 1 FROM users r WHERE r.id = t.requested_by AND r.name ILIKE sqlc.narg(pattern)::text)
        OR EXISTS (SELECT 1 FROM task_subtasks st
                   WHERE st.task_id = t.id AND st.title ILIKE sqlc.narg(pattern)::text)
        OR EXISTS (SELECT 1 FROM task_assignees tas
                   JOIN users u ON u.id = tas.user_id
                   WHERE tas.task_id = t.id AND u.name ILIKE sqlc.narg(pattern)::text)
        OR EXISTS (SELECT 1 FROM task_labels tl JOIN labels l ON l.id = tl.label_id
                   WHERE tl.task_id = t.id AND l.name ILIKE sqlc.narg(pattern)::text)
        OR EXISTS (SELECT 1 FROM task_attachments ta
                   WHERE ta.task_id = t.id AND NOT ta.is_deleted AND ta.original_name ILIKE sqlc.narg(pattern)::text)))
-- Tiga urutan dalam satu query: bobot urgency (FR-TASK-14), terlama selesai, dan terbaru selesai (bawaan).
ORDER BY
  CASE WHEN @sort::text = 'urgency'
       THEN CASE t.urgency WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END END,
  CASE WHEN @sort::text = 'oldest' THEN t.solved_date END ASC,
  CASE WHEN @sort::text <> 'oldest' THEN t.solved_date END DESC,
  t.id DESC
LIMIT @row_limit::int OFFSET @row_offset::int;

-- name: ListSubtaskPICs :many
-- Revisi 22: semua PIC sub task per task untuk kelompok "Sub task" di kartu, satu query untuk semua kartu.
-- Unik per orang, urut menurut posisi sub task pertama yang dia pegang.
SELECT st.task_id, u.id, u.name, u.username, u.avatar_id, min(st.position)::int AS first_position
FROM task_subtasks st
JOIN subtask_assignees sa ON sa.subtask_id = st.id
JOIN users u ON u.id = sa.user_id
WHERE st.task_id = ANY(@task_ids::text[])
GROUP BY st.task_id, u.id
ORDER BY st.task_id, first_position, u.id;

-- name: ListUserNames :many
-- Nama untuk id yang sudah diketahui, mis. PIC di templat seri (revisi 19). Akun nonaktif ikut, supaya
-- templat lama tetap bisa ditampilkan.
SELECT id, name FROM users WHERE id = ANY(@ids::bigint[]);

-- name: TaskLiveState :one
-- Revisi 47: versi terbaru dan nama orang terakhir yang mengubah task (baris audit terakhir), untuk event papan
-- realtime dan pesan 409. Task yang sudah dihapus tetap terbaca supaya event hapus membawa versinya.
SELECT t.version, COALESCE(last.user_id, 0)::bigint AS last_actor_id, COALESCE(last.name, '')::text AS last_actor
FROM tasks t
LEFT JOIN LATERAL (SELECT a.user_id, u.name FROM audit_logs a JOIN users u ON u.id = a.user_id
                   WHERE a.entity = 'task' AND a.entity_id = t.id
                   ORDER BY a.occurred_at DESC, a.id DESC LIMIT 1) last ON true
WHERE t.id = @id;
