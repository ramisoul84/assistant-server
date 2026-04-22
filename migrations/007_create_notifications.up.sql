-- Tracks sent notifications so we never send the same one twice.
CREATE TABLE IF NOT EXISTS notifications (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- type: appointment_24h | appointment_1h | budget_50 | budget_80 | budget_100
    type       TEXT        NOT NULL,
    -- ref_id: appointment id for appointment alerts, year-month for budget alerts (e.g. "2026-04")
    ref_id     TEXT        NOT NULL,
    sent_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, type, ref_id)
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_type    ON notifications(type);
