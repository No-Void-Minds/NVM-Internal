package account

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/db/sqlc"
	"nvm-internal/internal/storage"
)

var ErrNoAvatar = errors.New("akun belum punya foto profil")

const maxJobTitleLen = 100

type ProfileInput struct {
	Name     string
	JobTitle string
}

// UpdateOwnProfile hanya pernah mengubah akun pelaku sendiri: id sasaran diambil dari actor, bukan dari
// permintaan, jadi tidak ada jalan mengubah profil orang lain lewat endpoint ini. Divisi dan peran tidak disentuh.
func (s *Service) UpdateOwnProfile(ctx context.Context, actor Actor, in ProfileInput) error {
	name := strings.TrimSpace(in.Name)
	if err := validateName(name); err != nil {
		return withField(err, "name")
	}
	job := strings.TrimSpace(in.JobTitle)
	if utf8.RuneCountInString(job) > maxJobTitleLen {
		return invalidField("job_title", "Jabatan maksimal 100 karakter.")
	}

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		u, err := q.GetOwnProfileForUpdate(ctx, actor.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		before, after := map[string]any{}, map[string]any{}
		if u.Name != name {
			before["name"], after["name"] = u.Name, name
		}
		if u.JobTitle.String != job {
			before["job_title"], after["job_title"] = nullable(u.JobTitle.String), nullable(job)
		}
		if len(after) == 0 {
			return nil
		}
		// Jabatan kosong disimpan NULL; constraint di migrasi 000005 menolak teks kosong.
		if err := q.UpdateOwnProfile(ctx, sqlc.UpdateOwnProfileParams{
			ID: actor.ID, Name: name, JobTitle: pgtype.Text{String: job, Valid: job != ""},
		}); err != nil {
			return err
		}
		return record(ctx, q, actor, "profile_updated", actor.ID, before, after)
	})
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Avatars menyimpan dan membuka foto profil. Dipisah dari Service karena butuh penyimpanan berkas.
type Avatars struct {
	pool  *pgxpool.Pool
	files *storage.Store
	log   *slog.Logger
}

func NewAvatars(pool *pgxpool.Pool, files *storage.Store, log *slog.Logger) *Avatars {
	return &Avatars{pool: pool, files: files, log: log}
}

// Set menyimpan foto yang sudah diproses ProcessAvatar sebagai foto akun pelaku. Berkas ditulis lebih dulu, baru
// database diubah, supaya avatar_id tidak pernah menunjuk berkas yang belum ada.
func (a *Avatars) Set(ctx context.Context, actor Actor, jpeg []byte) (string, error) {
	id, err := newUUID()
	if err != nil {
		return "", err
	}
	path, err := a.files.AvatarPath(id.String())
	if err != nil {
		return "", err
	}
	if err := a.write(path, jpeg); err != nil {
		return "", err
	}

	var old pgtype.UUID
	err = pgx.BeginFunc(ctx, a.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		u, err := q.GetOwnProfileForUpdate(ctx, actor.ID)
		if err != nil {
			return err
		}
		old = u.AvatarID
		if err := q.SetUserAvatar(ctx, sqlc.SetUserAvatarParams{ID: actor.ID, AvatarID: id}); err != nil {
			return err
		}
		return record(ctx, q, actor, "profile_updated", actor.ID, nil, map[string]any{"avatar": "diganti"})
	})
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	a.removeOld(old)
	return id.String(), nil
}

// Delete menghapus foto akun pelaku. Akun tanpa foto dianggap sudah beres, tanpa audit.
func (a *Avatars) Delete(ctx context.Context, actor Actor) error {
	var old pgtype.UUID
	err := pgx.BeginFunc(ctx, a.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		u, err := q.GetOwnProfileForUpdate(ctx, actor.ID)
		if err != nil {
			return err
		}
		if !u.AvatarID.Valid {
			return nil
		}
		old = u.AvatarID
		if err := q.SetUserAvatar(ctx, sqlc.SetUserAvatarParams{ID: actor.ID}); err != nil {
			return err
		}
		return record(ctx, q, actor, "profile_updated", actor.ID, nil, map[string]any{"avatar": "dihapus"})
	})
	if err != nil {
		return err
	}
	a.removeOld(old)
	return nil
}

// Open membuka foto akun mana pun. Pemanggil wajib menutup berkasnya.
func (a *Avatars) Open(ctx context.Context, userID int64) (string, *os.File, error) {
	id, err := sqlc.New(a.pool).GetUserAvatar(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !id.Valid) {
		return "", nil, ErrNoAvatar
	}
	if err != nil {
		return "", nil, err
	}
	path, err := a.files.AvatarPath(id.String())
	if err != nil {
		return "", nil, err
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		// Database menunjuk berkas yang hilang (mis. folder dipulihkan dari backup lama): tampil sebagai inisial.
		a.log.Warn("berkas foto profil hilang", "user_id", userID, "avatar_id", id.String())
		return "", nil, ErrNoAvatar
	}
	if err != nil {
		return "", nil, err
	}
	return id.String(), f, nil
}

// write menulis ke berkas sementara di folder yang sama lalu rename, jadi GET foto tidak pernah membaca
// berkas setengah jadi. Berkas ditutup sebelum rename karena Windows menolak memindahkan berkas yang terbuka.
func (a *Avatars) write(path string, data []byte) error {
	f, err := a.files.CreateAvatarTemp()
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, werr := f.Write(data)
	cerr := f.Close()
	if err := errors.Join(werr, cerr); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// removeOld dipanggil setelah transaksi berhasil. Kalau gagal, berkas yatim dibiarkan untuk pembersih di S12-07,
// karena database sudah benar dan pengguna tidak perlu melihat galat.
func (a *Avatars) removeOld(old pgtype.UUID) {
	if !old.Valid {
		return
	}
	path, err := a.files.AvatarPath(old.String())
	if err == nil {
		err = os.Remove(path)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		a.log.Warn("foto profil lama gagal dihapus", "avatar_id", old.String(), "err", err)
	}
}

// newUUID membuat UUID versi 4. Nama berkas di disk harus lolos pola UUID di paket storage.
func newUUID() (pgtype.UUID, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return pgtype.UUID{}, err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return pgtype.UUID{Bytes: b, Valid: true}, nil
}
