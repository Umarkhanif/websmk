-- Jejak audit untuk setiap mutasi state. Tabel ini append-only:
-- trigger di bawah menolak UPDATE dan DELETE.
CREATE TABLE IF NOT EXISTS state_audit_log (
    id            BIGSERIAL PRIMARY KEY,
    actor_user_id TEXT NOT NULL DEFAULT '',
    actor_role    TEXT NOT NULL DEFAULT '',
    action        TEXT NOT NULL,
    detail        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_state_audit_log_created_at
    ON state_audit_log (created_at DESC);

CREATE OR REPLACE FUNCTION block_state_audit_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'SECURITY VIOLATION: state_audit_log is append-only.';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_lock_state_audit ON state_audit_log;
CREATE TRIGGER trg_lock_state_audit
    BEFORE UPDATE OR DELETE ON state_audit_log
    FOR EACH ROW EXECUTE FUNCTION block_state_audit_mutation();
