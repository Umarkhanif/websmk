-- TOTP MFA for privileged accounts. The application encrypts the secret with
-- AES-256-GCM (WESMK_MFA_KEY) before writing; legacy plaintext rows are
-- re-encrypted on startup.
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_secret TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_enabled BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS users_totp_enabled_idx ON users (totp_enabled) WHERE totp_enabled;
