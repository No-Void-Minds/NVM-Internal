package task

import (
	"context"

	"nvm-internal/internal/db/sqlc"
)

// Can adalah ringkasan hak pelaku atas satu task, dikirim ke frontend supaya tombol yang tidak boleh
// dipakai tidak ditampilkan. Backend tetap memeriksa ulang setiap aksi.
type Can struct {
	Edit            bool     `json:"edit"`
	Delete          bool     `json:"delete"`
	EditLabels      bool     `json:"edit_labels"`
	EditDeliverable bool     `json:"edit_deliverable"`
	AddSubtask      bool     `json:"add_subtask"`
	MoveTo          []Status `json:"move_to"`
	// SubtaskPIC: pelaku memegang salah satu sub task task ini (revisi 17). Hanya menentukan teks di detail;
	// haknya sendiri sudah tercermin di MoveTo.
	SubtaskPIC bool `json:"subtask_pic"`
}

// PermissionsFor menghitung hak pelaku. isSubtaskPIC hanya menambah perpindahan status (revisi 17);
// hak lain tetap milik PIC task dan Manajer.
func PermissionsFor(role Role, isPIC, isSubtaskPIC bool, status Status) Can {
	manager := role == RoleManager
	contributor := manager || (role == RoleStaff && isPIC)
	c := Can{
		Edit:            manager,
		Delete:          manager,
		EditLabels:      contributor,
		EditDeliverable: contributor,
		// Revisi 6: PIC menyusun sendiri sub task yang ia kerjakan, Manajer boleh ikut menambah.
		AddSubtask: contributor && SubtaskWindowOpen(status),
		MoveTo:     []Status{},
		SubtaskPIC: isSubtaskPIC,
	}
	for _, to := range []Status{Requested, OnProgress, Ready, Completed} {
		// Catatan diisi sembarang supaya aturan wajib-catatan tidak menyembunyikan tujuan yang sebenarnya boleh.
		if _, err := CheckTransition(role, isPIC, isSubtaskPIC, status, to, "catatan"); err == nil {
			c.MoveTo = append(c.MoveTo, to)
		}
	}
	return c
}

// SubtaskRef menyebut satu sub task yang bisa dipilih untuk dibuka lagi (revisi 22).
type SubtaskRef struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

// MoveOptions menghitung hak dari sub task yang dipegang pelaku (mine), sekaligus daftar sub task yang boleh
// dibuka lagi bila pelaku memakai jalur revisi PIC sub task (revisi 22). Jalur itu butuh minimal satu sub task
// miliknya yang tercentang; tanpa itu On Progress dibuang dari MoveTo, supaya move_to cocok dengan yang
// diterima Move.
func MoveOptions(role Role, isPIC bool, mine []sqlc.ListMySubtasksRow, status Status) (Can, []SubtaskRef) {
	can := PermissionsFor(role, isPIC, len(mine) > 0, status)
	choices := []SubtaskRef{}
	if !IsSubtaskRevision(role, isPIC, len(mine) > 0, status, OnProgress) {
		return can, choices
	}
	for _, m := range mine {
		if m.Done {
			choices = append(choices, SubtaskRef{ID: m.ID, Title: m.Title})
		}
	}
	if len(choices) == 0 {
		kept := []Status{}
		for _, to := range can.MoveTo {
			if to != OnProgress {
				kept = append(kept, to)
			}
		}
		can.MoveTo = kept
	}
	return can, choices
}

// MySubtasks mengambil sub task yang dipegang userID di task-task itu dalam satu query, untuk move_to papan dan
// Arsip (revisi 17 dan 22).
func (s *Service) MySubtasks(ctx context.Context, userID int64, ids []string) (map[string][]sqlc.ListMySubtasksRow, error) {
	out := map[string][]sqlc.ListMySubtasksRow{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := sqlc.New(s.pool).ListMySubtasks(ctx, sqlc.ListMySubtasksParams{TaskIds: ids, UserID: userID})
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.TaskID] = append(out[r.TaskID], r)
	}
	return out, nil
}
