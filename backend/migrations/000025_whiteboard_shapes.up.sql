-- Whiteboard W5 (revisi C, docs/whiteboard.md bagian 9): kartu jenis shape untuk bentuk, flowchart, UML, dan panah
-- blok. Isinya divalidasi aplikasi (internal/whiteboard/content_w5.go); ukuran memakai w dan h kartu yang sudah ada.
-- Garis tidak dimigrasi: port, jalur, titik belok, jenis ujung, dan tiga label sudah ada sejak 000024.
ALTER TABLE wb_cards DROP CONSTRAINT wb_cards_kind_check;
ALTER TABLE wb_cards
  ADD CONSTRAINT wb_cards_kind_check
    CHECK (kind IN ('note', 'heading', 'checklist', 'board', 'image', 'file', 'link', 'column', 'drawing', 'table',
                    'shape'));
