-- Revisi 30: halaman Timeline. Kalender bersama berisi launching, produksi, dan kehadiran, plus Important Notes.
-- Hapus memakai is_deleted, jadi app tidak butuh hak DELETE di kedua tabel ini (10-grants.sql tetap delapan tabel).
CREATE TYPE timeline_category AS ENUM ('launching', 'produksi', 'kehadiran');
CREATE TYPE absence_type AS ENUM ('sakit', 'izin_pribadi', 'cuti_pribadi', 'cuti_khusus', 'tukar_libur', 'tukar_shift');

CREATE TABLE timeline_events (
  id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  category     timeline_category NOT NULL,
  title        text CHECK (title IS NULL OR (btrim(title) <> '' AND char_length(title) <= 200)),
  description  text CHECK (description IS NULL OR char_length(description) <= 2000),
  start_date   date NOT NULL,
  -- NULL berarti satu hari. Rentang paling panjang 62 hari termasuk hari pertama dan terakhir.
  end_date     date CHECK (end_date IS NULL OR (end_date >= start_date AND end_date - start_date <= 61)),
  user_id      bigint REFERENCES users (id),
  absence_type absence_type,
  created_by   bigint NOT NULL REFERENCES users (id),
  updated_by   bigint NOT NULL REFERENCES users (id),
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  is_deleted   boolean NOT NULL DEFAULT false,
  -- Kehadiran milik satu orang dan wajib beralasan; launching dan produksi wajib berjudul dan tidak milik orang.
  CONSTRAINT timeline_events_kind CHECK (
    (category = 'kehadiran' AND user_id IS NOT NULL AND absence_type IS NOT NULL)
    OR (category <> 'kehadiran' AND title IS NOT NULL AND user_id IS NULL AND absence_type IS NULL)
  )
);

CREATE INDEX timeline_events_dates_idx ON timeline_events (start_date, end_date);

CREATE TRIGGER timeline_events_set_updated_at BEFORE UPDATE ON timeline_events
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE timeline_notes (
  id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  title      text NOT NULL CHECK (btrim(title) <> '' AND char_length(title) <= 120),
  body       text NOT NULL DEFAULT '' CHECK (char_length(body) <= 2000),
  position   integer NOT NULL CHECK (position >= 0),
  created_by bigint NOT NULL REFERENCES users (id),
  updated_by bigint NOT NULL REFERENCES users (id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  is_deleted boolean NOT NULL DEFAULT false
);

CREATE TRIGGER timeline_notes_set_updated_at BEFORE UPDATE ON timeline_notes
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
