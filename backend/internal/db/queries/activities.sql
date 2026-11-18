-- name: ListActivityPeople :many
-- Kolom tabel Aktivitas: semua Staf aktif, ditambah akun apa pun yang punya isian di rentang itu (mis. Staf
-- yang sudah dinonaktifkan). Manajer dan Admin tidak punya tabel aktivitas, jadi tanpa isian mereka tidak muncul.
SELECT u.id, u.name, u.role, u.avatar_id, u.is_active, d.name AS division_name
FROM users u
JOIN divisions d ON d.id = u.division_id
WHERE (u.is_active AND u.role = 'staff')
   OR EXISTS (
     SELECT 1 FROM daily_activities a
     WHERE a.user_id = u.id
       AND a.activity_date BETWEEN @from_date::date AND @to_date::date
       AND (a.activity IS NOT NULL OR a.output IS NOT NULL)
   )
ORDER BY lower(d.name), lower(u.name);

-- name: ListDailyActivities :many
-- Semua isian di rentang tanggal, termasuk sel yang isinya sudah dikosongkan (activity dan output NULL),
-- supaya "diubah <nama>" tetap terbaca setelah Manajer menghapus isi sel.
SELECT a.user_id, a.activity_date, a.slot, a.activity, a.output, a.updated_by, ub.name AS updated_by_name
FROM daily_activities a
JOIN users ub ON ub.id = a.updated_by
WHERE a.activity_date BETWEEN @from_date::date AND @to_date::date
ORDER BY a.activity_date, a.slot;

-- name: GetDailyActivity :one
SELECT activity, output FROM daily_activities
WHERE user_id = @user_id AND activity_date = @activity_date AND slot = @slot;

-- name: UpsertDailyActivity :exec
-- Satu baris per sel: unique (user_id, activity_date, slot) membuat simpan berulang tetap satu baris.
INSERT INTO daily_activities (user_id, activity_date, slot, activity, output, updated_by)
VALUES (@user_id, @activity_date, @slot, @activity, @output, @updated_by)
ON CONFLICT (user_id, activity_date, slot) DO UPDATE
SET activity = EXCLUDED.activity, output = EXCLUDED.output, updated_by = EXCLUDED.updated_by;
