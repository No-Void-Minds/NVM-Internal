-- Revisi 24: lampiran Aktivitas Harian. Berkas menempel pada satu jam (user_id, activity_date, slot), bukan
-- pada kolom Aktivitas atau Output. Tanpa is_deleted: hapus di sini menghapus baris dan berkasnya di disk,
-- jadi app butuh hak DELETE di tabel ini (lihat deploy/sql/10-grants.sql).
CREATE TABLE activity_attachments (
  -- Juga dipakai sebagai nama berkas di penyimpanan, sama seperti task_attachments.
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       bigint NOT NULL REFERENCES users (id),
  activity_date date NOT NULL,
  -- Nomor slot jam, daftarnya di kode backend (activity.Slots). Batas atas sama dengan daily_activities.
  slot          smallint NOT NULL CHECK (slot BETWEEN 1 AND 24),
  original_name text NOT NULL,
  mime_type     varchar(255) NOT NULL,
  -- Batas 95 MB: Cloudflare menolak unggahan di atas 100 MB. Dicek di backend dan Nginx; ini pengaman terakhir.
  size_bytes    bigint NOT NULL CHECK (size_bytes > 0 AND size_bytes <= 99614720),
  checksum      varchar(64) NOT NULL CHECK (checksum ~ '^[0-9a-f]{64}$'),
  uploaded_by   bigint NOT NULL REFERENCES users (id),
  created_at    timestamptz NOT NULL DEFAULT now()
);

-- Tabel dan halaman Aktivitas selalu membaca per orang dan tanggal.
CREATE INDEX activity_attachments_cell_idx ON activity_attachments (user_id, activity_date);
