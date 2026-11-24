package timeline

import (
	"context"
	"testing"
)

// Aturan tanggal dan judul launching atau produksi tidak menyentuh database, jadi bisa diuji tanpa TEST_DATABASE_URL.
func TestCleanRentang(t *testing.T) {
	s := &Service{}
	cases := []struct {
		name    string
		in      EventInput
		ok      bool
		singleD bool
	}{
		{"satu hari", EventInput{Category: "launching", Title: "Rilis", StartDate: "2026-11-10"}, true, true},
		{"sampai sama dengan mulai", EventInput{Category: "launching", Title: "Rilis", StartDate: "2026-11-10", EndDate: "2026-11-10"}, true, true},
		{"62 hari", EventInput{Category: "produksi", Title: "Syuting", StartDate: "2026-11-01", EndDate: "2027-01-01"}, true, false},
		{"63 hari", EventInput{Category: "produksi", Title: "Syuting", StartDate: "2026-11-01", EndDate: "2027-01-02"}, false, false},
		{"mundur", EventInput{Category: "produksi", Title: "Syuting", StartDate: "2026-11-10", EndDate: "2026-11-09"}, false, false},
		{"tanpa judul", EventInput{Category: "produksi", Title: "   ", StartDate: "2026-11-10"}, false, false},
		{"tanpa mulai", EventInput{Category: "produksi", Title: "Syuting"}, false, false},
		{"jenis salah", EventInput{Category: "rapat", Title: "Syuting", StartDate: "2026-11-10"}, false, false},
	}
	for _, c := range cases {
		got, err := s.clean(context.Background(), nil, c.in)
		if (err == nil) != c.ok {
			t.Errorf("%s: err %v, mau lolos %v", c.name, err, c.ok)
			continue
		}
		if c.ok && got.end.Valid == c.singleD {
			t.Errorf("%s: end_date terisi %v", c.name, got.end.Valid)
		}
	}
}

func TestMonthRange(t *testing.T) {
	if from, to, err := MonthRange("2026-02"); err != nil || from.Day() != 1 || to.Day() != 28 {
		t.Errorf("Februari 2026: %v %v %v", from, to, err)
	}
	if _, _, err := MonthRange("2026-13"); err == nil {
		t.Error("bulan 13 harus ditolak")
	}
}
