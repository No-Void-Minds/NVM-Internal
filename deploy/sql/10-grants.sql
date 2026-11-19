-- Dijalankan setelah setiap migrasi yang menambah tabel, sebagai nvm_internal_migrator (pemilik tabel):
--   psql -U nvm_internal_migrator -d nvm_internal_prod -f 10-grants.sql
-- Aman dijalankan berulang. Daftar DELETE mengikuti audit query di S12-09 (docs/sprint.md): hanya tabel yang
-- memang dihapus permanen oleh kode. Task dan lampiran memakai is_deleted, jadi tidak ada di sini.

\set ON_ERROR_STOP on

-- Disetel ulang dulu supaya tabel yang keluar dari daftar ini kehilangan DELETE-nya.
REVOKE DELETE ON ALL TABLES IN SCHEMA public FROM nvm_internal_app;

GRANT DELETE ON
  task_subtasks,         -- DeleteSubtasks
  subtask_assignees,     -- RemoveSubtaskAssignees (juga lewat ON DELETE CASCADE)
  task_assignees,        -- RemoveTaskAssignees saat Manajer mengganti PIC task (revisi 6)
  task_labels,           -- DeleteTaskLabels
  activity_attachments,  -- DeleteActivityAttachment: lampiran aktivitas dihapus permanen (revisi 24)
  user_todos,            -- DeleteTodo dan DeleteDoneTodos: To Do pribadi dihapus permanen (revisi 28)
  user_sessions,         -- job maintenance: sesi kedaluwarsa
  audit_logs             -- job maintenance: audit log login lebih dari 1 tahun
TO nvm_internal_app;

-- daily_activities (migrasi 000016, revisi 21) sengaja tidak ada di daftar DELETE: sel yang dikosongkan
-- disimpan sebagai NULL lewat UPDATE, barisnya tidak pernah dihapus. Lampiran aktivitas (migrasi 000017,
-- revisi 24) justru sebaliknya: berkasnya bisa 95 MB, jadi hapus berarti baris dan berkasnya ikut hilang,
-- dan tabelnya masuk daftar DELETE di atas. user_todos (migrasi 000019, revisi 28) juga: To Do pribadi
-- dihapus permanen, tanpa jejak, karena itu catatan pribadi. Daftar itu sekarang delapan tabel.

-- task_series (migrasi 000015, revisi 19) sengaja tidak ada di daftar DELETE: seri hanya dihentikan
-- (active = false) dan templatnya JSONB yang diganti utuh lewat UPDATE. SELECT, INSERT, dan UPDATE-nya datang dari
-- ALTER DEFAULT PRIVILEGES di 00-bootstrap.sql, termasuk sequence identity-nya.

-- drive_exports, drive_folders, dan drive_export_runs (migrasi 000018, revisi 25) juga tidak ada di daftar
-- DELETE: ketiganya hanya ditambah dan dibaca. Salinan ke Drive tidak pernah menghapus apa pun.

-- Bukti untuk runbook: daftar tabel yang boleh dihapus app.
SELECT table_name FROM information_schema.role_table_grants
WHERE grantee = 'nvm_internal_app' AND privilege_type = 'DELETE'
ORDER BY table_name;
