-- Revisi 19: task berulang. Satu seri menyimpan aturan dan templat; setiap kemunculan adalah task biasa dengan
-- ID sendiri. Seri tidak pernah dihapus, hanya dihentikan (active = false), jadi app tidak butuh DELETE di sini.
CREATE TYPE recurrence_freq AS ENUM ('daily', 'weekly', 'monthly');

CREATE TABLE task_series (
  id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  freq         recurrence_freq NOT NULL,
  skip_weekend boolean NOT NULL DEFAULT false,
  -- Hari ISO: 1 Senin sampai 7 Minggu. Hanya untuk mingguan.
  weekdays     smallint[] NOT NULL DEFAULT '{}',
  -- Hanya untuk bulanan. Bulan yang lebih pendek memakai hari terakhirnya (keputusan Sprint 11q).
  month_day    smallint CHECK (month_day BETWEEN 1 AND 31),
  start_date   date NOT NULL,
  end_date     date,
  active       boolean NOT NULL DEFAULT true,
  -- Templat kemunculan berikutnya, diganti utuh setiap kali diperbarui (lihat Keputusan Sprint 11q).
  template     jsonb NOT NULL,
  created_by   bigint NOT NULL REFERENCES users (id),
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT task_series_end_after_start CHECK (end_date IS NULL OR end_date >= start_date),
  CONSTRAINT task_series_weekly_days CHECK ((freq = 'weekly') = (cardinality(weekdays) > 0)),
  CONSTRAINT task_series_monthly_day CHECK ((freq = 'monthly') = (month_day IS NOT NULL))
);

CREATE TRIGGER task_series_set_updated_at BEFORE UPDATE ON task_series
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE tasks
  ADD COLUMN series_id bigint REFERENCES task_series (id),
  ADD COLUMN occurrence_date date,
  ADD CONSTRAINT tasks_series_pair CHECK ((series_id IS NULL) = (occurrence_date IS NULL)),
  -- Membuat pembuatan kemunculan idempoten, juga bila dua instance backend berjalan bersamaan. Index-nya
  -- sekaligus melayani filter Arsip per seri.
  ADD CONSTRAINT tasks_series_occurrence_key UNIQUE (series_id, occurrence_date);
