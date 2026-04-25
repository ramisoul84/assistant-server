CREATE TABLE IF NOT EXISTS budget_limits (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    amount     NUMERIC(12,2) NOT NULL,
    currency   TEXT          NOT NULL DEFAULT 'EUR',
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
