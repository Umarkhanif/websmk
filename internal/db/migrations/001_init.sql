CREATE TABLE IF NOT EXISTS users (
    id                 TEXT PRIMARY KEY,
    identifier         TEXT NOT NULL,
    full_name          TEXT NOT NULL,
    role               TEXT NOT NULL,
    phone_wa           TEXT NOT NULL DEFAULT '',
    avatar_url         TEXT NOT NULL DEFAULT '',
    student_profile_id TEXT,
    attributes         JSONB NOT NULL DEFAULT '{}'::jsonb,
    password_hash      TEXT NOT NULL,
    is_demo            BOOLEAN NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS users_identifier_key ON users (identifier);
CREATE INDEX IF NOT EXISTS users_role_idx ON users (role);

CREATE TABLE IF NOT EXISTS app_state (
    id         SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    data       JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
