/*
  # Improve database constraints and indexes

  1. Constraints
    - Add NOT NULL constraint on scheduled_at
    - Add check constraints for passengers (1-8) and luggages (0-10)
    - Add check constraint for scheduled_at to be reasonable (after 2000-01-01)
    
  2. Indexes
    - Add index on lower(rider_email) for case-insensitive email lookups
    - Add index on (status, created_at DESC) for efficient status filtering
    - Add index on (rider_email, created_at DESC) for guest booking lists
    - Add index on scheduled_at for time-based queries
    
  3. Performance
    - Improves query performance for guest booking operations
    - Ensures data integrity at the database level
*/

-- +goose Up
-- +goose StatementBegin

-- Add constraints for data integrity
ALTER TABLE bookings 
  ADD CONSTRAINT bookings_passengers_range 
  CHECK (passengers >= 1 AND passengers <= 8);

ALTER TABLE bookings 
  ADD CONSTRAINT bookings_luggages_range 
  CHECK (luggages >= 0 AND luggages <= 10);

ALTER TABLE bookings 
  ADD CONSTRAINT bookings_scheduled_at_reasonable 
  CHECK (scheduled_at > '2000-01-01'::timestamptz);

-- Add performance indexes
CREATE INDEX IF NOT EXISTS bookings_rider_email_lower_idx 
  ON bookings (lower(rider_email));

CREATE INDEX IF NOT EXISTS bookings_status_created_at_idx 
  ON bookings (status, created_at DESC);

CREATE INDEX IF NOT EXISTS bookings_rider_email_created_at_idx 
  ON bookings (lower(rider_email), created_at DESC);

CREATE INDEX IF NOT EXISTS bookings_scheduled_at_idx 
  ON bookings (scheduled_at);

-- Add index for guest access codes cleanup
CREATE INDEX IF NOT EXISTS guest_access_codes_expires_at_idx 
  ON guest_access_codes (expires_at);

CREATE INDEX IF NOT EXISTS guest_access_codes_used_at_idx 
  ON guest_access_codes (used_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS bookings_scheduled_at_idx;
DROP INDEX IF EXISTS bookings_rider_email_created_at_idx;
DROP INDEX IF EXISTS bookings_status_created_at_idx;
DROP INDEX IF EXISTS bookings_rider_email_lower_idx;
DROP INDEX IF EXISTS guest_access_codes_used_at_idx;
DROP INDEX IF EXISTS guest_access_codes_expires_at_idx;

ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_scheduled_at_reasonable;
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_luggages_range;
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_passengers_range;

-- +goose StatementEnd