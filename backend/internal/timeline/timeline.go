// Package timeline menyimpan kalender bersama tim (launching, produksi, kehadiran) dan Important Notes
// (revisi 30). Semua yang login boleh membaca; hanya Manajer yang menulis, dan itu diperiksa di sini.
package timeline

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

var (
	ErrForbidden = errors.New("hanya manajer yang menulis timeline")
	ErrNotFound  = errors.New("isi timeline tidak ditemukan")
)

const (
	// MaxRangeDays sama dengan CHECK di migrasi 000020: hari pertama dan terakhir ikut dihitung.
	MaxRangeDays = 62
	MaxTitle     = 200
	MaxText      = 2000
	MaxNoteTitle = 120
	MaxNotes     = 50
)

// officeTZ: tanggal di Timeline adalah tanggal kantor, sama dengan papan dan Aktivitas.
var officeTZ = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		panic(err)
	}
	return loc
}()

// Actor membawa akses tambahan pengguna (revisi 38) yang dibaca dari database di setiap permintaan bersama data
// sesi, jadi akses yang dicabut langsung berlaku tanpa login ulang.
type Actor struct {
	ID             int64
	Role           sqlc.UserRole
	IP             *netip.Addr
	ManageTimeline bool
	ManageNotes    bool
}

// Manajer selalu punya kedua akses. Akses tambahan hanya berlaku untuk Staf; Admin tetap hanya membaca walau
// datanya entah bagaimana berisi true.
func (a Actor) has(flag bool) bool {
	return a.Role == sqlc.UserRoleManager || (a.Role == sqlc.UserRoleStaff && flag)
}

func (a Actor) canWrite() bool      { return a.has(a.ManageTimeline) }
func (a Actor) canWriteNotes() bool { return a.has(a.ManageNotes) }

func (a Actor) audit() audit.Actor { return audit.Actor{ID: a.ID, IP: a.IP} }

// CanWrite dan CanWriteNotes dipakai handler untuk can_write di respons, supaya frontend tidak menghitung
// aturannya sendiri.
func CanWrite(a Actor) bool      { return a.canWrite() }
func CanWriteNotes(a Actor) bool { return a.canWriteNotes() }

var absenceLabels = map[sqlc.AbsenceType]string{
	sqlc.AbsenceTypeSakit:       "Sakit",
	sqlc.AbsenceTypeIzinPribadi: "Izin pribadi",
	sqlc.AbsenceTypeCutiPribadi: "Cuti pribadi",
	sqlc.AbsenceTypeCutiKhusus:  "Cuti khusus",
	sqlc.AbsenceTypeTukarLibur:  "Tukar libur",
	sqlc.AbsenceTypeTukarShift:  "Tukar shift",
}

