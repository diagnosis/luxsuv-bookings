-- +goose Up
-- +goose StatementBegin
CREATE TYPE booking_status AS ENUM ('pending','confirmed','assigned','on_trip','completed','canceled');

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'ride_type') THEN
CREATE TYPE ride_type AS ENUM ('per_ride','hourly');
END IF;
END$$;

CREATE TABLE IF NOT EXISTS bookings (
id            BIGSERIAL PRIMARY KEY,
manage_token  UUID        NOT NULL UNIQUE,
status        booking_status NOT NULL DEFAULT 'pending',

rider_name    TEXT        NOT NULL,
rider_email   TEXT        NOT NULL,
rider_phone   TEXT        NOT NULL,

pickup        TEXT        NOT NULL,
dropoff       TEXT        NOT NULL,
scheduled_at  TIMESTAMPTZ NOT NULL,
notes         TEXT        NOT NULL DEFAULT '',

passengers    INT         NOT NULL DEFAULT 1,
luggages      INT         NOT NULL DEFAULT 0,
ride_type     ride_type   NOT NULL DEFAULT 'per_ride',

driver_id     BIGINT      NULL,

created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
    );

CREATE INDEX IF NOT EXISTS bookings_status_idx ON bookings(status);
CREATE INDEX IF NOT EXISTS bookings_scheduled_at_idx ON bookings(scheduled_at);
CREATE INDEX IF NOT EXISTS bookings_ride_type_idx ON bookings(ride_type);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS bookings_set_updated_at ON bookings;
CREATE TRIGGER bookings_set_updated_at
    BEFORE UPDATE ON bookings
    FOR EACH ROW EXECUTE PROCEDURE set_updated_at();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS bookings_set_updated_at ON bookings;
DROP FUNCTION IF EXISTS set_updated_at();
DROP INDEX IF EXISTS bookings_ride_type_idx;
DROP INDEX IF EXISTS bookings_scheduled_at_idx;
DROP INDEX IF EXISTS bookings_status_idx;
DROP TABLE IF EXISTS bookings;
DROP TYPE IF EXISTS ride_type;
DROP TYPE IF EXISTS booking_status;
-- +goose StatementEnd
