package whiteboard

import (
	"strings"
	"testing"
)

func TestSnippet(t *testing.T) {
	long := strings.Repeat("awal ", 20) + "Rencana Kampanye Ramadan" + strings.Repeat(" akhir", 20)
	got := snippet(long, "kampanye")
	if !strings.Contains(got, "Kampanye") || !strings.HasPrefix(got, "...") || !strings.HasSuffix(got, "...") {
		t.Fatalf("potongan tengah: %q", got)
	}
	if n := len([]rune(strings.Trim(got, "."))); n > 80 {
		t.Fatalf("potongan %d karakter", n)
	}
	if got := snippet("Pendek  sekali\nbaris", "zzz"); got != "Pendek sekali baris" {
		t.Fatalf("teks pendek: %q", got)
	}
	// Tidak ketemu (cocok lewat judul pratinjau di tempat lain): awal teks.
	if got := snippet(long, "tidak ada"); !strings.HasPrefix(got, "awal") || !strings.HasSuffix(got, "...") {
		t.Fatalf("tanpa kecocokan: %q", got)
	}
	// Kecocokan di ujung tetap menampilkan 80 karakter terakhir.
	if got := snippet(long+" Penutup", "penutup"); !strings.HasSuffix(got, "Penutup") || strings.HasSuffix(got, "...") {
		t.Fatalf("kecocokan di ujung: %q", got)
	}
	if got := snippet(strings.Repeat("é", 100)+" Ü", "ü"); !strings.HasSuffix(got, "Ü") {
		t.Fatalf("huruf non-ASCII: %q", got)
	}
}

func TestSearchTerm(t *testing.T) {
	for _, bad := range []string{"", " a ", strings.Repeat("x", 101)} {
		if _, err := searchTerm(bad); !invalid(err) {
			t.Fatalf("%q: %v", bad, err)
		}
	}
	if got, _ := searchTerm(` 50%_a\ `); got != `50\%\_a\\` {
		t.Fatalf("escape: %q", got)
	}
}