type Person struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Event struct {
	ID          int64   `json:"id"`
	Category    string  `json:"category"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date"`
	User        *Person `json:"user"`
	// AbsenceType null bila alasannya disembunyikan dari pembaca ini; AbsenceLabel selalu terisi untuk kehadiran.
	AbsenceType  *string `json:"absence_type"`
	AbsenceLabel string  `json:"absence_label,omitempty"`
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// MonthRange membaca "YYYY-MM" menjadi hari pertama dan terakhir bulan itu.
func MonthRange(month string) (time.Time, time.Time, error) {
	first, err := time.ParseInLocation("2006-01", month, officeTZ)
	if err != nil {
		return time.Time{}, time.Time{}, apperr.Invalid("Bulan harus berformat YYYY-MM.")
	}
	return first, first.AddDate(0, 1, -1), nil
}

func pgDate(t time.Time) pgtype.Date { return pgtype.Date{Time: t, Valid: !t.IsZero()} }

func dateString(d pgtype.Date) string { return d.Time.Format("2006-01-02") }

// Events mengembalikan event yang beririsan dengan bulan itu. Alasan dan keterangan kehadiran dikirim lengkap ke
// semua pengguna yang login (revisi 38, transparan), menggantikan penyamaran "Tidak hadir" revisi 30.
func (s *Service) Events(ctx context.Context, month string) ([]Event, error) {
	from, to, err := MonthRange(month)
	if err != nil {
		return nil, err
	}
	rows, err := sqlc.New(s.pool).ListTimelineEvents(ctx, sqlc.ListTimelineEventsParams{
		FromDate: pgDate(from), ToDate: pgDate(to),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(rows))
	for _, r := range rows {
		e := Event{
			ID: r.ID, Category: string(r.Category), Title: r.Title.String, Description: r.Description.String,
			StartDate: dateString(r.StartDate),
		}
		if r.EndDate.Valid {
			v := dateString(r.EndDate)
			e.EndDate = &v
		}
		if r.UserID.Valid {
			e.User = &Person{ID: r.UserID.Int64, Name: r.UserName.String}
		}
		if r.AbsenceType.Valid {
			t := r.AbsenceType.AbsenceType
			v := string(t)
			e.AbsenceType = &v
			e.AbsenceLabel = absenceLabels[t]
		}
		out = append(out, e)
	}
	return out, nil
}

type EventInput struct {
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	UserID      int64  `json:"user_id"`
	AbsenceType string `json:"absence_type"`
}

type cleanEvent struct {
	category    sqlc.TimelineCategory
	title       pgtype.Text
	description pgtype.Text
	start, end  pgtype.Date
	userID      pgtype.Int8
	absence     sqlc.NullAbsenceType
	userName    string
}

func cleanText(v string, max int, label string) (pgtype.Text, error) {
	v = strings.TrimSpace(strings.ReplaceAll(v, "\r\n", "\n"))
	if utf8.RuneCountInString(v) > max {
		return pgtype.Text{}, apperr.Invalid(fmt.Sprintf("%s maksimal %d karakter.", label, max))
	}
	return pgtype.Text{String: v, Valid: v != ""}, nil
}

func parseDay(v, label string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(v), officeTZ)
	if err != nil {
		return t, apperr.Invalid(label + " tidak terbaca. Pakai format tanggal YYYY-MM-DD.")
	}
	return t, nil
}

func (s *Service) clean(ctx context.Context, q *sqlc.Queries, in EventInput) (cleanEvent, error) {
	var c cleanEvent
	switch sqlc.TimelineCategory(in.Category) {
	case sqlc.TimelineCategoryLaunching, sqlc.TimelineCategoryProduksi, sqlc.TimelineCategoryKehadiran:
		c.category = sqlc.TimelineCategory(in.Category)
	default:
		return c, apperr.Invalid("Pilih jenis kartu: Launching, Produksi, atau Kehadiran.")
	}
	if strings.TrimSpace(in.StartDate) == "" {
		return c, apperr.Invalid("Tanggal mulai wajib diisi.")
	}
	start, err := parseDay(in.StartDate, "Tanggal mulai")
	if err != nil {
		return c, err
	}
	c.start = pgDate(start)
	if strings.TrimSpace(in.EndDate) != "" {
		end, err := parseDay(in.EndDate, "Tanggal sampai")
		if err != nil {
			return c, err
		}
		if end.Before(start) {
			return c, apperr.Invalid("Tanggal sampai tidak boleh sebelum tanggal mulai.")
		}
		if days := int(end.Sub(start).Hours()/24) + 1; days > MaxRangeDays {
			return c, apperr.Invalid(fmt.Sprintf("Rentang paling panjang %d hari (yang diisi %d hari).", MaxRangeDays, days))
		}
		// Sampai yang sama dengan tanggal mulai berarti satu hari, disimpan seperti tanpa Sampai.
		if end.After(start) {
			c.end = pgDate(end)
		}
	}

	if c.category != sqlc.TimelineCategoryKehadiran {
		title, err := cleanText(strings.Join(strings.Fields(in.Title), " "), MaxTitle, "Judul")
		if err != nil {
			return c, err
		}
		if !title.Valid {
			return c, apperr.Invalid("Judul wajib diisi.")
		}
		c.title = title
		c.description, err = cleanText(in.Description, MaxText, "Deskripsi")
		return c, err
	}

	if in.UserID <= 0 {
		return c, apperr.Invalid("Pilih nama orang yang tidak hadir.")
	}
	name, err := q.GetActiveUserName(ctx, in.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, apperr.Invalid("Akun itu tidak ada atau sudah nonaktif. Pilih akun yang aktif.")
	}
	if err != nil {
		return c, err
	}
	c.userID = pgtype.Int8{Int64: in.UserID, Valid: true}
	c.userName = name
	t := sqlc.AbsenceType(in.AbsenceType)
	if _, ok := absenceLabels[t]; !ok {
		return c, apperr.Invalid("Pilih alasan tidak hadir.")
	}
	c.absence = sqlc.NullAbsenceType{AbsenceType: t, Valid: true}
	c.description, err = cleanText(in.Description, MaxText, "Keterangan")
	return c, err
}

// auditValues sengaja tanpa alasan dan keterangan kehadiran: Audit Log juga dibaca Admin, yang tidak berhak
// melihat alasan pribadi (keputusan revisi 30).
func auditValues(category sqlc.TimelineCategory, title pgtype.Text, start, end pgtype.Date, userID pgtype.Int8, name string) map[string]any {
	m := map[string]any{"category": string(category), "start_date": dateString(start)}
	if end.Valid {
		m["end_date"] = dateString(end)
	}
	if category == sqlc.TimelineCategoryKehadiran {
		m["user_id"] = userID.Int64
		m["name"] = name
	} else {
		m["title"] = title.String
	}
	return m
}

func (s *Service) CreateEvent(ctx context.Context, actor Actor, in EventInput) (int64, error) {
	if !actor.canWrite() {
		return 0, ErrForbidden
	}
	var id int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		c, err := s.clean(ctx, q, in)
		if err != nil {
			return err
		}
		id, err = q.InsertTimelineEvent(ctx, sqlc.InsertTimelineEventParams{
			Category: c.category, Title: c.title, Description: c.description, StartDate: c.start, EndDate: c.end,
			UserID: c.userID, AbsenceType: c.absence, ActorID: actor.ID,
		})
		if err != nil {
			return err
		}
		return audit.Change(ctx, q, actor.audit(), "timeline_event_created", "timeline_event", fmt.Sprint(id), nil,
			auditValues(c.category, c.title, c.start, c.end, c.userID, c.userName))
	})
	return id, err
}

