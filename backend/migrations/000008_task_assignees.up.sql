BEGIN;

-- Revisi 6 Manajer: PIC kembali menempel di task, dipilih Manajer saat membuat task. PIC yang sudah ada
-- disalin dari sub task supaya task lama tidak kehilangan pemiliknya.
CREATE TABLE task_assignees (
  task_id     varchar(10) NOT NULL REFERENCES tasks (id),
  user_id     bigint NOT NULL REFERENCES users (id),
  assigned_by bigint NOT NULL REFERENCES users (id),
  assigned_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (task_id, user_id)
);
CREATE INDEX task_assignees_user_id_idx ON task_assignees (user_id);

INSERT INTO task_assignees (task_id, user_id, assigned_by, assigned_at)
SELECT st.task_id, sa.user_id, min(sa.assigned_by), min(sa.assigned_at)
FROM task_subtasks st
JOIN subtask_assignees sa ON sa.subtask_id = st.id
GROUP BY st.task_id, sa.user_id;

COMMIT;
