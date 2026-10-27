BEGIN;

CREATE TYPE user_role AS ENUM ('admin', 'manager', 'staff');
CREATE TYPE task_status AS ENUM ('requested', 'on_progress', 'ready', 'completed');
CREATE TYPE task_urgency AS ENUM ('low', 'medium', 'high', 'urgent');
CREATE TYPE attachment_kind AS ENUM ('brief', 'deliverable');

-- Diisi lewat trigger supaya updated_at tidak bergantung pada setiap query UPDATE.
CREATE FUNCTION set_updated_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;
END;
$$;

CREATE TABLE divisions (
  id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name       varchar(50) NOT NULL CHECK (btrim(name) <> ''),
  is_active  boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX divisions_name_lower_key ON divisions (lower(name));

CREATE TABLE users (
  id                   bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name                 varchar(100) NOT NULL CHECK (btrim(name) <> ''),
  username             varchar(50) NOT NULL CHECK (username <> '' AND username = btrim(username)),
  password_hash        text NOT NULL,
  role                 user_role NOT NULL,
  division_id          bigint NOT NULL REFERENCES divisions (id),
  is_active            boolean NOT NULL DEFAULT true,
  must_change_password boolean NOT NULL DEFAULT true,
  failed_login_count   smallint NOT NULL DEFAULT 0 CHECK (failed_login_count >= 0),
  -- 0 belum pernah dikunci, 1 = 5 menit, 2 = 15 menit, 3 = 30 menit (FR-AUTH-09)
  lockout_level        smallint NOT NULL DEFAULT 0 CHECK (lockout_level BETWEEN 0 AND 3),
  locked_until         timestamptz,
  -- Disalin ke JWT; dinaikkan saat logout semua sesi, reset password, atau nonaktif.
  token_version        integer NOT NULL DEFAULT 0,
  last_login_at        timestamptz,
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now()
);
-- Login tidak membedakan huruf besar dan kecil, jadi keunikan juga begitu.
CREATE UNIQUE INDEX users_username_lower_key ON users (lower(username));

CREATE TABLE labels (
  id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name        varchar(50) NOT NULL CHECK (btrim(name) <> ''),
  color       varchar(7) NOT NULL CHECK (color ~ '^#[0-9A-Fa-f]{6}$'),
  -- NULL berarti label berlaku untuk semua divisi.
  division_id bigint REFERENCES divisions (id),
  is_active   boolean NOT NULL DEFAULT true,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX labels_name_lower_key ON labels (lower(name));

CREATE SEQUENCE task_number_seq AS bigint START 1;

-- lpad dengan lebar tetap memotong angka di atas 9999, jadi lebarnya minimal 4 dan boleh lebih.
CREATE FUNCTION next_task_id() RETURNS varchar
LANGUAGE sql AS $$
  SELECT 'T-' || lpad(n::text, greatest(4, length(n::text)), '0')
  FROM (SELECT nextval('task_number_seq') AS n) s;
$$;

CREATE TABLE tasks (
  id              varchar(10) PRIMARY KEY DEFAULT next_task_id(),
  title           varchar(200) NOT NULL CHECK (btrim(title) <> ''),
  description     text,
  division_id     bigint NOT NULL REFERENCES divisions (id),
  requested_by    bigint NOT NULL REFERENCES users (id),
  urgency         task_urgency NOT NULL,
  status          task_status NOT NULL DEFAULT 'requested',
  request_date    timestamptz NOT NULL DEFAULT now(),
  start_date      date,
  due_date        date,
  solved_date     timestamptz,
  deliverable_url text,
  is_migrated     boolean NOT NULL DEFAULT false,
  -- Optimistic locking: UPDATE wajib menyertakan versi yang dibaca klien.
  version         integer NOT NULL DEFAULT 1,
  is_deleted      boolean NOT NULL DEFAULT false,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT tasks_id_format CHECK (id ~ '^T-[0-9]{4,8}$'),
  CONSTRAINT tasks_due_after_start CHECK (due_date IS NULL OR start_date IS NULL OR due_date >= start_date),
  -- Dibuka kembali dari Completed berarti solved_date harus ikut dikosongkan.
  CONSTRAINT tasks_solved_only_when_completed CHECK ((status = 'completed') = (solved_date IS NOT NULL)),
  -- Mencegah tautan javascript: dan sejenisnya (NFR-SEC-06).
  CONSTRAINT tasks_deliverable_url_scheme CHECK (deliverable_url IS NULL OR deliverable_url ~* '^https?://')
);
CREATE INDEX tasks_status_idx ON tasks (status);
CREATE INDEX tasks_division_id_idx ON tasks (division_id);
CREATE INDEX tasks_due_date_idx ON tasks (due_date);

CREATE TABLE task_assignees (
  task_id     varchar(10) NOT NULL REFERENCES tasks (id),
  user_id     bigint NOT NULL REFERENCES users (id),
  assigned_by bigint NOT NULL REFERENCES users (id),
  assigned_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (task_id, user_id)
);
CREATE INDEX task_assignees_user_id_idx ON task_assignees (user_id);

CREATE TABLE task_labels (
  task_id  varchar(10) NOT NULL REFERENCES tasks (id),
  label_id bigint NOT NULL REFERENCES labels (id),
  PRIMARY KEY (task_id, label_id)
);
-- Dipakai saat mengecek apakah label masih terpasang sebelum dinonaktifkan.
CREATE INDEX task_labels_label_id_idx ON task_labels (label_id);

CREATE TABLE task_attachments (
  -- Juga dipakai sebagai nama berkas di penyimpanan.
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  task_id       varchar(10) NOT NULL REFERENCES tasks (id),
  kind          attachment_kind NOT NULL,
  original_name text NOT NULL,
  mime_type     varchar(255) NOT NULL,
  -- Batas 50 MB (FR-TASK-15) dicek di backend dan Nginx; ini pengaman terakhir.
  size_bytes    bigint NOT NULL CHECK (size_bytes > 0 AND size_bytes <= 52428800),
  checksum      varchar(64) NOT NULL CHECK (checksum ~ '^[0-9a-f]{64}$'),
  uploaded_by   bigint NOT NULL REFERENCES users (id),
  uploaded_at   timestamptz NOT NULL DEFAULT now(),
  is_deleted    boolean NOT NULL DEFAULT false
);
CREATE INDEX task_attachments_task_id_idx ON task_attachments (task_id);

-- Juga menyimpan riwayat status task (action = task_status_changed).
CREATE TABLE audit_logs (
  id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  -- NULL bila login gagal dengan username yang tidak terdaftar.
  user_id     bigint REFERENCES users (id),
  action      varchar(50) NOT NULL,
  entity      varchar(50),
  entity_id   varchar(50),
  old_value   jsonb,
  new_value   jsonb,
  ip_address  inet,
  occurred_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_logs_occurred_at_idx ON audit_logs (occurred_at);
CREATE INDEX audit_logs_entity_idx ON audit_logs (entity, entity_id);

CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER tasks_set_updated_at BEFORE UPDATE ON tasks
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;