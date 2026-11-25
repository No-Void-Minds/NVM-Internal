package task

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/audit"
	"nvm-internal/internal/db/sqlc"
)

// ScheduleInput adalah bagian Jadwal di form Buat task dan Ubah task (revisi 19).
type ScheduleInput struct {
	// Freq kosong atau "none" berarti tidak berulang.
	Freq        string
	SkipWeekend bool
	Weekdays    []int
	MonthDay    int
	EndDate     string
	// ApplyToSeries hanya untuk Ubah task yang termasuk seri: templat dan aturan seri ikut diperbarui
	// (keputusan 4). Kemunculan yang sudah ada tidak berubah.
	ApplyToSeries bool
}

func (in *ScheduleInput) recurring() bool {
	return in != nil && in.Freq != "" && in.Freq != "none"
}

// SeriesTemplate disimpan sebagai JSONB di task_series.template. Satu kolom yang diganti utuh, bukan tabel
// anak: templat selalu dibaca dan ditulis sekaligus, tidak pernah dicari per isinya, dan tanpa tabel anak app
// tidak butuh hak DELETE untuk mengganti daftar sub task-nya.
type SeriesTemplate struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Urgency     string  `json:"urgency"`
	PICIDs      []int64 `json:"pic_ids"`
	LabelIDs    []int64 `json:"label_ids"`
	RequestedBy int64   `json:"requested_by"`
	// DueOffsetDays: tenggat dikurangi tanggal kemunculan. nil berarti kemunculan tanpa tenggat.
	DueOffsetDays *int            `json:"due_offset_days"`
	Subtasks      []SeriesSubtask `json:"subtasks"`
}

// SeriesSubtask: lampiran tidak ikut (keputusan 3), tautan acuan ikut. Tautan dan catatan hasil juga tidak,
// karena itu diisi saat pengerjaan.
type SeriesSubtask struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	Reference     string `json:"reference_url"`
	PICID         int64  `json:"pic_id"`
	DueOffsetDays *int   `json:"due_offset_days"`
}

func scheduleRule(in *ScheduleInput, start time.Time) (Rule, error) {
	r := Rule{Freq: in.Freq, SkipWeekend: in.SkipWeekend, Weekdays: in.Weekdays, MonthDay: in.MonthDay, Start: start}
	end, err := parseDate(in.EndDate, "Tanggal berhenti")
	if err != nil {
		return r, err
	}
	if end.Valid {
		r.End = end.Time
	}
	return r, r.Validate()
}

func ruleOf(freq sqlc.RecurrenceFreq, skip bool, weekdays []int16, monthDay pgtype.Int2, start, end pgtype.Date) Rule {
	r := Rule{Freq: string(freq), SkipWeekend: skip, MonthDay: int(monthDay.Int16), Start: start.Time}
	for _, d := range weekdays {
		r.Weekdays = append(r.Weekdays, int(d))
	}
	if end.Valid {
		r.End = end.Time
	}
	return r
}

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: !t.IsZero()}
}

func offset(due pgtype.Date, from time.Time) *int {
	if !due.Valid {
		return nil
	}
	n := int(due.Time.Sub(from).Hours() / 24)
	return &n
}

// templateFromTask membaca templat dari isi task sekarang (sesudah dibuat atau diubah), supaya templat selalu
// sama dengan task yang dilihat Manajer saat menyimpan.
func templateFromTask(ctx context.Context, q *sqlc.Queries, taskID string, occurrence time.Time) (SeriesTemplate, error) {
	var t SeriesTemplate
	d, err := q.GetTaskDetail(ctx, taskID)
	if err != nil {
		return t, err
	}
	t = SeriesTemplate{
		Title: d.Title, Description: d.Description.String, Urgency: string(d.Urgency),
		RequestedBy: d.RequestedBy, DueOffsetDays: offset(d.DueDate, occurrence),
		PICIDs: []int64{}, LabelIDs: []int64{}, Subtasks: []SeriesSubtask{},
	}
	pics, err := q.ListTaskAssignees(ctx, []string{taskID})
	if err != nil {
		return t, err
	}
	for _, p := range pics {
		t.PICIDs = append(t.PICIDs, p.ID)
	}
	labels, err := q.ListTaskLabels(ctx, []string{taskID})
	if err != nil {
		return t, err
	}
	for _, l := range labels {
		t.LabelIDs = append(t.LabelIDs, l.ID)
	}
	t.Subtasks, err = seriesSubtasksOf(ctx, q, taskID, occurrence)
	return t, err
}

