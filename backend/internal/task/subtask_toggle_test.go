package task

import "testing"

// Aturan centang sub task (Sprint 11g, ditegaskan revisi 13): PIC sub task, pembuatnya, PIC task untuk
// sub task tanpa PIC, dan Manajer kecuali sub task itu dipegang Manajer lain.
func TestCanToggle(t *testing.T) {
	staf := Actor{ID: 1, Role: RoleStaff}
	manager := Actor{ID: 2, Role: RoleManager}
	cases := []struct {
		name                                                      string
		actor                                                     Actor
		owner, creator, ownerless, taskPIC, otherManagerPIC, want bool
	}{
		{"PIC sub task", staf, true, false, false, false, false, true},
		{"pembuat Staf", staf, false, true, false, false, false, true},
		{"PIC task, sub task tanpa PIC", staf, false, false, true, true, false, true},
		{"PIC task, sub task milik orang lain", staf, false, false, false, true, false, false},
		{"Staf lain", staf, false, false, false, false, false, false},
		{"Manajer, PIC Staf", manager, false, false, false, false, false, true},
		{"Manajer, PIC Manajer lain", manager, false, false, false, false, true, false},
		{"Manajer pembuat, PIC Manajer lain", manager, false, true, false, false, true, true},
		{"Manajer PIC sub task", manager, true, false, false, false, false, true},
	}
	for _, c := range cases {
		if got := canToggle(c.actor, c.owner, c.creator, c.ownerless, c.taskPIC, c.otherManagerPIC); got != c.want {
			t.Errorf("%s: %v, mau %v", c.name, got, c.want)
		}
	}
}
