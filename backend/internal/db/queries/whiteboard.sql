-- Whiteboard (docs/whiteboard.md). "Hidup" berarti deleted_at IS NULL; baris di Tempat sampah tidak pernah
-- dikembalikan query baca biasa. Semua adalah papan: papan akar tanpa induk, papan lain tampil sebagai kartu jenis
-- board di kanvas induknya.

-- name: GetWbRoot :one
SELECT id FROM wb_boards WHERE is_root;

-- name: InsertWbRoot :one
-- Hanya untuk database tanpa papan akar (test). ON CONFLICT menangani dua permintaan pertama yang bersamaan.
INSERT INTO wb_boards (is_root, title, created_by, updated_by)
VALUES (true, 'Whiteboard', @actor_id, @actor_id)
ON CONFLICT ((true)) WHERE is_root DO NOTHING
RETURNING id;

-- name: GetWbBoard :one
SELECT id, is_root, parent_id, title, seq, version, updated_at
FROM wb_boards WHERE id = @id AND deleted_at IS NULL;

-- name: GetWbBoardForUpdate :one
SELECT id, is_root, parent_id, title, seq, version
FROM wb_boards WHERE id = @id AND deleted_at IS NULL
FOR UPDATE;

-- name: ListWbAncestors :many
-- Jalur dari papan akar sampai papan ini (papan ini ikut), untuk breadcrumb dan batas kedalaman.
WITH RECURSIVE up AS (
  SELECT b.id, b.title, b.parent_id, 0 AS depth FROM wb_boards b WHERE b.id = @id
  UNION ALL
  SELECT p.id, p.title, p.parent_id, up.depth + 1
  FROM wb_boards p JOIN up ON p.id = up.parent_id
  WHERE up.depth < 64
)
SELECT id, title FROM up ORDER BY depth DESC;

-- name: InsertWbBoard :one
INSERT INTO wb_boards (parent_id, title, created_by, updated_by)
VALUES (@parent_id, @title, @actor_id, @actor_id)
RETURNING id;

-- name: RenameWbBoard :one
UPDATE wb_boards
SET title = @title, version = version + 1, updated_by = @actor_id
WHERE id = @id AND deleted_at IS NULL AND version = @version
RETURNING id, title, version;

-- name: TrashWbSubtree :many
-- Papan dan seluruh turunannya yang masih hidup masuk satu batch. Turunan yang sudah di Tempat sampah dari aksi
-- lain tetap memakai batch lamanya. Papan akar tidak pernah ikut (dijaga CHECK dan pemanggil).
WITH RECURSIVE sub AS (
  SELECT b.id FROM wb_boards b WHERE b.id = ANY(@ids::uuid[]) AND b.deleted_at IS NULL AND NOT b.is_root
  UNION ALL
  SELECT c.id FROM wb_boards c JOIN sub ON c.parent_id = sub.id WHERE c.deleted_at IS NULL
)
UPDATE wb_boards
SET deleted_at = now(), deleted_by = @actor_id::bigint, deleted_batch = @batch, updated_by = @actor_id
WHERE id IN (SELECT id FROM sub)
RETURNING id, title;

-- name: SearchWbBoards :many
SELECT b.id, b.title, b.parent_id
FROM wb_boards b
WHERE b.deleted_at IS NULL AND NOT b.is_root AND b.title ILIKE '%' || @q::text || '%'
ORDER BY lower(b.title), b.id
LIMIT 50;

-- name: BumpWbBoardSeq :one
-- Setiap perubahan isi papan lewat sini dulu: memastikan papan hidup, mengunci barisnya sampai transaksi selesai
-- (perubahan di satu papan berurutan), dan memberi seq baru untuk baris yang berubah.
UPDATE wb_boards SET seq = seq + 1
WHERE id = @id AND deleted_at IS NULL
RETURNING seq;

-- name: ListWbCards :many
-- Kartu papan membawa judul papan anak dan ringkasan isinya ("3 papan, 12 kartu").
SELECT c.id, c.board_id, c.kind, c.ref_board_id, c.x, c.y, c.w, c.h, c.z, c.color, c.content, c.version,
       c.content_version, c.seq, c.updated_by, c.updated_at,
       coalesce(rb.title, '')::text AS ref_title, coalesce(rb.version, 0)::int AS ref_version,
       (SELECT count(*) FROM wb_cards k WHERE k.board_id = c.ref_board_id AND k.deleted_at IS NULL AND k.kind = 'board')::bigint AS ref_boards,
       (SELECT count(*) FROM wb_cards k WHERE k.board_id = c.ref_board_id AND k.deleted_at IS NULL AND k.kind <> 'board')::bigint AS ref_cards
