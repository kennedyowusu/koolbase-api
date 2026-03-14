CREATE TABLE segments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id UUID NOT NULL REFERENCES environments (id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    description    TEXT NOT NULL DEFAULT '',
    rules          JSONB NOT NULL DEFAULT '[]',
    created_by     UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (environment_id, name)
);
CREATE INDEX idx_segments_environment_id ON segments (environment_id);
CREATE INDEX idx_segments_rules          ON segments USING GIN (rules);
