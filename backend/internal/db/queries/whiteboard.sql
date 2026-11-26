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
SELECT id, is_root, parent_id, title, seq, version, updated_at, is_template
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
       c.content_version, c.seq, c.updated_by, c.updated_at, c.file_id, c.column_id, c.col_order, c.style, c.locked, c.group_ids,
       coalesce(rb.title, '')::text AS ref_title, coalesce(rb.version, 0)::int AS ref_version,
       (SELECT count(*) FROM wb_cards k WHERE k.board_id = c.ref_board_id AND k.deleted_at IS NULL AND k.kind = 'board')::bigint AS ref_boards,
       (SELECT count(*) FROM wb_cards k WHERE k.board_id = c.ref_board_id AND k.deleted_at IS NULL AND k.kind <> 'board')::bigint AS ref_cards,
       coalesce(f.original_name, '')::text AS file_name, coalesce(f.mime_type, '')::text AS file_mime,
       coalesce(f.size_bytes, 0)::bigint AS file_size, coalesce(f.has_thumb, false)::bool AS file_thumb,
       coalesce(f.width, 0)::int AS file_width, coalesce(f.height, 0)::int AS file_height,
       lp.id AS preview_id, coalesce(lp.title, '')::text AS preview_title,
       coalesce(lp.description, '')::text AS preview_description, coalesce(lp.site_name, '')::text AS preview_site,
       coalesce(lp.final_url, '')::text AS preview_final_url, coalesce(lp.status, '')::text AS preview_status,
       lp.image_file_id AS preview_image,
       (SELECT count(*) FROM wb_comments m WHERE m.card_id = c.id AND m.deleted_at IS NULL)::bigint AS comment_count,
       coalesce(cu.name, '')::text AS created_by_name,
       coalesce(pu.id, 0)::bigint AS person_id, coalesce(pu.name, '')::text AS person_name, pu.job_title AS person_title,
       coalesce(pd.name, '')::text AS person_division, coalesce(pu.is_active, false)::bool AS person_active,
       pu.avatar_id AS person_avatar
FROM wb_cards c
LEFT JOIN wb_boards rb ON rb.id = c.ref_board_id
LEFT JOIN wb_files f ON f.id = c.file_id
LEFT JOIN wb_link_previews lp ON c.kind = 'link' AND lp.id = nullif(c.content->>'preview_id', '')::uuid
LEFT JOIN users cu ON cu.id = c.created_by
-- Kartu orang (revisi 51): data profil dibaca saat papan dimuat, tidak disalin ke isi kartu.
LEFT JOIN users pu ON c.kind = 'person' AND pu.id = (c.content->>'user_id')::bigint
LEFT JOIN divisions pd ON pd.id = pu.division_id
WHERE c.board_id = @board_id AND c.deleted_at IS NULL
ORDER BY c.z, c.created_at, c.id;

-- name: GetWbCardForUpdate :one
SELECT id, board_id, kind, ref_board_id, content, content_version, column_id, locked, file_id
FROM wb_cards
WHERE id = @id AND board_id = @board_id AND deleted_at IS NULL
FOR UPDATE;

-- name: ListWbLockedCards :many
-- Kartu terkunci di antara id ini (revisi 50): ditolak saat dihapus.
SELECT id FROM wb_cards
WHERE board_id = @board_id AND deleted_at IS NULL AND locked AND id = ANY(@ids::uuid[]);

-- name: WbCardExists :one
SELECT EXISTS (SELECT 1 FROM wb_cards WHERE id = @id)::bool;

-- name: MaxWbCardZ :one
SELECT coalesce(max(z), 0)::double precision FROM wb_cards WHERE board_id = @board_id AND deleted_at IS NULL;

-- name: InsertWbCard :exec
INSERT INTO wb_cards (id, board_id, kind, ref_board_id, file_id, x, y, w, h, z, color, content, plain, seq, created_by, updated_by,
                      style, locked, group_ids)
VALUES (@id, @board_id, @kind, sqlc.narg(ref_board_id), sqlc.narg(file_id), @x, @y, @w, @h, @z, sqlc.narg(color), @content,
        @plain, @seq, @actor_id, @actor_id, @style, @locked, @group_ids::uuid[]);

