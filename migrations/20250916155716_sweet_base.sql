-- +goose Up
-- +goose StatementBegin
-- Add idempotency support table (robust / re-runnable)

CREATE TABLE IF NOT EXISTS booking_idempotency (
                                                   key_hash    TEXT        PRIMARY KEY,                   -- SHA256 of idempotency key
                                                   booking_id  BIGINT      NOT NULL,                      -- FK added below
                                                   created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '24 hours')
    );

DO $$
BEGIN
ALTER TABLE booking_idempotency
    ADD CONSTRAINT booking_idempotency_booking_id_fk
        FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE;
EXCEPTION
    WHEN duplicate_object THEN
        NULL;
END $$;

CREATE INDEX IF NOT EXISTS booking_idempotency_expires_at_idx
    ON booking_idempotency (expires_at);

CREATE INDEX IF NOT EXISTS booking_idempotency_booking_id_idx
    ON booking_idempotency (booking_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS booking_idempotency;
-- +goose StatementEnd