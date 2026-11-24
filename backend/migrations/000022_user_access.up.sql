-- Revisi 38: akses tambahan per pengguna untuk Timeline dan Important Notes. Kolom di users, bukan tabel
-- terpisah, karena aksesnya tetap dua dan dibaca bersama data sesi di setiap permintaan, jadi pencabutan
-- langsung berlaku tanpa login ulang. Hanya Staf yang menyimpan akses: Manajer selalu punya keduanya (dihitung
-- di kode), Admin hanya membaca. CHECK menjaga aturan itu walau kode lupa mengosongkannya.
ALTER TABLE users
  ADD COLUMN can_manage_timeline boolean NOT NULL DEFAULT false,
  ADD COLUMN can_manage_notes boolean NOT NULL DEFAULT false,
  ADD CONSTRAINT users_access_staff_only CHECK (role = 'staff' OR NOT (can_manage_timeline OR can_manage_notes));
