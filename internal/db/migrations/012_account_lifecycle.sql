-- Account lifecycle: deactivated accounts keep their history but cannot log in.
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;
CREATE INDEX IF NOT EXISTS users_is_active_idx ON users (is_active);
