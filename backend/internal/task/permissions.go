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
		AddSubtask: contributor && subtaskWindowOpen(status),
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

// SubtaskPICTasks menjawab task mana di antara ids yang sub task-nya dipegang userID, dalam satu query,
// untuk menghitung move_to papan dan Arsip (revisi 17).
func (s *Service) SubtaskPICTasks(ctx context.Context, userID int64, ids []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := sqlc.New(s.pool).ListTasksWithSubtaskPIC(ctx, sqlc.ListTasksWithSubtaskPICParams{TaskIds: ids, UserID: userID})
	if err != nil {
		return nil, err
	}
	for _, id := range rows {
		out[id] = true
	}
	return out, nil
}
