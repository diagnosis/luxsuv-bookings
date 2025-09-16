-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS users (
                                     id            BIGSERIAL PRIMARY KEY,
                                     role          TEXT        NOT NULL DEFAULT 'rider', -- rider|driver|admin
                                     email         CITEXT      NOT NULL UNIQUE,
                                     password_hash TEXT        NOT NULL,
                                     name          TEXT        NOT NULL DEFAULT '',
                                     phone         TEXT        NOT NULL DEFAULT '',
                                     created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
    );

CREATE INDEX IF NOT EXISTS users_email_idx ON users(email);

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS bookings_user_id_idx ON bookings(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE bookings DROP COLUMN IF EXISTS user_id;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd