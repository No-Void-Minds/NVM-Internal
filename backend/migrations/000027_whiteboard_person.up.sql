-- Whiteboard revisi 51: kartu anggota tim (jenis person). Isinya hanya user_id dan pengaturan tampilan; nama, foto,
-- jabatan, dan divisi selalu dibaca dari users saat papan dimuat (internal/whiteboard/person.go), jadi tidak ada
-- data profil yang disalin ke kartu. user_id divalidasi aplikasi (harus ada). Tanpa tabel baru, jadi
-- deploy/sql/10-grants.sql tidak berubah.
ALTER TABLE wb_cards DROP CONSTRAINT wb_cards_kind_check;
ALTER TABLE wb_cards
  ADD CONSTRAINT wb_cards_kind_check
    CHECK (kind IN ('note', 'heading', 'checklist', 'board', 'image', 'file', 'link', 'column', 'drawing', 'table',
                    'shape', 'person'));
