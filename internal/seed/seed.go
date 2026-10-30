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
	DemoMode     bool
	DemoPassword string
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
	if options.DemoMode {
		// All demo accounts share one publicly documented password, so one hash
		// is enough; env accounts get their own salts.
		demoHash, err := auth.HashPassword(options.DemoPassword)
		if err != nil {
			return fmt.Errorf("hash demo password: %w", err)
		}
		for _, user := range demoUsers() {
			user.PasswordHash = demoHash
			user.IsDemo = true
			if err := store.UpsertUser(ctx, user); err != nil {
				return fmt.Errorf("seed demo user %s: %w", user.Username, err)
			}
		}
		log.Printf("seeded %d demo accounts", len(demoUsers()))
	} else if err := store.DeleteDemoUsers(ctx); err != nil {
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
		item.user.IsDemo = false
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

func demoUsers() []*model.User {
	user := func(id, username, fullName, role, phone, avatar, studentProfileID string, attributes map[string]any) *model.User {
		return &model.User{
			ID:               id,
			Username:         username,
			FullName:         fullName,
			Role:             role,
			PhoneWA:          phone,
			AvatarURL:        avatarBase + avatar + "?w=150",
			StudentProfileID: studentProfileID,
			Attributes:       attributes,
		}
	}

	return []*model.User{
		user("usr-student-1", "0058471290", "Muhammad Budi Santoso", "student", "081234567801",
			"1539571696357-5a69c17a67c6", "std-budi-01", map[string]any{
				"isKetuaKelas": true, "isSekretaris": false, "isAlumni": false,
				"classroomId": "cls-xii-tkj-1", "majorId": "TKJ",
			}),
		user("usr-student-2", "0058471291", "Siti Rahmawati", "student", "081234567802",
			"1494790108377-be9c29b29330", "std-rahma-02", map[string]any{
				"isKetuaKelas": false, "isSekretaris": true, "isAlumni": false,
				"classroomId": "cls-xii-tkj-1", "majorId": "TKJ",
			}),
		user("usr-parent-1", "081234567800", "Bapak Ahmad Santoso (Wali Budi)", "parent", "081234567800",
			"1507003211169-0a1dd7228f2d", "std-budi-01", map[string]any{}),
		user("usr-teacher-1", "197804152005011003", "Ir. Hendro Wijaya, S.T., M.Kom.", "teacher", "081398765401",
			"1500648767791-00dcc994a43e", "", map[string]any{
				"isProctor": true, "isHomeroom": true, "isPklSupervisor": true,
			}),
		user("usr-treasurer-1", "198509122010012005", "Hj. Ratna Sari, S.E. (Bendahara)", "treasurer", "081398765402",
			"1573496359142-b8d87734a5a2", "", map[string]any{}),
		user("usr-discipline-1", "197503112000031002", "Drs. Bambang Sudarmono (Kesiswaan & BK)", "discipline", "081398765403",
			"1472099645785-5658abf4ff4e", "", map[string]any{}),
		user("usr-piket-1", "198811202014021004", "Rian Pratama, S.Pd. (Guru Piket)", "piket", "081398765404",
			"1519085360753-af0119f7cbe7", "", map[string]any{}),
		user("usr-admin-1", "199201052019032011", "Nurul Hidayati, A.Md. (Admin TU & Dapodik)", "admin_tu", "081398765405",
			"1580489944761-15a19d654956", "", map[string]any{}),
		user("usr-headmaster-1", "196805121994121001", "Dr. H. Sulaiman Effendi, M.Pd. (Kepala Sekolah)", "headmaster", "081398765406",
			"1560250097-0b93528c311a", "", map[string]any{}),
		user("usr-developer-1", "developer.admin", "Tim Developer WESMK (Core Engineering)", "developer", "081299990001",
			"1534528741775-53994a69daeb", "", map[string]any{
				"isDeveloper": true,
			}),
	}
}
