-- Whiteboard tahap W1 (docs/whiteboard.md): semua adalah papan. Satu papan akar bersama ("Whiteboard") untuk semua
-- orang; papan lain selalu berada di dalam papan lain dan tampil sebagai kartu jenis board di kanvas induknya
-- (seperti Milanote). Kartu catatan, judul, checklist, dan papan; riwayat aktivitas per papan; Tempat sampah lewat
-- kolom deleted_*. Berkas, tautan, kolom, penghubung, gambar bebas, tabel, dan komentar menyusul di 000024 (W3).
--
-- Hapus permanen dilakukan job maintenance (role aplikasi) setelah 30 hari, jadi wb_boards dan wb_cards masuk
-- daftar DELETE di deploy/sql/10-grants.sql. wb_activity ikut terhapus lewat ON DELETE CASCADE saat papannya
-- dihapus permanen; aksi FK berjalan dengan hak pemilik tabel, jadi app tidak butuh DELETE di wb_activity.

CREATE TABLE wb_boards (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  -- Papan akar tidak punya induk dan tidak bisa dihapus atau dipindah; papan lain selalu punya induk.
  is_root       boolean NOT NULL DEFAULT false,
  -- CASCADE aman: hapus menandai seluruh subtree dalam satu batch, jadi saat induk dihapus permanen, anaknya sudah
  -- di batch yang sama atau batch yang lebih tua (yang terhapus lebih dulu). Papan hidup tidak pernah berada di
  -- bawah induk yang ada di Tempat sampah.
  parent_id     uuid REFERENCES wb_boards (id) ON DELETE CASCADE,
  title         text NOT NULL CHECK (btrim(title) <> '' AND char_length(title) <= 120),
  is_template   boolean NOT NULL DEFAULT false,
  -- Naik setiap ada perubahan di papan ini; dasar sinkron ulang setelah koneksi putus (W2).
  seq           bigint NOT NULL DEFAULT 0,
  version       integer NOT NULL DEFAULT 1,
  created_by    bigint NOT NULL REFERENCES users (id),
  updated_by    bigint NOT NULL REFERENCES users (id),
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  deleted_at    timestamptz,
  deleted_by    bigint REFERENCES users (id),
  deleted_batch uuid,
  CONSTRAINT wb_boards_root_shape CHECK (
    (is_root AND parent_id IS NULL AND deleted_at IS NULL) OR (NOT is_root AND parent_id IS NOT NULL)
  ),
  CONSTRAINT wb_boards_not_own_parent CHECK (parent_id IS NULL OR parent_id <> id),
  CONSTRAINT wb_boards_trash_complete CHECK (
    (deleted_at IS NULL AND deleted_by IS NULL AND deleted_batch IS NULL)
    OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND deleted_batch IS NOT NULL)
  )
);

CREATE UNIQUE INDEX wb_boards_one_root ON wb_boards ((true)) WHERE is_root;
CREATE INDEX wb_boards_parent_idx ON wb_boards (parent_id) WHERE deleted_at IS NULL;
CREATE INDEX wb_boards_trash_idx ON wb_boards (deleted_batch, deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX wb_boards_title_trgm_idx ON wb_boards USING gin (title gin_trgm_ops) WHERE deleted_at IS NULL;

CREATE TRIGGER wb_boards_set_updated_at BEFORE UPDATE ON wb_boards
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Papan akar dibuat di sini bila sudah ada akun (produksi dan dev). Database kosong (test) membuatnya lewat
-- Service.Root saat pertama dibuka, karena created_by wajib menunjuk akun.
INSERT INTO wb_boards (is_root, title, created_by, updated_by)
SELECT true, 'Whiteboard', u.id, u.id FROM users u ORDER BY u.id LIMIT 1;

CREATE TABLE wb_cards (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  board_id        uuid NOT NULL REFERENCES wb_boards (id) ON DELETE CASCADE,
  kind            text NOT NULL CHECK (kind IN ('note', 'heading', 'checklist', 'board')),
  -- Kartu papan menunjuk papan anaknya; satu papan anak punya tepat satu kartu, di kanvas induknya.
  ref_board_id    uuid UNIQUE REFERENCES wb_boards (id) ON DELETE CASCADE,
  -- Koordinat dunia pada zoom 100%. BETWEEN juga menolak NaN dan tak hingga.
  x               double precision NOT NULL CHECK (x BETWEEN -1000000 AND 1000000),
  y               double precision NOT NULL CHECK (y BETWEEN -1000000 AND 1000000),
  w               double precision NOT NULL CHECK (w BETWEEN 40 AND 4000),
  h               double precision NOT NULL CHECK (h BETWEEN 40 AND 4000),
  -- Urutan tumpuk pecahan: bawa ke depan cukup mengubah satu baris.
  z               double precision NOT NULL DEFAULT 0 CHECK (z BETWEEN -1000000000 AND 1000000000),
  color           text CHECK (color IN ('abu', 'hijau', 'kuning', 'biru', 'krem', 'merah')),
  -- Bentuknya per jenis dan divalidasi di Go (internal/whiteboard/content.go); batas di sini hanya pengaman.
  content         jsonb NOT NULL CHECK (octet_length(content::text) <= 70000),
  -- Teks polos dari content untuk pencarian, dibuat Go dari isi yang sudah lolos validasi.
  plain           text NOT NULL DEFAULT '',
  version         integer NOT NULL DEFAULT 1,
  -- Naik hanya saat isi berubah; patch isi dengan versi basi ditolak 409.
  content_version integer NOT NULL DEFAULT 1,
  seq             bigint NOT NULL DEFAULT 0,
  created_by      bigint NOT NULL REFERENCES users (id),
  updated_by      bigint NOT NULL REFERENCES users (id),
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now(),
  deleted_at      timestamptz,
  deleted_by      bigint REFERENCES users (id),
  deleted_batch   uuid,
  CONSTRAINT wb_cards_board_ref CHECK ((kind = 'board') = (ref_board_id IS NOT NULL)),
  CONSTRAINT wb_cards_trash_complete CHECK (
    (deleted_at IS NULL AND deleted_by IS NULL AND deleted_batch IS NULL)
    OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL AND deleted_batch IS NOT NULL)
  )
);

CREATE INDEX wb_cards_board_idx ON wb_cards (board_id) WHERE deleted_at IS NULL;
CREATE INDEX wb_cards_board_seq_idx ON wb_cards (board_id, seq);
CREATE INDEX wb_cards_trash_idx ON wb_cards (deleted_batch, deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX wb_cards_plain_trgm_idx ON wb_cards USING gin (plain gin_trgm_ops) WHERE deleted_at IS NULL;

CREATE TRIGGER wb_cards_set_updated_at BEFORE UPDATE ON wb_cards
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Riwayat per papan. Seret beruntun dan edit isi beruntun dari orang yang sama digabung ke satu baris selama 5
-- menit, jadi updated_at dipakai untuk jendela penggabungan.
CREATE TABLE wb_activity (
  id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  board_id   uuid NOT NULL REFERENCES wb_boards (id) ON DELETE CASCADE,
  user_id    bigint NOT NULL REFERENCES users (id),
  action     text NOT NULL,
  entity_id  uuid,
  summary    jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX wb_activity_board_idx ON wb_activity (board_id, created_at DESC);
