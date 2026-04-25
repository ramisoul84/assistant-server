CREATE TABLE IF NOT EXISTS expenses (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount      NUMERIC(12,2) NOT NULL,
    currency    TEXT          NOT NULL DEFAULT 'EUR',
    category    TEXT          NOT NULL DEFAULT 'other',
    description TEXT          NOT NULL DEFAULT '',
    happened_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_expenses_user_id     ON expenses(user_id);
CREATE INDEX IF NOT EXISTS idx_expenses_happened_at ON expenses(happened_at);
