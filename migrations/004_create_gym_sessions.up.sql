CREATE TABLE IF NOT EXISTS gym_sessions (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notes      TEXT        NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gym_sessions_user_id    ON gym_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_gym_sessions_started_at ON gym_sessions(started_at);
