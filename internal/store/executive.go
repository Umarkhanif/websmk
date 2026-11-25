package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type ExecutiveSignature struct {
	DocumentType  string
	DocumentHash  string
	SignatureHMAC string
	SignedAt      string
}

func (p *Postgres) RecordExecutiveSignature(ctx context.Context, signedBy, documentType, documentHash, signatureHMAC string) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO executive_signatures (signed_by, document_type, document_hash, signature_hmac)
		VALUES ($1, $2, $3, $4)
	`, signedBy, documentType, documentHash, signatureHMAC)
	return err
}

func (p *Postgres) LatestExecutiveSignature(ctx context.Context) (*ExecutiveSignature, error) {
	var signature ExecutiveSignature
	err := p.pool.QueryRow(ctx, `
		SELECT document_type, document_hash, signature_hmac,
			COALESCE(to_char(signed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
		FROM executive_signatures
		ORDER BY signed_at DESC
		LIMIT 1
	`).Scan(&signature.DocumentType, &signature.DocumentHash, &signature.SignatureHMAC, &signature.SignedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &signature, nil
}
