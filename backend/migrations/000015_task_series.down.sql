ALTER TABLE tasks
  DROP CONSTRAINT tasks_series_occurrence_key,
  DROP CONSTRAINT tasks_series_pair,
  DROP COLUMN occurrence_date,
  DROP COLUMN series_id;
DROP TABLE task_series;
DROP TYPE recurrence_freq;
