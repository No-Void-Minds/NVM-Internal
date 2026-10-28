-- name: ListUsers :many
SELECT u.id, u.name, u.username, u.role, u.division_id, d.name AS division_name,
       u.is_active, u.locked_until, u.last_login_at, u.created_at
FROM users u
JOIN divisions d ON d.id = u.division_id
ORDER BY u.is_active DESC, lower(u.name);

-- name: LockActiveManagers :many
-- Mengunci semua baris Admin dan Manajer aktif, selalu sebelum mengunci akun sasaran dan dalam urutan id
-- yang sama, supaya dua aksi bersamaan tidak sama-sama lolos cek pengelola terakhir dan tidak saling menunggu.
SELECT id FROM users
WHERE is_active AND role IN ('admin', 'manager')
ORDER BY id
FOR UPDATE;

-- name: GetUserForUpdate :one
SELECT id, name, username, role, division_id, is_active, locked_until
FROM users
WHERE id = @id
FOR UPDATE;

-- name: GetActiveDivision :one
SELECT id, name FROM divisions WHERE id = @id AND is_active;

-- name: CreateUser :one
INSERT INTO users (name, username, password_hash, role, division_id, must_change_password)
VALUES (@name, @username, @password_hash, @role, @division_id, true)
RETURNING id;

-- name: UpdateUserProfile :exec
UPDATE users
SET name = @name, role = @role, division_id = @division_id
WHERE id = @id;

-- name: SetUserActive :exec
-- token_version naik supaya sesi akun yang dinonaktifkan langsung berakhir (FR-AUTH-08).
UPDATE users
SET is_active = @is_active, token_version = token_version + 1
WHERE id = @id;

-- name: ResetUserPassword :exec
UPDATE users
SET password_hash        = @password_hash,
    must_change_password = true,
    token_version        = token_version + 1,
    failed_login_count   = 0,
    lockout_level        = 0,
    locked_until         = NULL
WHERE id = @id;

-- name: UnlockUser :exec
UPDATE users
SET failed_login_count = 0, lockout_level = 0, locked_until = NULL
WHERE id = @id;