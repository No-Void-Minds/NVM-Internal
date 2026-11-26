package whiteboard

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/db/sqlc"
)

// Kartu anggota tim (revisi 51). Isinya hanya user_id dan pengaturan tampilan. Nama, foto, jabatan, dan divisi tidak
// disalin ke kartu: dibaca dari users saat papan dimuat dan dikirim ulang lewat siaran saat profil berubah. Hanya data
// yang sudah terlihat di aplikasi (nama, foto, jabatan, divisi) yang dikirim; email, username, dan peran tidak.

const KindPerson = "person"

type PersonContent struct {
	UserID int64  `json:"user_id"`
	View   string `json:"view"`
	Size   string `json:"size"`
	Layout string `json:"layout"`
}

var (
	personViews   = map[string]bool{"foto_nama": true, "foto_nama_jabatan": true, "lengkap": true}
	personSizes   = map[string]bool{"kecil": true, "sedang": true, "besar": true}
	personLayouts = map[string]bool{"atas": true, "samping": true}
)

func cleanPerson(raw json.RawMessage) (PersonContent, string, error) {
	var in PersonContent
	if err := decodeStrict(raw, &in); err != nil {
		return in, "", err
	}
	switch {
	case in.UserID <= 0:
		return in, "", apperr.Invalid("Kartu orang harus menunjuk satu pengguna.")
	case !personViews[in.View]:
		return in, "", apperr.Invalid("Tampilan kartu orang tidak dikenal.")
	case !personSizes[in.Size]:
		return in, "", apperr.Invalid("Ukuran kartu orang tidak dikenal.")
	case !personLayouts[in.Layout]:
		return in, "", apperr.Invalid("Susunan kartu orang tidak dikenal.")
	}
	// plain kosong: pencarian kartu orang memakai nama terkini lewat JOIN (SearchWbCards), bukan salinan nama.
	return in, "", nil
}

// checkPersonUser memastikan pengguna yang ditunjuk kartu orang ada (aktif maupun nonaktif).
func checkPersonUser(ctx context.Context, q *sqlc.Queries, kind string, content []byte) error {
	if kind != KindPerson {
		return nil
	}
	var p PersonContent
	if err := json.Unmarshal(content, &p); err != nil {
		return apperr.Invalid("Isi kartu orang tidak terbaca.")
	}
	if _, err := q.GetWbPerson(ctx, p.UserID); err != nil {
		return apperr.Invalid("Pengguna untuk kartu orang ini tidak ditemukan.")
	}
	return nil
}

// PersonInfo: data profil yang tampil di kartu orang dan di daftar alat Tim.
type PersonInfo struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	JobTitle   *string `json:"job_title"`
	DivisionID int64   `json:"division_id,omitempty"`
	Division   string  `json:"division"`
	Active     bool    `json:"active"`
	AvatarURL  *string `json:"avatar_url"`
}

func avatarURL(id int64, avatar pgtype.UUID) *string {
	if !avatar.Valid {
		return nil
	}
	u := fmt.Sprintf("/api/users/%d/avatar?v=%s", id, avatar.String())
	return &u
}

func textOrNil(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	v := t.String
	return &v
}

func personOf(r sqlc.GetWbPersonRow) PersonInfo {
	return PersonInfo{ID: r.ID, Name: r.Name, JobTitle: textOrNil(r.JobTitle), DivisionID: r.DivisionID,
		Division: r.DivisionName, Active: r.IsActive, AvatarURL: avatarURL(r.ID, r.AvatarID)}
}

// People: daftar alat Tim, pengguna aktif dan nonaktif.
func (s *Service) People(ctx context.Context) ([]PersonInfo, error) {
	rows, err := sqlc.New(s.pool).ListWbPeople(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PersonInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, personOf(sqlc.GetWbPersonRow(r)))
	}
	return out, nil
}

// Person: data terkini satu pengguna, untuk siaran setelah profilnya berubah.
func (s *Service) Person(ctx context.Context, id int64) (PersonInfo, error) {
	r, err := sqlc.New(s.pool).GetWbPerson(ctx, id)
	if err != nil {
		return PersonInfo{}, notFound(err)
	}
	return personOf(r), nil
}