func (s *Service) UpdateEvent(ctx context.Context, actor Actor, id int64, in EventInput) error {
	if !actor.canWrite() {
		return ErrForbidden
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		old, err := q.GetTimelineEvent(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		c, err := s.clean(ctx, q, in)
		if err != nil {
			return err
		}
		if _, err := q.UpdateTimelineEvent(ctx, sqlc.UpdateTimelineEventParams{
			ID: id, Category: c.category, Title: c.title, Description: c.description, StartDate: c.start,
			EndDate: c.end, UserID: c.userID, AbsenceType: c.absence, ActorID: actor.ID,
		}); err != nil {
			return err
		}
		oldName := ""
		if old.UserID.Valid {
			oldName, _ = q.GetActiveUserName(ctx, old.UserID.Int64)
		}
		return audit.Change(ctx, q, actor.audit(), "timeline_event_updated", "timeline_event", fmt.Sprint(id),
			auditValues(old.Category, old.Title, old.StartDate, old.EndDate, old.UserID, oldName),
			auditValues(c.category, c.title, c.start, c.end, c.userID, c.userName))
	})
}

func (s *Service) DeleteEvent(ctx context.Context, actor Actor, id int64) error {
	if !actor.canWrite() {
		return ErrForbidden
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		old, err := q.GetTimelineEvent(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if _, err := q.DeleteTimelineEvent(ctx, sqlc.DeleteTimelineEventParams{ID: id, ActorID: actor.ID}); err != nil {
			return err
		}
		name := ""
		if old.UserID.Valid {
			name, _ = q.GetActiveUserName(ctx, old.UserID.Int64)
		}
		return audit.Change(ctx, q, actor.audit(), "timeline_event_deleted", "timeline_event", fmt.Sprint(id),
			auditValues(old.Category, old.Title, old.StartDate, old.EndDate, old.UserID, name), nil)
	})
}

type Note struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type NoteInput struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func cleanNote(in NoteInput) (string, string, error) {
	title, err := cleanText(strings.Join(strings.Fields(in.Title), " "), MaxNoteTitle, "Judul catatan")
	if err != nil {
		return "", "", err
	}
	if !title.Valid {
		return "", "", apperr.Invalid("Judul catatan wajib diisi.")
	}
	body, err := cleanText(in.Body, MaxText, "Isi catatan")
	return title.String, body.String, err
}

