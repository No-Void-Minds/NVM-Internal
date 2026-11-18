package task

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// occurrences mengumpulkan tanggal kemunculan di rentang [from, to].
func occurrences(r Rule, from, to string) []string {
	var out []string
	for d := day(from); !d.After(day(to)); d = d.AddDate(0, 0, 1) {
		if r.OccursOn(d) {
			out = append(out, d.Format("2006-01-02"))
		}
	}
	return out
}

func sameDays(t *testing.T, name string, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: dapat %v, mau %v", name, got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s: dapat %v, mau %v", name, got, want)
			return
		}
	}
}

func TestRuleDaily(t *testing.T) {
	// 2026-11-20 Jumat, 21 Sabtu, 22 Minggu, 23 Senin.
	r := Rule{Freq: FreqDaily, Start: day("2026-11-20")}
	sameDays(t, "harian", occurrences(r, "2026-11-19", "2026-11-23"), "2026-11-20", "2026-11-21", "2026-11-22", "2026-11-23")
	r.SkipWeekend = true
	sameDays(t, "harian lewati akhir pekan", occurrences(r, "2026-11-19", "2026-11-23"), "2026-11-20", "2026-11-23")
	if next, ok := r.Next(day("2026-11-20")); !ok || next != day("2026-11-23") {
		t.Errorf("berikutnya setelah Jumat dengan lewati akhir pekan: %v %v, mau Senin 23", next, ok)
	}
}

func TestRuleWeekly(t *testing.T) {
	// Senin dan Kamis.
	r := Rule{Freq: FreqWeekly, Weekdays: []int{1, 4}, Start: day("2026-11-16")}
	sameDays(t, "mingguan Senin dan Kamis", occurrences(r, "2026-11-16", "2026-11-29"),
		"2026-11-16", "2026-11-19", "2026-11-23", "2026-11-26")
	// Minggu memakai angka ISO 7.
	r = Rule{Freq: FreqWeekly, Weekdays: []int{7}, Start: day("2026-11-16")}
	sameDays(t, "mingguan Minggu", occurrences(r, "2026-11-16", "2026-11-29"), "2026-11-22", "2026-11-29")
}

func TestRuleMonthly(t *testing.T) {
	r := Rule{Freq: FreqMonthly, MonthDay: 31, Start: day("2026-01-01")}
	// Februari 2026 (bukan kabisat) ke 28, April ke 30.
	sameDays(t, "tanggal 31", occurrences(r, "2026-01-01", "2026-05-31"),
		"2026-01-31", "2026-02-28", "2026-03-31", "2026-04-30", "2026-05-31")
	// Tahun kabisat: tanggal 30 di Februari 2028 jatuh ke 29.
	r = Rule{Freq: FreqMonthly, MonthDay: 30, Start: day("2028-01-01")}
	sameDays(t, "kabisat", occurrences(r, "2028-02-01", "2028-03-31"), "2028-02-29", "2028-03-30")
	// Tanggal 29 di Februari bukan kabisat jatuh ke 28, dan tidak muncul dua kali.
	r = Rule{Freq: FreqMonthly, MonthDay: 29, Start: day("2027-01-01")}
	sameDays(t, "29 Februari bukan kabisat", occurrences(r, "2027-02-01", "2027-03-31"), "2027-02-28", "2027-03-29")
}

func TestRuleStartEnd(t *testing.T) {
	r := Rule{Freq: FreqDaily, Start: day("2026-11-20"), End: day("2026-11-22")}
	sameDays(t, "batas mulai dan berhenti", occurrences(r, "2026-11-18", "2026-11-25"), "2026-11-20", "2026-11-21", "2026-11-22")
	if _, ok := r.Next(day("2026-11-22")); ok {
		t.Error("tidak boleh ada kemunculan setelah end_date")
	}
	if next, ok := r.Next(day("2026-11-01")); !ok || next != day("2026-11-20") {
		t.Errorf("kemunculan pertama sebelum start: %v %v", next, ok)
	}
}

func TestRuleValidate(t *testing.T) {
	bad := []Rule{
		{Freq: "yearly", Start: day("2026-11-20")},
		{Freq: FreqWeekly, Start: day("2026-11-20")},
		{Freq: FreqWeekly, Weekdays: []int{0}, Start: day("2026-11-20")},
		{Freq: FreqMonthly, MonthDay: 32, Start: day("2026-11-20")},
		{Freq: FreqDaily, Start: day("2026-11-20"), End: day("2026-11-19")},
	}
	for i, r := range bad {
		if r.Validate() == nil {
			t.Errorf("aturan %d harus ditolak: %+v", i, r)
		}
	}
	if err := (Rule{Freq: FreqMonthly, MonthDay: 31, Start: day("2026-11-20")}).Validate(); err != nil {
		t.Errorf("bulanan 31 harus sah: %v", err)
	}
}

func TestOfficeDate(t *testing.T) {
	// 18.30 UTC tanggal 22 sudah tanggal 23 di Jakarta (UTC+7).
	now := time.Date(2026, 11, 22, 18, 30, 0, 0, time.UTC)
	if got := officeDate(now); got != day("2026-11-23") {
		t.Errorf("officeDate: %v, mau 2026-11-23", got)
	}
}
