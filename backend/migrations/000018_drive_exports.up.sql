-- Revisi 25: salinan berkas ke Google Drive akun tim. Tiga tabel kecil, semuanya hanya ditambah dan dibaca,
-- jadi app tidak butuh hak DELETE di sini (daftar DELETE di deploy/sql/10-grants.sql tetap tujuh tabel).

-- Satu baris per berkas yang sudah benar-benar terunggah. Dicatat setelah rclone selesai, bukan sebelumnya,
-- supaya unggahan yang gagal dicoba lagi malam berikutnya.
CREATE TABLE drive_exports (
  id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  -- 'task_attachment' atau 'activity_attachment'. Ekspor Excel bulanan dan dump sengaja tidak dicatat:
  -- keduanya memang ditimpa atau diunggah ulang, dan rclone sendiri melewati berkas yang sudah sama.
  source_kind varchar(32) NOT NULL,
  source_id   varchar(64) NOT NULL,
  drive_path  text NOT NULL,
  size_bytes  bigint NOT NULL CHECK (size_bytes >= 0),
  exported_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT drive_exports_source_key UNIQUE (source_kind, source_id)
);

-- Nama folder task dan sub task ditetapkan sekali saat pertama disalin. Ganti judul di aplikasi tidak
-- membuat folder baru di Drive, karena nama yang sudah dipakai dibaca dari sini.
CREATE TABLE drive_folders (
  id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  -- 'task' (source_id = id task) atau 'subtask' (source_id = id sub task).
  scope       varchar(16) NOT NULL,
  source_id   varchar(64) NOT NULL,
  -- Path relatif di Drive, mis. "Task/2026-11/T-0012 Foto katalog".
  folder_path text NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT drive_folders_scope_key UNIQUE (scope, source_id)
);

-- Satu baris per jalannya job. Halaman Status membaca baris terakhir, bukan berkas log.
CREATE TABLE drive_export_runs (
  id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  finished_at timestamptz NOT NULL DEFAULT now(),
  ok          boolean NOT NULL,
  files       integer NOT NULL DEFAULT 0,
  bytes       bigint NOT NULL DEFAULT 0,
  message     text
);
CREATE INDEX drive_export_runs_finished_at_idx ON drive_export_runs (finished_at DESC);
