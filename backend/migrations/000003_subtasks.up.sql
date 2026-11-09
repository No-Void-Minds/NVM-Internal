BEGIN;

-- PIC pindah dari task ke sub task. Aplikasi belum dipakai di produksi, jadi tabel lama dibuang tanpa menyalin datanya.
DROP TABLE task_assignees;

CREATE TABLE task_subtasks (
  id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  task_id    varchar(10) NOT NULL REFERENCES tasks (id),
  title      varchar(200) NOT NULL CHECK (btrim(title) <> ''),
  position   integer NOT NULL CHECK (position >= 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  -- Ditunda sampai commit supaya dua sub task bisa bertukar urutan dalam satu transaksi.
  -- Index-nya juga melayani pencarian sub task per task, jadi tidak perlu index terpisah.
  CONSTRAINT task_subtasks_position_key UNIQUE (task_id, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE TRIGGER task_subtasks_set_updated_at BEFORE UPDATE ON task_subtasks
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Sub task dihapus permanen (tanpa kolom is_deleted); jejaknya ada di audit_logs, jadi PIC-nya ikut terhapus.
CREATE TABLE subtask_assignees (
  subtask_id  bigint NOT NULL REFERENCES task_subtasks (id) ON DELETE CASCADE,
  user_id     bigint NOT NULL REFERENCES users (id),
  assigned_by bigint NOT NULL REFERENCES users (id),
  assigned_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (subtask_id, user_id)
);
CREATE INDEX subtask_assignees_user_id_idx ON subtask_assignees (user_id);

COMMIT;