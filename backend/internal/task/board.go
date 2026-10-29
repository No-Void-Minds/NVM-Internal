package task

import (
	"context"
	"sort"
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
	DueWeek    = "week"
	DueNone    = "none"
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
	DivisionID   *int64
	LabelID      *int64
	Urgency      string
	Due          string
}

type BoardColumn struct {
	Status Status
	Tasks  []Summary
}

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
	list := make([]sqlc.ListTasksRow, len(rows))
	for i, r := range rows {
		list[i] = sqlc.ListTasksRow(r)
	}
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
	// Di kolom Completed yang penting kapan selesainya, bukan urgency-nya.
	done := cols[index[Completed]].Tasks
	sort.SliceStable(done, func(a, b int) bool {
		return done[a].Task.SolvedDate.Time.After(done[b].Task.SolvedDate.Time)
	})
	return cols, nil
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
	case DueOverdue, DueWeek, DueNone:
		p.Due = pgtype.Text{String: f.Due, Valid: true}
	default:
		return p, apperr.Invalid("Filter tenggat tidak dikenal.")
	}
	if f.DivisionID != nil {
		p.DivisionID = pgtype.Int8{Int64: *f.DivisionID, Valid: true}
	}
	if f.LabelID != nil {
		p.LabelID = pgtype.Int8{Int64: *f.LabelID, Valid: true}
	}
	return p, nil
}

// People mengembalikan calon PIC untuk filter papan dan form task.
func (s *Service) People(ctx context.Context) ([]sqlc.ListPeopleRow, error) {
	return sqlc.New(s.pool).ListPeople(ctx)
}
