-- Pesan ke diri sendiri tidak memenuhi CHECK lama. Down tidak menghapusnya diam-diam: bila masih ada, down gagal
-- dengan jumlah barisnya, dan orang yang menurunkan versi memutuskan sendiri (hapus garisnya lewat aplikasi atau
-- jalankan DELETE dengan sadar) sebelum mengulang down.
DO $$
DECLARE n bigint;
BEGIN
  SELECT count(*) INTO n FROM wb_connectors WHERE from_card_id = to_card_id;
  IF n > 0 THEN
    RAISE EXCEPTION 'Masih ada % garis pesan ke diri sendiri (wb_connectors dengan from_card_id = to_card_id). Hapus dulu, lalu ulangi down.', n;
  END IF;
END $$;
ALTER TABLE wb_connectors
  DROP CONSTRAINT wb_connectors_two_cards,
  ADD CONSTRAINT wb_connectors_two_cards CHECK (from_card_id IS NULL OR to_card_id IS NULL OR from_card_id <> to_card_id);