// seriesSubtasksOf menyalin bagian rencana sub task task ini: centang, catatan hasil, tautan hasil, dan berkas
// diisi saat pengerjaan, jadi tidak ikut.
func seriesSubtasksOf(ctx context.Context, q *sqlc.Queries, taskID string, occurrence time.Time) ([]SeriesSubtask, error) {
	out := []SeriesSubtask{}
	subs, err := loadSubtaskViews(ctx, q, taskID)
	if err != nil {
		return nil, err
	}
	for _, st := range subs {
		ss := SeriesSubtask{
			Title: st.Title, Description: st.Description, Reference: st.ReferenceURL,
			DueOffsetDays: offset(st.DueDate, occurrence),
		}
		if len(st.Assignees) > 0 {
			ss.PICID = st.Assignees[0].ID
		}
		out = append(out, ss)
	}
	return out, nil
}

// syncTemplateSubtasks (revisi 29): perubahan sub task di kemunculan terbaru seri yang aktif langsung menjadi
// sub task templat, di transaksi pemanggil. Kemunculan lama dan seri yang dihentikan tidak menyentuh templat,
// supaya membereskan task minggu lalu tidak mengubah task minggu depan. Field task induk tetap lewat Ubah
// dengan centang "Pakai perubahan ini".
func syncTemplateSubtasks(ctx context.Context, q *sqlc.Queries, actor Actor, taskID string) error {
	ref, err := q.GetTaskSeriesRef(ctx, taskID)
	if err != nil {
		return err
	}
	if !ref.SeriesID.Valid || !ref.OccurrenceDate.Valid {
		return nil
	}
	cur, err := q.GetSeriesForUpdate(ctx, ref.SeriesID.Int64)
	if err != nil || !cur.Active {
		return err
	}
	latest, err := q.IsLatestOccurrence(ctx, sqlc.IsLatestOccurrenceParams{
		SeriesID: ref.SeriesID, OccurrenceDate: ref.OccurrenceDate,
	})
	if err != nil || !latest.Bool {
		return err
	}
	var tmpl SeriesTemplate
	if err := json.Unmarshal(cur.Template, &tmpl); err != nil {
		return err
	}
	subs, err := seriesSubtasksOf(ctx, q, taskID, ref.OccurrenceDate.Time)
	if err != nil {
		return err
	}
	// Mengubah catatan hasil atau tautan hasil saja tidak mengubah rencana, jadi tidak perlu ditulis dan dicatat.
	oldRaw, _ := json.Marshal(tmpl.Subtasks)
	newRaw, _ := json.Marshal(subs)
	if bytes.Equal(oldRaw, newRaw) {
		return nil
	}
	before := subtaskTitles(tmpl.Subtasks)
	tmpl.Subtasks = subs
	raw, err := json.Marshal(tmpl)
	if err != nil {
		return err
	}
	if err := q.UpdateSeries(ctx, sqlc.UpdateSeriesParams{
		ID: cur.ID, Freq: cur.Freq, SkipWeekend: cur.SkipWeekend, Weekdays: cur.Weekdays, MonthDay: cur.MonthDay,
		EndDate: cur.EndDate, Template: raw,
	}); err != nil {
		return err
	}
	return audit.Change(ctx, q, actor.auditActor(), "series_updated", "task", taskID,
		map[string]any{"series_id": cur.ID, "subtasks": before},
		map[string]any{"series_id": cur.ID, "subtasks": subtaskTitles(subs), "source": "subtask"})
}

func subtaskTitles(subs []SeriesSubtask) []string {
	out := []string{}
	for _, st := range subs {
		out = append(out, st.Title)
	}
	return out
}

func ruleAudit(r Rule) map[string]any {
	m := map[string]any{"freq": r.Freq}
	switch r.Freq {
	case FreqDaily:
		m["skip_weekend"] = r.SkipWeekend
	case FreqWeekly:
		m["weekdays"] = r.Weekdays
	case FreqMonthly:
		m["month_day"] = r.MonthDay
	}
	if !r.End.IsZero() {
		m["end_date"] = r.End.Format("2006-01-02")
	}
	return m
}

func monthDayParam(r Rule) pgtype.Int2 {
	return pgtype.Int2{Int16: int16(r.MonthDay), Valid: r.Freq == FreqMonthly}
}

// weekdaysParam: kosong selain mingguan, sesuai CHECK task_series_weekly_days.
func weekdaysParam(r Rule) []int16 {
	w := []int16{}
	if r.Freq == FreqWeekly {
		for _, d := range r.Weekdays {
			w = append(w, int16(d))
		}
	}
	return w
}

