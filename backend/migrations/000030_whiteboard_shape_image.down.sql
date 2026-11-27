-- Bentuk bergambar tidak memenuhi CHECK lama. Down tidak melepas gambarnya diam-diam: bila masih ada, down gagal
-- dengan jumlah barisnya, dan orang yang menurunkan versi memutuskan sendiri (lepas gambarnya lewat aplikasi atau
-- jalankan UPDATE dengan sadar) sebelum mengulang down.
DO $$
DECLARE n bigint;
BEGIN
  SELECT count(*) INTO n FROM wb_cards WHERE kind = 'shape' AND file_id IS NOT NULL;
  IF n > 0 THEN
    RAISE EXCEPTION 'Masih ada % bentuk bergambar (wb_cards kind shape dengan file_id). Lepas dulu gambarnya, lalu ulangi down.', n;
  END IF;
END $$;
ALTER TABLE wb_cards
  DROP CONSTRAINT wb_cards_file_ref,
  ADD CONSTRAINT wb_cards_file_ref CHECK ((kind IN ('image', 'file')) = (file_id IS NOT NULL));
