CREATE TABLE snapshots (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id UUID NOT NULL REFERENCES environments (id) ON DELETE CASCADE UNIQUE,
    payload        JSONB NOT NULL,
    version        BIGINT NOT NULL DEFAULT 1,
    built_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_snapshots_environment_id ON snapshots (environment_id);