func (s *Service) Notes(ctx context.Context) ([]Note, error) {
	rows, err := sqlc.New(s.pool).ListTimelineNotes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Note, 0, len(rows))
	for _, r := range rows {
		out = append(out, Note{ID: r.ID, Title: r.Title, Body: r.Body})
	}
	return out, nil
}

func (s *Service) CreateNote(ctx context.Context, actor Actor, in NoteInput) (Note, error) {
	if !actor.canWriteNotes() {
		return Note{}, ErrForbidden
	}
	title, body, err := cleanNote(in)
	if err != nil {
		return Note{}, err
	}
	var n Note
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		count, err := q.CountTimelineNotes(ctx)
		if err != nil {
			return err
		}
		if count >= MaxNotes {
			return apperr.Invalid(fmt.Sprintf("Important Notes sudah penuh (%d catatan). Hapus yang tidak berlaku dulu.", MaxNotes))
		}
		pos, err := q.NextTimelineNotePosition(ctx)
		if err != nil {
			return err
		}
		id, err := q.InsertTimelineNote(ctx, sqlc.InsertTimelineNoteParams{Title: title, Body: body, Position: pos, ActorID: actor.ID})
		if err != nil {
			return err
		}
		n = Note{ID: id, Title: title, Body: body}
		return audit.Change(ctx, q, actor.audit(), "timeline_note_created", "timeline_note", fmt.Sprint(id), nil,
			map[string]any{"title": title})
	})
	return n, err
}

func (s *Service) UpdateNote(ctx context.Context, actor Actor, id int64, in NoteInput) error {
	if !actor.canWriteNotes() {
		return ErrForbidden
	}
	title, body, err := cleanNote(in)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		old, err := q.GetTimelineNote(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if old.Title == title && old.Body == body {
			return nil
		}
		if _, err := q.UpdateTimelineNote(ctx, sqlc.UpdateTimelineNoteParams{ID: id, Title: title, Body: body, ActorID: actor.ID}); err != nil {
			return err
		}
		return audit.Change(ctx, q, actor.audit(), "timeline_note_updated", "timeline_note", fmt.Sprint(id),
			map[string]any{"title": old.Title}, map[string]any{"title": title})
	})
}

func (s *Service) DeleteNote(ctx context.Context, actor Actor, id int64) error {
	if !actor.canWriteNotes() {
		return ErrForbidden
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		old, err := q.GetTimelineNote(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if _, err := q.DeleteTimelineNote(ctx, sqlc.DeleteTimelineNoteParams{ID: id, ActorID: actor.ID}); err != nil {
			return err
		}
		return audit.Change(ctx, q, actor.audit(), "timeline_note_deleted", "timeline_note", fmt.Sprint(id),
			map[string]any{"title": old.Title}, nil)
	})
}

// ReorderNotes menyusun ulang semua catatan sesuai urutan id. Daftarnya harus lengkap, supaya dua Manajer yang
// mengurutkan bersamaan tidak meninggalkan posisi ganda. Urutan tidak dicatat di audit: isinya tidak berubah.
func (s *Service) ReorderNotes(ctx context.Context, actor Actor, ids []int64) error {
	if !actor.canWriteNotes() {
		return ErrForbidden
	}
	seen := map[int64]bool{}
	for _, id := range ids {
		if seen[id] {
			return apperr.Invalid("Urutan catatan memuat catatan yang sama dua kali.")
		}
		seen[id] = true
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		total, err := q.CountTimelineNotes(ctx)
		if err != nil {
			return err
		}
		found, err := q.CountTimelineNotesIn(ctx, ids)
		if err != nil {
			return err
		}
		if found != int64(len(ids)) || total != found {
			return apperr.Invalid("Daftar catatan sudah berubah. Muat ulang halaman, lalu urutkan lagi.")
		}
		for i, id := range ids {
			if err := q.SetTimelineNotePosition(ctx, sqlc.SetTimelineNotePositionParams{ID: id, Position: int32(i)}); err != nil {
				return err
			}
		}
		return nil
	})
}
