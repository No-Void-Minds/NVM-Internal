ALTER TABLE wb_boards DROP COLUMN resync_seq;
DROP TABLE wb_comments;
DROP TABLE wb_connectors;
DROP INDEX wb_cards_column_idx;
DROP INDEX wb_cards_file_idx;
-- Kartu jenis W3 dihapus dulu supaya CHECK jenis W1 bisa dipasang lagi.
DELETE FROM wb_cards WHERE kind IN ('image', 'file', 'link', 'column', 'drawing', 'table');
ALTER TABLE wb_cards
  DROP CONSTRAINT wb_cards_column_ref,
  DROP CONSTRAINT wb_cards_file_ref,
  DROP COLUMN col_order,
  DROP COLUMN column_id,
  DROP COLUMN file_id,
  DROP CONSTRAINT wb_cards_kind_check;
ALTER TABLE wb_cards ADD CONSTRAINT wb_cards_kind_check CHECK (kind IN ('note', 'heading', 'checklist', 'board'));
DROP TABLE wb_link_previews;
DROP TABLE wb_files;