// startSeries menjadikan task ini kemunculan pertama seri baru. Tanggal kemunculannya tanggal mulai task, atau
// hari ini bila kosong.
func (s *Service) startSeries(ctx context.Context, q *sqlc.Queries, actor Actor, taskID string, in *ScheduleInput) error {
	ref, err := q.GetTaskSeriesRef(ctx, taskID)
	if err != nil {
		return err
	}
	occurrence := officeDate(s.clock())
	if ref.StartDate.Valid {
		occurrence = ref.StartDate.Time
	}
	r, err := scheduleRule(in, occurrence)
	if err != nil {
		return err
	}
	tmpl, err := templateFromTask(ctx, q, taskID, occurrence)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(tmpl)
	if err != nil {
		return err
	}
	sid, err := q.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Freq: sqlc.RecurrenceFreq(r.Freq), SkipWeekend: r.Freq == FreqDaily && r.SkipWeekend,
		Weekdays: weekdaysParam(r), MonthDay: monthDayParam(r), StartDate: pgDate(r.Start), EndDate: pgDate(r.End),
		Template: raw, CreatedBy: actor.ID,
	})
	if err != nil {
		return err
	}
	if err := q.SetTaskSeries(ctx, sqlc.SetTaskSeriesParams{ID: taskID, SeriesID: pgtype.Int8{Int64: sid, Valid: true}, OccurrenceDate: pgDate(occurrence)}); err != nil {
		return err
	}
	after := ruleAudit(r)
	after["series_id"] = sid
	after["occurrence_date"] = occurrence.Format("2006-01-02")
	return audit.Change(ctx, q, actor.auditActor(), "series_created", "task", taskID, nil, after)
}

// applySchedule dipanggil di akhir Update, di transaksi yang sama, setelah isi task tertulis.
func (s *Service) applySchedule(ctx context.Context, q *sqlc.Queries, actor Actor, taskID string, in *ScheduleInput) error {
	if in == nil {
		return nil
	}
	ref, err := q.GetTaskSeriesRef(ctx, taskID)
	if err != nil {
		return err
	}
	if !ref.SeriesID.Valid {
		if in.recurring() {
			return s.startSeries(ctx, q, actor, taskID, in)
		}
		return nil
	}
	if !in.ApplyToSeries {
		return nil
	}
	if !in.recurring() {
		return stopSeries(ctx, q, actor, taskID, ref.SeriesID.Int64)
	}
	cur, err := q.GetSeriesForUpdate(ctx, ref.SeriesID.Int64)
	if err != nil {
		return err
	}
	if !cur.Active {
		return apperr.Invalid("Pengulangan task ini sudah dihentikan. Buat task berulang baru bila perlu.")
	}
	r, err := scheduleRule(in, cur.StartDate.Time)
	if err != nil {
		return err
	}
	tmpl, err := templateFromTask(ctx, q, taskID, ref.OccurrenceDate.Time)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(tmpl)
	if err != nil {
		return err
	}
	if err := q.UpdateSeries(ctx, sqlc.UpdateSeriesParams{
		ID: cur.ID, Freq: sqlc.RecurrenceFreq(r.Freq), SkipWeekend: r.Freq == FreqDaily && r.SkipWeekend,
		Weekdays: weekdaysParam(r), MonthDay: monthDayParam(r), EndDate: pgDate(r.End), Template: raw,
	}); err != nil {
		return err
	}
	before := ruleAudit(ruleOf(cur.Freq, cur.SkipWeekend, cur.Weekdays, cur.MonthDay, cur.StartDate, cur.EndDate))
	after := ruleAudit(r)
	before["series_id"], after["series_id"] = cur.ID, cur.ID
	return audit.Change(ctx, q, actor.auditActor(), "series_updated", "task", taskID, before, after)
}

func stopSeries(ctx context.Context, q *sqlc.Queries, actor Actor, taskID string, seriesID int64) error {
	n, err := q.StopSeries(ctx, seriesID)
	if err != nil || n == 0 {
		return err
	}
	return audit.Change(ctx, q, actor.auditActor(), "series_stopped", "task", taskID,
		nil, map[string]any{"series_id": seriesID})
}

