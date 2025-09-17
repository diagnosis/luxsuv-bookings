/*
  # Add reschedule count to bookings table

  1. Changes
     - Add `reschedule_count` column to bookings table with default value 0
     - Add check constraint to enforce maximum reschedule limit

  2. Business Rules
     - Maximum 2 reschedules allowed per booking
     - Counter increments automatically when scheduled_at is changed
*/

-- +goose Up
-- +goose StatementBegin
ALTER TABLE bookings 
ADD COLUMN IF NOT EXISTS reschedule_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE bookings 
ADD CONSTRAINT bookings_reschedule_count_range 
CHECK (reschedule_count >= 0 AND reschedule_count <= 2);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_reschedule_count_range;
ALTER TABLE bookings DROP COLUMN IF EXISTS reschedule_count;
-- +goose StatementEnd