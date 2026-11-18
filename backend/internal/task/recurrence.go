package task

import (
	"fmt"
	"slices"
	"time"

	"nvm-internal/internal/apperr"
)

// Frekuensi task berulang (revisi 19).
const (
	FreqDaily   = "daily"
	FreqWeekly  = "weekly"
	FreqMonthly = "monthly"
)

// Rule adalah aturan hari kemunculan sebuah seri. Semua tanggal berupa tanggal kantor (officeTZ) yang disimpan
// sebagai tengah malam UTC, sama dengan kolom date di database, supaya perbandingan tidak bergeser zona waktu.
type Rule struct {
	Freq        string
	SkipWeekend bool
	// Weekdays memakai hari ISO: 1 Senin sampai 7 Minggu.
	Weekdays []int
	MonthDay int
	Start    time.Time
	// End nol berarti tanpa batas.
	End time.Time
}

// Validate memeriksa isian aturan dari form. Pesannya untuk pengguna.
func (r Rule) Validate() error {
	switch r.Freq {
	case FreqDaily:
	case FreqWeekly:
		if len(r.Weekdays) == 0 {
			return apperr.Invalid("Pilih minimal satu hari untuk pengulangan mingguan.")
		}
		for _, d := range r.Weekdays {
			if d < 1 || d > 7 {
				return apperr.Invalid("Hari pengulangan mingguan tidak dikenal.")
			}
		}
	case FreqMonthly:
		if r.MonthDay < 1 || r.MonthDay > 31 {
			return apperr.Invalid("Tanggal pengulangan bulanan harus 1 sampai 31.")
		}
	default:
		return apperr.Invalid("Jenis pengulangan tidak dikenal.")
	}
	if !r.End.IsZero() && r.End.Before(r.Start) {
		return apperr.Invalid(fmt.Sprintf("Tanggal berhenti tidak boleh sebelum %s.", formatTime(r.Start)))
	}
	return nil
}

// OccursOn menjawab apakah seri ini punya kemunculan di tanggal d.
func (r Rule) OccursOn(d time.Time) bool {
	if d.Before(r.Start) || (!r.End.IsZero() && d.After(r.End)) {
		return false
	}
	switch r.Freq {
	case FreqDaily:
		wd := d.Weekday()
		return !r.SkipWeekend || (wd != time.Saturday && wd != time.Sunday)
	case FreqWeekly:
		return slices.Contains(r.Weekdays, isoWeekday(d))
	case FreqMonthly:
		// Keputusan 2: tanggal yang tidak ada di bulan itu (mis. 31 April) pindah ke hari terakhir bulan itu.
		return d.Day() == min(r.MonthDay, lastDay(d))
	}
	return false
}

// Next mengembalikan kemunculan pertama setelah tanggal after, atau false bila seri sudah habis.
// Dibatasi 400 hari: aturan yang sah selalu punya kemunculan dalam satu tahun lebih sedikit.
func (r Rule) Next(after time.Time) (time.Time, bool) {
	d := after.AddDate(0, 0, 1)
	if d.Before(r.Start) {
		d = r.Start
	}
	for range 400 {
		if !r.End.IsZero() && d.After(r.End) {
			return time.Time{}, false
		}
		if r.OccursOn(d) {
			return d, true
		}
		d = d.AddDate(0, 0, 1)
	}
	return time.Time{}, false
}

func isoWeekday(d time.Time) int {
	if d.Weekday() == time.Sunday {
		return 7
	}
	return int(d.Weekday())
}

func lastDay(d time.Time) int {
	return time.Date(d.Year(), d.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// officeDate mengubah waktu menjadi tanggal kantor (Asia/Jakarta) berbentuk tengah malam UTC.
func officeDate(now time.Time) time.Time {
	l := now.In(officeTZ)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}

func formatTime(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), monthNames[t.Month()-1], t.Year())
}

// OfficeTZ dipakai perintah dev `server occurrences --date` untuk membaca tanggal sebagai tanggal kantor.
func OfficeTZ() *time.Location { return officeTZ }
