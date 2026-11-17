-- Revisi 7: sub task bisa ditugaskan ke PIC task lain, jadi "pembuat" dan "yang mengerjakan" tidak lagi
-- orang yang sama. Baris di subtask_assignees sekarang berarti yang ditugaskan, created_by menyimpan
-- yang menugaskan. Keduanya dipakai menentukan siapa boleh mengubah atau menghapus sub task.
ALTER TABLE task_subtasks ADD COLUMN created_by bigint REFERENCES users (id);

-- Data lama: sebelum revisi 7 pembuat dan PIC sub task selalu orang yang sama, dan assigned_by menyimpannya.
-- min() dipakai karena data dari sebelum revisi 6 bisa punya lebih dari satu PIC per sub task.
UPDATE task_subtasks st
SET created_by = (SELECT min(sa.assigned_by) FROM subtask_assignees sa WHERE sa.subtask_id = st.id)
WHERE EXISTS (SELECT 1 FROM subtask_assignees sa WHERE sa.subtask_id = st.id);
