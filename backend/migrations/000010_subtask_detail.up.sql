-- Revisi 8: sub task jadi unit kerja yang lengkap, bukan sekadar baris centang. Isinya bertambah
-- deskripsi dan lampiran sendiri, supaya PIC yang ditugaskan tahu persis apa yang diminta dan
-- menaruh hasilnya menempel di pekerjaannya, bukan tercampur di lampiran task induk.
ALTER TABLE task_subtasks ADD COLUMN description text NOT NULL DEFAULT '';

-- Lampiran sub task ikut menumpang tabel lampiran task: aturan hak, batas ukuran, pemeriksaan tipe,
-- dan pembersih berkas yatim sudah berlaku untuk tabel ini. subtask_id NULL berarti lampiran task induk.
ALTER TABLE task_attachments ADD COLUMN subtask_id bigint REFERENCES task_subtasks (id) ON DELETE CASCADE;

CREATE INDEX task_attachments_subtask_id_idx ON task_attachments (subtask_id) WHERE subtask_id IS NOT NULL;
