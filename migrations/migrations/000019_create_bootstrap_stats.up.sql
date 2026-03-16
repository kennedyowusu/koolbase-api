CREATE TABLE bootstrap_stats (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    date           DATE NOT NULL DEFAULT CURRENT_DATE,
    platform       TEXT NOT NULL,
    request_count  INTEGER NOT NULL DEFAULT 1,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (environment_id, date, platform)
);
CREATE INDEX idx_bootstrap_stats_env_date ON bootstrap_stats(environment_id, date DESC);
