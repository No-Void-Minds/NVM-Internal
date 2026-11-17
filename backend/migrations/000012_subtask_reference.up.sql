-- Revisi 10: sub task punya dua tautan dengan arah berbeda. reference_url diisi yang menugaskan sebagai
-- acuan kerja (brief, contoh, aset), deliverable_url diisi yang mengerjakan sebagai hasilnya.
-- Dipisah supaya acuan tidak tertimpa saat hasilnya dimasukkan.
ALTER TABLE task_subtasks ADD COLUMN reference_url text NOT NULL DEFAULT '';
