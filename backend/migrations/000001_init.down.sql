BEGIN;

DROP TABLE IF EXISTS audit_logs, task_attachments, task_labels, task_assignees, tasks, labels, users, divisions;
DROP FUNCTION IF EXISTS next_task_id();
DROP SEQUENCE IF EXISTS task_number_seq;
DROP FUNCTION IF EXISTS set_updated_at();
DROP TYPE IF EXISTS attachment_kind, task_urgency, task_status, user_role;

COMMIT;