-- name: PatchWbCard :exec
-- Hanya field yang dikirim yang berubah (last-write-wins per field). Warna dikosongkan lewat clear_color, karena
-- NULL pada sqlc.narg berarti "tidak diubah".
UPDATE wb_cards
SET x = coalesce(sqlc.narg(x), x),
    y = coalesce(sqlc.narg(y), y),
    w = coalesce(sqlc.narg(w), w),
    h = coalesce(sqlc.narg(h), h),
    z = coalesce(sqlc.narg(z), z),
    column_id = CASE WHEN @clear_column::bool THEN NULL ELSE coalesce(sqlc.narg(column_id), column_id) END,
    col_order = CASE WHEN @clear_column::bool THEN NULL ELSE coalesce(sqlc.narg(col_order), col_order) END,
    color = CASE WHEN @clear_color::bool THEN NULL ELSE coalesce(sqlc.narg(color), color) END,
    content = coalesce(sqlc.narg(content), content),
    style = coalesce(sqlc.narg(style), style),
    locked = coalesce(sqlc.narg(locked), locked),
    group_ids = CASE WHEN @set_groups::bool THEN @group_ids::uuid[] ELSE group_ids END,
    file_id = coalesce(sqlc.narg(file_id), file_id),
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
  UNION ALL
  SELECT wn.deleted_batch, 0, 0 FROM wb_connectors wn WHERE wn.deleted_at < @before
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
       c.content_version, c.seq, c.updated_by, c.updated_at, c.file_id, c.column_id, c.col_order, c.style, c.locked, c.group_ids,
       coalesce(rb.title, '')::text AS ref_title, coalesce(rb.version, 0)::int AS ref_version,
       (SELECT count(*) FROM wb_cards k WHERE k.board_id = c.ref_board_id AND k.deleted_at IS NULL AND k.kind = 'board')::bigint AS ref_boards,
       (SELECT count(*) FROM wb_cards k WHERE k.board_id = c.ref_board_id AND k.deleted_at IS NULL AND k.kind <> 'board')::bigint AS ref_cards,
       coalesce(f.original_name, '')::text AS file_name, coalesce(f.mime_type, '')::text AS file_mime,
       coalesce(f.size_bytes, 0)::bigint AS file_size, coalesce(f.has_thumb, false)::bool AS file_thumb,
       coalesce(f.width, 0)::int AS file_width, coalesce(f.height, 0)::int AS file_height,
       lp.id AS preview_id, coalesce(lp.title, '')::text AS preview_title,
       coalesce(lp.description, '')::text AS preview_description, coalesce(lp.site_name, '')::text AS preview_site,
       coalesce(lp.final_url, '')::text AS preview_final_url, coalesce(lp.status, '')::text AS preview_status,
       lp.image_file_id AS preview_image,
       (SELECT count(*) FROM wb_comments m WHERE m.card_id = c.id AND m.deleted_at IS NULL)::bigint AS comment_count,
       coalesce(cu.name, '')::text AS created_by_name,
       coalesce(pu.id, 0)::bigint AS person_id, coalesce(pu.name, '')::text AS person_name, pu.job_title AS person_title,
       coalesce(pd.name, '')::text AS person_division, coalesce(pu.is_active, false)::bool AS person_active,
       pu.avatar_id AS person_avatar
FROM wb_cards c
LEFT JOIN wb_boards rb ON rb.id = c.ref_board_id
LEFT JOIN wb_files f ON f.id = c.file_id
LEFT JOIN wb_link_previews lp ON c.kind = 'link' AND lp.id = nullif(c.content->>'preview_id', '')::uuid
LEFT JOIN users cu ON cu.id = c.created_by
-- Kartu orang (revisi 51): data profil dibaca saat papan dimuat, tidak disalin ke isi kartu.
LEFT JOIN users pu ON c.kind = 'person' AND pu.id = (c.content->>'user_id')::bigint
LEFT JOIN divisions pd ON pd.id = pu.division_id
WHERE c.id = @id AND c.deleted_at IS NULL;

