-- Revisi 9: hasil sub task tidak selalu berupa berkas. Desain di Figma, video di Drive, dan draft di
-- Docs lebih sering ditautkan, jadi sub task punya tautan deliverable sendiri seperti task induk.
ALTER TABLE task_subtasks ADD COLUMN deliverable_url text NOT NULL DEFAULT '';
