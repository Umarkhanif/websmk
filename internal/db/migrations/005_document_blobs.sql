-- Dokumen siswa disimpan di server, bukan hanya metadata di app_state.
-- File divalidasi (ukuran & magic bytes) sebelum masuk ke tabel ini.
CREATE TABLE IF NOT EXISTS document_blobs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mime_type  TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    data       BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_document_blobs_created_at
    ON document_blobs (created_at DESC);
