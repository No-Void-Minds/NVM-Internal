package task

import (
	"slices"
	"testing"
)

func TestPermissionsFor(t *testing.T) {
	cases := []struct {
		name   string
		role   Role
		isPIC  bool
		status Status
		edit   bool
		labels bool
		moveTo []Status
	}{
		{"Admin hanya baca", RoleAdmin, false, Ready, false, false, []Status{}},
		{"Staf bukan PIC hanya baca", RoleStaff, false, Requested, false, false, []Status{}},
		{"Staf PIC di Ready", RoleStaff, true, Ready, false, true, []Status{OnProgress}},
		{"Manajer di Ready", RoleManager, false, Ready, true, true, []Status{OnProgress, Completed}},
		{"Manajer di Completed", RoleManager, false, Completed, true, true, []Status{OnProgress}},
	}
	for _, tc := range cases {
		c := PermissionsFor(tc.role, tc.isPIC, tc.status)
		if c.Edit != tc.edit || c.EditLabels != tc.labels || !slices.Equal(c.MoveTo, tc.moveTo) {
			t.Errorf("%s: dapat %+v", tc.name, c)
		}
	}
}
