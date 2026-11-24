-- name: ListUsers :many
SELECT u.id, u.name, u.username, u.role, u.division_id, d.name AS division_name,
       u.is_active, u.locked_until, u.last_login_at, u.created_at, u.job_title, u.avatar_id,
       u.can_manage_timeline, u.can_manage_notes
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
SELECT id, name, username, role, division_id, is_active, locked_until,
       can_manage_timeline, can_manage_notes
FROM users
WHERE id = @id
FOR UPDATE;

-- name: GetActiveDivision :one
SELECT id, name FROM divisions WHERE id = @id AND is_active;

-- name: CreateUser :one
INSERT INTO users (name, username, password_hash, role, division_id, must_change_password,
                   can_manage_timeline, can_manage_notes)
VALUES (@name, @username, @password_hash, @role, @division_id, true,
        @can_manage_timeline, @can_manage_notes)
RETURNING id;

-- name: UpdateUserProfile :exec
UPDATE users
SET name = @name, role = @role, division_id = @division_id,
    can_manage_timeline = @can_manage_timeline, can_manage_notes = @can_manage_notes
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
-- name: GetOwnProfileForUpdate :one
SELECT id, name, job_title, avatar_id
FROM users
WHERE id = @id
FOR UPDATE;

-- name: UpdateOwnProfile :exec
-- Hanya nama dan jabatan. Divisi dan peran tetap lewat UpdateUserProfile oleh Admin atau Manajer.
UPDATE users
SET name = @name, job_title = sqlc.narg(job_title)
WHERE id = @id;

-- name: SetUserAvatar :exec
UPDATE users
SET avatar_id = sqlc.narg(avatar_id)
WHERE id = @id;

-- name: GetUserAvatar :one
SELECT avatar_id FROM users WHERE id = @id;

-- name: GetPersonCard :one
-- Kartu profil (revisi 14) untuk semua pengguna yang login. Kolom dipilih satu per satu supaya username,
-- password, dan jejak login tidak pernah ikut terbaca.
SELECT u.id, u.name, d.name AS division_name, u.job_title, u.role, u.is_active, u.avatar_id
FROM users u
JOIN divisions d ON d.id = u.division_id
WHERE u.id = @id;
