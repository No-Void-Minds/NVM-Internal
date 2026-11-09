BEGIN;

DROP TABLE subtask_assignees;
DROP TABLE task_subtasks;

-- Hanya strukturnya yang kembali; PIC yang sudah ada tidak bisa dipulihkan.
CREATE TABLE task_assignees (
  task_id     varchar(10) NOT NULL REFERENCES tasks (id),
  user_id     bigint NOT NULL REFERENCES users (id),
  assigned_by bigint NOT NULL REFERENCES users (id),
  assigned_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (task_id, user_id)
);
CREATE INDEX task_assignees_user_id_idx ON task_assignees (user_id);

COMMIT;