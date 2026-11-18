// Package activity menyimpan Aktivitas Harian (revisi 21), pengganti spreadsheet "Daily Activity Monitoring".
// Satu sel adalah satu orang, satu tanggal, dan satu slot jam, dengan isi Aktivitas dan Output.
package activity

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
)

// Slots adalah jam kerja yang bisa diisi. Nomor slot yang tersimpan di database adalah indeks + 1, jadi
// urutan daftar ini tidak boleh diubah tanpa migrasi data. 12.00-13.00 istirahat, karena itu tidak ada.
var Slots = []string{
	"09.00-10.00", "10.00-11.00", "11.00-12.00",
	"13.00-14.00", "14.00-15.00", "15.00-16.00", "16.00-17.00", "17.00-18.00",
}

// MaxFieldLen membatasi satu kolom isian. Sama dengan CHECK di migrasi 000016.
const MaxFieldLen = 1000

// MaxExportDays membatasi rentang ekspor, supaya satu permintaan tidak membentuk sheet raksasa.
const MaxExportDays = 31

// officeTZ menentukan "hari ini": server bisa berjalan dalam UTC, sedangkan hari kerja dihitung tim dalam WIB.
var officeTZ = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		panic(err)
	}
	return loc
}()

const dayLayout = "2006-01-02"

var ErrForbidden = errors.New("tidak boleh mengubah isian ini")

// Actor adalah pengguna yang sedang masuk.
type Actor struct {
	ID   int64
	Role sqlc.UserRole
	IP   *netip.Addr
}

// Cell dikirim apa adanya sebagai JSON, satu per slot dan selalu lengkap walau belum ada isinya.
type Cell struct {
	Slot     int    `json:"slot"`
	Time     string `json:"time"`
	Activity string `json:"activity"`
	Output   string `json:"output"`
	// Nama penyimpan terakhir, hanya diisi bila bukan pemilik isian (Manajer yang mengubah isian orang lain).
	UpdatedBy string `json:"updated_by"`
}

type Person struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	DivisionName string  `json:"division_name"`
	AvatarID     *string `json:"avatar_id"`
	IsActive     bool    `json:"is_active"`
	// Filled false berarti orang itu belum mengisi apa pun di tanggal ini.
	Filled bool   `json:"filled"`
	Cells  []Cell `json:"cells"`
}

type Day struct {
	Date   string   `json:"date"`
	Slots  []string `json:"slots"`
	People []Person `json:"people"`
}

type Mine struct {
	Date  string   `json:"date"`
	Slots []string `json:"slots"`
	Cells []Cell   `json:"cells"`
}

type SaveInput struct {
	UserID   int64
	Date     string
	Slot     int
	Activity string
	Output   string
}

type Service struct {
	pool *pgxpool.Pool
	// now bisa diganti di test supaya aturan "hanya hari ini" bisa diuji tanpa mengubah jam sistem.
	now func() time.Time
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, now: time.Now}
}

// Today adalah tanggal kantor hari ini dalam bentuk "2006-01-02".
func (s *Service) Today() string {
	return s.now().In(officeTZ).Format(dayLayout)
}

// ParseDay menolak tanggal yang tidak berbentuk 2006-01-02. Tanggal kosong berarti hari ini.
func (s *Service) ParseDay(v string) (string, error) {
	if strings.TrimSpace(v) == "" {
		return s.Today(), nil
	}
	d, err := time.ParseInLocation(dayLayout, strings.TrimSpace(v), officeTZ)
	if err != nil {
		return "", apperr.Invalid("Tanggal tidak valid. Pakai bentuk 2026-11-18.")
	}
	return d.Format(dayLayout), nil
}

// CanEdit menentukan siapa yang boleh menyimpan satu sel. Staf hanya isian miliknya sendiri dan hanya untuk
// hari ini; Manajer isian siapa pun di tanggal berapa pun; Admin hanya membaca (keputusan revisi 21).
func CanEdit(actor Actor, targetID int64, date, today string) bool {
	switch actor.Role {
	case sqlc.UserRoleManager:
		return true
	case sqlc.UserRoleStaff:
		return targetID == actor.ID && date == today
	default:
		return false
	}
}

// Clean membuang spasi di tepi, menyeragamkan akhir baris, dan menolak isian yang melebihi batas.
func Clean(v, label string) (string, error) {
	v = strings.ReplaceAll(v, "\r\n", "\n")
	v = strings.Trim(v, " \t\r\n")
	if utf8.RuneCountInString(v) > MaxFieldLen {
		return "", apperr.Invalid(fmt.Sprintf("%s maksimal %d karakter.", label, MaxFieldLen))
	}
	return v, nil
}

