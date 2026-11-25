-- Whiteboard tahap W3 (docs/whiteboard.md): berkas dan gambar, tautan dengan pratinjau, kolom, garis penghubung,
-- gambar bebas, tabel, dan komentar.
--
-- DELETE untuk role aplikasi (deploy/sql/10-grants.sql) bertambah tiga: wb_connectors (Tempat sampah lebih dari 30
-- hari), wb_files (berkas yang tidak lagi dipakai kartu mana pun), dan wb_link_previews (cache lebih dari 7 hari),
-- semuanya oleh job maintenance. wb_comments tidak: komentar dihapus lewat deleted_at, dan barisnya ikut hilang
-- lewat ON DELETE CASCADE saat kartunya dihapus permanen.

-- Berkas unggahan Whiteboard dan gambar pratinjau tautan. id sama dengan nama berkas di storage (attachments/),
-- jadi wajib ikut ListAttachmentFileIDs; thumbnail gambar disimpan di sebelahnya sebagai <id>.thumb.jpg.
CREATE TABLE wb_files (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  -- NULL untuk gambar pratinjau tautan, yang dipakai bersama lintas papan.
  board_id      uuid REFERENCES wb_boards (id) ON DELETE SET NULL,
  kind          text NOT NULL CHECK (kind IN ('image', 'file', 'preview')),
  original_name text NOT NULL CHECK (btrim(original_name) <> '' AND char_length(original_name) <= 255),
  mime_type     text NOT NULL,
  size_bytes    bigint NOT NULL CHECK (size_bytes > 0 AND size_bytes <= 52428800),
  checksum      text NOT NULL,
  width         integer,
  height        integer,
  has_thumb     boolean NOT NULL DEFAULT false,
  uploaded_by   bigint REFERENCES users (id),
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX wb_files_board_idx ON wb_files (board_id);

-- Cache pratinjau tautan per URL (sha256 dari URL yang dinormalkan), dipakai ulang 7 hari.
CREATE TABLE wb_link_previews (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  url_hash      text NOT NULL UNIQUE,
  url           text NOT NULL CHECK (char_length(url) <= 2048),
  final_url     text NOT NULL DEFAULT '' CHECK (char_length(final_url) <= 2048),
  title         text NOT NULL DEFAULT '' CHECK (char_length(title) <= 300),
  description   text NOT NULL DEFAULT '' CHECK (char_length(description) <= 500),
  site_name     text NOT NULL DEFAULT '' CHECK (char_length(site_name) <= 120),
  image_file_id uuid REFERENCES wb_files (id) ON DELETE SET NULL,
  status        text NOT NULL CHECK (status IN ('ok', 'blocked', 'failed')),
  message       text NOT NULL DEFAULT '',
  fetched_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX wb_link_previews_fetched_idx ON wb_link_previews (fetched_at);
CREATE INDEX wb_link_previews_image_idx ON wb_link_previews (image_file_id);

ALTER TABLE wb_cards DROP CONSTRAINT wb_cards_kind_check;
ALTER TABLE wb_cards
  ADD CONSTRAINT wb_cards_kind_check
    CHECK (kind IN ('note', 'heading', 'checklist', 'board', 'image', 'file', 'link', 'column', 'drawing', 'table')),
  ADD COLUMN file_id uuid REFERENCES wb_files (id),
  -- Kartu yang ditumpuk di kartu kolom; posisinya mengikuti kolom, x dan y diabaikan selama di dalamnya.
  ADD COLUMN column_id uuid REFERENCES wb_cards (id) ON DELETE SET NULL,
  ADD COLUMN col_order double precision,
  ADD CONSTRAINT wb_cards_file_ref CHECK ((kind IN ('image', 'file')) = (file_id IS NOT NULL)),
  ADD CONSTRAINT wb_cards_column_ref CHECK (column_id IS NULL OR (kind <> 'column' AND column_id <> id AND col_order IS NOT NULL));

CREATE INDEX wb_cards_file_idx ON wb_cards (file_id);
CREATE INDEX wb_cards_column_idx ON wb_cards (column_id, col_order) WHERE column_id IS NOT NULL AND deleted_at IS NULL;

-- Garis dan panah antar dua kartu di papan yang sama (dicek Go).
CREATE TABLE wb_connectors (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  board_id      uuid NOT NULL REFERENCES wb_boards (id) ON DELETE CASCADE,
  -- NULL berarti ujung itu bebas (tidak menempel ke kartu); titiknya ada di geometry.
  from_card_id  uuid REFERENCES wb_cards (id) ON DELETE CASCADE,
  to_card_id    uuid REFERENCES wb_cards (id) ON DELETE CASCADE,
  -- Divalidasi di Go. geometry: port per ujung, titik ujung bebas, gaya jalur (lurus, siku, lengkung), titik belok.
  -- style: jenis ujung awal dan akhir, gaya garis, warna (nama token). Model ini sudah memuat semua yang dipakai
  -- diagram W5, jadi data garis W3 tidak perlu dimigrasi ulang.
  geometry      jsonb NOT NULL DEFAULT '{}' CHECK (octet_length(geometry::text) <= 8000),
  style         jsonb NOT NULL DEFAULT '{}' CHECK (octet_length(style::text) <= 1000),
  -- Label di tengah, dekat ujung awal, dan dekat ujung akhir.
  label         text NOT NULL DEFAULT '' CHECK (char_length(label) <= 200),
  label_start   text NOT NULL DEFAULT '' CHECK (char_length(label_start) <= 200),
  label_end     text NOT NULL DEFAULT '' CHECK (char_length(label_end) <= 200),
  version       integer NOT NULL DEFAULT 1,
  seq           bigint NOT NULL DEFAULT 0,
  created_by    bigint NOT NULL REFERENCES users (id),
  updated_by    bigint NOT NULL REFERENCES users (id),
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  deleted_at    timestamptz,
  deleted_by    bigint REFERENCES users (id),
  deleted_batch uuid,
  CONSTRAINT wb_connectors_two_cards CHECK (from_card_id IS NULL OR to_card_id IS NULL OR from_card_id <> to_card_id),
  CONSTRAINT wb_connectors_trash_complete CHECK (
    (deleted_at IS NULL AND deleted_by IS NULL AND deleted_batch IS NULL)
    OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND deleted_batch IS NOT NULL)
  )
);

CREATE INDEX wb_connectors_board_idx ON wb_connectors (board_id) WHERE deleted_at IS NULL;
CREATE INDEX wb_connectors_board_seq_idx ON wb_connectors (board_id, seq);
CREATE INDEX wb_connectors_from_idx ON wb_connectors (from_card_id);
CREATE INDEX wb_connectors_to_idx ON wb_connectors (to_card_id);
CREATE INDEX wb_connectors_trash_idx ON wb_connectors (deleted_batch, deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TRIGGER wb_connectors_set_updated_at BEFORE UPDATE ON wb_connectors
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Komentar teks polos di kartu. Hapus lewat deleted_at; baris hilang bersama kartunya saat dihapus permanen.
CREATE TABLE wb_comments (
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  card_id    uuid NOT NULL REFERENCES wb_cards (id) ON DELETE CASCADE,
  user_id    bigint NOT NULL REFERENCES users (id),
  body       text NOT NULL CHECK (btrim(body) <> '' AND char_length(body) <= 2000),
  created_at timestamptz NOT NULL DEFAULT now(),
  edited_at  timestamptz,
  deleted_at timestamptz,
  deleted_by bigint REFERENCES users (id)
);

CREATE INDEX wb_comments_card_idx ON wb_comments (card_id, created_at) WHERE deleted_at IS NULL;

-- Pindah kartu ke papan lain tidak bisa disusulkan per kartu di papan asal (kartunya sudah berpindah board_id).
-- Klien yang tersambung ulang dengan last_seq lebih kecil dari ini memuat ulang papan utuh.
ALTER TABLE wb_boards ADD COLUMN resync_seq bigint NOT NULL DEFAULT 0;