// StopSeries menghentikan pengulangan dari task mana pun di seri itu. Hak mengatur jadwal mengikuti hak membuat
// task (keputusan 5), jadi hanya Manajer. Kemunculan yang sudah ada tidak berubah.
func (s *Service) StopSeries(ctx context.Context, actor Actor, taskID string) error {
	if actor.Role != RoleManager {
		return errStopSeriesForbidden
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		ref, err := q.GetTaskSeriesRef(ctx, taskID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if !ref.SeriesID.Valid {
			return apperr.Invalid("Task ini tidak berulang.")
		}
		return stopSeries(ctx, q, actor, taskID, ref.SeriesID.Int64)
	})
}

// CreateDueOccurrences membuat kemunculan hari ini (menurut officeTZ dari now) untuk setiap seri aktif. Hanya hari
// ini: hari yang terlewat tidak disusulkan. Aman dipanggil berulang dan dari dua instance sekaligus: kemunculan
// yang sudah ada dilewati, dan bentrok unique (series_id, occurrence_date) dianggap "sudah dibuat".
// now disuntikkan supaya bisa diuji dengan jam palsu.
func (s *Service) CreateDueOccurrences(ctx context.Context, now time.Time) (created []string, err error) {
	created, _, err = s.createDueOccurrences(ctx, now)
	return created, err
}

// abandonedAfterDays: seri aktif yang semua kemunculannya sudah dihapus dan kemunculan terakhirnya lebih lama
// dari ini dianggap ditinggalkan (revisi 31). Ticker tidak membuat kemunculan baru dan mencatat peringatan, tetapi
// seri tidak dihentikan otomatis, karena yang berhak menghentikannya Manajer.
const abandonedAfterDays = 7

func (s *Service) createDueOccurrences(ctx context.Context, now time.Time) (created []string, abandoned []int64, err error) {
	today := officeDate(now)
	q := sqlc.New(s.pool)
	if _, err := q.EndExpiredSeries(ctx, pgDate(today)); err != nil {
		return nil, nil, err
	}
	list, err := q.ListActiveSeries(ctx, pgDate(today))
	if err != nil {
		return nil, nil, err
	}
	var errs []error
	for _, row := range list {
		r := ruleOf(row.Freq, row.SkipWeekend, row.Weekdays, row.MonthDay, row.StartDate, row.EndDate)
		if !r.OccursOn(today) {
			continue
		}
		exists, err := q.OccurrenceExists(ctx, sqlc.OccurrenceExistsParams{
			SeriesID: pgtype.Int8{Int64: row.ID, Valid: true}, OccurrenceDate: pgDate(today),
		})
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if exists {
			continue
		}
		st, err := q.SeriesLiveState(ctx, pgtype.Int8{Int64: row.ID, Valid: true})
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if st.Live == 0 && st.LastOccurrence.Valid && st.LastOccurrence.Time.Before(today.AddDate(0, 0, -abandonedAfterDays)) {
			abandoned = append(abandoned, row.ID)
			continue
		}
		var id string
		err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
			id, err = createOccurrence(ctx, sqlc.New(tx), sqlc.TaskSeries{
				ID: row.ID, Template: row.Template, CreatedBy: row.CreatedBy,
			}, today)
			return err
		})
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "tasks_series_occurrence_key":
			// Instance lain sudah membuatnya lebih dulu.
		case err != nil:
			errs = append(errs, fmt.Errorf("seri %d: %w", row.ID, err))
		default:
			created = append(created, id)
		}
	}
	return created, abandoned, errors.Join(errs...)
}

var errNoEligiblePIC = errors.New("tidak ada PIC templat yang masih aktif")

