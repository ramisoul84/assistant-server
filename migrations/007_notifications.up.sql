CREATE TABLE IF NOT EXISTS notifications (
    id      BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type    TEXT   NOT NULL,
    ref_id  TEXT   NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, type, ref_id)
);
