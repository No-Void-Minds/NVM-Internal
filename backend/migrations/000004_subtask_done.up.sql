BEGIN;

-- done_at dan done_by selalu terisi berpasangan, jadi tidak ada centang tanpa tahu siapa yang mencentang.
ALTER TABLE task_subtasks
  ADD COLUMN done_at timestamptz,
  ADD COLUMN done_by bigint REFERENCES users (id),
  ADD CONSTRAINT task_subtasks_done_pair CHECK ((done_at IS NULL) = (done_by IS NULL));

COMMIT;