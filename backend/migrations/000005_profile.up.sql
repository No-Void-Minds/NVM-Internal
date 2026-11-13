BEGIN;

-- Jabatan kosong disimpan sebagai NULL, bukan '', supaya UI cukup memeriksa satu keadaan untuk menampilkan ajakan mengisi.
-- avatar_id sekaligus nama berkas di STORAGE_DIR/avatars/ dan ETag foto: setiap unggahan memberi UUID baru,
-- jadi foto lama tidak pernah tertimpa di tempat dan cache browser otomatis basi.
ALTER TABLE users
  ADD COLUMN job_title varchar(100)
    CHECK (job_title IS NULL OR (btrim(job_title) <> '' AND job_title = btrim(job_title))),
  ADD COLUMN avatar_id uuid;

COMMIT;