-- name: GetWbCardKind :one
SELECT kind FROM wb_cards WHERE id = @id AND board_id = @board_id AND deleted_at IS NULL;

-- name: CountWbCardsSince :one
SELECT count(*)::bigint FROM wb_cards WHERE board_id = @board_id AND seq > @seq;

-- name: ListWbCardsSince :many
-- Sinkron ulang setelah koneksi putus: semua kartu yang berubah sesudah seq itu, termasuk yang masuk Tempat sampah
-- (deleted true) supaya klien bisa membuangnya.
SELECT c.id, c.board_id, c.kind, c.ref_board_id, c.x, c.y, c.w, c.h, c.z, c.color, c.content, c.version,
       c.content_version, c.seq, c.updated_by, c.updated_at, c.file_id, c.column_id, c.col_order, c.style, c.locked, c.group_ids,
       coalesce(rb.title, '')::text AS ref_title, coalesce(rb.version, 0)::int AS ref_version,
       (SELECT count(*) FROM wb_cards k WHERE k.board_id = c.ref_board_id AND k.deleted_at IS NULL AND k.kind = 'board')::bigint AS ref_boards,
       (SELECT count(*) FROM wb_cards k WHERE k.board_id = c.ref_board_id AND k.deleted_at IS NULL AND k.kind <> 'board')::bigint AS ref_cards,
       coalesce(f.original_name, '')::text AS file_name, coalesce(f.mime_type, '')::text AS file_mime,
       coalesce(f.size_bytes, 0)::bigint AS file_size, coalesce(f.has_thumb, false)::bool AS file_thumb,
       coalesce(f.width, 0)::int AS file_width, coalesce(f.height, 0)::int AS file_height,
       lp.id AS preview_id, coalesce(lp.title, '')::text AS preview_title,
       coalesce(lp.description, '')::text AS preview_description, coalesce(lp.site_name, '')::text AS preview_site,
       coalesce(lp.final_url, '')::text AS preview_final_url, coalesce(lp.status, '')::text AS preview_status,
       lp.image_file_id AS preview_image,
       (SELECT count(*) FROM wb_comments m WHERE m.card_id = c.id AND m.deleted_at IS NULL)::bigint AS comment_count,
       coalesce(cu.name, '')::text AS created_by_name,
       coalesce(pu.id, 0)::bigint AS person_id, coalesce(pu.name, '')::text AS person_name, pu.job_title AS person_title,
       coalesce(pd.name, '')::text AS person_division, coalesce(pu.is_active, false)::bool AS person_active,
       pu.avatar_id AS person_avatar,
       (c.deleted_at IS NOT NULL)::bool AS deleted
FROM wb_cards c
LEFT JOIN wb_boards rb ON rb.id = c.ref_board_id
LEFT JOIN wb_files f ON f.id = c.file_id
LEFT JOIN wb_link_previews lp ON c.kind = 'link' AND lp.id = nullif(c.content->>'preview_id', '')::uuid
LEFT JOIN users cu ON cu.id = c.created_by
-- Kartu orang (revisi 51): data profil dibaca saat papan dimuat, tidak disalin ke isi kartu.
LEFT JOIN users pu ON c.kind = 'person' AND pu.id = (c.content->>'user_id')::bigint
LEFT JOIN divisions pd ON pd.id = pu.division_id
WHERE c.board_id = @board_id AND c.seq > @seq
ORDER BY c.seq, c.id;

-- ---------------------------------------------------------------------------------------------------------------
-- W3: berkas, pratinjau tautan, kolom, penghubung, komentar, pindah antar papan

-- name: InsertWbFile :exec
INSERT INTO wb_files (id, board_id, kind, original_name, mime_type, size_bytes, checksum, width, height, has_thumb, uploaded_by)
VALUES (@id, sqlc.narg(board_id), @kind, @original_name, @mime_type, @size_bytes, @checksum, sqlc.narg(width),
        sqlc.narg(height), @has_thumb, sqlc.narg(uploaded_by));

-- name: GetWbFile :one
SELECT id, board_id, kind, original_name, mime_type, size_bytes, has_thumb, created_at FROM wb_files WHERE id = @id;

