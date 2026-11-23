package task

import "testing"

// Aturan centang sub task revisi 34: PIC sub task itu, PIC task mana pun, pembuat task (requested_by), dan Manajer.
// Admin tidak pernah, walau tercatat sebagai peminta task hasil impor Excel.
func TestCanToggle(t *testing.T) {
	staf := Actor{ID: 1, Role: RoleStaff}
	manager := Actor{ID: 2, Role: RoleManager}
	admin := Actor{ID: 3, Role: RoleAdmin}
	cases := []struct {
		name                            string
		actor                           Actor
		owner, taskPIC, requester, want bool
	}{
		{"PIC sub task", staf, true, false, false, true},
		{"PIC task, sub task milik orang lain", staf, false, true, false, true},
		{"pembuat task Staf", staf, false, false, true, true},
		{"Staf lain", staf, false, false, false, false},
		{"Manajer tanpa peran di task", manager, false, false, false, true},
		{"Admin peminta task", admin, false, false, true, false},
		{"Admin", admin, false, false, false, false},
	}
	for _, c := range cases {
		if got := canToggle(c.actor, c.owner, c.taskPIC, c.requester); got != c.want {
			t.Errorf("%s: %v, mau %v", c.name, got, c.want)
		}
	}
}
