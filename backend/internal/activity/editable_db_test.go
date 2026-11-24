package activity

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

func TestEditableDays(t *testing.T) {
	cases := map[string][]string{
		"2026-11-23": {"2026-11-23", "2026-11-20", "2026-11-19"}, // Senin: Jumat dan Kamis
		"2026-11-24": {"2026-11-24", "2026-11-23", "2026-11-20"}, // Selasa: Senin dan Jumat
		"2026-11-18": {"2026-11-18", "2026-11-17", "2026-11-16"}, // Rabu
		"2026-11-22": {"2026-11-22", "2026-11-20", "2026-11-19"}, // Minggu: hari ini tetap boleh
		"bukan":      {},
	}
	for today, want := range cases {
		if got := EditableDays(today); !slices.Equal(got, want) {
			t.Errorf("%s: %v, mau %v", today, got, want)
		}
	}
}

// Revisi 37 dengan jam Senin 23 November 10.00 WIB: Staf mengisi hari ini, Jumat, dan Kamis, termasuk lampiran
// dan rantai jam; Rabu pekan lalu dan besok ditolak; Manajer bebas; Admin ditolak.
func TestStafMengisiDuaHariKerjaKeBelakang(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	f.svc.now = func() time.Time { return time.Date(2026, 11, 23, 3, 0, 0, 0, time.UTC) }

	for _, d := range []string{"2026-11-23", "2026-11-20", "2026-11-19"} {
		if err := f.svc.Save(ctx, f.dina, SaveInput{UserID: f.dina.ID, Date: d, Slot: 1, Activity: "Isi " + d}); err != nil {
			t.Fatalf("Staf mengisi %s: %v", d, err)
		}
		if got := f.cell(t, f.dina.ID, d, 1).Activity; got != "Isi "+d {
			t.Fatalf("isian %s tersimpan %q", d, got)
		}
		if _, err := f.upload(t, f.dina, f.dina.ID, d, 1, "bukti.png", pngBytes()); err != nil {
			t.Fatalf("lampiran %s: %v", d, err)
		}
	}
	for _, d := range []string{"2026-11-18", "2026-11-24"} { // Rabu pekan lalu, besok
		if err := f.svc.Save(ctx, f.dina, SaveInput{UserID: f.dina.ID, Date: d, Slot: 1, Activity: "x"}); !errors.Is(err, ErrForbidden) {
			t.Fatalf("Staf mengisi %s: err %v, mau ErrForbidden", d, err)
		}
		if _, err := f.upload(t, f.dina, f.dina.ID, d, 1, "bukti.png", pngBytes()); !errors.Is(err, ErrForbidden) {
			t.Fatalf("lampiran %s: err %v, mau ErrForbidden", d, err)
		}
	}
	if err := f.svc.Save(ctx, f.manajer, SaveInput{UserID: f.dina.ID, Date: "2026-10-01", Slot: 1, Activity: "Manajer"}); err != nil {
		t.Fatalf("Manajer mengisi tanggal lama: %v", err)
	}
	if err := f.svc.Save(ctx, f.admin, SaveInput{UserID: f.dina.ID, Date: "2026-11-23", Slot: 2, Activity: "x"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("Admin: err %v, mau ErrForbidden", err)
	}

	// Rantai jam di Jumat: jam 2 kosong disambung ke jam 1, lalu dilepas.
	jumat := "2026-11-20"
	if err := f.svc.Link(ctx, f.dina, LinkInput{UserID: f.dina.ID, Date: jumat, Slot: 2, Linked: true}); err != nil {
		t.Fatalf("rantai di Jumat: %v", err)
	}
	if c := f.cell(t, f.dina.ID, jumat, 2); !c.Linked {
		t.Fatal("jam 2 Jumat harus tersambung")
	}
	if err := f.svc.Link(ctx, f.dina, LinkInput{UserID: f.dina.ID, Date: "2026-11-18", Slot: 2, Linked: true}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("rantai di luar rentang: err %v, mau ErrForbidden", err)
	}

	mine, err := f.svc.Mine(ctx, f.dina.ID, jumat)
	if err != nil {
		t.Fatal(err)
	}
	if mine.Today != "2026-11-23" || !slices.Equal(mine.EditableDates, []string{"2026-11-23", "2026-11-20", "2026-11-19"}) {
		t.Fatalf("Mine today %s editable %v", mine.Today, mine.EditableDates)
	}
}
