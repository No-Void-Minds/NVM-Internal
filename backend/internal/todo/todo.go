// Package todo menyimpan To Do pribadi per akun (revisi 28). Pribadi berarti pribadi: tidak ada endpoint
// untuk membaca milik orang lain, Manajer dan Admin tidak terkecuali, dan isinya tidak masuk audit log
// karena ini catatan pribadi, bukan pekerjaan yang perlu dilacak.
package todo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/db/sqlc"
)

// MaxItems menjaga daftar tetap bisa dibaca dan kolom kiri papan tidak jadi gudang catatan.
const MaxItems = 100

// MaxTitle sama dengan CHECK di migrasi 000019.
const MaxTitle = 200

// ErrNotFound juga dipakai untuk item milik orang lain, supaya jawabannya 404 dan bukan 403.
var ErrNotFound = errors.New("to do tidak ditemukan")

type Item struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func cleanTitle(v string) (string, error) {
	v = strings.Join(strings.Fields(strings.ReplaceAll(v, "\n", " ")), " ")
	if v == "" {
		return "", apperr.Invalid("Tulis dulu isi to do-nya.")
	}
	if utf8.RuneCountInString(v) > MaxTitle {
		return "", apperr.Invalid(fmt.Sprintf("To do maksimal %d karakter.", MaxTitle))
	}
	return v, nil
}

func itemOf(id int64, title string, done pgtype.Timestamptz) Item {
	return Item{ID: id, Title: title, Done: done.Valid}
}

func (s *Service) List(ctx context.Context, userID int64) ([]Item, error) {
	rows, err := sqlc.New(s.pool).ListTodos(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(rows))
	for _, r := range rows {
		out = append(out, itemOf(r.ID, r.Title, r.DoneAt))
	}
	return out, nil
}

// Add menambah satu item di akhir daftar. Batas 100 dicek di transaksi yang sama dengan penyisipan, supaya
// dua tab yang menambah bersamaan tidak bisa melewatinya.
func (s *Service) Add(ctx context.Context, userID int64, title string) (Item, error) {
	title, err := cleanTitle(title)
	if err != nil {
		return Item{}, err
	}
	var out Item
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		n, err := q.CountTodos(ctx, userID)
		if err != nil {
			return err
		}
		if n >= MaxItems {
			return apperr.Invalid(fmt.Sprintf("Daftar to do sudah penuh (%d item). Hapus atau bersihkan yang selesai dulu.", MaxItems))
		}
		pos, err := q.NextTodoPosition(ctx, userID)
		if err != nil {
			return err
		}
		row, err := q.InsertTodo(ctx, sqlc.InsertTodoParams{UserID: userID, Title: title, Position: pos})
		if err != nil {
			return err
		}
		out = itemOf(row.ID, row.Title, row.DoneAt)
		return nil
	})
	return out, err
}

func (s *Service) Rename(ctx context.Context, userID, id int64, title string) error {
	title, err := cleanTitle(title)
	if err != nil {
		return err
	}
	n, err := sqlc.New(s.pool).RenameTodo(ctx, sqlc.RenameTodoParams{ID: id, UserID: userID, Title: title})
	return missing(n, err)
}

// SetDone mencentang atau membatalkan centang. Waktunya dari jam server, bukan dari klien, supaya urutan
// item selesai tidak bisa diatur dari browser.
func (s *Service) SetDone(ctx context.Context, userID, id int64, done bool) error {
	var at pgtype.Timestamptz
	if done {
		at = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	}
	n, err := sqlc.New(s.pool).SetTodoDone(ctx, sqlc.SetTodoDoneParams{ID: id, UserID: userID, DoneAt: at})
	return missing(n, err)
}

func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	n, err := sqlc.New(s.pool).DeleteTodo(ctx, sqlc.DeleteTodoParams{ID: id, UserID: userID})
	return missing(n, err)
}

// ClearDone membuang semua yang sudah selesai milik akun itu, dan mengembalikan jumlahnya.
func (s *Service) ClearDone(ctx context.Context, userID int64) (int64, error) {
	return sqlc.New(s.pool).DeleteDoneTodos(ctx, userID)
}

// Reorder menyusun ulang daftar sesuai urutan id yang dikirim. Satu id milik orang lain membuat seluruh
// permintaan ditolak sebagai tidak ditemukan, jadi urutan tidak bisa dipakai untuk menebak id orang lain.
func (s *Service) Reorder(ctx context.Context, userID int64, ids []int64) error {
	if len(ids) == 0 {
		return apperr.Invalid("Urutan to do kosong.")
	}
	if len(ids) > MaxItems {
		return apperr.Invalid("Urutan to do terlalu panjang.")
	}
	seen := map[int64]bool{}
	for _, id := range ids {
		if seen[id] {
			return apperr.Invalid("Urutan to do memuat id yang sama dua kali.")
		}
		seen[id] = true
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		owned, err := q.CountTodosOwned(ctx, sqlc.CountTodosOwnedParams{UserID: userID, Ids: ids})
		if err != nil {
			return err
		}
		if owned != int64(len(ids)) {
			return ErrNotFound
		}
		for i, id := range ids {
			if _, err := q.SetTodoPosition(ctx, sqlc.SetTodoPositionParams{
				ID: id, UserID: userID, Position: int32(i),
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func missing(n int64, err error) error {
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
