package model

import (
	"encoding/json"
	"strings"
)

// User is the authenticated account. MarshalJSON preserves the exact shape the
// React client expects (username_nisn_nip, camelCase fields, optional flags).
type User struct {
	ID               string
	Username         string
	FullName         string
	Role             string
	PhoneWA          string
	AvatarURL        string
	StudentProfileID string
	Attributes       map[string]any
	PasswordHash     string
	IsActive         bool
	TOTPSecret       string
	TOTPEnabled      bool
}

func (u User) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"id":                u.ID,
		"username_nisn_nip": u.Username,
		"fullName":          u.FullName,
		"role":              u.Role,
		"phoneWa":           u.PhoneWA,
		"avatarUrl":         u.AvatarURL,
	}
	if u.StudentProfileID != "" {
		payload["studentProfileId"] = u.StudentProfileID
	}
	for key, value := range u.Attributes {
		if _, exists := payload[key]; !exists {
			payload[key] = value
		}
	}
	return json.Marshal(payload)
}

// NormalizePhone converts Indonesian numbers to the canonical 08... form.
func NormalizePhone(phone string) string {
	var digits strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	value := digits.String()
	if strings.HasPrefix(value, "62") {
		return "0" + value[2:]
	}
	return value
}