-- name: GetWbPreviewByHash :one
SELECT id, url, final_url, title, description, site_name, image_file_id, status, message, fetched_at
FROM wb_link_previews WHERE url_hash = @url_hash;

-- name: GetWbPreview :one
SELECT id, url, final_url, title, description, site_name, image_file_id, status, message, fetched_at
FROM wb_link_previews WHERE id = @id;

-- name: UpsertWbPreview :one
INSERT INTO wb_link_previews (url_hash, url, final_url, title, description, site_name, image_file_id, status, message, fetched_at)
VALUES (@url_hash, @url, @final_url, @title, @description, @site_name, sqlc.narg(image_file_id), @status, @message, now())
ON CONFLICT (url_hash) DO UPDATE
SET url = EXCLUDED.url, final_url = EXCLUDED.final_url, title = EXCLUDED.title, description = EXCLUDED.description,
    site_name = EXCLUDED.site_name, image_file_id = EXCLUDED.image_file_id, status = EXCLUDED.status,
    message = EXCLUDED.message, fetched_at = now()
RETURNING id, url, final_url, title, description, site_name, image_file_id, status, message, fetched_at;

-- name: ListWbColumnChildren :many
SELECT id FROM wb_cards WHERE column_id = ANY(@ids::uuid[]) AND deleted_at IS NULL;

-- name: MaxWbColOrder :one
SELECT coalesce(max(col_order), 0)::double precision FROM wb_cards WHERE column_id = @column_id AND deleted_at IS NULL;

-- name: ListWbConnectors :many
-- Penghubung yang ujung menempelnya hidup (ujung bebas tidak punya kartu); ujung di Tempat sampah menyembunyikannya.
SELECT n.id, n.board_id, n.from_card_id, n.to_card_id, n.geometry, n.style, n.label, n.label_start, n.label_end,
       n.version, n.seq
FROM wb_connectors n
LEFT JOIN wb_cards a ON a.id = n.from_card_id
LEFT JOIN wb_cards b ON b.id = n.to_card_id
WHERE n.board_id = @board_id AND n.deleted_at IS NULL
  AND (n.from_card_id IS NULL OR a.deleted_at IS NULL) AND (n.to_card_id IS NULL OR b.deleted_at IS NULL)
ORDER BY n.created_at, n.id;

-- name: ListWbConnectorsSince :many
SELECT n.id, n.board_id, n.from_card_id, n.to_card_id, n.geometry, n.style, n.label, n.label_start, n.label_end,
       n.version, n.seq,
       (n.deleted_at IS NOT NULL OR a.deleted_at IS NOT NULL OR b.deleted_at IS NOT NULL)::bool AS hidden
FROM wb_connectors n
LEFT JOIN wb_cards a ON a.id = n.from_card_id
LEFT JOIN wb_cards b ON b.id = n.to_card_id
WHERE n.board_id = @board_id AND n.seq > @seq;

-- name: InsertWbConnector :one
INSERT INTO wb_connectors (board_id, from_card_id, to_card_id, geometry, style, label, label_start, label_end, seq,
                           created_by, updated_by)
VALUES (@board_id, sqlc.narg(from_card_id), sqlc.narg(to_card_id), @geometry, @style, @label, @label_start, @label_end,
        @seq, @actor_id, @actor_id)
RETURNING id, board_id, from_card_id, to_card_id, geometry, style, label, label_start, label_end, version, seq;

-- name: UpdateWbConnector :one
UPDATE wb_connectors
SET style = coalesce(sqlc.narg(style), style), geometry = coalesce(sqlc.narg(geometry), geometry),
    label = coalesce(sqlc.narg(label), label), label_start = coalesce(sqlc.narg(label_start), label_start),
    label_end = coalesce(sqlc.narg(label_end), label_end), version = version + 1,
    seq = @seq, updated_by = @actor_id
WHERE id = @id AND board_id = @board_id AND deleted_at IS NULL
RETURNING id, board_id, from_card_id, to_card_id, geometry, style, label, label_start, label_end, version, seq;

