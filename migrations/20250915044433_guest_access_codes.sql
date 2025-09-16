-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS guest_access_codes (
                                                  id            BIGSERIAL PRIMARY KEY,
                                                  email         CITEXT      NOT NULL,
                                                  code_hash     TEXT        NOT NULL,         -- bcrypt(hash of 6-digit)
                                                  token         UUID        NOT NULL UNIQUE,  -- for magic link
                                                  purpose       TEXT        NOT NULL DEFAULT 'email_login',
                                                  expires_at    TIMESTAMPTZ NOT NULL,
                                                  used_at       TIMESTAMPTZ,
                                                  attempts      INT         NOT NULL DEFAULT 0,
                                                  ip_created    INET,
                                                  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
    );
CREATE INDEX IF NOT EXISTS guest_access_codes_email_idx ON guest_access_codes(email);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS guest_access_codes;
DROP EXTENSION IF EXISTS citext;
-- +goose StatementEnd