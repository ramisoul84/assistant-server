CREATE TABLE IF NOT EXISTS users (
    id          BIGSERIAL PRIMARY KEY,
    telegram_id BIGINT      NOT NULL UNIQUE,
    handle      TEXT        NOT NULL DEFAULT '',
    first_name  TEXT        NOT NULL DEFAULT '',
    timezone TEXT NOT NULL DEFAULT 'UTC',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id);
CREATE INDEX IF NOT EXISTS idx_users_handle      ON users(handle);
