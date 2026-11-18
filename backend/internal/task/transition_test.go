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

// allowed adalah tabel aturan dari SRS (BR-02, BR-03, FR-TASK-06, 07, 19) dan revisi 17, ditulis ulang secara
// eksplisit supaya test tidak sekadar menyalin isi transition.go. Format: peran, hubungan, asal, tujuan.
// Hubungan: pic (PIC task), subpic (hanya PIC sub task), bukan (tidak keduanya).
var allowed = map[string]bool{
	"staff,pic,requested,on_progress": true,
	"staff,pic,on_progress,ready":     true,
	"staff,pic,ready,on_progress":     true,

	// Revisi 17: PIC sub task boleh memulai dan menandai Ready. Revisi 22: juga mengembalikan Ready ke On Progress.
	"staff,subpic,requested,on_progress": true,
	"staff,subpic,on_progress,ready":     true,
	"staff,subpic,ready,on_progress":     true,

	"manager,subpic,requested,on_progress": true,
	"manager,subpic,on_progress,ready":     true,
	"manager,subpic,ready,on_progress":     true,
	"manager,subpic,ready,completed":       true,
	"manager,subpic,completed,on_progress": true,

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

// TestEveryTransition mencoba seluruh 144 kombinasi peran, hubungan, asal, dan tujuan.
func TestEveryTransition(t *testing.T) {
	relations := []struct {
		name         string
		isPIC, isSub bool
	}{{"pic", true, false}, {"subpic", false, true}, {"bukan", false, false}}
	for _, role := range allRoles {
		for _, rel := range relations {
			for _, from := range allStatuses {
				for _, to := range allStatuses {
					key := fmt.Sprintf("%s,%s,%s,%s", role, rel.name, from, to)
					_, err := CheckTransition(role, rel.isPIC, rel.isSub, from, to, "catatan")
					if got := err == nil; got != allowed[key] {
						t.Errorf("%s: boleh=%v, mau %v (err %v)", key, got, allowed[key], err)
					}
				}
			}
		}
	}
}

// PIC task yang juga PIC sub task memakai jalur PIC task: catatan tetap wajib. Hanya PIC sub task saja yang
// memakai jalur revisi sub task dengan alasan opsional (revisi 22), dan batas 500 karakter tetap berlaku.
func TestTaskPICAndSubtaskPIC(t *testing.T) {
	if _, err := CheckTransition(RoleStaff, true, true, Ready, OnProgress, ""); !errors.Is(err, ErrNoteRequired) {
		t.Fatalf("PIC task sekaligus PIC sub task tanpa catatan: err %v, mau ErrNoteRequired", err)
	}
	if IsSubtaskRevision(RoleStaff, true, true, Ready, OnProgress) || IsSubtaskRevision(RoleManager, false, true, Ready, OnProgress) {
		t.Fatal("PIC task dan Manajer tidak boleh masuk jalur revisi sub task")
	}
	if !IsSubtaskRevision(RoleStaff, false, true, Ready, OnProgress) {
		t.Fatal("PIC sub task saja harus masuk jalur revisi sub task")
	}
	if _, err := CheckTransition(RoleStaff, false, true, Ready, OnProgress, ""); err != nil {
		t.Fatalf("PIC sub task saja tanpa alasan: %v", err)
	}
	long := make([]rune, MaxNoteLength+1)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := CheckTransition(RoleStaff, false, true, Ready, OnProgress, string(long)); !errors.Is(err, ErrNoteTooLong) {
		t.Fatalf("alasan 501 karakter: err %v, mau ErrNoteTooLong", err)
	}
}

func TestUnknownTransitionVersusForbidden(t *testing.T) {
	if _, err := CheckTransition(RoleManager, true, false, Requested, Completed, ""); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("lompat Requested ke Completed: err %v, mau ErrInvalidTransition", err)
	}
	if _, err := CheckTransition(RoleStaff, true, false, Ready, Completed, ""); !errors.Is(err, ErrTransitionForbidden) {
		t.Fatalf("Staf menutup task: err %v, mau ErrTransitionForbidden", err)
	}
	if _, err := CheckTransition(RoleStaff, false, false, Requested, OnProgress, ""); !errors.Is(err, ErrTransitionForbidden) {
		t.Fatalf("Staf memindahkan task orang lain: err %v, mau ErrTransitionForbidden", err)
	}
	if _, err := CheckTransition(RoleAdmin, true, false, Requested, OnProgress, ""); !errors.Is(err, ErrTransitionForbidden) {
		t.Fatalf("Admin memindahkan task: err %v, mau ErrTransitionForbidden", err)
	}
}

func TestRevisionNeedsNote(t *testing.T) {
	for _, role := range []Role{RoleStaff, RoleManager} {
		if _, err := CheckTransition(role, true, false, Ready, OnProgress, "   "); !errors.Is(err, ErrNoteRequired) {
			t.Fatalf("%s mengembalikan Ready tanpa catatan: err %v, mau ErrNoteRequired", role, err)
		}
	}
	note, err := CheckTransition(RoleStaff, true, false, Ready, OnProgress, "  Warna logo kurang kontras  ")
	if err != nil || note != "Warna logo kurang kontras" {
		t.Fatalf("catatan harus dirapikan dan diterima, dapat %q err %v", note, err)
	}
	if _, err := CheckTransition(RoleManager, false, false, Completed, OnProgress, ""); err != nil {
		t.Fatalf("buka kembali tidak wajib catatan: %v", err)
	}
	long := make([]rune, MaxNoteLength+1)
	for i := range long {
		long[i] = 'a'
	}
	if _, err := CheckTransition(RoleStaff, true, false, Ready, OnProgress, string(long)); !errors.Is(err, ErrNoteTooLong) {
		t.Fatalf("catatan terlalu panjang: err %v, mau ErrNoteTooLong", err)
	}
}
