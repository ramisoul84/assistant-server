CREATE TABLE IF NOT EXISTS gym_exercises (
    id         BIGSERIAL PRIMARY KEY,
    session_id BIGINT        NOT NULL REFERENCES gym_sessions(id) ON DELETE CASCADE,
    name       TEXT          NOT NULL,
    sets       INT           NOT NULL DEFAULT 0,
    reps       INT           NOT NULL DEFAULT 0,
    weight_kg  NUMERIC(6,2)  NOT NULL DEFAULT 0,
    notes      TEXT          NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gym_exercises_session_id ON gym_exercises(session_id);
