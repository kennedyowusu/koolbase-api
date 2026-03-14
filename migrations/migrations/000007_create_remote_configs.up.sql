CREATE TABLE remote_configs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id UUID NOT NULL REFERENCES environments (id) ON DELETE CASCADE,
    key            TEXT NOT NULL,
    value          JSONB NOT NULL,
    description    TEXT NOT NULL DEFAULT '',
    created_by     UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (environment_id, key)
);
CREATE INDEX idx_remote_configs_environment_id ON remote_configs (environment_id);