FROM wb_cards c
LEFT JOIN wb_boards rb ON rb.id = c.ref_board_id
WHERE c.board_id = @board_id AND c.deleted_at IS NULL
ORDER BY c.z, c.created_at, c.id;

-- name: GetWbCardForUpdate :one
SELECT id, board_id, kind, ref_board_id, content, content_version
FROM wb_cards
WHERE id = @id AND board_id = @board_id AND deleted_at IS NULL
FOR UPDATE;

-- name: WbCardExists :one
SELECT EXISTS (SELECT 1 FROM wb_cards WHERE id = @id)::bool;

-- name: MaxWbCardZ :one
SELECT coalesce(max(z), 0)::double precision FROM wb_cards WHERE board_id = @board_id AND deleted_at IS NULL;

-- name: InsertWbCard :exec
INSERT INTO wb_cards (id, board_id, kind, ref_board_id, x, y, w, h, z, color, content, plain, seq, created_by, updated_by)
VALUES (@id, @board_id, @kind, sqlc.narg(ref_board_id), @x, @y, @w, @h, @z, sqlc.narg(color), @content, @plain, @seq,
        @actor_id, @actor_id);

-- name: PatchWbCard :exec
-- Hanya field yang dikirim yang berubah (last-write-wins per field). Warna dikosongkan lewat clear_color, karena
-- NULL pada sqlc.narg berarti "tidak diubah".
UPDATE wb_cards
SET x = coalesce(sqlc.narg(x), x),
    y = coalesce(sqlc.narg(y), y),
    w = coalesce(sqlc.narg(w), w),
    h = coalesce(sqlc.narg(h), h),
    z = coalesce(sqlc.narg(z), z),
    color = CASE WHEN @clear_color::bool THEN NULL ELSE coalesce(sqlc.narg(color), color) END,
    content = coalesce(sqlc.narg(content), content),
    plain = coalesce(sqlc.narg(plain), plain),
    content_version = content_version + CASE WHEN sqlc.narg(content)::jsonb IS NULL THEN 0 ELSE 1 END,
    version = version + 1,
    seq = @seq,
    updated_by = @actor_id
WHERE id = @id AND board_id = @board_id AND deleted_at IS NULL;

-- name: TrashWbCards :many
UPDATE wb_cards
SET deleted_at = now(), deleted_by = @actor_id::bigint, deleted_batch = @batch, seq = @seq, updated_by = @actor_id
WHERE board_id = @board_id AND deleted_at IS NULL AND id = ANY(@ids::uuid[])
RETURNING id, kind, plain, ref_board_id;

-- name: ListWbTrashBoards :many
SELECT b.id, b.title, b.parent_id, b.deleted_at, b.deleted_batch, u.name AS deleted_by_name,
       p.title AS parent_title, (p.deleted_at IS NOT NULL)::bool AS parent_in_trash,
       (SELECT count(*) FROM wb_cards c WHERE c.board_id = b.id AND c.deleted_at IS NULL AND c.kind <> 'board')::bigint AS card_count
FROM wb_boards b
JOIN users u ON u.id = b.deleted_by
LEFT JOIN wb_boards p ON p.id = b.parent_id
WHERE b.deleted_at IS NOT NULL
ORDER BY b.deleted_at DESC, b.id;

-- name: ListWbTrashCards :many
SELECT c.id, c.kind, c.plain, c.board_id, c.deleted_at, c.deleted_batch, u.name AS deleted_by_name,
       b.title AS board_title, (b.deleted_at IS NOT NULL)::bool AS board_in_trash
FROM wb_cards c
JOIN users u ON u.id = c.deleted_by
JOIN wb_boards b ON b.id = c.board_id
WHERE c.deleted_at IS NOT NULL
ORDER BY c.deleted_at DESC, c.id;

-- name: RestoreWbBoards :many
UPDATE wb_boards SET deleted_at = NULL, deleted_by = NULL, deleted_batch = NULL, updated_by = @actor_id
WHERE deleted_batch = @batch
RETURNING id, title;

-- name: RestoreWbCards :many
UPDATE wb_cards SET deleted_at = NULL, deleted_by = NULL, deleted_batch = NULL, updated_by = @actor_id,
                    seq = (SELECT b.seq FROM wb_boards b WHERE b.id = wb_cards.board_id)
WHERE wb_cards.deleted_batch = @batch
RETURNING id, board_id;

