package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"wesmk/internal/model"
	"wesmk/internal/secure"
)

// ErrNoState is returned when the school state has not been initialized yet.
var ErrNoState = errors.New("app_state is empty")

type Postgres struct {
	pool      *pgxpool.Pool
	mfaCipher *secure.Cipher
}

type Option func(*Postgres)

// WithMFACipher enables encryption for TOTP secrets written by this store.
func WithMFACipher(cipher *secure.Cipher) Option {
	return func(p *Postgres) { p.mfaCipher = cipher }
}

func NewPostgres(pool *pgxpool.Pool, options ...Option) *Postgres {
	p := &Postgres{pool: pool}
	for _, option := range options {
		option(p)
	}
	return p
}

func (p *Postgres) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }

const userColumns = `id, identifier, full_name, role, phone_wa, avatar_url,
	COALESCE(student_profile_id, ''), attributes, password_hash, is_active,
	totp_secret, totp_enabled`

func (p *Postgres) scanUser(row pgx.Row) (*model.User, error) {
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
		&user.IsActive,
		&user.TOTPSecret,
		&user.TOTPEnabled,
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
	if user.TOTPSecret != "" {
		if secure.IsEncrypted(user.TOTPSecret) && p.mfaCipher == nil {
			return nil, errors.New("TOTP secret is encrypted but no MFA key is configured")
		}
		if p.mfaCipher != nil {
			secret, err := p.mfaCipher.Decrypt(user.TOTPSecret)
			if err != nil {
				return nil, err
			}
			user.TOTPSecret = secret
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
			(id, identifier, full_name, role, phone_wa, avatar_url, student_profile_id, attributes, password_hash)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8::jsonb, $9)
		ON CONFLICT (identifier) DO UPDATE SET
			full_name = EXCLUDED.full_name,
			role = EXCLUDED.role,
			phone_wa = EXCLUDED.phone_wa,
			avatar_url = EXCLUDED.avatar_url,
			student_profile_id = EXCLUDED.student_profile_id,
			attributes = EXCLUDED.attributes,
			password_hash = EXCLUDED.password_hash,
			updated_at = now()
	`,
		user.ID, user.Username, user.FullName, user.Role, phone, user.AvatarURL,
		user.StudentProfileID, string(attributes), user.PasswordHash,
	)
	return err
}

func (p *Postgres) UserByIdentifier(ctx context.Context, identifier string) (*model.User, error) {
	row := p.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE identifier = $1`, identifier)
	return p.scanUser(row)
}

func (p *Postgres) UserByID(ctx context.Context, id string) (*model.User, error) {
	row := p.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	return p.scanUser(row)
}

func (p *Postgres) ParentByPhone(ctx context.Context, normalizedPhone string) (*model.User, error) {
	row := p.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE role = 'parent' AND phone_wa = $1 LIMIT 1`, normalizedPhone)
	return p.scanUser(row)
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

// StateAudit describes who performed a state mutation.
type StateAudit struct {
	ActorID string
	Role    string
	Action  string
	Detail  json.RawMessage
}

// UpdateState locks the single state row, applies mutate, and saves the result
// in one transaction. Server-authoritative writes (quiz, exam, payments) use
// this so two requests cannot overwrite each other's changes.
func (p *Postgres) UpdateState(ctx context.Context, mutate func(json.RawMessage) (json.RawMessage, error)) (json.RawMessage, error) {
	return p.UpdateStateAudited(ctx, StateAudit{}, mutate)
}

// UpdateStateAudited is UpdateState plus an append-only audit row written in the
// same transaction, so a state change can never exist without its audit trail.
func (p *Postgres) UpdateStateAudited(ctx context.Context, audit StateAudit, mutate func(json.RawMessage) (json.RawMessage, error)) (json.RawMessage, error) {
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

	detail := audit.Detail
	if len(detail) == 0 || !json.Valid(detail) {
		detail = json.RawMessage(`{}`)
	}
	action := audit.Action
	if action == "" {
		action = "state.update"
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO state_audit_log (actor_user_id, actor_role, action, detail) VALUES ($1, $2, $3, $4::jsonb)`,
		audit.ActorID, audit.Role, action, string(detail)); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return updated, nil
}

// RecordStateAudit writes a standalone audit entry for direct state writes that
// do not go through UpdateStateAudited (reset and raw snapshot saves).
func (p *Postgres) RecordStateAudit(ctx context.Context, audit StateAudit) error {
	detail := audit.Detail
	if len(detail) == 0 || !json.Valid(detail) {
		detail = json.RawMessage(`{}`)
	}
	action := audit.Action
	if action == "" {
		action = "state.write"
	}
	_, err := p.pool.Exec(ctx,
		`INSERT INTO state_audit_log (actor_user_id, actor_role, action, detail) VALUES ($1, $2, $3, $4::jsonb)`,
		audit.ActorID, audit.Role, action, string(detail))
	return err
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

// ErrBlobNotFound is returned when a document blob does not exist.
var ErrBlobNotFound = errors.New("document blob not found")

// PutDocumentBlob stores validated file bytes and returns its generated UUID.
func (p *Postgres) PutDocumentBlob(ctx context.Context, mimeType string, data []byte) (string, error) {
	var id string
	err := p.pool.QueryRow(ctx,
		`INSERT INTO document_blobs (mime_type, size_bytes, data) VALUES ($1, $2, $3) RETURNING id::text`,
		mimeType, len(data), data).Scan(&id)
	return id, err
}

// DeleteDocumentBlob removes an orphaned upload when the document record could
// not be created.
func (p *Postgres) DeleteDocumentBlob(ctx context.Context, id string) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM document_blobs WHERE id = $1::uuid`, id)
	return err
}

// DocumentBlob loads a stored file for an authorized reader.
func (p *Postgres) DocumentBlob(ctx context.Context, id string) (string, []byte, error) {
	var mimeType string
	var data []byte
	err := p.pool.QueryRow(ctx,
		`SELECT mime_type, data FROM document_blobs WHERE id = $1::uuid`, id).Scan(&mimeType, &data)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, ErrBlobNotFound
	}
	return mimeType, data, err
}
