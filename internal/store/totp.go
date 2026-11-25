package store

import (
	"context"
	"fmt"

	"wesmk/internal/secure"
)

// SetTOTP stores an encrypted pending secret or flips MFA on. Disabling clears
// the secret and recovery codes so re-enrollment always starts clean.
func (p *Postgres) SetTOTP(ctx context.Context, userID, secret string, enabled bool) (bool, error) {
	stored := secret
	if secret != "" && p.mfaCipher != nil {
		encrypted, err := p.mfaCipher.Encrypt(secret)
		if err != nil {
			return false, fmt.Errorf("encrypt totp secret: %w", err)
		}
		stored = encrypted
	}
	tag, err := p.pool.Exec(ctx, `
		UPDATE users
		SET totp_secret = $2, totp_enabled = $3, updated_at = now()
		WHERE id = $1
	`, userID, stored, enabled)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// EncryptLegacyTOTPSecrets rewrites plaintext secrets created before encryption
// was introduced. It is safe to call on every boot and returns the row count.
func (p *Postgres) EncryptLegacyTOTPSecrets(ctx context.Context) (int, error) {
	if p.mfaCipher == nil {
		return 0, nil
	}
	rows, err := p.pool.Query(ctx,
		`SELECT id, totp_secret FROM users WHERE totp_secret <> '' AND totp_secret NOT LIKE $1`,
		secure.MFAPrefix+"%")
	if err != nil {
		return 0, err
	}
	type legacySecret struct {
		id     string
		secret string
	}
	var legacy []legacySecret
	for rows.Next() {
		var item legacySecret
		if err := rows.Scan(&item.id, &item.secret); err != nil {
			rows.Close()
			return 0, err
		}
		legacy = append(legacy, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	if len(legacy) == 0 {
		return 0, nil
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, item := range legacy {
		encrypted, err := p.mfaCipher.Encrypt(item.secret)
		if err != nil {
			return 0, fmt.Errorf("encrypt legacy totp secret: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE users SET totp_secret = $2, updated_at = now() WHERE id = $1`, item.id, encrypted); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(legacy), nil
}
