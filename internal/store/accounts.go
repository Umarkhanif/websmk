package store

import (
	"context"
	"errors"

	"wesmk/internal/model"
)

var ErrAccountExists = errors.New("akun dengan identitas tersebut sudah ada")

// ListUsers returns every account so Admin TU can manage access without
// touching the state blob.
func (p *Postgres) ListUsers(ctx context.Context) ([]*model.User, error) {
	rows, err := p.pool.Query(ctx, `SELECT `+userColumns+` FROM users ORDER BY role, full_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*model.User, 0)
	for rows.Next() {
		user, err := p.scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

// CreateAccount inserts a new staff account. Existing identifiers are refused
// instead of being overwritten.
func (p *Postgres) CreateAccount(ctx context.Context, user *model.User) error {
	tag, err := p.pool.Exec(ctx, `
		INSERT INTO users (id, identifier, full_name, role, phone_wa, attributes, password_hash, is_active)
		VALUES ($1, $2, $3, $4, $5, '{}'::jsonb, $6, TRUE)
		ON CONFLICT (identifier) DO NOTHING
	`, user.ID, user.Username, user.FullName, user.Role, model.NormalizePhone(user.PhoneWA), user.PasswordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAccountExists
	}
	return nil
}

func (p *Postgres) SetUserActive(ctx context.Context, id string, active bool) (bool, error) {
	tag, err := p.pool.Exec(ctx, `UPDATE users SET is_active = $2, updated_at = now() WHERE id = $1`, id, active)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (p *Postgres) SetUserPasswordHash(ctx context.Context, id, hash string) (bool, error) {
	tag, err := p.pool.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, id, hash)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
