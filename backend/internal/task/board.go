package task

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"
	// Zona waktu ikut dikompilasi karena Windows dan image Docker yang ramping tidak selalu punya data zoneinfo.
	_ "time/tzdata"

	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/db/sqlc"
)

// CompletedDays adalah berapa hari task Completed masih tampil di papan. Setelah itu task hanya ada di Arsip
// (AC-08), tetapi Manajer tetap melihat hasil minggu ini tanpa pindah halaman.
const CompletedDays = 7

const (
	DueOverdue = "overdue"
	// DueToday dan DueWeek sengaja tumpang tindih: "7 hari ke depan" memang menghitung hari ini sebagai
	// hari pertama (due_date BETWEEN today AND today + 6), jadi task hari ini muncul di keduanya.
	DueToday = "today"
	DueWeek  = "week"
	DueNone  = "none"
)

// officeTZ dipakai untuk menentukan "hari ini". Server bisa berjalan dalam UTC, sedangkan tenggat dibaca tim
// dalam WIB; tanpa ini, antara pukul 00.00 dan 07.00 WIB "hari ini" masih terbaca kemarin.
var officeTZ = mustLoadLocation("Asia/Jakarta")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// BoardFilter mengikuti query string papan. Nilai kosong berarti filter itu tidak dipakai.
type BoardFilter struct {
	PICIDs       []int64
	PICUsernames []string
	// RequesterID: "Task saya" (revisi 45) juga memuat task yang diminta orang ini, selain yang dia pegang sebagai
	// PIC task atau PIC sub task. Hanya diisi bersama PICIDs; filter Orang tidak memakainya.
	RequesterID *int64
	DivisionID  *int64
	LabelID     *int64
	Urgency     string
	Due         string
	Keyword     string
}

type BoardColumn struct {
	Status Status
	Tasks  []Summary
	// Total: jumlah task yang cocok dengan filter, bisa lebih besar dari len(Tasks) di kolom Completed (revisi 40).
	Total int
}

// CompletedLimit: kolom Completed di papan paling banyak menampilkan sekian task terbaru (revisi 40). Sisanya
// dibuka lewat Arsip.
const CompletedLimit = 10

var boardOrder = []Status{Requested, OnProgress, Ready, Completed}

// Board mengembalikan empat kolom dalam urutan alur, termasuk kolom yang kosong, supaya frontend tidak perlu
// menebak kolom mana yang hilang.
func (s *Service) Board(ctx context.Context, f BoardFilter, now time.Time) ([]BoardColumn, error) {
	params, err := f.params(now)
	if err != nil {
		return nil, err
	}
	q := sqlc.New(s.pool)
	rows, err := q.ListBoardTasks(ctx, params)
	if err != nil {
		return nil, err
	}
	// Completed dipotong sebelum PIC dan label dimuat, supaya task yang tidak tampil tidak ikut diproses.
	list := make([]sqlc.ListTasksRow, 0, len(rows))
	var done []sqlc.ListTasksRow
	for _, r := range rows {
		if Status(r.Status) == Completed {
			done = append(done, sqlc.ListTasksRow(r))
		} else {
			list = append(list, sqlc.ListTasksRow(r))
		}
	}
	sortCompleted(done)
	completedTotal := len(done)
	if len(done) > CompletedLimit {
		done = done[:CompletedLimit]
	}
	list = append(list, done...)
	summaries, err := attachPeopleAndLabels(ctx, q, list)
	if err != nil {
		return nil, err
	}

	cols := make([]BoardColumn, len(boardOrder))
	index := make(map[Status]int, len(boardOrder))
	for i, st := range boardOrder {
		cols[i] = BoardColumn{Status: st, Tasks: []Summary{}}
		index[st] = i
	}
	for _, sm := range summaries {
		i := index[Status(sm.Task.Status)]
		cols[i].Tasks = append(cols[i].Tasks, sm)
	}
	for i := range cols {
		cols[i].Total = len(cols[i].Tasks)
	}
	cols[index[Completed]].Total = completedTotal
	return cols, nil
}

// sortCompleted: yang paling baru masuk Completed di atas. solved_date adalah timestamptz yang diisi now() setiap
// kali task masuk Completed dan dikosongkan saat keluar, jadi nilainya selalu waktu perpindahan terakhir, termasuk
// task yang pernah dikembalikan lalu Completed lagi. Seri dipecah dengan nomor task terbesar dulu.
func sortCompleted(done []sqlc.ListTasksRow) {
	sort.SliceStable(done, func(a, b int) bool {
		ta, tb := done[a].SolvedDate.Time, done[b].SolvedDate.Time
		if !ta.Equal(tb) {
			return ta.After(tb)
		}
		return taskNumber(done[a].ID) > taskNumber(done[b].ID)
	})
}

// taskNumber membaca angka dari "T-0012". Dibandingkan sebagai angka, bukan teks, supaya T-10000 tetap lebih
// besar dari T-9999.
func taskNumber(id string) int {
	n, _ := strconv.Atoi(strings.TrimPrefix(id, "T-"))
	return n
}

func (f BoardFilter) params(now time.Time) (sqlc.ListBoardTasksParams, error) {
	local := now.In(officeTZ)
	startOfToday := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, officeTZ)
	p := sqlc.ListBoardTasksParams{
		CompletedSince: pgtype.Timestamptz{Time: startOfToday.AddDate(0, 0, -(CompletedDays - 1)), Valid: true},
		// Kolom date tidak punya zona waktu; tanggal WIB dikirim sebagai tengah malam UTC supaya tidak bergeser.
		Today:        pgtype.Date{Time: time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC), Valid: true},
		PicIds:       append([]int64{}, f.PICIDs...),
		PicUsernames: []string{},
	}
	// Slice nil terkirim sebagai NULL, dan cardinality(NULL) bukan 0, sehingga semua task akan tersaring habis.
	for _, u := range f.PICUsernames {
		if u = strings.ToLower(strings.TrimSpace(u)); u != "" {
			p.PicUsernames = append(p.PicUsernames, u)
		}
	}
	switch f.Urgency {
	case "":
	case "low", "medium", "high", "urgent":
		p.Urgency = sqlc.NullTaskUrgency{TaskUrgency: sqlc.TaskUrgency(f.Urgency), Valid: true}
	default:
		return p, apperr.Invalid("Filter urgency tidak dikenal.")
	}
	switch f.Due {
	case "":
	case DueOverdue, DueToday, DueWeek, DueNone:
		p.Due = pgtype.Text{String: f.Due, Valid: true}
	default:
		return p, apperr.Invalid("Filter deadline tidak dikenal.")
	}
	if f.DivisionID != nil {
		p.DivisionID = pgtype.Int8{Int64: *f.DivisionID, Valid: true}
	}
	if f.LabelID != nil {
		p.LabelID = pgtype.Int8{Int64: *f.LabelID, Valid: true}
	}
	if f.RequesterID != nil {
		p.RequesterID = pgtype.Int8{Int64: *f.RequesterID, Valid: true}
	}
	pattern, err := keywordPattern(f.Keyword)
	if err != nil {
		return p, err
	}
	p.Pattern = pattern
	return p, nil
}

// People mengembalikan calon PIC untuk filter papan dan form task.
func (s *Service) People(ctx context.Context) ([]sqlc.ListPeopleRow, error) {
	return sqlc.New(s.pool).ListPeople(ctx)
}
