DROP INDEX IF EXISTS task_attachments_subtask_id_idx;
ALTER TABLE task_attachments DROP COLUMN subtask_id;
ALTER TABLE task_subtasks DROP COLUMN description;
