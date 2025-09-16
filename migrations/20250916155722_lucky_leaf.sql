-- +goose Up
-- +goose StatementBegin
-- Add rate limiting table (re-runnable; safe on multiple runs)

CREATE TABLE IF NOT EXISTS rate_limits (
                                           rl_key       TEXT        PRIMARY KEY,                             -- e.g. "ip:127.0.0.1" or "email:user@example.com"
                                           count        INTEGER     NOT NULL DEFAULT 1 CHECK (count >= 0),
    window_start TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '1 hour')
    );

CREATE INDEX IF NOT EXISTS rate_limits_expires_at_idx
    ON rate_limits (expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS rate_limits;
-- +goose StatementEnd