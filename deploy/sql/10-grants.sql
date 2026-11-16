-- Dijalankan setelah setiap migrasi yang menambah tabel, sebagai nvm_internal_migrator (pemilik tabel):
--   psql -U nvm_internal_migrator -d nvm_internal_prod -f 10-grants.sql
-- Aman dijalankan berulang. Daftar DELETE mengikuti audit query di S12-09 (docs/sprint.md): hanya tabel yang
-- memang dihapus permanen oleh kode. Task dan lampiran memakai is_deleted, jadi tidak ada di sini.

\set ON_ERROR_STOP on

-- Disetel ulang dulu supaya tabel yang keluar dari daftar ini kehilangan DELETE-nya.
REVOKE DELETE ON ALL TABLES IN SCHEMA public FROM nvm_internal_app;

GRANT DELETE ON
  task_subtasks,      -- DeleteSubtasks
  subtask_assignees,  -- RemoveSubtaskAssignees (juga lewat ON DELETE CASCADE)
  task_labels,        -- DeleteTaskLabels
  user_sessions,      -- job maintenance: sesi kedaluwarsa
  audit_logs          -- job maintenance: audit log login lebih dari 1 tahun
TO nvm_internal_app;

-- Bukti untuk runbook: daftar tabel yang boleh dihapus app.
SELECT table_name FROM information_schema.role_table_grants
WHERE grantee = 'nvm_internal_app' AND privilege_type = 'DELETE'
ORDER BY table_name;
