-- +goose Up
-- +goose StatementBegin

-- 1) Add NOT VALID constraints (don’t validate existing rows yet)
ALTER TABLE bookings
    ADD CONSTRAINT bookings_passengers_range
        CHECK (passengers >= 1 AND passengers <= 8) NOT VALID;

ALTER TABLE bookings
    ADD CONSTRAINT bookings_luggages_range
        CHECK (luggages >= 0 AND luggages <= 10) NOT VALID;

ALTER TABLE bookings
    ADD CONSTRAINT bookings_scheduled_at_reasonable
        CHECK (scheduled_at > '2000-01-01'::timestamptz) NOT VALID;

-- 2) Indexes (safe to create anytime)
CREATE INDEX IF NOT EXISTS bookings_rider_email_lower_idx
    ON bookings (lower(rider_email));

CREATE INDEX IF NOT EXISTS bookings_status_created_at_idx
    ON bookings (status, created_at DESC);

CREATE INDEX IF NOT EXISTS bookings_rider_email_created_at_idx
    ON bookings (lower(rider_email), created_at DESC);

CREATE INDEX IF NOT EXISTS bookings_scheduled_at_idx
    ON bookings (scheduled_at);

-- (Optional but recommended) quick lookups by manage_token
CREATE INDEX IF NOT EXISTS bookings_manage_token_idx
    ON bookings (manage_token);

-- Guest access housekeeping
CREATE INDEX IF NOT EXISTS guest_access_codes_expires_at_idx
    ON guest_access_codes (expires_at);

CREATE INDEX IF NOT EXISTS guest_access_codes_used_at_idx
    ON guest_access_codes (used_at);

-- 3) Backfill / clean data to satisfy constraints
--    a) scheduled_at: set any bad/NULL to a safe future default
UPDATE bookings
SET scheduled_at = COALESCE(created_at, now()) + interval '1 hour'
WHERE scheduled_at IS NULL
   OR scheduled_at <= '2000-01-01'::timestamptz;

--    b) passengers: clamp to minimum 1
UPDATE bookings
SET passengers = 1
WHERE passengers IS NULL OR passengers < 1;

--    c) luggages: clamp to minimum 0
UPDATE bookings
SET luggages = 0
WHERE luggages IS NULL OR luggages < 0;

-- 4) Validate constraints now that data is clean
ALTER TABLE bookings VALIDATE CONSTRAINT bookings_passengers_range;
ALTER TABLE bookings VALIDATE CONSTRAINT bookings_luggages_range;
ALTER TABLE bookings VALIDATE CONSTRAINT bookings_scheduled_at_reasonable;

-- 5) (Optional) enforce NOT NULL at the end (now it won’t fail)
ALTER TABLE bookings
    ALTER COLUMN scheduled_at SET NOT NULL;

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

-- Revert NOT NULL
ALTER TABLE bookings
    ALTER COLUMN scheduled_at DROP NOT NULL;

-- Drop indexes
DROP INDEX IF EXISTS bookings_manage_token_idx;
DROP INDEX IF EXISTS bookings_scheduled_at_idx;
DROP INDEX IF EXISTS bookings_rider_email_created_at_idx;
DROP INDEX IF EXISTS bookings_status_created_at_idx;
DROP INDEX IF EXISTS bookings_rider_email_lower_idx;
DROP INDEX IF EXISTS guest_access_codes_used_at_idx;
DROP INDEX IF EXISTS guest_access_codes_expires_at_idx;

-- Drop constraints
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_scheduled_at_reasonable;
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_luggages_range;
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_passengers_range;

-- +goose StatementEnd