func createOccurrence(ctx context.Context, q *sqlc.Queries, series sqlc.TaskSeries, day time.Time) (string, error) {
	var t SeriesTemplate
	if err := json.Unmarshal(series.Template, &t); err != nil {
		return "", err
	}
	// PIC yang sudah nonaktif dilewati; urutan templat dipertahankan karena PIC pertama menentukan divisi.
	eligible, err := q.GetEligiblePICs(ctx, t.PICIDs)
	if err != nil {
		return "", err
	}
	divisionOf := map[int64]int64{}
	for _, e := range eligible {
		divisionOf[e.ID] = e.DivisionID
	}
	var pics []int64
	for _, id := range t.PICIDs {
		if _, ok := divisionOf[id]; ok {
			pics = append(pics, id)
		}
	}
	if len(pics) == 0 {
		return "", errNoEligiblePIC
	}
	due := pgtype.Date{}
	if t.DueOffsetDays != nil {
		due = pgDate(day.AddDate(0, 0, *t.DueOffsetDays))
	}
	id, err := q.CreateTask(ctx, sqlc.CreateTaskParams{
		Title: t.Title, Description: pgtype.Text{String: t.Description, Valid: t.Description != ""},
		DivisionID: divisionOf[pics[0]], RequestedBy: t.RequestedBy, Urgency: sqlc.TaskUrgency(t.Urgency),
		StartDate: pgDate(day), DueDate: due,
	})
	if err != nil {
		return "", err
	}
	if err := q.SetTaskSeries(ctx, sqlc.SetTaskSeriesParams{
		ID: id, SeriesID: pgtype.Int8{Int64: series.ID, Valid: true}, OccurrenceDate: pgDate(day),
	}); err != nil {
		return "", err
	}
	if err := addTaskPICs(ctx, q, series.CreatedBy, id, pics); err != nil {
		return "", err
	}
	labels, err := q.GetLabelsByIDs(ctx, t.LabelIDs)
	if err != nil {
		return "", err
	}
	for _, l := range labels {
		if l.IsActive {
			if err := q.AddTaskLabel(ctx, sqlc.AddTaskLabelParams{TaskID: id, LabelID: l.ID}); err != nil {
				return "", err
			}
		}
	}
	for i, st := range t.Subtasks {
		subDue := pgtype.Date{}
		if st.DueOffsetDays != nil {
			subDue = pgDate(day.AddDate(0, 0, *st.DueOffsetDays))
		}
		sid, err := q.CreateSubtask(ctx, sqlc.CreateSubtaskParams{
			TaskID: id, Title: st.Title, Description: st.Description, ReferenceUrl: st.Reference,
			Position: int32(i), CreatedBy: userRef(series.CreatedBy), DueDate: subDue,
		})
		if err != nil {
			return "", err
		}
		if st.PICID == 0 {
			continue
		}
		// PIC sub task yang sudah tidak memenuhi syarat dibiarkan kosong, bukan menggagalkan kemunculan.
		ok, err := q.GetEligiblePICs(ctx, []int64{st.PICID})
		if err != nil {
			return "", err
		}
		if len(ok) == 1 {
			if err := q.AddSubtaskAssignee(ctx, sqlc.AddSubtaskAssigneeParams{
				SubtaskID: sid, UserID: st.PICID, AssignedBy: series.CreatedBy,
			}); err != nil {
				return "", err
			}
		}
	}
	return id, audit.Record(ctx, q, audit.Entry{
		Action: "occurrence_created", Entity: "task", EntityID: id,
		New: map[string]any{"series_id": series.ID, "occurrence_date": day.Format("2006-01-02")},
	})
}

// RunOccurrenceTicker menjalankan CreateDueOccurrences sekali saat mulai lalu setiap interval, sampai ctx
// selesai. Galat dicatat dan tidak menghentikan ticker. onCreated (boleh nil) menerima id kemunculan baru setelah
// transaksinya commit, untuk papan realtime (revisi 47).
func (s *Service) RunOccurrenceTicker(ctx context.Context, log *slog.Logger, every time.Duration, onCreated func([]string)) {
	run := func() {
		created, abandoned, err := s.createDueOccurrences(ctx, s.clock())
		if len(created) > 0 {
			log.Info("kemunculan task berulang dibuat", "tasks", created)
			if onCreated != nil {
				onCreated(created)
			}
		}
		if len(abandoned) > 0 {
			log.Warn("seri aktif tanpa kemunculan tersisa dilewati, minta Manajer menghentikannya", "series", abandoned)
		}
		if err != nil {
			log.Error("pembuat kemunculan task berulang", "err", err)
		}
	}
	run()
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}

// NextOccurrence: kemunculan berikutnya setelah hari ini dan setelah tanggal kemunculan task yang dilihat.
func NextOccurrence(sr sqlc.GetSeriesRow, occurrence pgtype.Date, now time.Time) (time.Time, bool) {
	if !sr.Active {
		return time.Time{}, false
	}
	after := officeDate(now)
	if occurrence.Valid && occurrence.Time.After(after) {
		after = occurrence.Time
	}
	return ruleOf(sr.Freq, sr.SkipWeekend, sr.Weekdays, sr.MonthDay, sr.StartDate, sr.EndDate).Next(after)
}

// TemplateDueOffset membaca jarak tenggat dari templat untuk tampilan; nil bila tanpa tenggat atau rusak.
func TemplateDueOffset(raw []byte) *int {
	var t SeriesTemplate
	if json.Unmarshal(raw, &t) != nil {
		return nil
	}
	return t.DueOffsetDays
}
