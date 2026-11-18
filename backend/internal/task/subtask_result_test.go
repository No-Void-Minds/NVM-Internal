package task

import (
	"strings"
	"testing"
)

func TestCleanResultNote(t *testing.T) {
	cases := []struct {
		name, in, want string
		valid, ok      bool
	}{
		{"kosong jadi NULL", "", "", false, true},
		{"spasi saja jadi NULL", "  \n ", "", false, true},
		{"tepi dirapikan, baris baru di tengah dipertahankan", "  baris 1\nbaris 2  ", "baris 1\nbaris 2", true, true},
		{"tepat 2000 huruf", strings.Repeat("é", 2000), strings.Repeat("é", 2000), true, true},
		{"2001 huruf ditolak", strings.Repeat("a", 2001), "", false, false},
	}
	for _, c := range cases {
		got, err := cleanResultNote(c.in)
		if (err == nil) != c.ok || (c.ok && (got.Valid != c.valid || got.String != c.want)) {
			t.Errorf("%s: dapat %+v err %v", c.name, got, err)
		}
	}
}
