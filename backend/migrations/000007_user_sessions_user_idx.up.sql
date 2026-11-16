BEGIN;

-- Pencarian sesi per akun (pembersihan, dan pencabutan per akun bila kelak dibutuhkan). Pencarian per sesi sudah
-- memakai primary key id. Dipisah dari 000006 karena 000006 sudah terpasang di database dev.
CREATE INDEX user_sessions_user_id_idx ON user_sessions (user_id);

COMMIT;
