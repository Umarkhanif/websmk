package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"wesmk/internal/auth"
	"wesmk/internal/model"
)

type Store interface {
	UpsertUser(ctx context.Context, user *model.User) error
	DeleteDemoUsers(ctx context.Context) error
}

type Options struct {
	EnvUsersJSON string
}

var validRoles = map[string]bool{
	"student":    true,
	"parent":     true,
	"treasurer":  true,
	"teacher":    true,
	"discipline": true,
	"piket":      true,
	"admin_tu":   true,
	"headmaster": true,
	"developer":  true,
}

const avatarBase = "https://images.unsplash.com/photo-"

func Run(ctx context.Context, store Store, options Options) error {
	// Purge any demo accounts left over from an earlier deployment.
	if err := store.DeleteDemoUsers(ctx); err != nil {
		return fmt.Errorf("remove demo accounts: %w", err)
	}

	if strings.TrimSpace(options.EnvUsersJSON) == "" {
		return nil
	}

	users, err := parseEnvUsers(options.EnvUsersJSON)
	if err != nil {
		return err
	}
	for _, item := range users {
		hash, err := auth.HashPassword(item.password)
		if err != nil {
			return fmt.Errorf("hash password for %s: %w", item.user.Username, err)
		}
		item.user.PasswordHash = hash
		if err := store.UpsertUser(ctx, item.user); err != nil {
			return fmt.Errorf("seed env user %s: %w", item.user.Username, err)
		}
	}
	log.Printf("seeded %d env accounts", len(users))
	return nil
}

type envUserJSON struct {
	ID               string `json:"id"`
	Identifier       string `json:"identifier"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	FullName         string `json:"fullName"`
	Role             string `json:"role"`
	PhoneWA          string `json:"phoneWa"`
	AvatarURL        string `json:"avatarUrl"`
	StudentProfileID string `json:"studentProfileId"`
}

type parsedUser struct {
	user     *model.User
	password string
}

func parseEnvUsers(raw string) ([]parsedUser, error) {
	var entries []envUserJSON
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return nil, fmt.Errorf("WESMK_USERS bukan JSON array yang valid: %w", err)
	}

	users := make([]parsedUser, 0, len(entries))
	for index, entry := range entries {
		identifier := strings.TrimSpace(entry.Identifier)
		if identifier == "" {
			identifier = strings.TrimSpace(entry.Username)
		}
		if identifier == "" || len(entry.Password) < 8 || !validRoles[entry.Role] {
			log.Printf("WESMK_USERS[%d] tidak valid (butuh identifier, password >= 8 karakter, role); dilewati", index)
			continue
		}

		id := entry.ID
		if id == "" {
			id = fmt.Sprintf("usr-env-%d", index+1)
		}
		fullName := entry.FullName
		if fullName == "" {
			fullName = "Pengguna " + identifier
		}
		avatar := entry.AvatarURL
		if avatar == "" {
			avatar = avatarBase + "1560250097-0b93528c311a?w=150"
		}

		users = append(users, parsedUser{
			password: entry.Password,
			user: &model.User{
				ID:               id,
				Username:         identifier,
				FullName:         fullName,
				Role:             entry.Role,
				PhoneWA:          entry.PhoneWA,
				AvatarURL:        avatar,
				StudentProfileID: entry.StudentProfileID,
				Attributes:       map[string]any{},
			},
		})
	}
	return users, nil
}