-- name: TrashWbConnectors :many
-- Penghubung yang dipilih atau yang menempel di kartu yang dihapus atau dipindah, ke batch yang sama.
UPDATE wb_connectors
SET deleted_at = now(), deleted_by = @actor_id::bigint, deleted_batch = @batch, seq = @seq, updated_by = @actor_id
WHERE board_id = @board_id AND deleted_at IS NULL
  AND (id = ANY(@ids::uuid[]) OR from_card_id = ANY(@cards::uuid[]) OR to_card_id = ANY(@cards::uuid[]))
RETURNING id;

-- name: RestoreWbConnectors :many
UPDATE wb_connectors SET deleted_at = NULL, deleted_by = NULL, deleted_batch = NULL, updated_by = @actor_id,
                         seq = (SELECT b.seq FROM wb_boards b WHERE b.id = wb_connectors.board_id)
WHERE wb_connectors.deleted_batch = @batch
RETURNING id, board_id;

-- name: ListWbTrashConnectors :many
SELECT n.deleted_batch, n.deleted_at, n.board_id, u.name AS deleted_by_name, b.title AS board_title,
       (b.deleted_at IS NOT NULL)::bool AS board_in_trash
FROM wb_connectors n
JOIN users u ON u.id = n.deleted_by
JOIN wb_boards b ON b.id = n.board_id
WHERE n.deleted_at IS NOT NULL;

-- name: DeleteWbExpiredConnectors :execrows
DELETE FROM wb_connectors WHERE deleted_at < @before;

-- name: DeleteWbUnusedFiles :execrows
-- Berkas yang tidak lagi dipakai kartu mana pun (termasuk kartu di Tempat sampah) atau pratinjau, dan sudah berumur
-- lebih dari satu hari (unggahan yang sedang dipakai kartu baru tidak tersentuh). Berkas di disk dibersihkan pembersih
-- berkas yatim pada run berikutnya.
DELETE FROM wb_files f
WHERE f.created_at < @before
  AND NOT EXISTS (SELECT 1 FROM wb_cards c WHERE c.file_id = f.id)
  AND NOT EXISTS (SELECT 1 FROM wb_link_previews p WHERE p.image_file_id = f.id);

-- name: DeleteWbOldPreviews :execrows
-- Cache pratinjau lebih dari 7 hari yang tidak dipakai kartu tautan mana pun (termasuk di Tempat sampah).
DELETE FROM wb_link_previews p
WHERE p.fetched_at < @before
  AND NOT EXISTS (SELECT 1 FROM wb_cards c WHERE c.kind = 'link' AND c.content->>'preview_id' = p.id::text);

-- name: ListWbComments :many
SELECT m.id, m.card_id, m.user_id, u.name AS user_name, u.avatar_id, m.body, m.created_at, m.edited_at
FROM wb_comments m
JOIN users u ON u.id = m.user_id
WHERE m.card_id = @card_id AND m.deleted_at IS NULL
ORDER BY m.created_at, m.id;

-- name: InsertWbComment :one
INSERT INTO wb_comments (card_id, user_id, body) VALUES (@card_id, @user_id, @body)
RETURNING id, created_at;

-- name: GetWbCommentForUpdate :one
SELECT m.id, m.card_id, m.user_id, m.body, c.board_id
FROM wb_comments m JOIN wb_cards c ON c.id = m.card_id
WHERE m.id = @id AND m.deleted_at IS NULL AND c.deleted_at IS NULL
FOR UPDATE OF m;

-- name: UpdateWbComment :exec
UPDATE wb_comments SET body = @body, edited_at = now() WHERE id = @id;

-- name: SoftDeleteWbComment :exec
UPDATE wb_comments SET deleted_at = now(), deleted_by = @actor_id::bigint WHERE id = @id;

-- name: ListWbBoards :many
-- Semua papan hidup untuk pilihan "Pindahkan ke papan"; jumlah papan tim kecil, jadi pohonnya disusun di klien.
SELECT id, title, parent_id, is_root FROM wb_boards WHERE deleted_at IS NULL ORDER BY lower(title), id;

