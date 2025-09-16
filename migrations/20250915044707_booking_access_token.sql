-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS booking_access_tokens (
    id            BIGSERIAL PRIMARY KEY,
    booking_id    BIGINT      NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    token         UUID        NOT NULL UNIQUE,     -- one-time token
    expires_at    TIMESTAMPTZ NOT NULL,
    used_at       TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
    );
CREATE INDEX IF NOT EXISTS booking_access_tokens_booking_id_idx ON booking_access_tokens(booking_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS booking_access_tokens;
-- +goose StatementEnd