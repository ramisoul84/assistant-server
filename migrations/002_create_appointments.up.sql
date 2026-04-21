CREATE TABLE IF NOT EXISTS appointments (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title      TEXT        NOT NULL,
    datetime   TIMESTAMPTZ NOT NULL,
    notes      TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_appointments_user_id  ON appointments(user_id);
CREATE INDEX IF NOT EXISTS idx_appointments_datetime ON appointments(datetime);
