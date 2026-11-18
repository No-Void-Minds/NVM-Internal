-- Revisi 16: tenggat sub task, opsional. Tidak dipaksa lewat CHECK terhadap tenggat task induk, karena
-- tenggat task boleh dimajukan setelahnya; batas itu dicek di aplikasi saat tenggat sub task diisi atau diubah.
ALTER TABLE task_subtasks ADD COLUMN due_date date;
