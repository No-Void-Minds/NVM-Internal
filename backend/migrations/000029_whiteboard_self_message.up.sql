-- Whiteboard revisi 54d: pesan ke diri sendiri di sequence diagram adalah garis yang awal dan ujungnya lifeline yang
-- sama. CHECK lama melarang semua garis seperti itu; sekarang hanya jalur message (pesan sequence) yang boleh.
-- Hanya constraint yang berubah, tanpa tabel baru, jadi deploy/sql/10-grants.sql tidak berubah.
ALTER TABLE wb_connectors
  DROP CONSTRAINT wb_connectors_two_cards,
  ADD CONSTRAINT wb_connectors_two_cards CHECK (
    from_card_id IS NULL OR to_card_id IS NULL OR from_card_id <> to_card_id OR geometry->>'route' = 'message'
  );
