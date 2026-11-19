-- Revisi 28: To Do pribadi di papan. Catatan pribadi per akun, bukan task: tidak punya PIC, status, atau
-- riwayat, dan tidak masuk audit log. Hapus di sini menghapus barisnya, jadi app butuh hak DELETE
-- (lihat deploy/sql/10-grants.sql, daftarnya jadi delapan tabel).
CREATE TABLE user_todos (
  id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id    bigint NOT NULL REFERENCES users (id),
  title      text NOT NULL CHECK (btrim(title) <> '' AND char_length(title) <= 200),
  -- done_at NULL berarti belum selesai. Waktunya disimpan, bukan hanya boolean, supaya "bersihkan yang
  -- selesai" dan urutan item selesai bisa memakai waktunya tanpa kolom tambahan.
  done_at    timestamptz,
  position   integer NOT NULL CHECK (position >= 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- Daftar selalu dibaca per akun dan urut posisi.
CREATE INDEX user_todos_user_position_idx ON user_todos (user_id, position);

CREATE TRIGGER user_todos_set_updated_at BEFORE UPDATE ON user_todos
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
