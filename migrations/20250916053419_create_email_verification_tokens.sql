-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS email_verification_tokens (
                                                         id          BIGSERIAL PRIMARY KEY,
                                                         user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       UUID   NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
    );

CREATE INDEX IF NOT EXISTS email_verification_tokens_user_id_idx
    ON email_verification_tokens(user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS email_verification_tokens;
-- +goose StatementEnd