-- name: MoveWbCards :many
-- Pindah kartu ke papan lain. Kartu yang tetap berada di kolomnya (kolomnya ikut pindah) mempertahankan column_id;
-- yang lain keluar dari kolom.
UPDATE wb_cards
SET board_id = @to_board, seq = @seq, updated_by = @actor_id, version = version + 1,
    x = x + @dx::double precision, y = y + @dy::double precision,
    column_id = CASE WHEN column_id = ANY(@ids::uuid[]) THEN column_id ELSE NULL END,
    col_order = CASE WHEN column_id = ANY(@ids::uuid[]) THEN col_order ELSE NULL END
WHERE board_id = @from_board AND deleted_at IS NULL AND id = ANY(@ids::uuid[])
RETURNING id, ref_board_id;

-- name: ReparentWbBoards :exec
UPDATE wb_boards SET parent_id = @parent_id, updated_by = @actor_id WHERE id = ANY(@ids::uuid[]) AND NOT is_root;

-- name: WbSubtreeDepth :one
-- Kedalaman terdalam di bawah papan ini (0 bila tanpa papan anak), supaya pindah kartu papan tidak melewati batas.
WITH RECURSIVE down AS (
  SELECT b.id, 0 AS depth FROM wb_boards b WHERE b.id = @id
  UNION ALL
  SELECT c.id, down.depth + 1
  FROM wb_boards c JOIN down ON c.parent_id = down.id
  WHERE c.deleted_at IS NULL AND down.depth < 64
)
SELECT max(depth)::int FROM down;

-- name: MoveWbConnectors :many
-- Penghubung yang kedua ujungnya ikut pindah, pindah bersama kartunya.
UPDATE wb_connectors SET board_id = @to_board, seq = @seq, updated_by = @actor_id
WHERE board_id = @from_board AND deleted_at IS NULL
  AND from_card_id = ANY(@ids::uuid[]) AND to_card_id = ANY(@ids::uuid[])
RETURNING id;

-- name: MarkWbResync :exec
UPDATE wb_boards SET resync_seq = seq WHERE id = @id;

-- name: GetWbResyncSeq :one
SELECT resync_seq FROM wb_boards WHERE id = @id;

-- name: ListWbCardBounds :one
-- Batas kanan dan atas isi papan, untuk menaruh kartu pindahan di sebelah kanan isi yang sudah ada.
SELECT coalesce(max(x + w), 0)::double precision AS right_edge, coalesce(min(y), 0)::double precision AS top_edge,
       count(*)::bigint AS n
FROM wb_cards WHERE board_id = @board_id AND deleted_at IS NULL AND column_id IS NULL;

-- name: TouchWbCard :one
-- Menaikkan seq kartu tanpa mengubah isinya (jumlah komentar berubah), supaya ikut sinkron ulang.
UPDATE wb_cards SET seq = @seq WHERE id = @id AND deleted_at IS NULL RETURNING board_id;

-- name: GetWbConnectorEnds :one
SELECT from_card_id, to_card_id FROM wb_connectors
WHERE id = @id AND board_id = @board_id AND deleted_at IS NULL
FOR UPDATE;

-- W4: template, pencarian isi.

-- name: SetWbTemplate :one
-- Papan akar tidak bisa menjadi template (yang disalin selalu papan anak beserta turunannya).
UPDATE wb_boards SET is_template = @is_template, updated_by = @actor_id
WHERE id = @id AND deleted_at IS NULL AND NOT is_root
RETURNING id, is_template;

-- name: ListWbTemplates :many
SELECT b.id, b.title, b.parent_id,
       (SELECT count(*) FROM wb_cards k WHERE k.board_id = b.id AND k.deleted_at IS NULL)::bigint AS cards
FROM wb_boards b
WHERE b.is_template AND b.deleted_at IS NULL
ORDER BY lower(b.title), b.id;

-- name: ListWbSubtreeBoards :many
-- Papan ini dan semua turunannya yang hidup, induk selalu sebelum anaknya.
WITH RECURSIVE down AS (
  SELECT b.id, b.parent_id, b.title, b.is_template, 0 AS depth FROM wb_boards b WHERE b.id = @id AND b.deleted_at IS NULL
  UNION ALL
  SELECT c.id, c.parent_id, c.title, c.is_template, down.depth + 1
  FROM wb_boards c JOIN down ON c.parent_id = down.id
  WHERE c.deleted_at IS NULL AND down.depth < 64
)
SELECT id, parent_id, title, is_template, depth::int AS depth FROM down ORDER BY depth, id;

