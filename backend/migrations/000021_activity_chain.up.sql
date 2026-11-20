-- Revisi 33: rantai sambung jam. Jam yang menyambung ke jam sebelumnya tidak menyimpan isi sendiri; isinya
-- diturunkan dari jam kepala rantainya di backend (activity.resolveChains), jadi edit di jam kepala langsung
-- tampil di semua jam yang tersambung. Melepas sambungan cukup mengubah penanda ini, bukan menghapus baris,
-- jadi hak DELETE tetap tidak dibutuhkan di tabel ini.
ALTER TABLE daily_activities
  ADD COLUMN continues_previous boolean NOT NULL DEFAULT false,
  -- Pengaman terakhir: jam tersambung tidak boleh punya isi sendiri, mis. karena simpan dan sambung balapan.
  ADD CONSTRAINT daily_activities_chain_empty
    CHECK (NOT continues_previous OR (activity IS NULL AND output IS NULL));
