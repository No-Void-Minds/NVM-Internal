BEGIN;

ALTER TABLE task_subtasks
  DROP CONSTRAINT task_subtasks_done_pair,
  DROP COLUMN done_by,
  DROP COLUMN done_at;

COMMIT;