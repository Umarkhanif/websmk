-- Recovery codes are stored only as hashes and can be consumed once.
CREATE TABLE IF NOT EXISTS mfa_recovery_codes (
    id         BIGSERIAL PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash  TEXT NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS mfa_recovery_codes_user_idx
    ON mfa_recovery_codes (user_id) WHERE used_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS mfa_recovery_codes_unique_hash
    ON mfa_recovery_codes (user_id, code_hash);

-- Login audit is append-only like state_audit_log. Failed logins keep the
-- attempted identifier but user_id stays empty when no account matched.
CREATE TABLE IF NOT EXISTS auth_login_events (
    id         BIGSERIAL PRIMARY KEY,
    user_id    TEXT NOT NULL DEFAULT '',
    identifier TEXT NOT NULL DEFAULT '',
    role       TEXT NOT NULL DEFAULT '',
    method     TEXT NOT NULL DEFAULT '',
    success    BOOLEAN NOT NULL,
    reason     TEXT NOT NULL DEFAULT '',
    ip         TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS auth_login_events_created_idx
    ON auth_login_events (created_at DESC);
CREATE INDEX IF NOT EXISTS auth_login_events_user_idx
    ON auth_login_events (user_id, created_at DESC);

CREATE OR REPLACE FUNCTION block_auth_login_event_mutation() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'SECURITY VIOLATION: auth_login_events is append-only.';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_lock_auth_login_events ON auth_login_events;
CREATE TRIGGER trg_lock_auth_login_events
    BEFORE UPDATE OR DELETE ON auth_login_events
    FOR EACH ROW EXECUTE FUNCTION block_auth_login_event_mutation();
