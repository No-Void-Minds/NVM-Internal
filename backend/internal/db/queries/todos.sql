-- name: ListTodos :many
-- Belum selesai di atas, lalu yang selesai; keduanya urut posisi. Satu urutan untuk semua pemakai daftar ini.
SELECT id, title, done_at, position FROM user_todos
WHERE user_id = @user_id
ORDER BY (done_at IS NOT NULL), position, id;

-- name: CountTodos :one
SELECT count(*) FROM user_todos WHERE user_id = @user_id;

-- name: NextTodoPosition :one
SELECT coalesce(max(position) + 1, 0)::int FROM user_todos WHERE user_id = @user_id;

-- name: InsertTodo :one
INSERT INTO user_todos (user_id, title, position) VALUES (@user_id, @title, @position)
RETURNING id, title, done_at, position;

-- name: GetTodo :one
-- user_id ikut di WHERE: item milik orang lain diperlakukan sama dengan item yang tidak ada (404, bukan 403),
-- supaya keberadaannya tidak bocor.
SELECT id, title, done_at, position FROM user_todos WHERE id = @id AND user_id = @user_id;

-- name: RenameTodo :execrows
UPDATE user_todos SET title = @title WHERE id = @id AND user_id = @user_id;

-- name: SetTodoDone :execrows
UPDATE user_todos SET done_at = @done_at WHERE id = @id AND user_id = @user_id;

-- name: DeleteTodo :execrows
DELETE FROM user_todos WHERE id = @id AND user_id = @user_id;

-- name: DeleteDoneTodos :execrows
DELETE FROM user_todos WHERE user_id = @user_id AND done_at IS NOT NULL;

-- name: SetTodoPosition :execrows
UPDATE user_todos SET position = @position WHERE id = @id AND user_id = @user_id;

-- name: CountTodosOwned :one
-- Berapa dari daftar id ini yang memang milik akun itu. Dipakai urutkan ulang: kalau jumlahnya tidak sama
-- dengan jumlah id yang dikirim, ada id milik orang lain dan seluruh permintaan ditolak.
SELECT count(*) FROM user_todos WHERE user_id = @user_id AND id = ANY(@ids::bigint[]);
