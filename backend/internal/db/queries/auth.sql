-- name: GetUserForLogin :one
-- FOR UPDATE mengunci baris selama transaksi login, supaya dua percobaan bersamaan
-- tidak sama-sama membaca hitungan gagal yang lama.
SELECT id, name, username, password_hash, role, division_id, is_active, must_change_password,
       failed_login_count, lockout_level, locked_until, token_version
FROM users
WHERE lower(username) = lower(@username)
FOR UPDATE;

-- name: SaveLockState :exec
UPDATE users
SET failed_login_count = @failed_login_count,
    lockout_level      = @lockout_level,
    locked_until       = @locked_until
WHERE id = @id;

-- name: RecordLoginSuccess :exec
UPDATE users
SET failed_login_count = 0,
    lockout_level      = 0,
    locked_until       = NULL,
    last_login_at      = now()
WHERE id = @id;

-- name: GetSessionUser :one
SELECT u.id, u.name, u.username, u.role, u.division_id, d.name AS division_name,
       u.is_active, u.must_change_password, u.token_version, u.job_title, u.avatar_id,
       u.can_manage_timeline, u.can_manage_notes
FROM users u
JOIN divisions d ON d.id = u.division_id
WHERE u.id = @id;

-- name: BumpTokenVersion :one
UPDATE users
SET token_version = token_version + 1
WHERE id = @id
RETURNING token_version;

-- name: GetPasswordHash :one
SELECT password_hash FROM users WHERE id = @id;

-- name: ChangePassword :one
-- token_version ikut naik supaya sesi di perangkat lain berakhir setelah password diganti.
UPDATE users
SET password_hash        = @password_hash,
    must_change_password = false,
    token_version        = token_version + 1
WHERE id = @id
RETURNING token_version;

-- name: InsertAuditLog :exec
INSERT INTO audit_logs (user_id, action, entity, entity_id, old_value, new_value, ip_address)
VALUES (@user_id, @action, @entity, @entity_id, @old_value, @new_value, @ip_address);
-- name: CreateUserSession :one
-- Waktu dari jam aplikasi, bukan now() database, supaya perbandingan idle tidak terganggu selisih jam dua mesin.
INSERT INTO user_sessions (user_id, created_at, last_seen_at) VALUES (@user_id, @now, @now) RETURNING id;

-- name: GetUserSession :one
-- user_id ikut dicocokkan supaya sid dari token akun lain tidak bisa dipakai.
SELECT created_at, last_seen_at FROM user_sessions WHERE id = @id AND user_id = @user_id;

-- name: TouchUserSession :exec
UPDATE user_sessions SET last_seen_at = @now WHERE id = @id;
