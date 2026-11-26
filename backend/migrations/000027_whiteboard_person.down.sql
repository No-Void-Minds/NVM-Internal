-- Kartu orang dihapus dulu supaya CHECK jenis W5 bisa dipasang lagi (pola yang sama dengan 000025). Garis yang
-- menempel ke kartu orang ikut terhapus lewat ON DELETE CASCADE.
DELETE FROM wb_cards WHERE kind = 'person';
ALTER TABLE wb_cards DROP CONSTRAINT wb_cards_kind_check;
ALTER TABLE wb_cards
  ADD CONSTRAINT wb_cards_kind_check
    CHECK (kind IN ('note', 'heading', 'checklist', 'board', 'image', 'file', 'link', 'column', 'drawing', 'table',
                    'shape'));
