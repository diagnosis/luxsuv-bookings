/*
  # Add rate limiting table

  1. New Tables
    - `rate_limits` 
      - `key` (text, primary key) - Rate limit key (IP, email, etc.)
      - `count` (integer) - Current request count  
      - `window_start` (timestamp) - Start of current time window
      - `expires_at` (timestamp) - When this record expires
      
  2. Security
    - Enables rate limiting per IP and email for guest access requests
    - Automatic cleanup via expires_at
    
  3. Performance
    - Primary key on key for fast lookups
    - Index on expires_at for cleanup
*/

-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS rate_limits (
    key          TEXT        PRIMARY KEY,  -- e.g. "ip:127.0.0.1" or "email:user@example.com"
    count        INTEGER     NOT NULL DEFAULT 1,
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