-- Revisi 21: Aktivitas Harian, pengganti spreadsheet "Daily Activity Monitoring". Satu baris per orang,
-- tanggal, dan slot jam. Isian kosong disimpan sebagai NULL dan barisnya dibiarkan, jadi app tidak pernah
-- menghapus di sini dan tidak butuh hak DELETE.
CREATE TABLE daily_activities (
  id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id       bigint NOT NULL REFERENCES users (id),
  activity_date date NOT NULL,
  -- Nomor slot jam, 1 sampai 8, daftarnya ada di kode backend (activity.Slots). Batas atas dilonggarkan
  -- supaya menambah slot nanti tidak perlu migrasi, tetapi angka liar tetap tertolak.
  slot          smallint NOT NULL CHECK (slot BETWEEN 1 AND 24),
  activity      text CHECK (char_length(activity) <= 1000),
  output        text CHECK (char_length(output) <= 1000),
  -- Siapa yang terakhir menyimpan. Berbeda dari user_id hanya bila Manajer mengubah isian orang lain.
  updated_by    bigint NOT NULL REFERENCES users (id),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT daily_activities_cell_key UNIQUE (user_id, activity_date, slot)
);

-- Halaman Aktivitas dan ekspor selalu menyaring per tanggal lebih dulu.
CREATE INDEX daily_activities_date_idx ON daily_activities (activity_date);

CREATE TRIGGER daily_activities_set_updated_at BEFORE UPDATE ON daily_activities
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();
