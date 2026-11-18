package task

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// forbiddenCases adalah semua penolakan hak akses di package ini. subtask menandai aksi atas sub task.
func forbiddenCases() []struct {
	name    string
	err     error
	subtask bool
} {
	cases := []struct {
		name    string
		err     error
		subtask bool
	}{
		{"buat task", errCreateForbidden, false},
		{"ubah task", errUpdateForbidden, false},
		{"hapus task", errDeleteForbidden, false},
		{"label", errLabelsForbidden, false},
		{"deliverable", errDeliverableForbidden, false},
		{"hentikan pengulangan", errStopSeriesForbidden, false},
		{"unggah brief", attachmentForbidden("mengunggah", KindBrief, 0), false},
		{"hapus brief", attachmentForbidden("menghapus", KindBrief, 0), false},
		{"unggah deliverable", attachmentForbidden("mengunggah", KindDeliverable, 0), false},
		{"hapus deliverable", attachmentForbidden("menghapus", KindDeliverable, 0), false},
		{"tambah sub task", errAddSubtaskForbidden, true},
		{"ubah sub task", subtaskForbidden("mengubah"), true},
		{"hapus sub task", subtaskForbidden("menghapus"), true},
		{"sub task orang lain", ErrSubtaskLocked, true},
		{"centang sub task", ErrNotSubtaskPIC, true},
		{"unggah berkas sub task", attachmentForbidden("mengunggah", KindDeliverable, 7), true},
		{"hapus berkas sub task", attachmentForbidden("menghapus", KindBrief, 7), true},
	}
	for _, st := range []Status{Requested, OnProgress, Ready, Completed} {
		cases = append(cases, struct {
			name    string
			err     error
			subtask bool
		}{"pindah ke " + string(st), moveForbidden(st), false})
	}
	return cases
}

// Setiap jenis penolakan membawa pesannya sendiri: tetap terbaca sebagai ErrForbidden, pesannya unik, dan aksi
// yang bukan atas sub task tidak menyebut sub task (dulu semua penolakan memakai pesan centang sub task).
func TestForbiddenMessagesPerAction(t *testing.T) {
	seen := map[string]string{}
	for _, tc := range forbiddenCases() {
		var fe *ForbiddenError
		if !errors.Is(tc.err, ErrForbidden) || !errors.As(tc.err, &fe) {
			t.Errorf("%s: bukan ForbiddenError yang cocok dengan ErrForbidden", tc.name)
			continue
		}
		if prev, dup := seen[fe.Message]; dup {
			t.Errorf("%s: pesan sama dengan %s: %q", tc.name, prev, fe.Message)
		}
		seen[fe.Message] = tc.name
		if mentions := strings.Contains(strings.ToLower(fe.Message), "sub task"); mentions != tc.subtask {
			t.Errorf("%s: menyebut sub task %v, mau %v: %q", tc.name, mentions, tc.subtask, fe.Message)
		}
	}
	if got := moveForbidden(OnProgress).Error(); got != "Kamu tidak punya hak memindahkan task ini ke On Progress." {
		t.Errorf("pesan pindah ke On Progress: %q", got)
	}
}

// Penjaga: tidak ada lagi `return ErrForbidden` polos di kode package ini, supaya penolakan baru wajib memilih
// pesan aksinya sendiri lewat forbidden.go.
func TestNoBareErrForbidden(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	bare := regexp.MustCompile(`return\b[^\n]*\bErrForbidden\b`)
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range bare.FindAll(src, -1) {
			if !strings.Contains(string(m), "target == ErrForbidden") {
				t.Errorf("%s: penolakan tanpa pesan aksi: %s", f, m)
			}
		}
	}
}
