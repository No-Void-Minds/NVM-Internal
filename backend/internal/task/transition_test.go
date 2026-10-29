package task

import (
	"errors"
	"fmt"
	"testing"
)

var (
	allStatuses = []Status{Requested, OnProgress, Ready, Completed}
	allRoles    = []Role{RoleAdmin, RoleManager, RoleStaff}
)

// allowed adalah tabel aturan dari SRS (BR-02, BR-03, FR-TASK-06, 07, 19), ditulis ulang secara eksplisit
// supaya test tidak sekadar menyalin isi transition.go. Format: peran, PIC atau bukan, asal, tujuan.
var allowed = map[string]bool{
	"staff,pic,requested,on_progress": true,
	"staff,pic,on_progress,ready":     true,
	"staff,pic,ready,on_progress":     true,

	"manager,pic,requested,on_progress":   true,
	"manager,pic,on_progress,ready":       true,
	"manager,pic,ready,on_progress":       true,
	"manager,pic,ready,completed":         true,
	"manager,pic,completed,on_progress":   true,
	"manager,bukan,requested,on_progress": true,
	"manager,bukan,on_progress,ready":     true,
	"manager,bukan,ready,on_progress":     true,
	"manager,bukan,ready,completed":       true,
	"manager,bukan,completed,on_progress": true,
}

// TestEveryTransition mencoba seluruh 96 kombinasi peran, PIC, asal, dan tujuan.
func TestEveryTransition(t *testing.T) {
	for _, role := range allRoles {
		for _, isPIC := range []bool{true, false} {
			pic := "bukan"
			if isPIC {
				pic = "pic"
			}
			for _, from := range allStatuses {
				for _, to := range allStatuses {
					key := fmt.Sprintf("%s,%s,%s,%s", role, pic, from, to)
					_, err := CheckTransition(role, isPIC, from, to, "catatan")
					if got := err == nil; got != allowed[key] {
						t.Errorf("%s: boleh=%v, mau %v (err %v)", key, got, allowed[key], err)
					}
				}
			}
		}
	}
}

func TestUnknownTransitionVersusForbidden(t *testing.T) {
	if _, err := CheckTransition(RoleManager, true, Requested, Completed, ""); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("lompat Requested ke Completed: err %v, mau ErrInvalidTransition", err)
	}
	if _, err := CheckTransition(RoleStaff, true, Ready, Completed, ""); !errors.Is(err, ErrTransitionForbidden) {
		t.Fatalf("Staf menutup task: err %v, mau ErrTransitionForbidden", err)
	}
	if _, err := CheckTransition(RoleStaff, false, Requested, OnProgress, ""); !errors.Is(err, ErrTransitionForbidden) {
		t.Fatalf("Staf memindahkan task orang lain: err %v, mau ErrTransitionForbidden", err)
	}
	if _, err := CheckTransition(RoleAdmin, true, Requested, OnProgress, ""); !errors.Is(err, ErrTransitionForbidden) {
		t.Fatalf("Admin memindahkan task: err %v, mau ErrTransitionForbidden", err)
	}
}

func TestRevisionNeedsNote(t *testing.T) {
	for _, role := range []Role{RoleStaff, RoleManager} {
		if _, err := CheckTransition(role, true, Ready, OnProgress, "   "); !errors.Is(err, ErrNoteRequired) {
			t.Fatalf("%s mengembalikan Ready tanpa catatan: err %v, mau ErrNoteRequired", role, err)
		}
	}
	note, err := CheckTransition(RoleStaff, true, Ready, OnProgress, "  Warna logo kurang kontras  ")
	if err != nil || note != "Warna logo kurang kontras" {
		t.Fatalf("catatan harus dirapikan dan diterima, dapat %q err %v", note, err)
	}
	if _, err := CheckTransition(RoleManager, false, Completed, OnProgress, ""); err != nil {
		t.Fatalf("buka kembali tidak wajib catatan: %v", err)
	}
	long := make([]rune, MaxNoteLength+1)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := CheckTransition(RoleStaff, true, Ready, OnProgress, string(long)); !errors.Is(err, ErrNoteTooLong) {
		t.Fatalf("catatan terlalu panjang: err %v, mau ErrNoteTooLong", err)
	}
}
