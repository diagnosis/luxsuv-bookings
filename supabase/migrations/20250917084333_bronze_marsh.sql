-- +goose Up
-- +goose StatementBegin

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS citext;

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    role          TEXT        NOT NULL DEFAULT 'rider' CHECK (role IN ('guest', 'rider', 'driver', 'dispatcher', 'admin')),
    email         CITEXT      NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    name          TEXT        NOT NULL DEFAULT '',
    phone         TEXT        NOT NULL DEFAULT '',
    is_verified   BOOLEAN     NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Email verification tokens
CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       UUID        NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Guest access codes
CREATE TABLE IF NOT EXISTS guest_access_codes (
    id          BIGSERIAL PRIMARY KEY,
    email       CITEXT      NOT NULL,
    code_hash   TEXT        NOT NULL,         -- bcrypt hash of 6-digit code
    token       UUID        NOT NULL UNIQUE,  -- magic link token
    purpose     TEXT        NOT NULL DEFAULT 'email_login',
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    attempts    INT         NOT NULL DEFAULT 0,
    ip_created  INET,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Rate limits table
CREATE TABLE IF NOT EXISTS rate_limits (
    rl_key       TEXT        PRIMARY KEY,  -- hashed rate limit key
    count        INTEGER     NOT NULL DEFAULT 1 CHECK (count >= 0),
    window_start TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '1 hour')
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS users_email_idx ON users(email);
CREATE INDEX IF NOT EXISTS users_role_idx ON users(role);
CREATE INDEX IF NOT EXISTS users_created_at_idx ON users(created_at DESC);

CREATE INDEX IF NOT EXISTS email_verification_tokens_user_id_idx ON email_verification_tokens(user_id);
CREATE INDEX IF NOT EXISTS email_verification_tokens_token_idx ON email_verification_tokens(token);
CREATE INDEX IF NOT EXISTS email_verification_tokens_expires_at_idx ON email_verification_tokens(expires_at);

CREATE INDEX IF NOT EXISTS guest_access_codes_email_idx ON guest_access_codes(email);
CREATE INDEX IF NOT EXISTS guest_access_codes_token_idx ON guest_access_codes(token);
CREATE INDEX IF NOT EXISTS guest_access_codes_expires_at_idx ON guest_access_codes(expires_at);
CREATE INDEX IF NOT EXISTS guest_access_codes_used_at_idx ON guest_access_codes(used_at);

CREATE INDEX IF NOT EXISTS rate_limits_expires_at_idx ON rate_limits(expires_at);

-- Updated at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to users table
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop triggers
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS rate_limits_expires_at_idx;
DROP INDEX IF EXISTS guest_access_codes_used_at_idx;
DROP INDEX IF EXISTS guest_access_codes_expires_at_idx;
DROP INDEX IF EXISTS guest_access_codes_token_idx;
DROP INDEX IF EXISTS guest_access_codes_email_idx;
DROP INDEX IF EXISTS email_verification_tokens_expires_at_idx;
DROP INDEX IF EXISTS email_verification_tokens_token_idx;
DROP INDEX IF EXISTS email_verification_tokens_user_id_idx;
DROP INDEX IF EXISTS users_created_at_idx;
DROP INDEX IF EXISTS users_role_idx;
DROP INDEX IF EXISTS users_email_idx;

-- Drop tables
DROP TABLE IF EXISTS rate_limits;
DROP TABLE IF EXISTS guest_access_codes;
DROP TABLE IF EXISTS email_verification_tokens;
DROP TABLE IF EXISTS users;

-- Drop extensions
DROP EXTENSION IF EXISTS citext;

-- +goose StatementEnd