-- name: ListWbCardsForCopy :many
-- Kolom lebih dulu, supaya column_id isi kolom salinan menunjuk kolom yang sudah ada.
SELECT id, board_id, kind, ref_board_id, file_id, column_id, col_order, x, y, w, h, z, color, content, plain,
       style, locked, group_ids
FROM wb_cards
WHERE board_id = ANY(@board_ids::uuid[]) AND deleted_at IS NULL
ORDER BY (column_id IS NOT NULL), board_id, z, id;

-- name: InsertWbCardCopy :exec
INSERT INTO wb_cards (id, board_id, kind, ref_board_id, file_id, column_id, col_order, x, y, w, h, z, color, content,
                      plain, created_by, updated_by, style, locked, group_ids)
VALUES (@id, @board_id, @kind, sqlc.narg(ref_board_id), sqlc.narg(file_id), sqlc.narg(column_id), sqlc.narg(col_order),
        @x, @y, @w, @h, @z, sqlc.narg(color), @content, @plain, @actor_id, @actor_id, @style, @locked, @group_ids::uuid[]);

-- name: ListWbConnectorsForCopy :many
SELECT board_id, from_card_id, to_card_id, geometry, style, label, label_start, label_end
FROM wb_connectors
WHERE board_id = ANY(@board_ids::uuid[]) AND deleted_at IS NULL
ORDER BY created_at, id;

-- name: SearchWbCards :many
-- Isi kartu (plain memuat teks catatan, judul, checklist, tabel, keterangan atau nama berkas, URL tautan) lewat
-- indeks trigram, ditambah judul pratinjau tautan. Dua cabang UNION supaya cabang plain tetap memakai indeksnya.
WITH hits AS (
  SELECT c.id FROM wb_cards c
  WHERE c.deleted_at IS NULL AND c.kind <> 'board' AND c.plain ILIKE '%' || @q::text || '%'
  UNION
  SELECT c.id FROM wb_link_previews lp
  JOIN wb_cards c ON c.kind = 'link' AND c.deleted_at IS NULL AND c.content->>'preview_id' = lp.id::text
  WHERE lp.title ILIKE '%' || @q::text || '%'
  UNION
  -- Kartu orang dicari lewat nama terkini penggunanya (revisi 51), karena namanya tidak disalin ke kartu.
  SELECT c.id FROM users u
  JOIN wb_cards c ON c.kind = 'person' AND c.deleted_at IS NULL AND (c.content->>'user_id')::bigint = u.id
  WHERE u.name ILIKE '%' || @q::text || '%'
)
SELECT c.id, c.board_id, c.kind,
       (CASE WHEN c.kind = 'person' THEN coalesce(pu.name, '') ELSE c.plain END)::text AS plain,
       coalesce(lp.title, '')::text AS preview_title
FROM hits
JOIN wb_cards c ON c.id = hits.id
JOIN wb_boards b ON b.id = c.board_id AND b.deleted_at IS NULL
LEFT JOIN wb_link_previews lp ON c.kind = 'link' AND lp.id = nullif(c.content->>'preview_id', '')::uuid
LEFT JOIN users pu ON c.kind = 'person' AND pu.id = (c.content->>'user_id')::bigint
ORDER BY c.updated_at DESC, c.id
LIMIT 50;

-- name: ListWbPeople :many
-- Daftar alat Tim (revisi 51): pengguna aktif dan nonaktif. Kolom dipilih satu per satu supaya username, peran,
-- email, dan data akun lain tidak pernah ikut terbaca.
SELECT u.id, u.name, u.job_title, u.division_id, d.name AS division_name, u.is_active, u.avatar_id
FROM users u
JOIN divisions d ON d.id = u.division_id
ORDER BY u.is_active DESC, lower(u.name), u.id;

-- name: GetWbPerson :one
SELECT u.id, u.name, u.job_title, u.division_id, d.name AS division_name, u.is_active, u.avatar_id
FROM users u
JOIN divisions d ON d.id = u.division_id
WHERE u.id = @id;
