package task

import (
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/apperr"
	"nvm-internal/internal/db/sqlc"
)

// fields adalah atribut task yang diisi Manajer, sudah dirapikan dan siap disimpan.
type fields struct {
	title          string
	description    pgtype.Text
	urgency        sqlc.TaskUrgency
	startDate      pgtype.Date
	dueDate        pgtype.Date
	deliverableURL pgtype.Text
}

func cleanFields(title, description, urgency, start, due, deliverable string) (fields, error) {
	var f fields

	f.title = strings.TrimSpace(title)
	if f.title == "" {
		return f, apperr.Invalid("Judul wajib diisi.")
	}
	if utf8.RuneCountInString(f.title) > 200 {
		return f, apperr.Invalid("Judul maksimal 200 karakter.")
	}

	if d := strings.TrimSpace(description); d != "" {
		if utf8.RuneCountInString(d) > 5000 {
			return f, apperr.Invalid("Deskripsi maksimal 5000 karakter.")
		}
		f.description = pgtype.Text{String: d, Valid: true}
	}

	switch u := sqlc.TaskUrgency(urgency); u {
	case sqlc.TaskUrgencyLow, sqlc.TaskUrgencyMedium, sqlc.TaskUrgencyHigh, sqlc.TaskUrgencyUrgent:
		f.urgency = u
	default:
		return f, apperr.Invalid("Pilih urgency: Low, Medium, High, atau Urgent.")
	}

	var err error
	if f.startDate, err = parseDate(start, "Tanggal mulai"); err != nil {
		return f, err
	}
	if f.dueDate, err = parseDate(due, "Tenggat"); err != nil {
		return f, err
	}
	if f.startDate.Valid && f.dueDate.Valid && f.dueDate.Time.Before(f.startDate.Time) {
		return f, apperr.Invalid("Tenggat tidak boleh lebih awal dari tanggal mulai.")
	}

	if f.deliverableURL, err = cleanURL(deliverable); err != nil {
		return f, err
	}
	return f, nil
}

func parseDate(v, label string) (pgtype.Date, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return pgtype.Date{}, nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return pgtype.Date{}, apperr.Invalid(label + " tidak valid.")
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

// cleanURL hanya menerima tautan http dan https (NFR-SEC-06), supaya tautan javascript: tidak bisa disisipkan.
func cleanURL(v string) (pgtype.Text, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return pgtype.Text{}, nil
	}
	if len(v) > 2000 {
		return pgtype.Text{}, apperr.Invalid("Tautan deliverable terlalu panjang.")
	}
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return pgtype.Text{}, apperr.Invalid("Tautan deliverable harus diawali http:// atau https://.")
	}
	return pgtype.Text{String: v, Valid: true}, nil
}

func uniqueIDs(ids []int64) []int64 {
	seen := make(map[int64]bool, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

var monthNames = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus",
	"September", "Oktober", "November", "Desember"}

// formatDay menulis tanggal untuk pesan galat dalam bentuk yang dibaca tim, mis. "20 November 2026".
func formatDay(d pgtype.Date) string {
	return fmt.Sprintf("%d %s %d", d.Time.Day(), monthNames[d.Time.Month()-1], d.Time.Year())
}

// checkSubtaskDue: tenggat sub task tidak boleh melewati tenggat task induk (revisi 16). Task tanpa tenggat
// tidak membatasi apa pun.
func checkSubtaskDue(sub, taskDue pgtype.Date) error {
	if sub.Valid && taskDue.Valid && sub.Time.After(taskDue.Time) {
		return apperr.Invalid(fmt.Sprintf("Tenggat sub task tidak boleh melewati tenggat task (%s).", formatDay(taskDue)))
	}
	return nil
}

// dateValue menulis tanggal untuk audit log sebagai YYYY-MM-DD, atau nil bila kosong.
func dateValue(d pgtype.Date) any {
	if !d.Valid {
		return nil
	}
	return d.Time.Format("2006-01-02")
}