-- name: ListWbExpiredBatches :many
-- Batch Tempat sampah yang lebih tua dari batas, dengan isinya untuk audit hapus permanen.
SELECT t.batch::uuid AS batch, sum(t.nb)::bigint AS boards, sum(t.nc)::bigint AS cards
FROM (
  SELECT wb.deleted_batch AS batch, 1 AS nb, 0 AS nc FROM wb_boards wb WHERE wb.deleted_at < @before
  UNION ALL
  SELECT wc.deleted_batch, 0, 1 FROM wb_cards wc WHERE wc.deleted_at < @before
) t
GROUP BY t.batch
ORDER BY t.batch;

-- name: DeleteWbExpiredCards :execrows
DELETE FROM wb_cards WHERE deleted_at < @before;

-- name: DeleteWbExpiredBoards :execrows
-- Anak dalam batch yang sama terhapus di pernyataan yang sama. Kartu hidup di papan yang dihapus ikut hilang lewat
-- ON DELETE CASCADE.
DELETE FROM wb_boards WHERE deleted_at < @before;

-- name: GetWbActivityToMerge :one
-- Baris riwayat terakhir orang yang sama untuk aksi yang sama dalam jendela penggabungan.
SELECT id, summary FROM wb_activity
WHERE board_id = @board_id AND user_id = @user_id AND action = @action
  AND entity_id IS NOT DISTINCT FROM sqlc.narg(entity_id)::uuid AND updated_at >= @since
ORDER BY updated_at DESC
LIMIT 1
FOR UPDATE;

-- name: InsertWbActivity :exec
INSERT INTO wb_activity (board_id, user_id, action, entity_id, summary)
VALUES (@board_id, @user_id, @action, sqlc.narg(entity_id), @summary);

-- name: UpdateWbActivitySummary :exec
UPDATE wb_activity SET summary = @summary, updated_at = now() WHERE id = @id;

-- name: ListWbActivity :many
SELECT a.id, a.action, a.entity_id, a.summary, a.created_at, a.updated_at, a.user_id, u.name AS user_name
FROM wb_activity a
JOIN users u ON u.id = a.user_id
WHERE a.board_id = @board_id
ORDER BY a.updated_at DESC, a.id DESC
LIMIT 200;

-- name: GetWbCard :one
SELECT c.id, c.board_id, c.kind, c.ref_board_id, c.x, c.y, c.w, c.h, c.z, c.color, c.content, c.version,
       c.content_version, c.seq, c.updated_by, c.updated_at,
       coalesce(rb.title, '')::text AS ref_title, coalesce(rb.version, 0)::int AS ref_version,
       (SELECT count(*) FROM wb_cards k WHERE k.board_id = c.ref_board_id AND k.deleted_at IS NULL AND k.kind = 'board')::bigint AS ref_boards,
       (SELECT count(*) FROM wb_cards k WHERE k.board_id = c.ref_board_id AND k.deleted_at IS NULL AND k.kind <> 'board')::bigint AS ref_cards
FROM wb_cards c
LEFT JOIN wb_boards rb ON rb.id = c.ref_board_id
WHERE c.id = @id AND c.deleted_at IS NULL;

-- name: GetWbCardKind :one
SELECT kind FROM wb_cards WHERE id = @id AND board_id = @board_id AND deleted_at IS NULL;

-- name: CountWbCardsSince :one
SELECT count(*)::bigint FROM wb_cards WHERE board_id = @board_id AND seq > @seq;

-- name: ListWbCardsSince :many
-- Sinkron ulang setelah koneksi putus: semua kartu yang berubah sesudah seq itu, termasuk yang masuk Tempat sampah
-- (deleted true) supaya klien bisa membuangnya.
SELECT c.id, c.board_id, c.kind, c.ref_board_id, c.x, c.y, c.w, c.h, c.z, c.color, c.content, c.version,
       c.content_version, c.seq, c.updated_by, c.updated_at,
       coalesce(rb.title, '')::text AS ref_title, coalesce(rb.version, 0)::int AS ref_version,
       (SELECT count(*) FROM wb_cards k WHERE k.board_id = c.ref_board_id AND k.deleted_at IS NULL AND k.kind = 'board')::bigint AS ref_boards,
       (SELECT count(*) FROM wb_cards k WHERE k.board_id = c.ref_board_id AND k.deleted_at IS NULL AND k.kind <> 'board')::bigint AS ref_cards,
       (c.deleted_at IS NOT NULL)::bool AS deleted
FROM wb_cards c
LEFT JOIN wb_boards rb ON rb.id = c.ref_board_id
WHERE c.board_id = @board_id AND c.seq > @seq
ORDER BY c.seq, c.id;
