/*
  # Add idempotency support table

  1. New Tables
    - `booking_idempotency` 
      - `key_hash` (text, primary key) - SHA256 hash of the idempotency key
      - `booking_id` (bigint) - Reference to the created booking
      - `created_at` (timestamp) - When this idempotency record was created
      - `expires_at` (timestamp) - When this record should be cleaned up
      
  2. Security
    - Uses hash of idempotency key to prevent key enumeration
    - Includes expiry for automatic cleanup
    
  3. Performance  
    - Primary key on key_hash for fast lookups
    - Index on expires_at for cleanup operations
*/

-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS booking_idempotency (
    key_hash    TEXT        PRIMARY KEY,  -- SHA256 of idempotency key
    booking_id  BIGINT      NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '24 hours')
);

CREATE INDEX IF NOT EXISTS booking_idempotency_expires_at_idx 
    ON booking_idempotency (expires_at);

CREATE INDEX IF NOT EXISTS booking_idempotency_booking_id_idx 
    ON booking_idempotency (booking_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS booking_idempotency;

-- +goose StatementEnd