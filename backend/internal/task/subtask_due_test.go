package task

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"nvm-internal/internal/apperr"
)

func TestCheckSubtaskDue(t *testing.T) {
	day := func(s string) pgtype.Date {
		d, err := parseDate(s, "tanggal")
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	taskDue := day("2026-11-20")
	cases := []struct {
		name     string
		sub, due pgtype.Date
		ok       bool
	}{
		{"tanpa tenggat sub task", day(""), taskDue, true},
		{"task tanpa tenggat", day("2027-01-01"), day(""), true},
		{"sama dengan tenggat task", day("2026-11-20"), taskDue, true},
		{"sebelum tenggat task", day("2026-11-19"), taskDue, true},
		{"melewati tenggat task", day("2026-11-21"), taskDue, false},
	}
	for _, c := range cases {
		err := checkSubtaskDue(c.sub, c.due)
		if (err == nil) != c.ok {
			t.Errorf("%s: err %v, mau lolos %v", c.name, err, c.ok)
		}
	}
	var ve *apperr.Validation
	err := checkSubtaskDue(day("2026-11-21"), taskDue)
	if !errors.As(err, &ve) || ve.Message != "Deadline sub task tidak boleh melewati deadline task (20 November 2026)." {
		t.Fatalf("pesan galat: %v", err)
	}
	if err := numbered(1, err); err.Error() != "Sub task nomor 2: "+ve.Message {
		t.Fatalf("numbered: %q", err.Error())
	}
}
