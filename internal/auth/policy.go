package auth

import "wesmk/internal/model"

// PrivilegedRoles have access to accounts, money, or school-wide data. They
// must enroll TOTP before the portal lets them do anything beyond enrollment.
var PrivilegedRoles = map[string]bool{
	"admin_tu":   true,
	"treasurer":  true,
	"headmaster": true,
	"developer":  true,
}

// RequiresMFA reports whether the account is subject to mandatory enrollment.
func RequiresMFA(user *model.User) bool {
	if user == nil {
		return false
	}
	return PrivilegedRoles[user.Role]
}
