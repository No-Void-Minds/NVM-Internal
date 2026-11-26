-- Whiteboard revisi 50: gaya kartu, grup, dan kunci posisi. Semuanya kolom di wb_cards, tanpa tabel baru, jadi
-- deploy/sql/10-grants.sql tidak berubah dan Tempat sampah, pulihkan, salin papan, serta template membawanya tanpa
-- jalur baru.
-- style: gaya kartu dari daftar putih (internal/whiteboard/style.go); batas di sini hanya pengaman.
-- locked: kunci posisi; kartu terkunci tidak bisa digeser, diubah ukuran, diputar, atau dihapus.
-- group_ids: jalur grup kartu ini dari grup terluar ke terdalam. Grup bersarang tidak butuh tabel: grup A berisi
-- grup B berarti setiap anggota B punya jalur yang diawali A. Id yang sama dua kali di satu jalur adalah siklus dan
-- ditolak aplikasi.
ALTER TABLE wb_cards
  ADD COLUMN style jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (octet_length(style::text) <= 1000),
  ADD COLUMN locked boolean NOT NULL DEFAULT false,
  ADD COLUMN group_ids uuid[] NOT NULL DEFAULT '{}' CHECK (cardinality(group_ids) <= 8);
