DROP INDEX IF EXISTS tasks_archive_idx;
DROP INDEX IF EXISTS tasks_requested_by_idx;
DROP INDEX IF EXISTS task_attachments_name_trgm_idx;
DROP INDEX IF EXISTS tasks_deliverable_url_trgm_idx;
DROP INDEX IF EXISTS tasks_description_trgm_idx;
DROP INDEX IF EXISTS tasks_title_trgm_idx;
DROP INDEX IF EXISTS tasks_id_trgm_idx;
DROP EXTENSION IF EXISTS pg_trgm;