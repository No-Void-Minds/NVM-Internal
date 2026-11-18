-- Revisi 18: catatan hasil sub task, penjelasan dari yang mengerjakan (output di mana, dikirim ke mana, versi
-- mana yang dipakai). Kosong disimpan NULL; batas 2000 karakter dicek di aplikasi, sama dengan deskripsi.
ALTER TABLE task_subtasks ADD COLUMN result_note text;
