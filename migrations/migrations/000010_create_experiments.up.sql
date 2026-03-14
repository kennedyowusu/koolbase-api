CREATE TABLE experiments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    environment_id UUID NOT NULL REFERENCES environments (id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    hypothesis     TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'running', 'paused', 'completed')),
    segment_id     UUID REFERENCES segments (id) ON DELETE SET NULL,
    start_at       TIMESTAMPTZ,
    end_at         TIMESTAMPTZ,
    created_by     UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_experiments_environment_id ON experiments (environment_id);

CREATE TABLE experiment_variants (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id UUID NOT NULL REFERENCES experiments (id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    weight        INT NOT NULL CHECK (weight BETWEEN 0 AND 100),
    config        JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_experiment_variants_experiment_id ON experiment_variants (experiment_id);
