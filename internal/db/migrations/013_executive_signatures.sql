-- Executive report e-signatures: hash + HMAC are stored server-side so a
-- signed report can be verified instead of only flashing a success message.
CREATE TABLE IF NOT EXISTS executive_signatures (
    id             BIGSERIAL PRIMARY KEY,
    signed_by      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    document_type  TEXT NOT NULL DEFAULT 'laporan-eksekutif',
    document_hash  TEXT NOT NULL,
    signature_hmac TEXT NOT NULL,
    signed_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_executive_signatures_signed_at
    ON executive_signatures (signed_at DESC);
