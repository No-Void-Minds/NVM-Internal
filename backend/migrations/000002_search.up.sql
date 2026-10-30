-- Pencarian arsip (FR-TASK-13, NFR-PERF-03). pg_trgm dipilih, bukan tsvector: PostgreSQL tidak punya kamus
-- bahasa Indonesia, dan tim mencari potongan kata atau ID ("bann", "T-007") yang tidak dilayani full-text search.
-- Trusted extension sejak PostgreSQL 13, jadi cukup dipasang oleh pemilik database.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX tasks_id_trgm_idx ON tasks USING gin (id gin_trgm_ops);
CREATE INDEX tasks_title_trgm_idx ON tasks USING gin (title gin_trgm_ops);
CREATE INDEX tasks_description_trgm_idx ON tasks USING gin (description gin_trgm_ops);
CREATE INDEX tasks_deliverable_url_trgm_idx ON tasks USING gin (deliverable_url gin_trgm_ops);
CREATE INDEX task_attachments_name_trgm_idx ON task_attachments USING gin (original_name gin_trgm_ops);

-- Dari nama pemohon yang cocok ke task-nya. Index PIC dan label sudah ada sejak migrasi pertama.
-- users dan labels tidak diberi index trigram: isinya puluhan baris, dan pemindaian langsung lebih cepat.
CREATE INDEX tasks_requested_by_idx ON tasks (requested_by);

-- Urutan bawaan Arsip: yang terakhir selesai di atas. Parsial supaya index hanya berisi task Completed.
CREATE INDEX tasks_archive_idx ON tasks (solved_date DESC) WHERE status = 'completed' AND NOT is_deleted;