BEGIN;

-- Satu baris per login. Idle timeout 30 menit ditegakkan dari last_seen_at di server, bukan dari umur cookie,
-- supaya pemakaian dashboard lewat Caddy forward_auth (yang tidak meneruskan Set-Cookie) tetap terhitung aktivitas.
-- Sesi tidak dihapus saat logout: logout menaikkan token_version, yang sudah membatalkan semua token akun itu.
-- Baris lama dibersihkan job harian S12-07.
CREATE TABLE user_sessions (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      bigint NOT NULL REFERENCES users (id),
  created_at   timestamptz NOT NULL DEFAULT now(),
  last_seen_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX user_sessions_last_seen_idx ON user_sessions (last_seen_at);

COMMIT;
