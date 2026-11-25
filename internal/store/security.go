package store

import (
	"context"
	"fmt"
	"strings"

	"wesmk/internal/model"
)

// ReplaceRecoveryCodes swaps the unused set atomically. Old codes stop working
// immediately when a user regenerates them.
func (p *Postgres) ReplaceRecoveryCodes(ctx context.Context, userID string, hashes []string) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM mfa_recovery_codes WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, hash := range hashes {
		if strings.TrimSpace(hash) == "" {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO mfa_recovery_codes (user_id, code_hash) VALUES ($1, $2)`, userID, hash); err != nil {
			return fmt.Errorf("insert recovery code: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// ConsumeRecoveryCode marks a matching unused code as used. The conditional
// update makes concurrent redemption of the same code safe.
func (p *Postgres) ConsumeRecoveryCode(ctx context.Context, userID, codeHash string) (bool, error) {
	tag, err := p.pool.Exec(ctx, `
		UPDATE mfa_recovery_codes
		SET used_at = now()
		WHERE user_id = $1 AND code_hash = $2 AND used_at IS NULL
	`, userID, codeHash)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (p *Postgres) DeleteRecoveryCodes(ctx context.Context, userID string) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM mfa_recovery_codes WHERE user_id = $1`, userID)
	return err
}

func (p *Postgres) CountUnusedRecoveryCodes(ctx context.Context, userID string) (int, error) {
	var count int
	err := p.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM mfa_recovery_codes WHERE user_id = $1 AND used_at IS NULL`, userID).Scan(&count)
	return count, err
}

// RecordLoginEvent writes one row to the append-only login audit. Audit failure
// must never block authentication, so callers log the error instead.
func (p *Postgres) RecordLoginEvent(ctx context.Context, event model.LoginEvent) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO auth_login_events
			(user_id, identifier, role, method, success, reason, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.UserID, event.Identifier, event.Role, event.Method, event.Success,
		event.Reason, event.IP, truncate(event.UserAgent, 240))
	return err
}

func (p *Postgres) LoginEventsForUser(ctx context.Context, userID string, limit int) ([]model.LoginEvent, error) {
	return p.queryLoginEvents(ctx, `
		SELECT id, user_id, identifier, role, method, success, reason, ip, user_agent, created_at
		FROM auth_login_events WHERE user_id = $1
		ORDER BY created_at DESC, id DESC LIMIT $2
	`, userID, normalizeLimit(limit))
}

func (p *Postgres) RecentLoginEvents(ctx context.Context, limit int) ([]model.LoginEvent, error) {
	return p.queryLoginEvents(ctx, `
		SELECT id, user_id, identifier, role, method, success, reason, ip, user_agent, created_at
		FROM auth_login_events
		ORDER BY created_at DESC, id DESC LIMIT $1
	`, normalizeLimit(limit))
}

func (p *Postgres) queryLoginEvents(ctx context.Context, query string, args ...any) ([]model.LoginEvent, error) {
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]model.LoginEvent, 0, 16)
	for rows.Next() {
		var event model.LoginEvent
		if err := rows.Scan(&event.ID, &event.UserID, &event.Identifier, &event.Role,
			&event.Method, &event.Success, &event.Reason, &event.IP,
			&event.UserAgent, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func normalizeLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 20
	}
	return limit
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
