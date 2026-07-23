package middleware

import "strings"

var rolePermissions = map[string][]string{
	"super-admin": {"*"},
	"admin":       {"*"},
	"supervisor": {
		"dashboard.view", "search.view", "profile.*", "conversations.*",
		"contacts.*", "companies.*", "teams.view", "teams.members.manage",
		"funnels.*", "deals.*", "metrics.view", "glpi.*",
		"announcements.view", "quick_replies.view", "users.view",
	},
	"agent": {
		"dashboard.view", "search.view", "profile.*", "conversations.own",
		"conversations.create",
		"contacts.view", "contacts.edit", "companies.view", "teams.view",
		"funnels.view", "deals.view", "glpi.*", "announcements.view",
		"quick_replies.view", "users.view",
	},
	"sales": {
		"dashboard.view", "search.view", "profile.*", "conversations.own",
		"conversations.create",
		"contacts.*", "companies.*", "teams.view", "funnels.*", "deals.*",
		"announcements.view", "quick_replies.view", "users.view",
	},
	"marketing": {
		"dashboard.view", "search.view", "profile.*", "contacts.view",
		"campaigns.*", "metrics.campaigns", "announcements.view",
	},
	"finance": {
		"dashboard.view", "search.view", "profile.*", "metrics.view",
		"deals.view", "funnels.view", "announcements.view",
	},
}

func RoleHasPermission(role, required string) bool {
	for _, granted := range rolePermissions[strings.ToLower(role)] {
		if granted == "*" || granted == required {
			return true
		}
		if strings.HasSuffix(granted, ".*") &&
			strings.HasPrefix(required, strings.TrimSuffix(granted, "*")) {
			return true
		}
	}
	return false
}
