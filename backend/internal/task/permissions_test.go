package task

import (
	"slices"
	"testing"

	"nvm-internal/internal/db/sqlc"
)

// Papan dan Arsip tidak menghitung PIC sub task untuk Manajer (mySubtasks), dengan anggapan isSubtaskPIC
// tidak mengubah hak Manajer. Test ini mengunci anggapan itu: Manajer yang hanya PIC sub task (bukan pembuat,
// bukan PIC task) mendapat move_to yang sama persis dengan Manajer lain di setiap status.
func TestManagerMoveToIgnoresSubtaskPIC(t *testing.T) {
	for _, st := range []Status{Requested, OnProgress, Ready, Completed} {
		plain := PermissionsFor(RoleManager, false, false, st).MoveTo
		subPIC := PermissionsFor(RoleManager, false, true, st).MoveTo
		if !slices.Equal(plain, subPIC) || len(plain) == 0 {
			t.Errorf("%s: Manajer %v, Manajer PIC sub task %v, mau sama dan tidak kosong", st, plain, subPIC)
		}
	}
}

func TestPermissionsFor(t *testing.T) {
	cases := []struct {
		name   string
		role   Role
		isPIC  bool
		isSub  bool
		status Status
		edit   bool
		labels bool
		moveTo []Status
	}{
		{"Admin hanya baca", RoleAdmin, false, false, Ready, false, false, []Status{}},
		{"Staf bukan PIC hanya baca", RoleStaff, false, false, Requested, false, false, []Status{}},
		{"Staf PIC di Ready", RoleStaff, true, false, Ready, false, true, []Status{OnProgress}},
		{"Manajer di Ready", RoleManager, false, false, Ready, true, true, []Status{OnProgress, Completed}},
		{"Manajer di Completed", RoleManager, false, false, Completed, true, true, []Status{OnProgress}},
		// Revisi 17: PIC sub task hanya mendapat perpindahan, hak lain tidak.
		{"PIC sub task di Requested", RoleStaff, false, true, Requested, false, false, []Status{OnProgress}},
		{"PIC sub task di On Progress", RoleStaff, false, true, OnProgress, false, false, []Status{Ready}},
		{"PIC sub task di Ready", RoleStaff, false, true, Ready, false, false, []Status{OnProgress}},
		{"PIC sub task di Completed", RoleStaff, false, true, Completed, false, false, []Status{}},
	}
	for _, tc := range cases {
		c := PermissionsFor(tc.role, tc.isPIC, tc.isSub, tc.status)
		if c.Edit != tc.edit || c.EditLabels != tc.labels || !slices.Equal(c.MoveTo, tc.moveTo) ||
			c.AddSubtask && tc.isSub && !tc.isPIC || c.SubtaskPIC != tc.isSub {
			t.Errorf("%s: dapat %+v", tc.name, c)
		}
	}
}

// Revisi 22: jalur revisi PIC sub task butuh minimal satu sub task miliknya yang tercentang. Tanpa itu On Progress
// tidak ditawarkan, supaya move_to cocok dengan yang diterima Move. Daftar pilihan hanya berisi yang tercentang.
func TestMoveOptionsSubtaskRevision(t *testing.T) {
	mine := []sqlc.ListMySubtasksRow{{ID: 1, Title: "Naskah", Done: true}, {ID: 2, Title: "Desain", Done: false}}
	can, choices := MoveOptions(RoleStaff, false, mine, Ready)
	if !slices.Equal(can.MoveTo, []Status{OnProgress}) || len(choices) != 1 || choices[0].ID != 1 {
		t.Fatalf("satu tercentang: move_to %v, pilihan %v", can.MoveTo, choices)
	}
	can, choices = MoveOptions(RoleStaff, false, mine[1:], Ready)
	if len(can.MoveTo) != 0 || len(choices) != 0 {
		t.Fatalf("tanpa yang tercentang: move_to %v, pilihan %v, mau kosong", can.MoveTo, choices)
	}
	// PIC task dan Manajer tidak mendapat daftar pilihan; perilakunya tidak berubah.
	for _, tc := range []struct {
		role  Role
		isPIC bool
	}{{RoleStaff, true}, {RoleManager, false}} {
		can, choices := MoveOptions(tc.role, tc.isPIC, mine, Ready)
		if len(choices) != 0 || !slices.Contains(can.MoveTo, OnProgress) {
			t.Errorf("%s pic=%v: move_to %v, pilihan %v", tc.role, tc.isPIC, can.MoveTo, choices)
		}
	}
	// Di status lain daftar pilihan selalu kosong.
	if _, choices := MoveOptions(RoleStaff, false, mine, OnProgress); len(choices) != 0 {
		t.Fatalf("On Progress: pilihan %v, mau kosong", choices)
	}
}
