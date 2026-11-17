package task

// Can adalah ringkasan hak pelaku atas satu task, dikirim ke frontend supaya tombol yang tidak boleh
// dipakai tidak ditampilkan. Backend tetap memeriksa ulang setiap aksi.
type Can struct {
	Edit            bool     `json:"edit"`
	Delete          bool     `json:"delete"`
	EditLabels      bool     `json:"edit_labels"`
	EditDeliverable bool     `json:"edit_deliverable"`
	AddSubtask      bool     `json:"add_subtask"`
	MoveTo          []Status `json:"move_to"`
}

func PermissionsFor(role Role, isPIC bool, status Status) Can {
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
	}
	for _, to := range []Status{Requested, OnProgress, Ready, Completed} {
		// Catatan diisi sembarang supaya aturan wajib-catatan tidak menyembunyikan tujuan yang sebenarnya boleh.
		if _, err := CheckTransition(role, isPIC, status, to, "catatan"); err == nil {
			c.MoveTo = append(c.MoveTo, to)
		}
	}
	return c
}