func (s *Service) Save(ctx context.Context, actor Actor, in SaveInput) error {
	date, err := s.ParseDay(in.Date)
	if err != nil {
		return err
	}
	if in.Slot < 1 || in.Slot > len(Slots) {
		return apperr.Invalid("Slot jam tidak dikenal.")
	}
	activity, err := Clean(in.Activity, "Aktivitas")
	if err != nil {
		return err
	}
	output, err := Clean(in.Output, "Output")
	if err != nil {
		return err
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
		owner, err := q.GetUserForUpdate(ctx, in.UserID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.Invalid("Akun tidak ditemukan.")
		}
		if err != nil {
			return err
		}
		// Tabel aktivitas hanya ada untuk Staf, jadi Manajer pun tidak bisa menitipkan isian ke Manajer lain.
		if owner.Role != sqlc.UserRoleStaff {
			return apperr.Invalid("Isian aktivitas hanya ada untuk akun Staf.")
		}
		if !owner.IsActive {
			return apperr.Invalid("Akun ini sudah nonaktif, isiannya tidak bisa diubah.")
		}

		before, err := q.GetDailyActivity(ctx, sqlc.GetDailyActivityParams{
			UserID: in.UserID, ActivityDate: day, Slot: int16(in.Slot),
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err := q.UpsertDailyActivity(ctx, sqlc.UpsertDailyActivityParams{
			UserID: in.UserID, ActivityDate: day, Slot: int16(in.Slot),
			Activity: pgText(activity), Output: pgText(output), UpdatedBy: actor.ID,
		}); err != nil {
			return err
		}
		// Hanya perubahan Manajer pada isian orang lain yang dicatat; isian sendiri tidak masuk audit log.
		if in.UserID == actor.ID {
			return nil
		}
		return audit.Change(ctx, q, audit.Actor{ID: actor.ID, IP: actor.IP},
			"activity_updated", "daily_activity", fmt.Sprintf("%d:%s:%d", in.UserID, date, in.Slot),
			map[string]any{"activity": before.Activity.String, "output": before.Output.String},
			map[string]any{
				"name": owner.Name, "date": date, "time": Slots[in.Slot-1],
				"activity": activity, "output": output,
			},
		)
	})
}

// Mine mengembalikan isian satu orang di satu tanggal, lengkap dengan daftar slot supaya frontend tidak
// menulis sendiri jam kerjanya.
func (s *Service) Mine(ctx context.Context, userID int64, date string) (Mine, error) {
	out := Mine{Date: date, Slots: Slots, Cells: emptyCells()}
	rows, err := s.rows(ctx, date, date)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		if r.UserID == userID {
			fill(out.Cells, r)
		}
	}
	return out, nil
}

// Day mengembalikan satu kolom per orang: semua Staf aktif, ditambah akun yang punya isian di tanggal itu.
func (s *Service) Day(ctx context.Context, date string) (Day, error) {
	out := Day{Date: date, Slots: Slots, People: []Person{}}
	people, err := s.people(ctx, date, date)
	if err != nil {
		return out, err
	}
	rows, err := s.rows(ctx, date, date)
	if err != nil {
		return out, err
	}
	index := make(map[int64]int, len(people))
	for i, p := range people {
		index[p.ID] = i
	}
	for _, r := range rows {
		i, ok := index[r.UserID]
		if !ok {
			continue
		}
		fill(people[i].Cells, r)
		if r.Activity.String != "" || r.Output.String != "" {
			people[i].Filled = true
		}
	}
	out.People = people
	return out, nil
}

// people adalah kolom tabel: dipakai halaman harian (from sama dengan to) dan ekspor rentang tanggal.
func (s *Service) people(ctx context.Context, from, to string) ([]Person, error) {
	fromDay, err := pgDate(from)
	if err != nil {
		return nil, err
	}
	toDay, err := pgDate(to)
	if err != nil {
		return nil, err
	}
	rows, err := sqlc.New(s.pool).ListActivityPeople(ctx, sqlc.ListActivityPeopleParams{FromDate: fromDay, ToDate: toDay})
	if err != nil {
		return nil, err
	}
	people := make([]Person, 0, len(rows))
	for _, r := range rows {
		people = append(people, Person{
			ID: r.ID, Name: r.Name, DivisionName: r.DivisionName, IsActive: r.IsActive,
			AvatarID: avatarID(r.AvatarID), Cells: emptyCells(),
		})
	}
	return people, nil
}

func (s *Service) rows(ctx context.Context, from, to string) ([]sqlc.ListDailyActivitiesRow, error) {
	fromDay, err := pgDate(from)
	if err != nil {
		return nil, err
	}
	toDay, err := pgDate(to)
	if err != nil {
		return nil, err
	}
	return sqlc.New(s.pool).ListDailyActivities(ctx, sqlc.ListDailyActivitiesParams{FromDate: fromDay, ToDate: toDay})
}

func emptyCells() []Cell {
	cells := make([]Cell, len(Slots))
	for i, label := range Slots {
		cells[i] = Cell{Slot: i + 1, Time: label}
	}
	return cells
}

func fill(cells []Cell, r sqlc.ListDailyActivitiesRow) {
	i := int(r.Slot) - 1
	if i < 0 || i >= len(cells) {
		return
	}
	cells[i].Activity = r.Activity.String
	cells[i].Output = r.Output.String
	if r.UpdatedBy != r.UserID {
		cells[i].UpdatedBy = r.UpdatedByName
	}
}

// pgDate mengirim tanggal WIB sebagai tengah malam UTC: kolom date tidak punya zona waktu, jadi tanpa ini
// tanggalnya bisa bergeser satu hari.
func pgDate(v string) (pgtype.Date, error) {
	d, err := time.Parse(dayLayout, v)
	if err != nil {
		return pgtype.Date{}, apperr.Invalid("Tanggal tidak valid. Pakai bentuk 2026-11-18.")
	}
	return pgtype.Date{Time: d, Valid: true}, nil
}

// Isian kosong disimpan sebagai NULL, bukan string kosong, supaya sel yang dikosongkan tidak perlu dihapus
// dan app tetap tidak butuh hak DELETE di tabel ini.
func pgText(v string) pgtype.Text {
	if v == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: v, Valid: true}
}

func avatarID(v pgtype.UUID) *string {
	if !v.Valid {
		return nil
	}
	s := fmt.Sprintf("%x-%x-%x-%x-%x", v.Bytes[0:4], v.Bytes[4:6], v.Bytes[6:8], v.Bytes[8:10], v.Bytes[10:16])
	return &s
}
