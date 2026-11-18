// Package activity menyimpan Aktivitas Harian (revisi 21), pengganti spreadsheet "Daily Activity Monitoring".
// Satu sel adalah satu orang, satu tanggal, dan satu slot jam, dengan isi Aktivitas dan Output.
package activity

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"slices"
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

// MaxRangeDays membatasi rentang yang bisa dibaca atau diekspor sekali jalan, supaya satu permintaan tidak
// membentuk tabel atau sheet raksasa.
const MaxRangeDays = 31

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

// Person adalah satu kolom tabel Aktivitas: isinya seluruh sel orang itu di rentang yang diminta, dikelompokkan
// per tanggal, supaya halaman mingguan cukup satu panggilan API.
type Person struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	DivisionName string  `json:"division_name"`
	AvatarID     *string `json:"avatar_id"`
	IsActive     bool    `json:"is_active"`
	// Filled dan Total mengisi ringkasan "n dari N jam terisi" untuk rentang yang tampil.
	Filled int               `json:"filled"`
	Total  int               `json:"total"`
	Days   map[string][]Cell `json:"days"`
}

type Range struct {
	From string `json:"from"`
	To   string `json:"to"`
	// Dates adalah tanggal yang benar-benar ditampilkan: hari kerja selalu ada, Sabtu dan Minggu hanya bila
	// ada isinya (keputusan revisi 21).
	Dates  []string `json:"dates"`
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

// WeekRange mengembalikan Senin sampai Jumat dari minggu tanggal itu. Minggu kerja dihitung mulai Senin,
// jadi tanggal hari Minggu masuk minggu yang Seninnya besok, bukan yang sudah lewat.
func WeekRange(date string) (string, string, error) {
	d, err := time.Parse(dayLayout, date)
	if err != nil {
		return "", "", apperr.Invalid("Tanggal tidak valid. Pakai bentuk 2026-11-18.")
	}
	// Weekday: 0 Minggu sampai 6 Sabtu. Senin jadi titik nol.
	back := (int(d.Weekday()) + 6) % 7
	monday := d.AddDate(0, 0, -back)
	return monday.Format(dayLayout), monday.AddDate(0, 0, 4).Format(dayLayout), nil
}

// Range mengembalikan satu kolom per orang beserta seluruh selnya di rentang tanggal, cukup untuk tampilan
// harian (from sama dengan to) maupun mingguan dalam satu panggilan.
func (s *Service) Range(ctx context.Context, from, to string) (Range, error) {
	out := Range{From: from, To: to, Slots: Slots, Dates: []string{}, People: []Person{}}
	people, dates, err := s.collect(ctx, from, to)
	if err != nil {
		return out, err
	}
	out.Dates, out.People = dates, people
	return out, nil
}

// collect adalah isi tabel untuk satu rentang: kolom orang, sel per tanggal, dan daftar tanggal yang tampil.
// Dipakai halaman Aktivitas dan ekspor, supaya aturan akhir pekan dan aturan kolom hanya ada di satu tempat.
func (s *Service) collect(ctx context.Context, from, to string) ([]Person, []string, error) {
	start, end, err := parseSpan(from, to)
	if err != nil {
		return nil, nil, err
	}
	fromDay, toDay := pgtype.Date{Time: start, Valid: true}, pgtype.Date{Time: end, Valid: true}
	q := sqlc.New(s.pool)
	rows, err := q.ListActivityPeople(ctx, sqlc.ListActivityPeopleParams{FromDate: fromDay, ToDate: toDay})
	if err != nil {
		return nil, nil, err
	}
	cells, err := q.ListDailyActivities(ctx, sqlc.ListDailyActivitiesParams{FromDate: fromDay, ToDate: toDay})
	if err != nil {
		return nil, nil, err
	}

	people := make([]Person, 0, len(rows))
	index := make(map[int64]int, len(rows))
	for i, r := range rows {
		index[r.ID] = i
		people = append(people, Person{
			ID: r.ID, Name: r.Name, DivisionName: r.DivisionName, IsActive: r.IsActive,
			AvatarID: avatarID(r.AvatarID), Days: map[string][]Cell{},
		})
	}
	// Sel diisi lebih dulu, karena daftar tanggal butuh tahu akhir pekan mana yang ada isinya.
	for _, r := range cells {
		i, ok := index[r.UserID]
		if !ok {
			continue
		}
		key := r.ActivityDate.Time.Format(dayLayout)
		if people[i].Days[key] == nil {
			people[i].Days[key] = emptyCells()
		}
		fill(people[i].Days[key], r)
	}

	dates := []string{}
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		key := d.Format(dayLayout)
		if weekend(d) && !dayHasContent(people, key) {
			continue
		}
		dates = append(dates, key)
	}
	// Setiap orang punya sel lengkap di setiap tanggal yang tampil, jadi frontend tidak perlu menebak sel kosong.
	for i := range people {
		for _, key := range dates {
			if people[i].Days[key] == nil {
				people[i].Days[key] = emptyCells()
			}
			for _, c := range people[i].Days[key] {
				if c.Activity != "" || c.Output != "" {
					people[i].Filled++
				}
			}
		}
		// Tanggal di luar daftar (akhir pekan kosong yang tersaring) dibuang supaya tidak terkirim sia-sia.
		for key := range people[i].Days {
			if !slices.Contains(dates, key) {
				delete(people[i].Days, key)
			}
		}
		people[i].Total = len(dates) * len(Slots)
	}
	return people, dates, nil
}

// parseSpan memeriksa bentuk tanggal, urutannya, dan batas panjang rentang.
func parseSpan(from, to string) (time.Time, time.Time, error) {
	start, err := time.Parse(dayLayout, from)
	if err != nil {
		return start, start, apperr.Invalid("Tanggal awal tidak valid. Pakai bentuk 2026-11-18.")
	}
	end, err := time.Parse(dayLayout, to)
	if err != nil {
		return start, end, apperr.Invalid("Tanggal akhir tidak valid. Pakai bentuk 2026-11-18.")
	}
	if end.Before(start) {
		return start, end, apperr.Invalid("Tanggal akhir tidak boleh lebih awal dari tanggal awal.")
	}
	if int(end.Sub(start).Hours()/24)+1 > MaxRangeDays {
		return start, end, apperr.Invalid(fmt.Sprintf("Rentang maksimal %d hari.", MaxRangeDays))
	}
	return start, end, nil
}

func dayHasContent(people []Person, date string) bool {
	for _, p := range people {
		for _, c := range p.Days[date] {
			if c.Activity != "" || c.Output != "" {
				return true
			}
		}
	}
	return false
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
