package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"wesmk/internal/model"
)

// ErrNoState is returned when the school state has not been initialized yet.
var ErrNoState = errors.New("app_state is empty")

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (p *Postgres) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }

const userColumns = `id, identifier, full_name, role, phone_wa, avatar_url,
	COALESCE(student_profile_id, ''), attributes, password_hash, is_demo`

func scanUser(row pgx.Row) (*model.User, error) {
	var user model.User
	var attributes []byte

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.FullName,
		&user.Role,
		&user.PhoneWA,
		&user.AvatarURL,
		&user.StudentProfileID,
		&attributes,
		&user.PasswordHash,
		&user.IsDemo,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	user.Attributes = map[string]any{}
	if len(attributes) > 0 {
		if err := json.Unmarshal(attributes, &user.Attributes); err != nil {
			return nil, fmt.Errorf("decode user attributes: %w", err)
		}
	}
	return &user, nil
}

func (p *Postgres) UpsertUser(ctx context.Context, user *model.User) error {
	attributes, err := json.Marshal(user.Attributes)
	if err != nil {
		return err
	}
	phone := model.NormalizePhone(user.PhoneWA)

	_, err = p.pool.Exec(ctx, `
		INSERT INTO users
			(id, identifier, full_name, role, phone_wa, avatar_url, student_profile_id, attributes, password_hash, is_demo)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8::jsonb, $9, $10)
		ON CONFLICT (identifier) DO UPDATE SET
			full_name = EXCLUDED.full_name,
			role = EXCLUDED.role,
			phone_wa = EXCLUDED.phone_wa,
			avatar_url = EXCLUDED.avatar_url,
			student_profile_id = EXCLUDED.student_profile_id,
			attributes = EXCLUDED.attributes,
			password_hash = EXCLUDED.password_hash,
			is_demo = EXCLUDED.is_demo,
			updated_at = now()
	`,
		user.ID, user.Username, user.FullName, user.Role, phone, user.AvatarURL,
		user.StudentProfileID, string(attributes), user.PasswordHash, user.IsDemo,
	)
	return err
}

func (p *Postgres) UserByIdentifier(ctx context.Context, identifier string) (*model.User, error) {
	row := p.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE identifier = $1`, identifier)
	return scanUser(row)
}

func (p *Postgres) UserByID(ctx context.Context, id string) (*model.User, error) {
	row := p.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	return scanUser(row)
}

func (p *Postgres) ParentByPhone(ctx context.Context, normalizedPhone string) (*model.User, error) {
	row := p.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE role = 'parent' AND phone_wa = $1 LIMIT 1`, normalizedPhone)
	return scanUser(row)
}

func (p *Postgres) DemoUserByRole(ctx context.Context, role string) (*model.User, error) {
	row := p.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE role = $1 AND is_demo = true ORDER BY id ASC LIMIT 1`, role)
	return scanUser(row)
}

func (p *Postgres) DeleteDemoUsers(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM users WHERE is_demo = true`)
	return err
}

func (p *Postgres) GetState(ctx context.Context) (json.RawMessage, error) {
	var data []byte
	err := p.pool.QueryRow(ctx, `SELECT data FROM app_state WHERE id = 1`).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// UpdateState locks the single state row, applies mutate, and saves the result
// in one transaction. Server-authoritative writes (quiz, exam, payments) use
// this so two requests cannot overwrite each other's changes.
func (p *Postgres) UpdateState(ctx context.Context, mutate func(json.RawMessage) (json.RawMessage, error)) (json.RawMessage, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var data []byte
	if err := tx.QueryRow(ctx, `SELECT data FROM app_state WHERE id = 1 FOR UPDATE`).Scan(&data); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoState
		}
		return nil, err
	}

	updated, err := mutate(json.RawMessage(data))
	if err != nil {
		return nil, err
	}
	if !json.Valid(updated) {
		return nil, errors.New("state is not valid JSON")
	}

	if _, err := tx.Exec(ctx, `UPDATE app_state SET data = $1::jsonb, updated_at = now() WHERE id = 1`, string(updated)); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

func (p *Postgres) PutState(ctx context.Context, data json.RawMessage) error {
	if !json.Valid(data) {
		return errors.New("state is not valid JSON")
	}
	_, err := p.pool.Exec(ctx, `
		INSERT INTO app_state (id, data) VALUES (1, $1::jsonb)
		ON CONFLICT (id) DO UPDATE SET data = EXCLUDED.data, updated_at = now()
	`, string(data))
	return err
}

func (p *Postgres) ResetState(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM app_state WHERE id = 1`)
	return err
}

func (p *Postgres) HasState(ctx context.Context) (bool, error) {
	var exists bool
	err := p.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM app_state WHERE id = 1)`).Scan(&exists)
	return exists, err
}
