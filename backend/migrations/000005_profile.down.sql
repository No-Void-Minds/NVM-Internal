BEGIN;

ALTER TABLE users
  DROP COLUMN avatar_id,
  DROP COLUMN job_title;

COMMIT;
