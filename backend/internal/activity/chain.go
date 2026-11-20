package activity

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
)

// Revisi 33: rantai sambung jam. Jam yang tersambung tidak menyimpan isi sendiri (kolom continues_previous,
// migrasi 000021); Aktivitas, Output, dan berkasnya diturunkan dari jam kepala rantainya saat dibaca.

const (
	hintNotEmpty  = "Kosongkan jam ini dulu untuk menyambung."
	hintPrevEmpty = "Isi jam sebelumnya dulu untuk menyambung."
)

// Conflict adalah permintaan yang sah tetapi bertabrakan dengan isi jam saat ini. Handler memetakannya ke 409.
type Conflict struct{ Message string }

func (e *Conflict) Error() string { return e.Message }

// Joinable: slot ini punya jam sebelumnya di Slots, yaitu semua slot kecuali yang pertama. Revisi 33 (ubahan):
// rantai boleh melintasi jam istirahat, jadi 11.00-12.00 dan 13.00-14.00 juga bisa disambung.
func Joinable(slot int) bool {
	return slot >= 2 && slot <= len(Slots)
}

func hasContent(c Cell) bool {
	return c.Activity != "" || c.Output != "" || len(c.Files) > 0
}

// resolveChains mengisi jam tersambung dengan isi jam kepalanya, lalu menghitung rentang dan petunjuk tombol
// rantai. Dipanggil setelah isian dan berkas satu hari ditempel ke selnya.
func resolveChains(cells []Cell) {
	for i := range cells {
		c := &cells[i]
		// Penanda dari data lama yang tidak sah lagi (mis. daftar slot berubah) diabaikan, bukan dipercaya.
		if c.Linked && !Joinable(c.Slot) {
			c.Linked = false
		}
		if !c.Linked {
			continue
		}
		h := cells[cells[i-1].Head-1]
		c.Head = h.Slot
		c.Activity, c.Output, c.UpdatedBy, c.Files = h.Activity, h.Output, h.UpdatedBy, h.Files
	}
	for i := 0; i < len(cells); {
		end := i
		for end+1 < len(cells) && cells[end+1].Head == cells[i].Slot {
			end++
		}
		start, _, _ := strings.Cut(Slots[i], "-")
		_, last, _ := strings.Cut(Slots[end], "-")
		for j := i; j <= end; j++ {
			cells[j].Span = start + "-" + last
		}
		i = end + 1
	}
	for i := range cells {
		c := &cells[i]
		switch {
		case !c.Joinable || c.Linked:
		case hasContent(*c):
			c.LinkHint = hintNotEmpty
		case !hasContent(cells[i-1]):
			c.LinkHint = hintPrevEmpty
		}
	}
}

// dayCells menyusun delapan sel satu orang di satu tanggal, rantainya sudah diturunkan. Menerima Queries
// supaya bisa dipakai di dalam transaksi sambung.
func dayCells(ctx context.Context, q *sqlc.Queries, userID int64, date string) ([]Cell, error) {
	day, err := pgDate(date)
	if err != nil {
		return nil, err
	}
	cells := emptyCells()
	rows, err := q.ListDailyActivities(ctx, sqlc.ListDailyActivitiesParams{FromDate: day, ToDate: day})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.UserID == userID {
			fill(cells, r)
		}
	}
	files, err := q.ListActivityAttachments(ctx, sqlc.ListActivityAttachmentsParams{FromDate: day, ToDate: day})
	if err != nil {
		return nil, err
	}
	for _, r := range files {
		if r.UserID == userID {
			addFile(cells, r)
		}
	}
	resolveChains(cells)
	return cells, nil
}

// rejectChained menolak isian dan unggahan ke jam yang tersambung: isinya milik jam kepala.
func rejectChained(ctx context.Context, q *sqlc.Queries, p sqlc.GetDailyActivityParams) error {
	row, err := q.GetDailyActivity(ctx, p)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if row.ContinuesPrevious {
		return &Conflict{Message: "Jam ini tersambung ke jam sebelumnya. Lepas sambungan dulu untuk mengisinya."}
	}
	return nil
}

type LinkInput struct {
	UserID int64
	Date   string
	Slot   int
	Linked bool
}

// Link memasang atau melepas sambungan satu jam ke jam sebelumnya. Haknya sama dengan mengisi jam (CanEdit).
// Melepas jam di tengah rantai ikut melepas jam setelahnya yang tersambung ke jam itu; jam kepala tidak berubah.
// Memasang atau melepas yang sudah dalam keadaan itu tidak mengubah apa pun, jadi klik ganda aman.
func (s *Service) Link(ctx context.Context, actor Actor, in LinkInput) error {
	date, err := s.ParseDay(in.Date)
	if err != nil {
		return err
	}
	if in.Slot < 1 || in.Slot > len(Slots) {
		return apperr.Invalid("Slot jam tidak dikenal.")
	}
	if !Joinable(in.Slot) {
		return apperr.Invalid("Jam pertama tidak punya jam sebelumnya untuk disambung.")
	}
	if !CanEdit(actor, in.UserID, date, s.Today()) {
		return ErrForbidden
	}
	day, err := pgDate(date)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		// Mengunci baris akun pemilik, jadi simpan, unggah, dan sambung untuk orang yang sama berjalan bergiliran.
		owner, err := requireStaffOwner(ctx, q, in.UserID)
		if err != nil {
			return err
		}
		cells, err := dayCells(ctx, q, in.UserID, date)
		if err != nil {
			return err
		}
		c := cells[in.Slot-1]
		if c.Linked == in.Linked {
			return nil
		}
		if in.Linked {
			if c.LinkHint != "" {
				return &Conflict{Message: c.LinkHint}
			}
			err = q.LinkDailyActivity(ctx, sqlc.LinkDailyActivityParams{
				UserID: in.UserID, ActivityDate: day, Slot: int16(in.Slot), UpdatedBy: actor.ID,
			})
		} else {
			slots := []int16{}
			for j := in.Slot - 1; j < len(cells) && cells[j].Linked; j++ {
				slots = append(slots, int16(cells[j].Slot))
			}
			err = q.UnlinkDailyActivities(ctx, sqlc.UnlinkDailyActivitiesParams{
				UserID: in.UserID, ActivityDate: day, Slots: slots, UpdatedBy: actor.ID,
			})
		}
		if err != nil || in.UserID == actor.ID {
			return err
		}
		// Sama dengan isian: hanya perubahan Manajer pada jam orang lain yang masuk audit log.
		return audit.Change(ctx, q, audit.Actor{ID: actor.ID, IP: actor.IP},
			"activity_updated", "daily_activity", fmt.Sprintf("%d:%s:%d", in.UserID, date, in.Slot),
			map[string]any{"linked": !in.Linked},
			map[string]any{"name": owner.Name, "date": date, "time": Slots[in.Slot-1], "linked": in.Linked},
		)
	})
}
