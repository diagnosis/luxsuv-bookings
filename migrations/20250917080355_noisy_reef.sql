-- +goose Up
-- +goose StatementBegin
-- Add reschedule_count column and constraint (max 2)
ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS reschedule_count INTEGER NOT NULL DEFAULT 0;

ALTER TABLE bookings
    ADD CONSTRAINT bookings_reschedule_count_range
        CHECK (reschedule_count >= 0 AND reschedule_count <= 2);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop constraint and column
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_reschedule_count_range;
ALTER TABLE bookings DROP COLUMN IF EXISTS reschedule_count;
-- +goose StatementEnd