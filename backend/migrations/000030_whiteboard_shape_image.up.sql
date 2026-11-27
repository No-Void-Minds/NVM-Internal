-- Whiteboard revisi 57b: bingkai ponsel, jendela browser, dan placeholder gambar (kartu bentuk) bisa diisi gambar.
-- Referensi berkasnya di wb_cards.file_id yang sudah ada, bukan di jsonb, supaya FK tanpa ON DELETE menjaga baris
-- wb_files selama kartu ada (juga di Tempat sampah dan di salinan template) dan job pembersih (DeleteWbUnusedFiles,
-- ListAttachmentFileIDs) melihatnya tanpa perubahan. Kartu bentuk boleh punya file_id kosong atau terisi; bentuk mana
-- yang boleh dan jenis berkasnya dicek Go. Hanya constraint yang berubah, jadi deploy/sql/10-grants.sql tidak berubah.
ALTER TABLE wb_cards
  DROP CONSTRAINT wb_cards_file_ref,
  ADD CONSTRAINT wb_cards_file_ref CHECK (
    kind = 'shape' OR (kind IN ('image', 'file')) = (file_id IS NOT NULL)
  );
