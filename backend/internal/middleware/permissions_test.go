package middleware

import "testing"

func TestRoleHasPermission(t *testing.T) {
	tests := []struct {
		role       string
		permission string
		want       bool
	}{
		{"admin", "anything.delete", true},
		{"agent", "conversations.own", true},
		{"agent", "contacts.edit", true},
		{"agent", "contacts.delete", false},
		{"agent", "campaigns.view", false},
		{"supervisor", "conversations.create", true},
		{"supervisor", "channels.manage", false},
		{"marketing", "campaigns.start", true},
		{"marketing", "contacts.edit", false},
	}
	for _, test := range tests {
		t.Run(test.role+"/"+test.permission, func(t *testing.T) {
			if got := RoleHasPermission(test.role, test.permission); got != test.want {
				t.Fatalf("RoleHasPermission() = %v, want %v", got, test.want)
			}
		})
	}
}
